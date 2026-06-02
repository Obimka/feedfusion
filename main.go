package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"rss-aggregator/internal/auth"
	"rss-aggregator/internal/config"
	"rss-aggregator/internal/handlers"
	"rss-aggregator/internal/tts"
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

	// Run migration for multi-user support
	migrateToMultiUser(db)

	// Initialize JWT config from environment or defaults
	jwtConfig := auth.DefaultJWTConfig()
	if secretKey := os.Getenv("JWT_SECRET"); secretKey != "" {
		jwtConfig.SecretKey = secretKey
	}

	// Create first admin user if none exists
	createFirstAdmin(db)

	// Load config and add preconfigured feeds
	loadConfigFeeds(db)

	// Configure registration
	configureRegistration()

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

	// Initialize TTS handler if API key is provided (from config or env)
	mistralAPIKey := ""
	cfg, err := config.LoadConfig("data/config.yaml")
	if err != nil {
	    log.Printf("Warning: failed to load config: %v", err)
	} else {
	    log.Printf("Config loaded, MistralAPIKey present: %v", cfg.MistralAPIKey != "")
	}
	if err == nil && cfg != nil && cfg.MistralAPIKey != "" {
	    mistralAPIKey = cfg.MistralAPIKey
	    log.Printf("TTS: Using API key from config file")
	} else if apiKey := os.Getenv("MISTRAL_API_KEY"); apiKey != "" {
	    mistralAPIKey = apiKey
	    log.Printf("TTS: Using API key from environment")
	}
	if mistralAPIKey != "" {
		ttsHandler := tts.NewTTSHandler(mistralAPIKey)
		ttsHandler.RegisterRoutes(r)
		log.Println("TTS enabled - Mistral API available")
	}

	// Register main handlers
	handlers.RegisterRoutes(r, db)

	// Serve web
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", newStaticHandler("web/static")))

	// Apply auth middleware
	r.Use(auth.AuthMiddleware(jwtConfig))

	// Determine server port: config > env > default
	port := "8081"
	if cfg, err := config.LoadConfig("data/config.yaml"); err == nil && cfg.ServerPort > 0 {
		port = fmt.Sprintf("%d", cfg.ServerPort)
	} else if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("Server started on :%s", port)
	log.Println("WebSub support enabled - feeds will receive push notifications when available")
	log.Println("Authentication enabled - use /api/login endpoint")
	log.Fatal(http.ListenAndServe(":"+port, r))
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

	// Get admin user ID to assign feeds to
	var adminUser models.User
	if err := db.Where("is_admin = ?", true).Order("id ASC").First(&adminUser).Error; err != nil {
		log.Printf("Warning: No admin user found, cannot assign preconfigured feeds: %v", err)
		// Try to find any user as fallback
		if err := db.Order("id ASC").First(&adminUser).Error; err != nil {
			log.Printf("Warning: No users found at all, cannot assign preconfigured feeds")
			return
		}
		log.Printf("Warning: Using first user (ID: %d, admin: %v) for preconfigured feeds", adminUser.ID, adminUser.IsAdmin)
	}

	for _, feedCfg := range cfg.Feeds {
		// Check if feed already exists for this admin
		var existingFeed models.Feed
		err := db.Where("url = ? AND user_id = ?", feedCfg.URL, adminUser.ID).First(&existingFeed).Error
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
			// Create new feed for admin
			feed := &models.Feed{
				URL:      feedCfg.URL,
				Title:    feedCfg.Title,
				Include:  feedCfg.Include,
				IsActive: true,
				UserID:   adminUser.ID,
			}
			// Parse and add items
			_, items, err := parser.ParseFeed(feed.URL)
			if err != nil {
				log.Printf("Warning: could not parse feed %s: %v", feed.URL, err)
				// Still save the feed even if we can't fetch it now
				db.Create(feed)
			} else {
				storage.AddFeedForUser(db, feed, items, adminUser.ID)
			}
			log.Printf("Added new feed: %s", feedCfg.URL)
		}
	}
}

// migrateToMultiUser handles database migration for multi-user support
func migrateToMultiUser(db *gorm.DB) {
	// Check if user_id column exists in feeds table
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('feeds') WHERE name = 'user_id'").Count(&count)
	
	if count == 0 {
		// Add user_id column with default value 1
		// Using default value allows adding to existing table
		if err := db.Exec("ALTER TABLE feeds ADD COLUMN user_id INTEGER DEFAULT 1").Error; err != nil {
			log.Printf("Error adding user_id column: %v", err)
			return
		}
		
		// Update existing rows to have user_id = 1 (will be assigned to admin)
		if err := db.Exec("UPDATE feeds SET user_id = 1 WHERE user_id IS NULL").Error; err != nil {
			log.Printf("Error updating existing feeds: %v", err)
			return
		}
		
		log.Println("Successfully added user_id column to feeds table")
	}
	
	// Create index on user_id for better performance
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id)").Error; err != nil {
		log.Printf("Error creating index: %v", err)
	}
	
	// Auto-migrate User table
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Printf("Warning: User table migration failed: %v", err)
	}
	
	// Assign existing feeds (with user_id=1) to admin - will be done in createFirstAdmin
}

// createFirstAdmin creates a default admin user if no users exist
// Returns the admin user's ID
func createFirstAdmin(db *gorm.DB) uint {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		// Admin already exists, return first admin user's ID
		var admin models.User
		if err := db.Where("is_admin = ?", true).First(&admin).Error; err == nil {
			return admin.ID
		}
		// If no admin but users exist, return 0
		return 0
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
		return 0
	}

	admin := models.User{
		Username: adminUsername,
		Password: hashedPassword,
		IsAdmin:  true,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Printf("Error creating admin user: %v", err)
		return 0
	}

	log.Printf("Created first admin user: %s", adminUsername)
	
	// Assign existing feeds to this admin
	if err := db.Model(&models.Feed{}).
		Where("user_id IS NULL OR user_id = 0 OR user_id = 1").
		Update("user_id", admin.ID).Error; err != nil {
		log.Printf("Warning: Could not assign existing feeds to admin: %v", err)
	}
	
	return admin.ID
}

// configureRegistration sets up the registration flag based on environment/config
func configureRegistration() {
	// Check environment variable first
	allowReg := os.Getenv("ALLOW_REGISTRATION")
	if allowReg == "true" {
		auth.SetAllowRegistration(true)
		log.Println("Public registration is enabled (via ALLOW_REGISTRATION=true)")
		return
	}

	// Check config file
	cfg, err := config.LoadConfig("data/config.yaml")
	if err == nil && cfg.AllowRegistration {
		auth.SetAllowRegistration(true)
		log.Println("Public registration is enabled (via config file)")
		return
	}

	// Default: disabled
	auth.SetAllowRegistration(false)
	log.Println("Public registration is disabled (admin must create users)")
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
