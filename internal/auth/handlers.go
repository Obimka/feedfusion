package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// AuthHandler holds dependencies for auth handlers
type AuthHandler struct {
	db     *gorm.DB
	config JWTConfig
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(db *gorm.DB, config JWTConfig) *AuthHandler {
	return &AuthHandler{
		db:     db,
		config: config,
	}
}

// LoginHandler handles user login
func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if creds.Username == "" || creds.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Find user
	var user User
	err := h.db.Where("username = ?", creds.Username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check password
	if !CheckPassword(creds.Password, user.Password) {
		http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := GenerateJWT(&user, h.config)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Generate refresh token
	refreshToken, err := GenerateRefreshToken(&user, h.config)
	if err != nil {
		http.Error(w, "Error generating refresh token", http.StatusInternalServerError)
		return
	}

	// Set token cookie (optional, for web interface)
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(h.config.Expiration),
		HttpOnly: true,
		Secure:   true, // Enable in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	// Return tokens
	response := map[string]string{
		"token":         token,
		"refresh_token": refreshToken,
		"username":      user.Username,
		"is_admin":      strconvFormatBool(user.IsAdmin),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RefreshHandler handles token refresh
func (h *AuthHandler) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate refresh token
	claims, err := ValidateJWT(request.RefreshToken, h.config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Find user
	var user User
	err = h.db.First(&user, claims.UserID).Error
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Generate new JWT token
	token, err := GenerateJWT(&user, h.config)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Return new token
	response := map[string]string{
		"token":    token,
		"username": claims.Username,
		"is_admin": strconvFormatBool(claims.IsAdmin),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LogoutHandler handles user logout (clears cookie)
func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	// For web requests (form submission), redirect to logout page
	// For API requests with Authorization header, return JSON
	if r.Header.Get("Authorization") == "" && r.Header.Get("Content-Type") != "application/json" {
		http.Redirect(w, r, "/logged-out", http.StatusSeeOther)
		return
	}

	// For API requests, return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

// strconvFormatBool is a helper to format bool as "true"/"false"
func strconvFormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// RegisterRoutes registers auth routes on the router
func (h *AuthHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/login", h.LoginHandler).Methods("POST")
	router.HandleFunc("/api/refresh", h.RefreshHandler).Methods("POST")
	router.HandleFunc("/api/logout", h.LogoutHandler).Methods("POST")
}
