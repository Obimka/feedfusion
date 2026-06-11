package handlers

import (
	"net/http"
	"net/url"
	"rss-aggregator/internal/auth"
	"rss-aggregator/internal/storage"
	"strconv"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func registerItemRoutes(r *mux.Router, db *gorm.DB) {
	r.HandleFunc("/item/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		if err := storage.MarkItemAsReadForUser(db, uint(id), claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		// Preserve query params and redirect back
		query := r.URL.Query()
		http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
	})

	r.HandleFunc("/item/{id}/read-and-redirect", func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		if err := storage.MarkItemAsReadForUser(db, uint(id), claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		// Get the target URL from query param
		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Decode the URL (it was encoded when passed as param)
		decodedURL, err := url.QueryUnescape(targetURL)
		if err != nil {
			decodedURL = targetURL
		}

		http.Redirect(w, r, decodedURL, http.StatusSeeOther)
	})

	// Mark as unread
	r.HandleFunc("/item/{id}/unread", func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		if err := storage.MarkItemAsUnreadForUser(db, uint(id), claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		// Preserve query params
		query := r.URL.Query()
		http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
	})

	// Toggle show read
	r.HandleFunc("/toggle-read", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		// The button already sends the toggled value, just use it
		http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
	})
}
