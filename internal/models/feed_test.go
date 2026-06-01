package models

import (
	"testing"
	"time"
)

func TestFeed_Model(t *testing.T) {
	// Test basic feed creation
	feed := Feed{
		URL:       "https://example.com/feed.xml",
		Title:     "Example Feed",
		Link:      "https://example.com",
		Type:      "rss",
		LastFetch: time.Now(),
		HubURL:    "https://hub.example.com",
		TopicURL:  "https://example.com/feed.xml",
		Secret:    "test-secret",
	}

	if feed.URL != "https://example.com/feed.xml" {
		t.Errorf("Expected URL to be https://example.com/feed.xml, got: %s", feed.URL)
	}
	if feed.Title != "Example Feed" {
		t.Errorf("Expected Title to be Example Feed, got: %s", feed.Title)
	}
	if feed.Type != "rss" {
		t.Errorf("Expected Type to be rss, got: %s", feed.Type)
	}
}

func TestItem_Model(t *testing.T) {
	now := time.Now()
	item := Item{
		Title:       "Test Article",
		Link:        "https://example.com/article",
		Description: "This is a test description",
		Content:     "Full content here",
		ImageURL:    "https://example.com/image.jpg",
		PublishedAt: now,
		Guid:        "unique-guid-123",
	}

	if item.Title != "Test Article" {
		t.Errorf("Expected Title to be Test Article, got: %s", item.Title)
	}
	if item.Guid != "unique-guid-123" {
		t.Errorf("Expected Guid to be unique-guid-123, got: %s", item.Guid)
	}
	if item.PublishedAt != now {
		t.Errorf("Expected PublishedAt to be %v, got: %v", now, item.PublishedAt)
	}
}
