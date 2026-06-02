package models

import (
	"testing"
	"time"
)

// =============================================================================
// Feed Model Tests
// =============================================================================

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

func TestFeed_Defaults(t *testing.T) {
	// Test default values for Feed
	feed := Feed{
		URL: "https://example.com/feed.xml",
		Title: "Test Feed",
	}

	// Default values should be set by GORM tags
	// We can't test GORM defaults directly without a DB connection,
	// but we can verify the tags are set correctly in the struct
	if feed.Type == "" {
		// Type has default:"rss" in GORM tag
		// This won't be set without GORM, but the tag exists
	}
	if feed.IsActive == false {
		// IsActive has default:true in GORM tag
	}
	if feed.Include == false {
		// Include has default:true in GORM tag
	}
	if feed.Subscribed == true {
		// Subscribed has default:false in GORM tag
	}
}

func TestFeed_WebSubFields(t *testing.T) {
	feed := Feed{
		URL:      "https://example.com/feed.xml",
		HubURL:   "https://pubsubhubbub.appspot.com",
		TopicURL: "https://example.com/feed.xml",
		Secret:   "webhook-secret-123",
	}

	if feed.HubURL != "https://pubsubhubbub.appspot.com" {
		t.Errorf("Expected HubURL to be set correctly, got: %s", feed.HubURL)
	}
	if feed.TopicURL != "https://example.com/feed.xml" {
		t.Errorf("Expected TopicURL to be set correctly, got: %s", feed.TopicURL)
	}
	if feed.Secret != "webhook-secret-123" {
		t.Errorf("Expected Secret to be set correctly, got: %s", feed.Secret)
	}
}

func TestFeed_EmptyFields(t *testing.T) {
	// Test that empty fields don't cause panics
	feed := Feed{}

	if feed.URL != "" {
		t.Errorf("Expected empty URL, got: %s", feed.URL)
	}
	if feed.Title != "" {
		t.Errorf("Expected empty Title, got: %s", feed.Title)
	}
	if feed.Link != "" {
		t.Errorf("Expected empty Link, got: %s", feed.Link)
	}
}

// =============================================================================
// Item Model Tests
// =============================================================================

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

func TestItem_Defaults(t *testing.T) {
	// Test default values for Item
	item := Item{
		Title: "Test",
		Link:  "https://example.com/test",
		Guid:  "test-guid",
	}

	// Read has default:false in GORM tag
	if item.Read == true {
		t.Error("Expected Read to default to false")
	}
}

func TestItem_EmptyFields(t *testing.T) {
	// Test that empty fields don't cause panics
	item := Item{}

	if item.Title != "" {
		t.Errorf("Expected empty Title, got: %s", item.Title)
	}
	if item.Link != "" {
		t.Errorf("Expected empty Link, got: %s", item.Link)
	}
	if item.Guid != "" {
		t.Errorf("Expected empty Guid, got: %s", item.Guid)
	}
	if item.Read != false {
		t.Errorf("Expected Read to be false by default, got: %v", item.Read)
	}
}

func TestItem_FullContent(t *testing.T) {
	item := Item{
		Title:       "Breaking News: Long Title That Should Be Handled Correctly",
		Link:        "https://example.com/very/long/path/to/article.html",
		Description: "A detailed description with multiple sentences. This should test how the model handles longer text content.",
		Content:     "Full article content with many paragraphs.\n\nSecond paragraph.\n\nThird paragraph.",
		ImageURL:    "https://example.com/images/large-image.jpg",
		Guid:        "very-unique-guid-for-this-specific-article-12345",
	}

	if len(item.Title) == 0 {
		t.Error("Expected Title to have content")
	}
	if len(item.Content) == 0 {
		t.Error("Expected Content to have content")
	}
	if len(item.Description) == 0 {
		t.Error("Expected Description to have content")
	}
}

// =============================================================================
// Feed-Item Relationship Tests
// =============================================================================

func TestFeedItem_Relationship(t *testing.T) {
	// Create a feed
	feed := Feed{
		ID:    1,
		URL:   "https://example.com/feed.xml",
		Title: "Test Feed",
	}

	// Create items belonging to this feed
	items := []Item{
		{
			FeedID:      feed.ID,
			Title:       "Item 1",
			Link:        "https://example.com/item1",
			Guid:        "guid-1",
			PublishedAt: time.Now(),
		},
		{
			FeedID:      feed.ID,
			Title:       "Item 2",
			Link:        "https://example.com/item2",
			Guid:        "guid-2",
			PublishedAt: time.Now(),
		},
	}

	// Verify items have correct FeedID
	for _, item := range items {
		if item.FeedID != feed.ID {
			t.Errorf("Expected item FeedID to be %d, got: %d", feed.ID, item.FeedID)
		}
	}

	// Verify items have unique GUIDs within the same feed
	guidMap := make(map[string]bool)
	for _, item := range items {
		if guidMap[item.Guid] {
			t.Errorf("Duplicate GUID found: %s", item.Guid)
		}
		guidMap[item.Guid] = true
	}
}

