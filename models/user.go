package models

import "time"

// User represents a user in the multi-user system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose in JSON
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"` // "admin", "editor", "viewer"
	Language     string    `json:"language"`
	Theme        string    `json:"theme"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLoginAt  time.Time `json:"last_login_at,omitempty"`
}

// UserSettings represents user-specific settings
type UserSettings struct {
	UserID              string `json:"user_id"`
	EmailsPerPage       int    `json:"emails_per_page"`
	DefaultFolder       string `json:"default_folder"`
	ShowPreview         bool   `json:"show_preview"`
	AutoMarkAsRead      bool   `json:"auto_mark_as_read"`
	EnableNotifications bool   `json:"enable_notifications"`
}
