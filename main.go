package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"rss-aggregator/internal/auth"
	"rss-aggregator/internal/config"
	"rss-aggregator/internal/handlers"
	"rss-aggregator/internal/models"
	"rss-aggregator/internal/parser"
	"rss-aggregator/internal/storage"
	"rss-aggregator/internal/websub"
	"rss-aggregator/internal/worker"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func main() {
	// Init storage
	db, err := storage.InitDB("rss.db")
	if err != nil {
		log.Fatal(err)
	}

	// Auto-migrate with new WebSub fields and auth tables
	err = db.AutoMigrate(&models.Feed{}, &models.Item{}, &auth.User{})
	if err != nil {
		log.Printf("Warning: auto-migration failed: %v", err)
	}

	// Initialize JWT config from environment or defaults
	jwtConfig := auth.DefaultJWTConfig()
	if secretKey := os.Getenv("JWT_SECRET"); secretKey != "" {
		jwtConfig.SecretKey = secretKey
	}

	// Create first admin user if none exists
	createFirstAdmin(db)

	// Load config and add preconfigured feeds
	loadConfigFeeds(db)

	// Start WebSub manager
	callbackBaseURL := "http://localhost:8080"
	go websub.StartWebSubManager(db, callbackBaseURL)

	// Start worker (fetch feeds every 30min)
	go worker.StartFeedFetcher(db, 30)

	// Setup routes
	r := mux.NewRouter()

	// Initialize auth handler and register routes
	authHandler := auth.NewAuthHandler(db, jwtConfig)
	authHandler.RegisterRoutes(r)

	// Register main handlers
	handlers.RegisterRoutes(r, db)

	// Serve web
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", newStaticHandler("web/static")))

	// Apply auth middleware
	r.Use(auth.AuthMiddleware(jwtConfig))

	http.Handle("/", r)

	log.Println("Server started on :8081")
	log.Println("WebSub support enabled - feeds will receive push notifications when available")
	log.Println("Authentication enabled - use /api/login endpoint")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func loadConfigFeeds(db *gorm.DB) {
	cfg, err := config.LoadConfig("data/config.yaml")
	if err != nil {
		log.Printf("Warning: could not load config: %v", err)
		return
	}

	if len(cfg.Feeds) == 0 {
		return
	}

	log.Printf("Loading %d preconfigured feeds from data/config.yaml", len(cfg.Feeds))

	for _, feedCfg := range cfg.Feeds {
		// Check if feed already exists
		var existingFeed models.Feed
		err := db.Where("url = ?", feedCfg.URL).First(&existingFeed).Error
		if err == nil {
			// Feed exists, just update if needed
			if feedCfg.Title != "" && feedCfg.Title != existingFeed.Title {
				existingFeed.Title = feedCfg.Title
			}
			if existingFeed.Include != feedCfg.Include {
				existingFeed.Include = feedCfg.Include
			}
			db.Save(&existingFeed)
			log.Printf("Updated existing feed: %s", feedCfg.URL)
		} else {
			// Create new feed
			feed := &models.Feed{
				URL:      feedCfg.URL,
				Title:    feedCfg.Title,
				Include:  feedCfg.Include,
				IsActive: true,
			}
			// Parse and add items
			_, items, err := parser.ParseFeed(feed.URL)
			if err != nil {
				log.Printf("Warning: could not parse feed %s: %v", feed.URL, err)
				// Still save the feed even if we can't fetch it now
				db.Create(feed)
			} else {
				storage.AddFeed(db, feed, items)
			}
			log.Printf("Added new feed: %s", feedCfg.URL)
		}
	}
}

// createFirstAdmin creates a default admin user if no users exist
func createFirstAdmin(db *gorm.DB) {
	var count int64
	db.Model(&auth.User{}).Count(&count)
	if count > 0 {
		return
	}

	// Create default admin user
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if adminUsername == "" || adminPassword == "" {
		// Generate a random password if not set
		adminUsername = "admin"
		adminPassword = "admin123"
		log.Printf("WARNING: No ADMIN_USERNAME/ADMIN_PASSWORD set, created default admin/admin123")
		log.Printf("PLEASE CHANGE THIS PASSWORD IMMEDIATELY!")
	}

	hashedPassword, err := auth.HashPassword(adminPassword)
	if err != nil {
		log.Printf("Error creating admin user: %v", err)
		return
	}

	admin := auth.User{
		Username: adminUsername,
		Password: hashedPassword,
		IsAdmin:  true,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Printf("Error creating admin user: %v", err)
	}

	log.Printf("Created first admin user: %s", adminUsername)
}

// newStaticHandler creates a file server that sets proper Content-Type headers
func newStaticHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set Content-Type based on file extension
		path := r.URL.Path
		if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		} else if strings.HasSuffix(path, ".png") {
			w.Header().Set("Content-Type", "image/png")
		} else if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
			w.Header().Set("Content-Type", "image/jpeg")
		} else if strings.HasSuffix(path, ".gif") {
			w.Header().Set("Content-Type", "image/gif")
		} else if strings.HasSuffix(path, ".svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		} else if strings.HasSuffix(path, ".json") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		fs.ServeHTTP(w, r)
	})
}
