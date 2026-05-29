package models

import "time"

type Feed struct {
	ID        uint   `gorm:"primarykey"`
	URL       string `gorm:"unique;not null"`
	Title     string
	Link      string
	Type      string `gorm:"default:rss"` // rss, atom, json
	IsActive  bool   `gorm:"default:true"`
	Include   bool   `gorm:"default:true"` // Include in fusion
	LastFetch time.Time
	Error     string
	// WebSub fields
	HubURL     string // URL of the WebSub hub
	TopicURL   string // Topic URL for this feed
	Subscribed bool   `gorm:"default:false"` // Whether we're subscribed to WebSub
	Secret     string // Secret for WebSub verification
}

type Item struct {
	ID          uint   `gorm:"primarykey"`
	FeedID      uint   `gorm:"index"`
	FeedTitle   string `gorm:"-"` // Transient, from Feed
	Title       string
	Link        string    `gorm:"unique;not null"`
	Description string    `gorm:"type:text"`
	Content     string    `gorm:"type:text"`
	ImageURL    string    `gorm:"type:text"` // URL de l'image/thumbnail
	PublishedAt time.Time `gorm:"index"`
	Read        bool      `gorm:"default:false"`
	Guid        string    `gorm:"unique;not null"` // Unique identifier from feed
}
