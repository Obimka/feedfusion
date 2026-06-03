package models

// Category represents a user-defined category for organizing feeds
type Category struct {
	ID      uint   `gorm:"primarykey"`
	UserID  uint   `gorm:"index;not null"`  // Owner of this category
	Name    string `gorm:"not null"`
	Color   string `gorm:"default:#6B8EBA"`  // Hex color code (e.g., #FF5733)
	Feeds   []Feed `gorm:"foreignKey:CategoryID"`
}
