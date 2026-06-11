package handlers

import (
	"net/http"
	"rss-aggregator/internal/auth"
	"rss-aggregator/internal/logger"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func registerAPIRoutes(r *mux.Router, db *gorm.DB, jwtConfig auth.JWTConfig) {
	// Category API routes
	r.HandleFunc("/api/categories", ListCategoriesHandler(db)).Methods("GET")
	r.HandleFunc("/api/categories", CreateCategoryHandler(db)).Methods("POST")
	r.HandleFunc("/api/categories/{id}", UpdateCategoryHandler(db)).Methods("PUT")
	r.HandleFunc("/api/categories/{id}", DeleteCategoryHandler(db)).Methods("DELETE")
	r.HandleFunc("/api/feed/{feedId}/category/{categoryId}", AssignFeedToCategoryHandler(db)).Methods("POST")
	r.HandleFunc("/api/feed/{feedId}/category", RemoveFeedFromCategoryHandler(db)).Methods("DELETE")

	// Settings page - requires authentication
	r.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("Accessing settings page")
		// Check if user is authenticated via context (from auth middleware)
		if _, ok := auth.GetUser(r.Context()); !ok {
			// Fallback: check cookie for backward compatibility
			if _, err := r.Cookie("token"); err != nil {
				logger.Debugf("User not authenticated, redirecting to login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
		}
		Tmpl.ExecuteTemplate(w, "settings.html", nil)
	})

	// Users management page - admin only
	r.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("Accessing users management page")
		// Check if user is authenticated and admin
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			logger.Debugf("User not authenticated, redirecting to login")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Check if admin
		if !claims.IsAdmin {
			logger.Debugf("User %d is not admin, access denied", claims.UserID)
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		Tmpl.ExecuteTemplate(w, "users.html", nil)
	})
}