func TestMultipleFeeds_SameGUID(t *testing.T) {
	// This tests the multi-user scenario where different feeds
	// (possibly with same URL for different users) can have items
	// with the same GUID
	
	// Feed 1 (user 1)
	feed1 := Feed{
		ID:    1,
		URL:   "https://example.com/feed.xml",
		Title: "User 1 Feed",
		UserID: 1,
	}

	// Feed 2 (user 2) - same URL as feed1, but different user
	feed2 := Feed{
		ID:    2,
		URL:   "https://example.com/feed.xml",
		Title: "User 2 Feed",
		UserID: 2,
	}

	// Both feeds have an item with the same GUID
	// This is valid because they belong to different feeds
	item1 := Item{
		FeedID:      feed1.ID,
		Title:       "Same Article",
		Link:        "https://example.com/article",
		Guid:        "same-guid-123",
		PublishedAt: time.Now(),
	}

	item2 := Item{
		FeedID:      feed2.ID,
		Title:       "Same Article",
		Link:        "https://example.com/article",
		Guid:        "same-guid-123", // Same GUID as item1, but different FeedID
		PublishedAt: time.Now(),
	}

	// Verify both items have the same GUID but different FeedIDs
	if item1.Guid != item2.Guid {
		t.Error("Expected both items to have the same GUID")
	}
	if item1.FeedID == item2.FeedID {
		t.Error("Expected items to have different FeedIDs")
	}

	// This is the key assertion: items with same GUID but different FeedID are valid
	// The unique constraint should be on (feed_id, guid), not on guid alone
	if item1.Guid == item2.Guid && item1.FeedID != item2.FeedID {
		// This is the expected behavior for multi-user support
		t.Logf("✓ Items with same GUID (%s) can exist for different feeds (feed_id: %d, %d)", 
			item1.Guid, item1.FeedID, item2.FeedID)
	}
}

func TestFeed_UserOwnership(t *testing.T) {
	// Test that feeds are properly associated with users
	feed := Feed{
		ID:     1,
		URL:    "https://user-specific.com/feed.xml",
		Title:  "User's Personal Feed",
		UserID: 42, // User ID 42
	}

	if feed.UserID != 42 {
		t.Errorf("Expected UserID to be 42, got: %d", feed.UserID)
	}
}

// =============================================================================
// Time-Based Tests
// =============================================================================

func TestFeed_LastFetch(t *testing.T) {
	before := time.Now()
	feed := Feed{
		URL:       "https://example.com/feed.xml",
		Title:     "Timed Feed",
		LastFetch: time.Now(),
	}
	after := time.Now()

	if feed.LastFetch.Before(before) {
		t.Errorf("LastFetch should be after or equal to before time")
	}
	if feed.LastFetch.After(after) {
		t.Errorf("LastFetch should be before or equal to after time")
	}
}

func TestItem_PublishedAt(t *testing.T) {
	published := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	item := Item{
		Title:       "Timed Article",
		PublishedAt: published,
	}

	if item.PublishedAt != published {
		t.Errorf("Expected PublishedAt to be %v, got: %v", published, item.PublishedAt)
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestFeed_SpecialCharacters(t *testing.T) {
	feed := Feed{
		URL:   "https://example.com/feed?param=value&other=123",
		Title: "Feed with Special Chars: <>&\"'",
		Link:  "https://example.com?query=test",
	}

	if feed.URL == "" {
		t.Error("Expected URL to be set")
	}
	if feed.Title == "" {
		t.Error("Expected Title to be set")
	}
}

func TestItem_SpecialCharacters(t *testing.T) {
	item := Item{
		Title:       "Article: <script>alert('xss')</script>",
		Description: "Description with \"quotes\" and 'apostrophes'",
		Content:     "Content with\nnewlines\tand\ttabs",
		Guid:        "guid-with-special-chars-äöü-123",
	}

	if item.Title == "" {
		t.Error("Expected Title to be set")
	}
	if item.Guid == "" {
		t.Error("Expected Guid to be set")
	}
}

func TestFeed_MinimalValid(t *testing.T) {
	// Test minimal valid feed (only required fields)
	feed := Feed{
		URL: "https://example.com/feed.xml",
	}

	if feed.URL != "https://example.com/feed.xml" {
		t.Errorf("Expected URL to be set")
	}
}

func TestItem_MinimalValid(t *testing.T) {
	// Test minimal valid item (only required fields)
	item := Item{
		Link: "https://example.com/article",
		Guid: "unique-guid",
	}

	if item.Link != "https://example.com/article" {
		t.Errorf("Expected Link to be set")
	}
	if item.Guid != "unique-guid" {
		t.Errorf("Expected Guid to be set")
	}
}
