package handlers

import (
	"encoding/json"
	"net/http"
	"rss-aggregator/internal/auth"
	"rss-aggregator/internal/models"
	"rss-aggregator/internal/storage"
	"strconv"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// ListCategoriesHandler returns all categories for the authenticated user
func ListCategoriesHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		categories, err := storage.GetAllCategoriesByUser(db, claims.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(categories)
	}
}

// CreateCategoryHandler creates a new category for the authenticated user
func CreateCategoryHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var category models.Category
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if category.Name == "" {
			http.Error(w, "Category name is required", http.StatusBadRequest)
			return
		}

		if category.Color == "" {
			category.Color = "#6B8EBA" // Default color
		}

		if err := storage.CreateCategoryForUser(db, &category, claims.UserID); err != nil {
			if err == storage.ErrCategoryNameExists {
				http.Error(w, "A category with this name already exists", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(category)
	}
}

// UpdateCategoryHandler updates an existing category
func UpdateCategoryHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		id, err := strconv.ParseUint(vars["id"], 10, 32)
		if err != nil {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}

		var category models.Category
		if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if category.Name == "" {
			http.Error(w, "Category name is required", http.StatusBadRequest)
			return
		}

		category.ID = uint(id)

		if err := storage.UpdateCategoryForUser(db, &category, claims.UserID); err != nil {
			if err == storage.ErrCategoryNotFound {
				http.Error(w, "Category not found", http.StatusNotFound)
				return
			}
			if err == storage.ErrCategoryNameExists {
				http.Error(w, "A category with this name already exists", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(category)
	}
}

// DeleteCategoryHandler deletes a category
func DeleteCategoryHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		id, err := strconv.ParseUint(vars["id"], 10, 32)
		if err != nil {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}

		if err := storage.DeleteCategoryForUser(db, uint(id), claims.UserID); err != nil {
			if err == storage.ErrCategoryNotFound {
				http.Error(w, "Category not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// AssignFeedToCategoryHandler assigns a feed to a category
func AssignFeedToCategoryHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		feedID, err := strconv.ParseUint(vars["feedId"], 10, 32)
		if err != nil {
			http.Error(w, "Invalid feed ID", http.StatusBadRequest)
			return
		}

		categoryID, err := strconv.ParseUint(vars["categoryId"], 10, 32)
		if err != nil {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}

		if err := storage.AssignFeedToCategory(db, uint(feedID), uint(categoryID), claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// RemoveFeedFromCategoryHandler removes a feed from its category
func RemoveFeedFromCategoryHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetUser(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		feedID, err := strconv.ParseUint(vars["feedId"], 10, 32)
		if err != nil {
			http.Error(w, "Invalid feed ID", http.StatusBadRequest)
			return
		}

		if err := storage.RemoveFeedFromCategory(db, uint(feedID), claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
