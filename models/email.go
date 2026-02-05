package models

import (
	"html/template"
	"time"
)

// Email represents an email message
type Email struct {
	ID              string        `json:"id"`
	From            string        `json:"from"`
	FromName        string        `json:"from_name"`
	To              string        `json:"to"`
	ToNames         []string      `json:"to_names"`
	Cc              string        `json:"cc"`
	Subject         string        `json:"subject"`
	Date            time.Time     `json:"date"`
	Body            string        `json:"body"`
	HTML            template.HTML `json:"html"`
	Preview         string        `json:"preview"`
	Flags           []string      `json:"flags"`
	Attachments     []Attachment  `json:"attachments"`
	HasAttachments  bool          `json:"has_attachments"`
	
	// Threading fields
	MessageID       string        `json:"message_id"`
	InReplyTo       string        `json:"in_reply_to"`
	References      []string      `json:"references"`
	ThreadID        string        `json:"thread_id"`
	
	// Labels
	Labels          []Label       `json:"labels"`
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Content     []byte `json:"-"` // Excluded from JSON
}
