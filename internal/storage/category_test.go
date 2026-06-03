package storage

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"rss-aggregator/internal/models"
)

// Test helper to set up an in-memory database with Category support
func setupTestDBWithCategories() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Auto-migrate all tables including Category
	err = db.AutoMigrate(&models.Feed{}, &models.Item{}, &models.User{}, &models.Category{})
	if err != nil {
		panic(err)
	}

	// Create indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uni_feeds_url_user ON feeds(url, user_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uni_items_feed_guid ON items(feed_id, guid)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name_user ON categories(name, user_id)")

	return db
}

// =============================================================================
// GetAllCategoriesByUser Tests
// =============================================================================

func TestGetAllCategoriesByUser_Empty(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	categories, err := GetAllCategoriesByUser(db, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(categories))
	}
}

func TestGetAllCategoriesByUser_WithCategories(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	
	// Create test categories
	db.Create(&models.Category{UserID: userID, Name: "Technology", Color: "#FF5733"})
	db.Create(&models.Category{UserID: userID, Name: "Sports", Color: "#33FF57"})
	db.Create(&models.Category{UserID: userID, Name: "News", Color: "#3357FF"})

	categories, err := GetAllCategoriesByUser(db, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(categories))
	}
}

func TestGetAllCategoriesByUser_OrderedByName(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	
	// Create categories in non-alphabetical order
	db.Create(&models.Category{UserID: userID, Name: "Zebra", Color: "#000000"})
	db.Create(&models.Category{UserID: userID, Name: "Alpha", Color: "#000000"})
	db.Create(&models.Category{UserID: userID, Name: "Beta", Color: "#000000"})

	categories, err := GetAllCategoriesByUser(db, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(categories))
	}
	// Check ordering
	if categories[0].Name != "Alpha" {
		t.Errorf("Expected first category to be 'Alpha', got '%s'", categories[0].Name)
	}
	if categories[1].Name != "Beta" {
		t.Errorf("Expected second category to be 'Beta', got '%s'", categories[1].Name)
	}
	if categories[2].Name != "Zebra" {
		t.Errorf("Expected third category to be 'Zebra', got '%s'", categories[2].Name)
	}
}

// =============================================================================
// GetCategoryByIDAndUser Tests
// =============================================================================

func TestGetCategoryByIDAndUser_NotFound(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	category, err := GetCategoryByIDAndUser(db, 999, userID)

	if err != ErrCategoryNotFound {
		t.Errorf("Expected ErrCategoryNotFound, got %v", err)
	}
	if category != nil {
		t.Errorf("Expected nil category, got %v", category)
	}
}

func TestGetCategoryByIDAndUser_Found(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	created := &models.Category{UserID: userID, Name: "Test Category", Color: "#123456"}
	db.Create(created)

	category, err := GetCategoryByIDAndUser(db, created.ID, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if category == nil {
		t.Errorf("Expected category, got nil")
	}
	if category.Name != "Test Category" {
		t.Errorf("Expected name 'Test Category', got '%s'", category.Name)
	}
	if category.Color != "#123456" {
		t.Errorf("Expected color '#123456', got '%s'", category.Color)
	}
}

func TestGetCategoryByIDAndUser_WrongUser(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	created := &models.Category{UserID: userID, Name: "Test Category", Color: "#123456"}
	db.Create(created)

	// Try to access with different user
	category, err := GetCategoryByIDAndUser(db, created.ID, 2)

	if err != ErrCategoryNotFound {
		t.Errorf("Expected ErrCategoryNotFound for wrong user, got %v", err)
	}
	if category != nil {
		t.Errorf("Expected nil category for wrong user, got %v", category)
	}
}

// =============================================================================
// CreateCategoryForUser Tests
// =============================================================================

func TestCreateCategoryForUser_Success(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	category := &models.Category{Name: "New Category", Color: "#ABCDEF"}

	err := CreateCategoryForUser(db, category, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if category.ID == 0 {
		t.Error("Expected category ID to be set")
	}
	if category.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, category.UserID)
	}

	// Verify it was created
	Retrieved, err := GetCategoryByIDAndUser(db, category.ID, userID)
	if err != nil {
		t.Errorf("Expected to retrieve created category, got %v", err)
	}
	if Retrieved.Name != "New Category" {
		t.Errorf("Expected name 'New Category', got '%s'", Retrieved.Name)
	}
}

