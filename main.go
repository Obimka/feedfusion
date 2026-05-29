package main

import (
	"log"
	"net/http"
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

	// Auto-migrate with new WebSub fields
	err = db.AutoMigrate(&models.Feed{}, &models.Item{})
	if err != nil {
		log.Printf("Warning: auto-migration failed: %v", err)
	}

	// Load config and add preconfigured feeds
	loadConfigFeeds(db)

	// Start WebSub manager
	callbackBaseURL := "http://localhost:8080"
	go websub.StartWebSubManager(db, callbackBaseURL)

	// Start worker (fetch feeds every 30min)
	go worker.StartFeedFetcher(db, 30)

	// Setup routes
	r := mux.NewRouter()
	handlers.RegisterRoutes(r, db)

	// Serve web
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	http.Handle("/", r)

	log.Println("Server started on :8080")
	log.Println("WebSub support enabled - feeds will receive push notifications when available")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadConfigFeeds(db *gorm.DB) {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Printf("Warning: could not load config: %v", err)
		return
	}

	if len(cfg.Feeds) == 0 {
		return
	}

	log.Printf("Loading %d preconfigured feeds from config.yaml", len(cfg.Feeds))

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
