package models

import "time"

// User represents an authenticated user with their feeds
type User struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"` // Never return password in JSON
	IsAdmin   bool      `json:"is_admin" gorm:"default:false"`
	Feeds     []Feed    `json:"feeds,omitempty" gorm:"foreignKey:UserID"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// UserCredentials for login and registration
type UserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserUpdate for updating user information (without password)
type UserUpdate struct {
	Username string `json:"username,omitempty"`
	IsAdmin  *bool  `json:"is_admin,omitempty"`
}

// PasswordChange request
type PasswordChange struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