func TestCreateCategoryForUser_DuplicateName(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	
	// Create first category
	first := &models.Category{Name: "Duplicate", Color: "#000000"}
	CreateCategoryForUser(db, first, userID)

	// Try to create another with same name
	second := &models.Category{Name: "Duplicate", Color: "#FFFFFF"}
	err := CreateCategoryForUser(db, second, userID)

	if err != ErrCategoryNameExists {
		t.Errorf("Expected ErrCategoryNameExists, got %v", err)
	}
}

func TestCreateCategoryForUser_DifferentUsersSameName(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create category for user 1
	category1 := &models.Category{Name: "Same Name", Color: "#000000"}
	CreateCategoryForUser(db, category1, 1)

	// Create category with same name for user 2 (should succeed)
	category2 := &models.Category{Name: "Same Name", Color: "#FFFFFF"}
	err := CreateCategoryForUser(db, category2, 2)

	if err != nil {
		t.Errorf("Expected no error for different users with same name, got %v", err)
	}
}

// =============================================================================
// UpdateCategoryForUser Tests
// =============================================================================

func TestUpdateCategoryForUser_Success(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	created := &models.Category{UserID: userID, Name: "Original", Color: "#000000"}
	db.Create(created)

	// Update the category
	created.Name = "Updated"
	created.Color = "#FFFFFF"

	err := UpdateCategoryForUser(db, created, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify update
	retrieved, err := GetCategoryByIDAndUser(db, created.ID, userID)
	if err != nil {
		t.Errorf("Expected to retrieve updated category, got %v", err)
	}
	if retrieved.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", retrieved.Name)
	}
	if retrieved.Color != "#FFFFFF" {
		t.Errorf("Expected color '#FFFFFF', got '%s'", retrieved.Color)
	}
}

func TestUpdateCategoryForUser_NotFound(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	category := &models.Category{ID: 999, UserID: userID, Name: "Non-existent", Color: "#000000"}

	err := UpdateCategoryForUser(db, category, userID)

	if err != ErrCategoryNotFound {
		t.Errorf("Expected ErrCategoryNotFound, got %v", err)
	}
}

func TestUpdateCategoryForUser_DuplicateName(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	
	// Create two categories
	first := &models.Category{UserID: userID, Name: "First", Color: "#000000"}
	db.Create(first)
	
	second := &models.Category{UserID: userID, Name: "Second", Color: "#111111"}
	db.Create(second)

	// Try to update second to have same name as first
	second.Name = "First"
	err := UpdateCategoryForUser(db, second, userID)

	if err != ErrCategoryNameExists {
		t.Errorf("Expected ErrCategoryNameExists, got %v", err)
	}
}

// =============================================================================
// DeleteCategoryForUser Tests
// =============================================================================

