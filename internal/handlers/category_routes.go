package handlers

import (
	"net/http"
	"rss-aggregator/internal/auth"
	"rss-aggregator/internal/models"
	"rss-aggregator/internal/storage"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func registerCategoryRoutes(r *mux.Router, db *gorm.DB) {
	// Category management page
	r.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		categories, _ := storage.GetAllCategoriesByUser(db, claims.UserID)
		feeds, _ := storage.GetAllFeedsByUser(db, claims.UserID)

		Tmpl.ExecuteTemplate(w, "categories.html", struct {
			Categories []models.Category
			Feeds      []models.Feed
		}{
			Categories: categories,
			Feeds:      feeds,
		})
	})
}
