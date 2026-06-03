package storage

import (
	"errors"
	"rss-aggregator/internal/logger"
	"rss-aggregator/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB(path string) (*gorm.DB, error) {
	logger.Infof("Initializing database at: %s", path)
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		logger.Errorf("Failed to open database: %v", err)
		return nil, err
	}
	logger.Debugf("Database connection established successfully")

	err = db.AutoMigrate(&models.Feed{}, &models.Item{}, &models.User{}, &models.Category{})
	if err != nil {
		return nil, err
	}

	// Create index on feeds.user_id for better query performance
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id)").Error; err != nil {
		return nil, err
	}

	// Create unique index on categories (user_id, name) to prevent duplicate names per user
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uni_categories_user_name ON categories(user_id, name)").Error; err != nil {
		return nil, err
	}

	// Create index on categories.user_id for better query performance
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id)").Error; err != nil {
		return nil, err
	}

	// Create index on feeds.category_id for better query performance
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_feeds_category_id ON feeds(category_id)").Error; err != nil {
		return nil, err
	}

	// Create composite unique index on (url, user_id) to allow same URL for different users
	// This replaces the single-column unique constraint on url
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uni_feeds_url_user ON feeds(url, user_id)").Error; err != nil {
		return nil, err
	}

	// Create composite unique index on (feed_id, guid) for items
	// This allows same GUID for different feeds (e.g., same feed URL for different users)
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uni_items_feed_guid ON items(feed_id, guid)").Error; err != nil {
		return nil, err
	}

	return db, nil
}

// ErrUnauthorized is returned when a user tries to access another user's resources
var ErrUnauthorized = errors.New("unauthorized: resource does not belong to user")

func AddFeed(db *gorm.DB, feed *models.Feed, items []models.Item) error {
	// For backward compatibility, use AddFeedForUser with userID from feed or default to 1
	if feed.UserID == 0 {
		feed.UserID = 1 // Default to admin for backward compatibility
	}
	return AddFeedForUser(db, feed, items, feed.UserID)
}

