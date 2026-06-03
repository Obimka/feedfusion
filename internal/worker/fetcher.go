package worker

import (
	"time"

	"rss-aggregator/internal/logger"
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

	for i := range feeds {
		feed := &feeds[i]
		if !feed.IsActive {
			continue
		}

		parsedFeed, items, err := parser.ParseFeed(feed.URL)
		if err != nil {
			logger.Errorf("Error fetching %s: %v", feed.URL, err)
			feed.Error = err.Error()
			db.Save(&feed)
			continue
		}

		// Update WebSub info from parsed feed
		if parsedFeed.HubURL != "" {
			feed.HubURL = parsedFeed.HubURL
			feed.TopicURL = parsedFeed.TopicURL
			if feed.Secret == "" {
				feed.Secret = parsedFeed.Secret
			}
		}

		feed.Error = ""
		feed.LastFetch = time.Now()
		db.Save(&feed)
		storage.AddFeed(db, parsedFeed, items)
		logger.Infof("Fetched %d items from %s (Hub: %v)", len(items), feed.URL, parsedFeed.HubURL != "")
	}
}
