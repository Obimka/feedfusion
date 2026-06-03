package handlers

import (
	"net/http"
	"rss-aggregator/internal/auth"
	"rss-aggregator/internal/storage"
	"rss-aggregator/internal/websub"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"strconv"
)

func registerWebSubRoutes(r *mux.Router, db *gorm.DB) {
	// WebSub callback endpoint
	r.HandleFunc("/websub/callback", websub.HandleCallback(db)).Methods("GET", "POST")

	// WebSub subscription management
	r.HandleFunc("/websub/subscribe/{id}", func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		feed, err := storage.GetFeedByIDAndUser(db, uint(id), claims.UserID)
		if err != nil || feed == nil {
			http.Error(w, "Feed not found or access denied", http.StatusNotFound)
			return
		}

		// Get the callback base URL from the host
		callbackBaseURL := "http://" + r.Host
		if r.TLS != nil {
			callbackBaseURL = "https://" + r.Host
		}

		if err := websub.SubscribeFeedToWebSub(db, feed, callbackBaseURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/feeds", http.StatusSeeOther)
	})

	r.HandleFunc("/websub/unsubscribe/{id}", func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)
		feed, err := storage.GetFeedByIDAndUser(db, uint(id), claims.UserID)
		if err != nil || feed == nil {
			http.Error(w, "Feed not found or access denied", http.StatusNotFound)
			return
		}

		callbackBaseURL := "http://" + r.Host
		if r.TLS != nil {
			callbackBaseURL = "https://" + r.Host
		}

		if err := websub.UnsubscribeFeedFromWebSub(db, feed, callbackBaseURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/feeds", http.StatusSeeOther)
	})
}
