package storage

import (
	"errors"
	"testing"
	"time"

	"rss-aggregator/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test helper to set up an in-memory database
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Auto-migrate tables
	err = db.AutoMigrate(&models.Feed{}, &models.Item{}, &models.User{})
	if err != nil {
		panic(err)
	}

	// Create indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uni_feeds_url_user ON feeds(url, user_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uni_items_feed_guid ON items(feed_id, guid)")

	return db
}

// =============================================================================
// AddFeedForUser Tests
// =============================================================================

func TestAddFeedForUser_NewFeed(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	feed := &models.Feed{
		URL:   "https://example.com/new-feed.xml",
		Title: "New Feed",
		Link:  "https://example.com/new-feed",
		Type:  "rss",
	}

	items := []models.Item{
		{
			Title:       "Item 1",
			Link:        "https://example.com/item1",
			Description: "Description 1",
			Guid:        "guid-1",
			PublishedAt: time.Now(),
		},
		{
			Title:       "Item 2",
			Link:        "https://example.com/item2",
			Description: "Description 2",
			Guid:        "guid-2",
			PublishedAt: time.Now(),
		},
	}

	err := AddFeedForUser(db, feed, items, userID)
	if err != nil {
		t.Fatalf("Failed to add feed: %v", err)
	}

	// Verify feed was created
	var createdFeed models.Feed
	err = db.Where("url = ? AND user_id = ?", feed.URL, userID).First(&createdFeed).Error
	if err != nil {
		t.Fatalf("Feed not found in database: %v", err)
	}
	if createdFeed.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, createdFeed.UserID)
	}

	// Verify items were created
	var count int64
	db.Model(&models.Item{}).Where("feed_id = ?", createdFeed.ID).Count(&count)
	if count != int64(len(items)) {
		t.Errorf("Expected %d items, got %d", len(items), count)
	}
}

func TestAddFeedForUser_ExistingFeed(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	// Create initial feed
	feed := &models.Feed{
		URL:     "https://example.com/existing-feed.xml",
		Title:   "Initial Title",
		Link:    "https://example.com/existing-feed",
		Type:    "rss",
		Include: true,
	}

	items := []models.Item{
		{
			Title:       "Initial Item",
			Link:        "https://example.com/initial-item",
			Guid:        "initial-guid",
			PublishedAt: time.Now(),
		},
	}

	err := AddFeedForUser(db, feed, items, userID)
	if err != nil {
		t.Fatalf("Failed to add initial feed: %v", err)
	}

	// Add the same feed URL again with updated items
	updatedFeed := &models.Feed{
		URL:     "https://example.com/existing-feed.xml",
		Title:   "Updated Title",
		Link:    "https://example.com/existing-feed",
		Type:    "rss",
		Include: false, // Changed
	}

	updatedItems := []models.Item{
		{
			Title:       "Initial Item",
			Link:        "https://example.com/initial-item",
			Guid:        "initial-guid", // Same GUID
			PublishedAt: time.Now(),
		},
		{
			Title:       "New Item",
			Link:        "https://example.com/new-item",
			Guid:        "new-guid",
			PublishedAt: time.Now(),
		},
	}

	err = AddFeedForUser(db, updatedFeed, updatedItems, userID)
	if err != nil {
		t.Fatalf("Failed to add updated feed: %v", err)
	}

	// Verify feed was updated, not duplicated
	var count int64
	db.Model(&models.Feed{}).Where("url = ? AND user_id = ?", feed.URL, userID).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 feed, got %d (duplicate feeds created)", count)
	}

	// Verify items count
	var itemCount int64
	db.Model(&models.Item{}).Where("feed_id = ?", updatedFeed.ID).Count(&itemCount)
	if itemCount != 2 {
		t.Errorf("Expected 2 items after update, got %d", itemCount)
	}
}

