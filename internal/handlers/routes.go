package handlers

import (
	"rss-aggregator/internal/auth"
	"rss-aggregator/internal/logger"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// RegisterRoutes registers all the routes for the application
func RegisterRoutes(r *mux.Router, db *gorm.DB, jwtConfig auth.JWTConfig) {
	logger.Infof("Registering routes...")
	
	// Auth routes
	registerAuthRoutes(r, db)

	// Feed routes
	registerFeedRoutes(r, db)

	// Item routes
	registerItemRoutes(r, db)

	// WebSub routes
	registerWebSubRoutes(r, db)

	// Category routes
	registerCategoryRoutes(r, db)

	// API routes
	registerAPIRoutes(r, db, jwtConfig)

	// SSE endpoint for real-time notifications
	r.HandleFunc("/api/events", SSEHandler(db, jwtConfig)).Methods("GET")
}