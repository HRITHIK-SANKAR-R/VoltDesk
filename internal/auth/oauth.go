package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"voltdesk/internal/models"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  "http://localhost:8081/api/auth/google/callback",
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
	Endpoint:     google.Endpoint,
}

var aesSessionKey []byte

type SessionClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

func InitOAuth() {
	googleOauthConfig.ClientID = os.Getenv("GOOGLE_CLIENT_ID")
	googleOauthConfig.ClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	
	keyHex := os.Getenv("AES_SESSION_KEY")
	if keyHex == "" {
		// Fallback for dev - 32 byte key for AES-256
		aesSessionKey = []byte("0123456789abcdef0123456789abcdef") 
	} else {
		key, err := hex.DecodeString(keyHex)
		if err != nil || len(key) != 32 {
			panic("AES_SESSION_KEY must be a valid 32-byte hex string (64 characters)")
		}
		aesSessionKey = key
	}
}

func EncryptSession(claims SessionClaims) (string, error) {
	plaintext, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(aesSessionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return hex.EncodeToString(ciphertext), nil
}

func DecryptSession(encryptedHex string) (*SessionClaims, error) {
	encrypted, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesSessionKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var claims SessionClaims
	if err := json.Unmarshal(plaintext, &claims); err != nil {
		return nil, err
	}

	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("session expired")
	}

	return &claims, nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func CallbackHandler(queries *models.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("state") != "state-token" {
			http.Error(w, "State invalid", http.StatusBadRequest)
			return
		}

		token, err := googleOauthConfig.Exchange(context.Background(), r.FormValue("code"))
		if err != nil {
			http.Error(w, "Code exchange failed", http.StatusInternalServerError)
			return
		}

		response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
		if err != nil {
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}
		defer response.Body.Close()

		var userInfo struct {
			ID      string `json:"id"`
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
		}
		if err := json.NewDecoder(response.Body).Decode(&userInfo); err != nil {
			http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
			return
		}

		// DB Insert/Fetch
		user, err := queries.GetOrCreateUserByGoogleID(userInfo.Email, userInfo.ID, userInfo.Name, userInfo.Picture)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		// Generate AES Session
		expirationTime := time.Now().Add(7 * 24 * time.Hour) // 7 days
		claims := SessionClaims{
			UserID: user.ID,
			Role:   user.Role,
			Exp:    expirationTime.Unix(),
		}

		sessionStr, err := EncryptSession(claims)
		if err != nil {
			http.Error(w, "Failed to encrypt session", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "aes_session",
			Value:    sessionStr,
			Expires:  expirationTime,
			MaxAge:   int(7 * 24 * time.Hour / time.Second),
			HttpOnly: true,
			Secure:   false, // Must be false for local HTTP
			Path:     "/",
			SameSite: http.SameSiteLaxMode, // Lax allows cross-port local dev
		})
		
		// For the customer, ensure they have an open conversation ready
		if user.Role == "customer" {
			_, err = queries.GetOrCreateOpenConversation(user.ID)
			if err != nil {
				http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
				return
			}
		}

		// Redirect to frontend
		http.Redirect(w, r, "http://localhost:5173", http.StatusTemporaryRedirect)
	}
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("aes_session")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := DecryptSession(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-User-Role", claims.Role)
		next.ServeHTTP(w, r)
	})
}
