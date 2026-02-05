package models

import "time"

// Thread represents an email thread
type EmailThread struct {
	ID           string    `json:"id"`
	Subject      string    `json:"subject"`
	Participants []string  `json:"participants"`
	MessageCount int       `json:"message_count"`
	LastDate     time.Time `json:"last_date"`
	Messages     []Email   `json:"messages"`
	Unread       bool      `json:"unread"`
	HasAttachment bool      `json:"has_attachment"`
}

// Label represents an email label/tag
type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"` // Hex color code
}

// EmailLabel represents the association between an email and a label
type EmailLabel struct {
	EmailID string `json:"email_id"`
	LabelID string `json:"label_id"`
}
