package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"voltdesk/internal/ai"
	"voltdesk/internal/database"
	"voltdesk/internal/models"
	"voltdesk/internal/websocket"
	"voltdesk/internal/worker"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // Load .env file if it exists

	// Init DB
	db := database.Connect()
	defer db.Close()
	queries := models.NewQueries(db)

	// Init AI
	ai.InitGemini()

	// Init WebSocket Hub
	hub := websocket.NewHub(queries)
	go hub.Run()

	// Start Idle Worker
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			worker.CheckIdleConversations(queries)
		}
	}()

	// Routes
	http.HandleFunc("/api/auth/customer", func(w http.ResponseWriter, r *http.Request) {
		// Mock simple auth
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "email is required", http.StatusBadRequest)
			return
		}
		user, err := queries.GetOrCreateCustomer(email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		conv, err := queries.GetOrCreateOpenConversation(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id": user.ID,
			"conversation_id": conv.ID,
		})
	})

	http.HandleFunc("/api/conversations/", func(w http.ResponseWriter, r *http.Request) {
		// e.g. /api/conversations/{id}/messages
		// For simplicity, grab the path manually
		path := r.URL.Path
		if len(path) > len("/api/conversations/") {
			idAndRest := path[len("/api/conversations/"):]
			// Split and find id
			// In production, we use proper routing
			if len(idAndRest) > 36 { // length of uuid
				id := idAndRest[:36]
				messages, err := queries.GetMessages(id, 50)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(messages)
				return
			}
		}
		
		// If just /api/conversations return open ones (for agent)
		convs, err := queries.GetOpenConversations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(convs)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		role := r.URL.Query().Get("role")
		convID := r.URL.Query().Get("conversation_id")
		
		if userID == "" || role == "" {
			http.Error(w, "Missing user_id or role", http.StatusBadRequest)
			return
		}
		websocket.ServeWs(hub, w, r, userID, role, convID)
	})

	// CORS Middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins for dev
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(http.DefaultServeMux)); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
