package auth

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"rss-aggregator/internal/models"
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

// AllowRegistration is a global flag for public registration (can be set via config)
var AllowRegistration = false

// SetAllowRegistration sets the global registration flag
func SetAllowRegistration(allow bool) {
	AllowRegistration = allow
}

// RegisterHandler handles new user registration (public if enabled)
func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if !AllowRegistration {
		http.Error(w, "Registration is disabled", http.StatusForbidden)
		return
	}

	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if creds.Username == "" || creds.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	var existingUser User
	err := h.db.Where("username = ?", creds.Username).First(&existingUser).Error
	if err == nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := HashPassword(creds.Password)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Create user (non-admin by default)
	user := User{
		Username: creds.Username,
		Password: hashedPassword,
		IsAdmin:  false,
	}

	if err := h.db.Create(&user).Error; err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
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

	// Set token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(h.config.Expiration),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	response := map[string]interface{}{
		"token":         token,
		"refresh_token": refreshToken,
		"username":      user.Username,
		"is_admin":      strconvFormatBool(user.IsAdmin),
		"message":       "User created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListUsersHandler returns all users (admin only)
func (h *AuthHandler) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	claims, ok := GetUser(r.Context())
	if !ok {
		http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	if !claims.IsAdmin {
		http.Error(w, ErrUnauthorized.Error(), http.StatusForbidden)
		return
	}

	var users []User
	if err := h.db.Find(&users).Error; err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	// Sanitize: remove passwords
	sanitizedUsers := make([]map[string]interface{}, len(users))
	for i, u := range users {
		sanitizedUsers[i] = map[string]interface{}{
			"id":       u.ID,
			"username": u.Username,
			"is_admin": u.IsAdmin,
			"created_at": u.CreatedAt,
			"updated_at": u.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizedUsers)
}

// GetUserHandler returns a specific user (admin or self)
func (h *AuthHandler) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconvParseUint(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if user is admin or requesting their own info
	claims, ok := GetUser(r.Context())
	if !ok {
		http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	if claims.UserID != uint(userID) && !claims.IsAdmin {
		http.Error(w, ErrUnauthorized.Error(), http.StatusForbidden)
		return
	}

	var user User
	if err := h.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error fetching user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateUserHandler creates a new user (admin only)
func (h *AuthHandler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	claims, ok := GetUser(r.Context())
	if !ok {
		http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	if !claims.IsAdmin {
		http.Error(w, ErrUnauthorized.Error(), http.StatusForbidden)
		return
	}

	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if request.Username == "" || request.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	var existingUser User
	err := h.db.Where("username = ?", request.Username).First(&existingUser).Error
	if err == nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := HashPassword(request.Password)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	user := User{
		Username: request.Username,
		Password: hashedPassword,
		IsAdmin:  request.IsAdmin,
	}

	if err := h.db.Create(&user).Error; err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin,
		"message":  "User created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateUserHandler updates a user (admin or self, limited fields)
func (h *AuthHandler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconvParseUint(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if user is admin or updating their own info
	claims, ok := GetUser(r.Context())
	if !ok {
		http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	if claims.UserID != uint(userID) && !claims.IsAdmin {
		http.Error(w, ErrUnauthorized.Error(), http.StatusForbidden)
		return
	}

	var request models.UserUpdate
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Only admin can change is_admin field
	if request.IsAdmin != nil && !claims.IsAdmin {
		http.Error(w, "Only admin can change admin status", http.StatusForbidden)
		return
	}

	var user User
	if err := h.db.First(&user, userID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update fields
	if request.Username != "" {
		// Check if new username already exists
		var existingUser User
		if err := h.db.Where("username = ? AND id != ?", request.Username, userID).First(&existingUser).Error; err == nil {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}
		user.Username = request.Username
	}

	if request.IsAdmin != nil {
		user.IsAdmin = *request.IsAdmin
	}

	if err := h.db.Save(&user).Error; err != nil {
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin,
		"message":  "User updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteUserHandler deletes a user (admin only)
func (h *AuthHandler) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconvParseUint(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if user is admin
	claims, ok := GetUser(r.Context())
	if !ok {
		http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	if !claims.IsAdmin {
		http.Error(w, ErrUnauthorized.Error(), http.StatusForbidden)
		return
	}

	// Prevent admin from deleting themselves
	if claims.UserID == uint(userID) {
		http.Error(w, "Cannot delete your own account", http.StatusForbidden)
		return
	}

	// Delete user's feeds and items first
	if err := h.db.Where("user_id = ?", userID).Delete(&models.Feed{}).Error; err != nil {
		http.Error(w, "Error deleting user feeds", http.StatusInternalServerError)
		return
	}

	// Delete user
	if err := h.db.Delete(&User{}, userID).Error; err != nil {
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
}

// ChangePasswordHandler changes a user's password (admin or self)
func (h *AuthHandler) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconvParseUint(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if user is admin or changing their own password
	claims, ok := GetUser(r.Context())
	if !ok {
		http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	if claims.UserID != uint(userID) && !claims.IsAdmin {
		http.Error(w, ErrUnauthorized.Error(), http.StatusForbidden)
		return
	}

	var request models.PasswordChange
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if request.NewPassword == "" {
		http.Error(w, "New password is required", http.StatusBadRequest)
		return
	}

	// For self-change, verify current password
	if claims.UserID == uint(userID) {
		var user User
		if err := h.db.First(&user, userID).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if !CheckPassword(request.CurrentPassword, user.Password) {
			http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
			return
		}
	}

	// Hash new password
	hashedPassword, err := HashPassword(request.NewPassword)
	if err != nil {
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}

	if err := h.db.Model(&User{}).Where("id = ?", userID).Update("password", hashedPassword).Error; err != nil {
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password changed successfully"})
}

// Helper function to parse uint from string
func strconvParseUint(s string) (uint, error) {
	u64, err := strconv.ParseUint(s, 10, 32)
	return uint(u64), err
}

// RegisterRoutes registers auth routes on the router
func (h *AuthHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/login", h.LoginHandler).Methods("POST")
	router.HandleFunc("/api/refresh", h.RefreshHandler).Methods("POST")
	router.HandleFunc("/api/logout", h.LogoutHandler).Methods("POST")
	router.HandleFunc("/api/register", h.RegisterHandler).Methods("POST")
	router.HandleFunc("/api/users", h.ListUsersHandler).Methods("GET")
	router.HandleFunc("/api/users", h.CreateUserHandler).Methods("POST")
	router.HandleFunc("/api/users/{id}", h.GetUserHandler).Methods("GET")
	router.HandleFunc("/api/users/{id}", h.UpdateUserHandler).Methods("PUT")
	router.HandleFunc("/api/users/{id}", h.DeleteUserHandler).Methods("DELETE")
	router.HandleFunc("/api/users/{id}/password", h.ChangePasswordHandler).Methods("POST")
}
