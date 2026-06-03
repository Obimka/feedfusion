package storage

import (
	"errors"
	"rss-aggregator/internal/models"

	"gorm.io/gorm"
)

// ErrCategoryNotFound is returned when a category is not found
var ErrCategoryNotFound = errors.New("category not found")

// ErrCategoryNameExists is returned when trying to create a category with a name that already exists for the user
var ErrCategoryNameExists = errors.New("a category with this name already exists for this user")

// GetAllCategoriesByUser returns all categories for a specific user, ordered by name
func GetAllCategoriesByUser(db *gorm.DB, userID uint) ([]models.Category, error) {
	var categories []models.Category
	err := db.Where("user_id = ?", userID).Order("name ASC").Find(&categories).Error
	return categories, err
}

// GetCategoryByIDAndUser returns a category by ID for a specific user
func GetCategoryByIDAndUser(db *gorm.DB, id uint, userID uint) (*models.Category, error) {
	var category models.Category
	err := db.Where("id = ? AND user_id = ?", id, userID).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

// CreateCategoryForUser creates a new category for a user
func CreateCategoryForUser(db *gorm.DB, category *models.Category, userID uint) error {
	category.UserID = userID

	// Check if category with same name already exists for this user
	var existing models.Category
	err := db.Where("name = ? AND user_id = ?", category.Name, userID).First(&existing).Error
	if err == nil {
		return ErrCategoryNameExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return db.Create(category).Error
}

// UpdateCategoryForUser updates a category for a user
func UpdateCategoryForUser(db *gorm.DB, category *models.Category, userID uint) error {
	// Verify category belongs to user
	_, err := GetCategoryByIDAndUser(db, category.ID, userID)
	if err != nil {
		return err
	}

	// Check if another category with same name exists for this user
	var count int64
	db.Model(&models.Category{}).Where("name = ? AND user_id = ? AND id != ?",
		category.Name, userID, category.ID).Count(&count)
	if count > 0 {
		return ErrCategoryNameExists
	}

	category.UserID = userID
	return db.Save(category).Error
}

// DeleteCategoryForUser deletes a category for a user
// Also removes the category association from all feeds
func DeleteCategoryForUser(db *gorm.DB, id uint, userID uint) error {
	// Verify category belongs to user
	_, err := GetCategoryByIDAndUser(db, id, userID)
	if err != nil {
		return err
	}

	// Remove category association from all feeds that belong to this user
	err = db.Model(&models.Feed{}).Where("category_id = ? AND user_id = ?", id, userID).Update("category_id", nil).Error
	if err != nil {
		return err
	}

	// Delete the category
	return db.Delete(&models.Category{}, id).Error
}

// AssignFeedToCategory assigns a feed to a category
func AssignFeedToCategory(db *gorm.DB, feedID, categoryID, userID uint) error {
	// Verify feed belongs to user
	feed, err := GetFeedByIDAndUser(db, feedID, userID)
	if err != nil {
		return err
	}

	// Verify category belongs to user
	_, err = GetCategoryByIDAndUser(db, categoryID, userID)
	if err != nil {
		return err
	}

	feed.CategoryID = &categoryID
	return db.Save(feed).Error
}

// RemoveFeedFromCategory removes a feed from its category
func RemoveFeedFromCategory(db *gorm.DB, feedID, userID uint) error {
	// Verify feed belongs to user
	feed, err := GetFeedByIDAndUser(db, feedID, userID)
	if err != nil {
		return err
	}

	feed.CategoryID = nil
	return db.Save(feed).Error
}

// GetFeedsByCategoryAndUser returns all feeds in a category for a specific user
func GetFeedsByCategoryAndUser(db *gorm.DB, categoryID, userID uint) ([]models.Feed, error) {
	var feeds []models.Feed
	err := db.Where("category_id = ? AND user_id = ?", categoryID, userID).Find(&feeds).Error
	return feeds, err
}

// GetCategoryForFeed returns the category of a feed (if any)
func GetCategoryForFeed(db *gorm.DB, feedID uint) (*models.Category, error) {
	var feed models.Feed
	err := db.First(&feed, feedID).Error
	if err != nil {
		return nil, err
	}

	if feed.CategoryID == nil || *feed.CategoryID == 0 {
		return nil, nil
	}

	return GetCategoryByIDAndUser(db, *feed.CategoryID, feed.UserID)
}

// GetUncategorizedFeedsByUser returns all feeds without a category for a specific user
func GetUncategorizedFeedsByUser(db *gorm.DB, userID uint) ([]models.Feed, error) {
	var feeds []models.Feed
	err := db.Where("user_id = ? AND (category_id IS NULL OR category_id = 0)", userID).Find(&feeds).Error
	return feeds, err
}
