package handlers

import (
	"net/http"
	"rss-aggregator/internal/logger"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func registerAuthRoutes(r *mux.Router, db *gorm.DB) {
	// Login page - public access
	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("Accessing login page (Method: %s)", r.Method)
		// If already logged in, redirect to home
		if _, err := r.Cookie("token"); err == nil {
			logger.Debugf("User already logged in, redirecting to home")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		Tmpl.ExecuteTemplate(w, "login.html", nil)
	})

	// Logout page - public access
	r.HandleFunc("/logged-out", func(w http.ResponseWriter, r *http.Request) {
		logger.Debugf("Accessing logged-out page")
		Tmpl.ExecuteTemplate(w, "logout.html", nil)
	})

	// Logout endpoint - protected
	r.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "token",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/logged-out", http.StatusSeeOther)
	})
}
