package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"voltdesk/internal/ai"
	"voltdesk/internal/auth"
	"voltdesk/internal/database"

	"github.com/golang-jwt/jwt/v5"
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

	// Start Idle & Archiver Worker
	worker.InitArchiver()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			worker.CheckIdleConversations(queries)
			worker.ArchiveOldConversations(queries)
		}
	}()

	// Init Auth
	auth.InitOAuth()

	// Routes
	http.HandleFunc("/api/auth/google/login", auth.LoginHandler)
	http.HandleFunc("/api/auth/google/callback", auth.CallbackHandler(queries))

	// Get current user session info
	http.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		claims := &auth.Claims{}
		tkn, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			secret := os.Getenv("JWT_SECRET")
			if secret == "" {
				secret = "super-secret-default-key-for-dev"
			}
			return []byte(secret), nil
		})

		if err != nil || !tkn.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var convID string
		if claims.Role == "customer" {
			// In production, queries should be accessible or use a global db instance.
			// Luckily queries is captured in the scope from main()
			conv, err := queries.GetOrCreateOpenConversation(claims.UserID)
			if err == nil {
				convID = conv.ID
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id":         claims.UserID,
			"role":            claims.Role,
			"conversation_id": convID,
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
