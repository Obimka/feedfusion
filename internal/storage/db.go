package storage

import (
	"rss-aggregator/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&models.Feed{}, &models.Item{})
	return db, err
}

func AddFeed(db *gorm.DB, feed *models.Feed, items []models.Item) error {
	tx := db.Begin()

	// Try to find existing feed
	var existingFeed models.Feed
	if err := tx.Where(models.Feed{URL: feed.URL}).First(&existingFeed).Error; err == nil {
		feed.ID = existingFeed.ID
		// Preserve Include setting from existing feed
		if feed.Include == false {
			feed.Include = existingFeed.Include
		}
		// Update other fields
		if err := tx.Model(feed).Select("title", "link", "type", "is_active", "last_fetch", "error", "include").Updates(feed).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Create(feed).Error; err != nil {
			return err
		}
	}

	for i := range items {
		items[i].FeedID = feed.ID
		// Try to find existing item
		var existingItem models.Item
		err := tx.Where(models.Item{Guid: items[i].Guid}).First(&existingItem).Error
		if err == nil {
			// Item exists, update if image URL is missing or different
			if existingItem.ImageURL == "" && items[i].ImageURL != "" {
				existingItem.ImageURL = items[i].ImageURL
				if err := tx.Save(&existingItem).Error; err != nil {
					return err
				}
			}
		} else {
			// Create new item
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
	}

	return tx.Commit().Error
}

func GetAllItems(db *gorm.DB, limit, offset int) ([]models.Item, int64, error) {
	return GetFilteredItems(db, limit, offset, nil)
}

func GetFilteredItems(db *gorm.DB, limit, offset int, feedIDs []uint) ([]models.Item, int64, error) {
	var items []models.Item
	var count int64
	query := db.Model(&models.Item{})

	// If feedIDs is provided and not empty, filter by those feeds
	// If feedIDs is empty slice but we want to filter, we need to handle it differently
	if len(feedIDs) > 0 {
		query = query.Where("feed_id IN ?", feedIDs)
	} else {
		// Check if this is an "included only" filter with no feeds included
		// In this case, we should return empty results
		// But we can't distinguish here, so we let the caller handle it
	}

	query.Count(&count)
	query.Order("published_at desc").Limit(limit).Offset(offset).Find(&items)

	return items, count, nil
}

func GetFilteredItemsByInclude(db *gorm.DB, limit, offset int, includeOnly bool) ([]models.Item, int64, error) {
	var items []models.Item
	var count int64
	query := db.Model(&models.Item{})

	if includeOnly {
		// Filter items from feeds where include = true
		query = query.Where("feed_id IN (SELECT id FROM feeds WHERE include = true)")
	}

	query.Count(&count)
	query.Order("published_at desc").Limit(limit).Offset(offset).Find(&items)

	return items, count, nil
}

func ToggleFeedInclude(db *gorm.DB, feedID uint) error {
	return db.Model(&models.Feed{}).Where("id = ?", feedID).Update("include", gorm.Expr("NOT include")).Error
}

func GetIncludedFeeds(db *gorm.DB) ([]models.Feed, error) {
	var feeds []models.Feed
	err := db.Where("include = ?", true).Find(&feeds).Error
	return feeds, err
}

func GetFeedByID(db *gorm.DB, id uint) (*models.Feed, error) {
	var feed models.Feed
	err := db.First(&feed, id).Error
	return &feed, err
}

func GetAllFeeds(db *gorm.DB) ([]models.Feed, error) {
	var feeds []models.Feed
	err := db.Find(&feeds).Error
	return feeds, err
}

func AddFeedURL(db *gorm.DB, url string) (*models.Feed, []models.Item, error) {
	feed := &models.Feed{URL: url, IsActive: true}
	err := db.FirstOrCreate(feed, models.Feed{URL: url}).Error
	if err != nil {
		return nil, nil, err
	}
	return feed, nil, nil
}

func MarkItemAsRead(db *gorm.DB, id uint) error {
	return db.Model(&models.Item{}).Where("id = ?", id).Update("read", true).Error
}

func MarkItemAsUnread(db *gorm.DB, id uint) error {
	return db.Model(&models.Item{}).Where("id = ?", id).Update("read", false).Error
}

func GetFilteredItemsByRead(db *gorm.DB, limit, offset int, showRead bool, feedIDs []uint) ([]models.Item, int64, error) {
	var items []models.Item
	var count int64
	query := db.Model(&models.Item{})

	// Filter by read status
	if !showRead {
		query = query.Where("read = ?", false)
	}

	// Filter by feed IDs
	if len(feedIDs) > 0 {
		query = query.Where("feed_id IN ?", feedIDs)
	}

	query.Count(&count)
	query.Order("published_at desc").Limit(limit).Offset(offset).Find(&items)

	return items, count, nil
}

func DeleteFeed(db *gorm.DB, feedID uint) error {
	// Delete all items from this feed first
	if err := db.Where("feed_id = ?", feedID).Delete(&models.Item{}).Error; err != nil {
		return err
	}
	// Then delete the feed
	return db.Delete(&models.Feed{}, feedID).Error
}

func UpdateFeed(db *gorm.DB, feed *models.Feed) error {
	return db.Save(feed).Error
}