func TestDeleteCategoryForUser_Success(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	category := &models.Category{UserID: userID, Name: "To Delete", Color: "#000000"}
	db.Create(category)

	// Create a feed and assign it to the category
	feed := &models.Feed{
		URL:       "https://example.com/feed.xml",
		Title:     "Test Feed",
		UserID:    userID,
		CategoryID: &category.ID,
	}
	db.Create(feed)

	// Delete the category
	err := DeleteCategoryForUser(db, category.ID, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify category is deleted
	_, err = GetCategoryByIDAndUser(db, category.ID, userID)
	if err != ErrCategoryNotFound {
		t.Errorf("Expected ErrCategoryNotFound after deletion, got %v", err)
	}

	// Verify feed's category ID is set to null
	var updatedFeed models.Feed
	db.First(&updatedFeed, feed.ID)
	if updatedFeed.CategoryID != nil {
		t.Errorf("Expected feed CategoryID to be nil after category deletion, got %v", updatedFeed.CategoryID)
	}
}

func TestDeleteCategoryForUser_NotFound(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	err := DeleteCategoryForUser(db, 999, userID)

	if err != ErrCategoryNotFound {
		t.Errorf("Expected ErrCategoryNotFound, got %v", err)
	}
}

// =============================================================================
// AssignFeedToCategory Tests
// =============================================================================

func TestAssignFeedToCategory_Success(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	
	// Create category
	category := &models.Category{UserID: userID, Name: "Test Category", Color: "#000000"}
	db.Create(category)

	// Create feed
	feed := &models.Feed{
		URL:    "https://example.com/feed.xml",
		Title:  "Test Feed",
		UserID: userID,
	}
	db.Create(feed)

	// Assign feed to category
	err := AssignFeedToCategory(db, feed.ID, category.ID, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify assignment
	var updatedFeed models.Feed
	db.First(&updatedFeed, feed.ID)
	if updatedFeed.CategoryID == nil {
		t.Error("Expected CategoryID to be set")
	}
	if *updatedFeed.CategoryID != category.ID {
		t.Errorf("Expected CategoryID %d, got %d", category.ID, *updatedFeed.CategoryID)
	}
}

func TestAssignFeedToCategory_WrongUser(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	otherUserID := uint(2)
	
	// Create category for user 1
	category := &models.Category{UserID: userID, Name: "Test Category", Color: "#000000"}
	db.Create(category)

	// Create feed for user 2
	feed := &models.Feed{
		URL:    "https://example.com/feed.xml",
		Title:  "Test Feed",
		UserID: otherUserID,
	}
	db.Create(feed)

	// Try to assign with wrong user
	err := AssignFeedToCategory(db, feed.ID, category.ID, userID)

	if err == nil {
		t.Error("Expected error when assigning feed from different user")
	}
}

// =============================================================================
// RemoveFeedFromCategory Tests
// =============================================================================

func TestRemoveFeedFromCategory_Success(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	
	// Create category
	category := &models.Category{UserID: userID, Name: "Test Category", Color: "#000000"}
	db.Create(category)

	// Create feed with category
	feed := &models.Feed{
		URL:       "https://example.com/feed.xml",
		Title:     "Test Feed",
		UserID:    userID,
		CategoryID: &category.ID,
	}
	db.Create(feed)

	// Remove feed from category
	err := RemoveFeedFromCategory(db, feed.ID, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify removal
	var updatedFeed models.Feed
	db.First(&updatedFeed, feed.ID)
	if updatedFeed.CategoryID != nil {
		t.Errorf("Expected CategoryID to be nil, got %v", updatedFeed.CategoryID)
	}
}

// =============================================================================
// GetFeedsByCategoryAndUser Tests
// =============================================================================

func TestGetFeedsByCategoryAndUser_Success(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	
	// Create category
	category := &models.Category{UserID: userID, Name: "Test Category", Color: "#000000"}
	db.Create(category)

	// Create feeds
	feed1 := &models.Feed{
		URL:       "https://example.com/feed1.xml",
		Title:     "Feed 1",
		UserID:    userID,
		CategoryID: &category.ID,
	}
	feed2 := &models.Feed{
		URL:       "https://example.com/feed2.xml",
		Title:     "Feed 2",
		UserID:    userID,
		CategoryID: &category.ID,
	}
	feed3 := &models.Feed{
		URL:    "https://example.com/feed3.xml",
		Title:  "Feed 3",
		UserID: userID,
		// No category
	}
	db.Create(feed1)
	db.Create(feed2)
	db.Create(feed3)

	// Get feeds by category
	feeds, err := GetFeedsByCategoryAndUser(db, category.ID, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("Expected 2 feeds, got %d", len(feeds))
	}
}

func TestGetFeedsByCategoryAndUser_Empty(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)

	feeds, err := GetFeedsByCategoryAndUser(db, 999, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("Expected 0 feeds, got %d", len(feeds))
	}
}

// =============================================================================
// GetUncategorizedFeedsByUser Tests
// =============================================================================

func TestGetUncategorizedFeedsByUser_Success(t *testing.T) {
	db := setupTestDBWithCategories()
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	userID := uint(1)
	
	// Create category
	category := &models.Category{UserID: userID, Name: "Test Category", Color: "#000000"}
	db.Create(category)

	// Create feeds
	feed1 := &models.Feed{
		URL:       "https://example.com/feed1.xml",
		Title:     "Feed 1",
		UserID:    userID,
		CategoryID: &category.ID,
	}
	feed2 := &models.Feed{
		URL:    "https://example.com/feed2.xml",
		Title:  "Feed 2",
		UserID: userID,
		// No category
	}
	feed3 := &models.Feed{
		URL:    "https://example.com/feed3.xml",
		Title:  "Feed 3",
		UserID: userID,
		// No category
	}
	db.Create(feed1)
	db.Create(feed2)
	db.Create(feed3)

	// Get uncategorized feeds
	feeds, err := GetUncategorizedFeedsByUser(db, userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("Expected 2 uncategorized feeds, got %d", len(feeds))
	}
}