func TestAddFeedForUser_DifferentUsersSameURL(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// User 1 adds a feed
	feed1 := &models.Feed{
		URL:   "https://example.com/shared-feed.xml",
		Title: "Shared Feed User 1",
	}

	items1 := []models.Item{
		{
			Title:       "Shared Item",
			Link:        "https://example.com/shared-item",
			Guid:        "shared-guid",
			PublishedAt: time.Now(),
		},
	}

	err := AddFeedForUser(db, feed1, items1, 1)
	if err != nil {
		t.Fatalf("User 1 failed to add feed: %v", err)
	}

	// User 2 adds the same feed URL
	feed2 := &models.Feed{
		URL:   "https://example.com/shared-feed.xml",
		Title: "Shared Feed User 2",
	}

	items2 := []models.Item{
		{
			Title:       "Shared Item",
			Link:        "https://example.com/shared-item",
			Guid:        "shared-guid", // Same GUID as user 1
			PublishedAt: time.Now(),
		},
	}

	err = AddFeedForUser(db, feed2, items2, 2)
	if err != nil {
		t.Fatalf("User 2 failed to add feed: %v", err)
	}

	// Verify both feeds exist
	var feedCount int64
	db.Model(&models.Feed{}).Where("url = ?", "https://example.com/shared-feed.xml").Count(&feedCount)
	if feedCount != 2 {
		t.Errorf("Expected 2 feeds (one per user), got %d", feedCount)
	}

	// Verify both sets of items exist
	var itemCount int64
	db.Model(&models.Item{}).Where("guid = ?", "shared-guid").Count(&itemCount)
	if itemCount != 2 {
		t.Errorf("Expected 2 items (one per feed), got %d", itemCount)
	}
}

// =============================================================================
// GetFilteredItemsByUser Tests
// =============================================================================

func TestGetFilteredItemsByUser(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	// Create a feed for the user
	feed := models.Feed{
		URL:    "https://example.com/test-feed.xml",
		Title:  "Test Feed",
		UserID: userID,
	}
	db.Create(&feed)

	// Add items to the feed
	items := []models.Item{
		{FeedID: feed.ID, Title: "Item 1", Guid: "guid-1", PublishedAt: time.Now()},
		{FeedID: feed.ID, Title: "Item 2", Guid: "guid-2", PublishedAt: time.Now().Add(-1 * time.Hour)},
		{FeedID: feed.ID, Title: "Item 3", Guid: "guid-3", PublishedAt: time.Now().Add(-2 * time.Hour)},
	}
	db.Create(&items)

	// Test: Get all items for user
	result, count, err := GetFilteredItemsByUser(db, 10, 0, nil, userID)
	if err != nil {
		t.Fatalf("Failed to get items: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 items, got %d", count)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items in result, got %d", len(result))
	}
}

func TestGetFilteredItemsByUser_WithFeedIDs(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	// Create two feeds for the user
	feed1 := models.Feed{URL: "https://feed1.com", Title: "Feed 1", UserID: userID}
	feed2 := models.Feed{URL: "https://feed2.com", Title: "Feed 2", UserID: userID}
	db.Create(&feed1)
	db.Create(&feed2)

	// Add items to both feeds
	db.Create(&models.Item{FeedID: feed1.ID, Title: "Feed1 Item", Guid: "f1-guid-1"})
	db.Create(&models.Item{FeedID: feed1.ID, Title: "Feed1 Item 2", Guid: "f1-guid-2"})
	db.Create(&models.Item{FeedID: feed2.ID, Title: "Feed2 Item", Guid: "f2-guid-1"})

	// Test: Get items for specific feed
	feedIDs := []uint{feed1.ID}
	result, count, err := GetFilteredItemsByUser(db, 10, 0, feedIDs, userID)
	if err != nil {
		t.Fatalf("Failed to get filtered items: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 items for feed1, got %d", count)
	}

	// Verify all returned items are from feed1
	for _, item := range result {
		if item.FeedID != feed1.ID {
			t.Errorf("Expected item from feed %d, got feed %d", feed1.ID, item.FeedID)
		}
	}
}

func TestGetFilteredItemsByUser_DifferentUsers(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create feeds for two users
	feed1 := models.Feed{URL: "https://feed.com", Title: "Feed", UserID: 1}
	feed2 := models.Feed{URL: "https://feed.com", Title: "Feed", UserID: 2}
	db.Create(&feed1)
	db.Create(&feed2)

	// Add items to both feeds with same GUID
	db.Create(&models.Item{FeedID: feed1.ID, Title: "Item", Guid: "same-guid"})
	db.Create(&models.Item{FeedID: feed2.ID, Title: "Item", Guid: "same-guid"})

	// Test: User 1 should only see their items
	result1, _, err := GetFilteredItemsByUser(db, 10, 0, nil, 1)
	if err != nil {
		t.Fatalf("Failed to get items for user 1: %v", err)
	}

	if len(result1) != 1 {
		t.Errorf("Expected 1 item for user 1, got %d", len(result1))
	}

	if result1[0].FeedID != feed1.ID {
		t.Errorf("User 1 should only see items from their own feed")
	}

	// Test: User 2 should only see their items
	result2, _, err := GetFilteredItemsByUser(db, 10, 0, nil, 2)
	if err != nil {
		t.Fatalf("Failed to get items for user 2: %v", err)
	}

	if len(result2) != 1 {
		t.Errorf("Expected 1 item for user 2, got %d", len(result2))
	}

	if result2[0].FeedID != feed2.ID {
		t.Errorf("User 2 should only see items from their own feed")
	}
}

