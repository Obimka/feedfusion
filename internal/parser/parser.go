package parser

import (
	"rss-aggregator/internal/models"
	"time"

	"github.com/mmcdole/gofeed"
)

func ParseFeed(url string) (*models.Feed, []models.Item, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, nil, err
	}

	resultFeed := &models.Feed{
		URL:       url,
		Title:     feed.Title,
		Link:      feed.Link,
		Type:      feed.FeedType,
		LastFetch: time.Now(),
	}

	var items []models.Item
	for _, item := range feed.Items {
		published := time.Now()
		if item.PublishedParsed != nil {
			published = *item.PublishedParsed
		}

		// Extract image URL from various sources
		imageURL := ""
		if item.Image != nil && item.Image.URL != "" {
			imageURL = item.Image.URL
		}
		// Also check for Content encoded images or other fields
		if imageURL == "" && len(item.Enclosures) > 0 {
			// Use first enclosure as image
			imageURL = item.Enclosures[0].URL
		}

		items = append(items, models.Item{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     item.Content,
			ImageURL:    imageURL,
			PublishedAt: published,
			Guid:        item.GUID,
		})
	}

	return resultFeed, items, nil
}
