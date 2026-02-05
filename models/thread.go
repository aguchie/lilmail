package models

import "time"

// Thread represents an email thread
type EmailThread struct {
	ID           string    `json:"id"`
	Subject      string    `json:"subject"`
	Folder       string    `json:"folder"`
	UserID       string    `json:"user_id"`
	MessageIDs   []string  `json:"message_ids"` // UIDs of emails in thread
	Participants []string  `json:"participants"`
	MessageCount int       `json:"message_count"`
	Count        int       `json:"count"`
	Unread       int       `json:"unread"`
	LastDate     time.Time `json:"last_date"`
	LatestDate   time.Time `json:"latest_date"`
	Messages     []Email   `json:"messages"`
	HasAttachment bool      `json:"has_attachment"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ThreadContainer is used by the JWZ threading algorithm
type ThreadContainer struct {
	Message   *Email
	MessageID string
	Parent    *ThreadContainer
	Children  []*ThreadContainer
	IsDummy   bool
}