// =============================================================================
// GetAllFeedsByUser Tests
// =============================================================================

func TestGetAllFeedsByUser(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	// Create feeds for user
	feeds := []models.Feed{
		{URL: "https://feed1.com", Title: "Feed 1", UserID: userID},
		{URL: "https://feed2.com", Title: "Feed 2", UserID: userID},
		{URL: "https://feed3.com", Title: "Feed 3", UserID: userID},
	}

	for i := range feeds {
		db.Create(&feeds[i])
	}

	// Test: Get all feeds for user
	result, err := GetAllFeedsByUser(db, userID)
	if err != nil {
		t.Fatalf("Failed to get feeds: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 feeds, got %d", len(result))
	}

	// Verify all feeds belong to the user
	for _, feed := range result {
		if feed.UserID != userID {
			t.Errorf("Feed %d does not belong to user %d", feed.ID, userID)
		}
	}
}

func TestGetAllFeedsByUser_Isolation(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create feeds for two users
	db.Create(&models.Feed{URL: "https://feed1.com", Title: "User1 Feed", UserID: 1})
	db.Create(&models.Feed{URL: "https://feed2.com", Title: "User1 Feed 2", UserID: 1})
	db.Create(&models.Feed{URL: "https://feed3.com", Title: "User2 Feed", UserID: 2})

	// Test: User 1 should only see their feeds
	feeds1, err := GetAllFeedsByUser(db, 1)
	if err != nil {
		t.Fatalf("Failed to get feeds for user 1: %v", err)
	}

	if len(feeds1) != 2 {
		t.Errorf("Expected 2 feeds for user 1, got %d", len(feeds1))
	}

	// Test: User 2 should only see their feed
	feeds2, err := GetAllFeedsByUser(db, 2)
	if err != nil {
		t.Fatalf("Failed to get feeds for user 2: %v", err)
	}

	if len(feeds2) != 1 {
		t.Errorf("Expected 1 feed for user 2, got %d", len(feeds2))
	}
}

// =============================================================================
// GetFeedByIDAndUser Tests
// =============================================================================

func TestGetFeedByIDAndUser(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	// Create a feed for the user
	feed := models.Feed{URL: "https://test.com", Title: "Test", UserID: userID}
	db.Create(&feed)

	// Test: Get feed by ID for the user
	result, err := GetFeedByIDAndUser(db, feed.ID, userID)
	if err != nil {
		t.Fatalf("Failed to get feed: %v", err)
	}

	if result.ID != feed.ID {
		t.Errorf("Expected feed ID %d, got %d", feed.ID, result.ID)
	}
}

func TestGetFeedByIDAndUser_NotFound(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create a feed for user 1
	feed := models.Feed{URL: "https://test.com", Title: "Test", UserID: 1}
	db.Create(&feed)

	// Test: User 2 tries to access user 1's feed
	_, err := GetFeedByIDAndUser(db, feed.ID, 2)
	if err == nil {
		t.Error("Expected error when user tries to access another user's feed")
	}

	// When a feed doesn't exist or doesn't belong to the user,
	// GORM returns gorm.ErrRecordNotFound
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected gorm.ErrRecordNotFound, got: %v", err)
	}
}

func TestGetFeedByIDAndUser_NoUserFilter(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create a feed for user 1
	feed := models.Feed{URL: "https://test.com", Title: "Test", UserID: 1}
	db.Create(&feed)

	// Test: Get feed without user filter (userID = 0)
	result, err := GetFeedByIDAndUser(db, feed.ID, 0)
	if err != nil {
		t.Fatalf("Failed to get feed without user filter: %v", err)
	}

	if result.ID != feed.ID {
		t.Errorf("Expected feed ID %d, got %d", feed.ID, result.ID)
	}
}

// =============================================================================
// ToggleFeedIncludeForUser Tests
// =============================================================================