// AddFeedForUser adds a feed for a specific user
func AddFeedForUser(db *gorm.DB, feed *models.Feed, items []models.Item, userID uint) error {
	feed.UserID = userID
	tx := db.Begin()

	// Try to find existing feed for this user
	var existingFeed models.Feed
	if err := tx.Where("url = ? AND user_id = ?", feed.URL, userID).First(&existingFeed).Error; err == nil {
		feed.ID = existingFeed.ID
		// Preserve Include setting from existing feed
		if feed.Include == false {
			feed.Include = existingFeed.Include
		}
		// Update other fields
		if err := tx.Model(feed).Select("title", "link", "type", "is_active", "last_fetch", "error", "include", "user_id").Updates(feed).Error; err != nil {
			return err
		}
	} else {
		// Check if URL exists for another user (we allow same URL for different users)
		if err := tx.Create(feed).Error; err != nil {
			return err
		}
	}

	for i := range items {
		items[i].FeedID = feed.ID
		// Try to find existing item for this specific feed (by feed_id + guid)
		// This allows different feeds (even with same URL for different users) to have their own items
		var existingItem models.Item
		err := tx.Where("feed_id = ? AND guid = ?", feed.ID, items[i].Guid).First(&existingItem).Error
		if err == nil {
			// Item exists for this feed, update if image URL is missing or different
			if existingItem.ImageURL == "" && items[i].ImageURL != "" {
				existingItem.ImageURL = items[i].ImageURL
				if err := tx.Save(&existingItem).Error; err != nil {
					return err
				}
			}
		} else {
			// Create new item for this feed
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
	}

	return tx.Commit().Error
}

func GetAllItems(db *gorm.DB, limit, offset int) ([]models.Item, int64, error) {
	// For backward compatibility, get items for all users
	return GetFilteredItemsByUser(db, limit, offset, nil, 0)
}

func GetFilteredItems(db *gorm.DB, limit, offset int, feedIDs []uint) ([]models.Item, int64, error) {
	// For backward compatibility
	return GetFilteredItemsByUser(db, limit, offset, feedIDs, 0)
}

func GetFilteredItemsByUser(db *gorm.DB, limit, offset int, feedIDs []uint, userID uint) ([]models.Item, int64, error) {
	var items []models.Item
	var count int64
	query := db.Model(&models.Item{}).Joins("JOIN feeds ON feeds.id = items.feed_id")

	// Filter by user if userID > 0
	if userID > 0 {
		query = query.Where("feeds.user_id = ?", userID)
	}

	// If feedIDs is provided and not empty, filter by those feeds
	if len(feedIDs) > 0 {
		query = query.Where("items.feed_id IN ?", feedIDs)
	}

	query.Count(&count)
	query.Order("published_at desc").Limit(limit).Offset(offset).Find(&items)

	return items, count, nil
}

func GetFilteredItemsByInclude(db *gorm.DB, limit, offset int, includeOnly bool) ([]models.Item, int64, error) {
	// For backward compatibility, include all users
	return GetFilteredItemsByIncludeAndUser(db, limit, offset, includeOnly, 0)
}

// GetFilteredItemsByIncludeAndUser filters items by include status and user
func GetFilteredItemsByIncludeAndUser(db *gorm.DB, limit, offset int, includeOnly bool, userID uint) ([]models.Item, int64, error) {
	var items []models.Item
	var count int64
	query := db.Model(&models.Item{}).Joins("JOIN feeds ON feeds.id = items.feed_id")

	// Filter by user if userID > 0
	if userID > 0 {
		query = query.Where("feeds.user_id = ?", userID)
	}

	if includeOnly {
		// Filter items from feeds where include = true
		query = query.Where("feeds.include = ?", true)
	}

	query.Count(&count)
	query.Order("published_at desc").Limit(limit).Offset(offset).Find(&items)

	return items, count, nil
}

func ToggleFeedInclude(db *gorm.DB, feedID uint) error {
	// For backward compatibility
	return ToggleFeedIncludeForUser(db, feedID, 0)
}

// ToggleFeedIncludeForUser toggles include status for a feed owned by user
func ToggleFeedIncludeForUser(db *gorm.DB, feedID uint, userID uint) error {
	// Verify feed belongs to user if userID > 0
	if userID > 0 {
		var count int64
		db.Model(&models.Feed{}).Where("id = ? AND user_id = ?", feedID, userID).Count(&count)
		if count == 0 {
			return ErrUnauthorized
		}
	}
	return db.Model(&models.Feed{}).Where("id = ?", feedID).Update("include", gorm.Expr("NOT include")).Error
}

func GetIncludedFeeds(db *gorm.DB) ([]models.Feed, error) {
	// For backward compatibility, get feeds for all users
	return GetIncludedFeedsByUser(db, 0)
}

// GetIncludedFeedsByUser returns included feeds for a specific user
func GetIncludedFeedsByUser(db *gorm.DB, userID uint) ([]models.Feed, error) {
	var feeds []models.Feed
	query := db.Where("include = ?", true)
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	err := query.Find(&feeds).Error
	return feeds, err
}

func GetFeedByID(db *gorm.DB, id uint) (*models.Feed, error) {
	// For backward compatibility
	return GetFeedByIDAndUser(db, id, 0)
}

// GetFeedByIDAndUser returns a feed by ID, optionally filtered by user
func GetFeedByIDAndUser(db *gorm.DB, id uint, userID uint) (*models.Feed, error) {
	var feed models.Feed
	query := db
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	err := query.First(&feed, id).Error
	return &feed, err
}

func GetAllFeeds(db *gorm.DB) ([]models.Feed, error) {
	// For backward compatibility, get feeds for all users
	return GetAllFeedsByUser(db, 0)
}

// GetAllFeedsByUser returns all feeds for a specific user
func GetAllFeedsByUser(db *gorm.DB, userID uint) ([]models.Feed, error) {
	var feeds []models.Feed
	query := db
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	err := query.Find(&feeds).Error
	return feeds, err
}

func AddFeedURL(db *gorm.DB, url string) (*models.Feed, []models.Item, error) {
	// For backward compatibility, create feed for admin (userID=1)
	feed := &models.Feed{URL: url, IsActive: true, UserID: 1}
	err := db.FirstOrCreate(feed, models.Feed{URL: url, UserID: 1}).Error
	if err != nil {
		return nil, nil, err
	}
	return feed, nil, nil
}

// AddFeedURLForUser adds a feed URL for a specific user
func AddFeedURLForUser(db *gorm.DB, url string, userID uint) (*models.Feed, []models.Item, error) {
	feed := &models.Feed{URL: url, IsActive: true, UserID: userID}
	err := db.FirstOrCreate(feed, models.Feed{URL: url, UserID: userID}).Error
	if err != nil {
		return nil, nil, err
	}
	return feed, nil, nil
}

func MarkItemAsRead(db *gorm.DB, id uint) error {
	// For backward compatibility
	return MarkItemAsReadForUser(db, id, 0)
}

// MarkItemAsReadForUser marks an item as read, verifying user ownership
func MarkItemAsReadForUser(db *gorm.DB, id uint, userID uint) error {
	// If userID > 0, verify the item belongs to the user
	if userID > 0 {
		var count int64
		db.Model(&models.Item{}).Joins("JOIN feeds ON feeds.id = items.feed_id").
			Where("items.id = ? AND feeds.user_id = ?", id, userID).
			Count(&count)
		if count == 0 {
			return ErrUnauthorized
		}
	}
	return db.Model(&models.Item{}).Where("id = ?", id).Update("read", true).Error
}

func MarkItemAsUnread(db *gorm.DB, id uint) error {
	// For backward compatibility
	return MarkItemAsUnreadForUser(db, id, 0)
}

// MarkItemAsUnreadForUser marks an item as unread, verifying user ownership
func MarkItemAsUnreadForUser(db *gorm.DB, id uint, userID uint) error {
	// If userID > 0, verify the item belongs to the user
	if userID > 0 {
		var count int64
		db.Model(&models.Item{}).Joins("JOIN feeds ON feeds.id = items.feed_id").
			Where("items.id = ? AND feeds.user_id = ?", id, userID).
			Count(&count)
		if count == 0 {
			return ErrUnauthorized
		}
	}
	return db.Model(&models.Item{}).Where("id = ?", id).Update("read", false).Error
}

func GetFilteredItemsByRead(db *gorm.DB, limit, offset int, showRead bool, feedIDs []uint) ([]models.Item, int64, error) {
	// For backward compatibility
	return GetFilteredItemsByReadAndUser(db, limit, offset, showRead, feedIDs, 0)
}

// GetFilteredItemsByReadAndUser filters items by read status, feed IDs, and user
func GetFilteredItemsByReadAndUser(db *gorm.DB, limit, offset int, showRead bool, feedIDs []uint, userID uint) ([]models.Item, int64, error) {
	var items []models.Item
	var count int64
	query := db.Model(&models.Item{}).Joins("JOIN feeds AS f ON f.id = items.feed_id")

	// Filter by user if userID > 0
	if userID > 0 {
		query = query.Where("f.user_id = ?", userID)
	}

	// Filter by read status
	if !showRead {
		query = query.Where("items.read = ?", false)
	}

	// Filter by feed IDs
	if len(feedIDs) > 0 {
		query = query.Where("items.feed_id IN ?", feedIDs)
	}

	query.Count(&count)
	query.Order("published_at desc").Limit(limit).Offset(offset).Find(&items)

	return items, count, nil
}

func DeleteFeed(db *gorm.DB, feedID uint) error {
	// For backward compatibility
	return DeleteFeedForUser(db, feedID, 0)
}

// DeleteFeedForUser deletes a feed owned by a specific user
func DeleteFeedForUser(db *gorm.DB, feedID uint, userID uint) error {
	// Verify feed belongs to user if userID > 0
	if userID > 0 {
		var count int64
		db.Model(&models.Feed{}).Where("id = ? AND user_id = ?", feedID, userID).Count(&count)
		if count == 0 {
			return ErrUnauthorized
		}
	}
	// Delete all items from this feed first
	if err := db.Where("feed_id = ?", feedID).Delete(&models.Item{}).Error; err != nil {
		return err
	}
	// Then delete the feed
	return db.Delete(&models.Feed{}, feedID).Error
}

func UpdateFeed(db *gorm.DB, feed *models.Feed) error {
	// For backward compatibility, check if user has access
	// If feed.UserID is set, verify ownership
	if feed.UserID > 0 {
		var existingFeed models.Feed
		if err := db.First(&existingFeed, feed.ID).Error; err == nil {
			if existingFeed.UserID != feed.UserID {
				return ErrUnauthorized
			}
		}
	}
	return db.Save(feed).Error
}

// UpdateFeedForUser updates a feed owned by a specific user
func UpdateFeedForUser(db *gorm.DB, feed *models.Feed, userID uint) error {
	// Verify feed belongs to user
	var existingFeed models.Feed
	if err := db.First(&existingFeed, feed.ID).Error; err != nil {
		return err
	}
	if existingFeed.UserID != userID {
		return ErrUnauthorized
	}
	feed.UserID = userID
	return db.Save(feed).Error
}
