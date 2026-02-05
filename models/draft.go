package models

import "time"

// Draft represents a saved email draft
type Draft struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	To        string    `json:"to"`
	Cc        string    `json:"cc"`
	Bcc       string    `json:"bcc"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	IsHTML    bool      `json:"is_html"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