func TestToggleFeedIncludeForUser(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	// Create a feed for the user
	feed := models.Feed{URL: "https://test.com", Title: "Test", Include: true, UserID: userID}
	db.Create(&feed)

	// Test: Toggle include to false
	err := ToggleFeedIncludeForUser(db, feed.ID, userID)
	if err != nil {
		t.Fatalf("Failed to toggle feed: %v", err)
	}

	// Verify the feed was toggled
	var updatedFeed models.Feed
	db.First(&updatedFeed, feed.ID)
	if updatedFeed.Include != false {
		t.Errorf("Expected Include to be false, got %v", updatedFeed.Include)
	}

	// Test: Toggle back to true
	err = ToggleFeedIncludeForUser(db, feed.ID, userID)
	if err != nil {
		t.Fatalf("Failed to toggle feed back: %v", err)
	}

	db.First(&updatedFeed, feed.ID)
	if updatedFeed.Include != true {
		t.Errorf("Expected Include to be true, got %v", updatedFeed.Include)
	}
}

func TestToggleFeedIncludeForUser_Unauthorized(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create a feed for user 1
	feed := models.Feed{URL: "https://test.com", Title: "Test", UserID: 1}
	db.Create(&feed)

	// Test: User 2 tries to toggle user 1's feed
	err := ToggleFeedIncludeForUser(db, feed.ID, 2)
	if err == nil {
		t.Error("Expected error when user tries to toggle another user's feed")
	}

	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("Expected ErrUnauthorized, got: %v", err)
	}
}

// =============================================================================
// MarkItemAsReadForUser Tests
// =============================================================================

func TestMarkItemAsReadForUser(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	// Create feed and item
	feed := models.Feed{URL: "https://test.com", Title: "Test", UserID: userID}
	db.Create(&feed)

	item := models.Item{FeedID: feed.ID, Title: "Test Item", Read: false}
	db.Create(&item)

	// Test: Mark item as read
	err := MarkItemAsReadForUser(db, item.ID, userID)
	if err != nil {
		t.Fatalf("Failed to mark item as read: %v", err)
	}

	// Verify the item was marked as read
	var updatedItem models.Item
	db.First(&updatedItem, item.ID)
	if updatedItem.Read != true {
		t.Errorf("Expected Read to be true, got %v", updatedItem.Read)
	}
}

func TestMarkItemAsReadForUser_Unauthorized(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create feed and item for user 1
	feed := models.Feed{URL: "https://test.com", Title: "Test", UserID: 1}
	db.Create(&feed)

	item := models.Item{FeedID: feed.ID, Title: "Test Item"}
	db.Create(&item)

	// Test: User 2 tries to mark user 1's item as read
	err := MarkItemAsReadForUser(db, item.ID, 2)
	if err == nil {
		t.Error("Expected error when user tries to mark another user's item")
	}

	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("Expected ErrUnauthorized, got: %v", err)
	}
}

// =============================================================================
// DeleteFeedForUser Tests
// =============================================================================

func TestDeleteFeedForUser(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	// Create feed with items
	feed := models.Feed{URL: "https://test.com", Title: "Test", UserID: userID}
	db.Create(&feed)

	db.Create(&models.Item{FeedID: feed.ID, Title: "Item 1", Guid: "guid-1"})
	db.Create(&models.Item{FeedID: feed.ID, Title: "Item 2", Guid: "guid-2"})

	// Test: Delete feed
	err := DeleteFeedForUser(db, feed.ID, userID)
	if err != nil {
		t.Fatalf("Failed to delete feed: %v", err)
	}

	// Verify feed was deleted
	var count int64
	db.Model(&models.Feed{}).Where("id = ?", feed.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected feed to be deleted, but it still exists")
	}

	// Verify items were deleted
	db.Model(&models.Item{}).Where("feed_id = ?", feed.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected items to be deleted, but %d still exist", count)
	}
}

func TestDeleteFeedForUser_Unauthorized(t *testing.T) {
	db := setupTestDB()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create feed for user 1
	feed := models.Feed{URL: "https://test.com", Title: "Test", UserID: 1}
	db.Create(&feed)

	// Test: User 2 tries to delete user 1's feed
	err := DeleteFeedForUser(db, feed.ID, 2)
	if err == nil {
		t.Error("Expected error when user tries to delete another user's feed")
	}

	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("Expected ErrUnauthorized, got: %v", err)
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestErrUnauthorized(t *testing.T) {
	// Test that ErrUnauthorized is properly defined
	if ErrUnauthorized == nil {
		t.Error("ErrUnauthorized should be defined")
	}

	err := ErrUnauthorized
	if err.Error() == "" {
		t.Error("ErrUnauthorized should have an error message")
	}
}
