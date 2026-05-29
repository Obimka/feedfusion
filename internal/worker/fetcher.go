package worker

import (
	"log"
	"time"

	"rss-aggregator/internal/models"
	"rss-aggregator/internal/parser"
	"rss-aggregator/internal/storage"

	"gorm.io/gorm"
)

func StartFeedFetcher(db *gorm.DB, minutes int) {
	ticker := time.NewTicker(time.Duration(minutes) * time.Minute)
	defer ticker.Stop()

	// Fetch immediately on start
	fetchAllFeeds(db)

	for {
		<-ticker.C
		fetchAllFeeds(db)
	}
}

func fetchAllFeeds(db *gorm.DB) {
	var feeds []models.Feed
	db.Find(&feeds)

	for _, feed := range feeds {
		if !feed.IsActive {
			continue
		}

		parsedFeed, items, err := parser.ParseFeed(feed.URL)
		if err != nil {
			log.Printf("Error fetching %s: %v", feed.URL, err)
			feed.Error = err.Error()
			db.Save(&feed)
			continue
		}

		feed.Error = ""
		feed.LastFetch = time.Now()
		db.Save(&feed)
		storage.AddFeed(db, parsedFeed, items)
		log.Printf("Fetched %d items from %s", len(items), feed.URL)
	}
}
