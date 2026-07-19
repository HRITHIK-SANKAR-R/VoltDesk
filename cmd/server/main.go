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
		cookie, err := r.Cookie("aes_session")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := auth.DecryptSession(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Fetch full profile from DB
		var user models.User
		err = db.QueryRow(`
			SELECT id, email, role, name, avatar_url, created_at 
			FROM users WHERE id = $1
		`, claims.UserID).Scan(&user.ID, &user.Email, &user.Role, &user.Name, &user.AvatarURL, &user.CreatedAt)

		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		var convID string
		if claims.Role == "customer" {
			conv, err := queries.GetOrCreateOpenConversation(claims.UserID)
			if err == nil {
				convID = conv.ID
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id":         user.ID,
			"role":            user.Role,
			"email":           user.Email,
			"name":            user.Name,
			"avatar_url":      user.AvatarURL,
			"conversation_id": convID,
		})
	})

	// Logout handler
	http.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "aes_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})
		w.WriteHeader(http.StatusOK)
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
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
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
