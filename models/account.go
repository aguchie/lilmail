package models

import "time"

// Account represents an email account configuration
type Account struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	IMAPServer  string    `json:"imap_server"`
	IMAPPort    int       `json:"imap_port"`
	IMAPSSL     bool      `json:"imap_ssl"`
	SMTPServer  string    `json:"smtp_server"`
	SMTPPort    int       `json:"smtp_port"`
	SMTPSSL     bool      `json:"smtp_ssl"`
	Username    string    `json:"username"`
	Password    string    `json:"-"` // Never expose in JSON
	DisplayName string    `json:"display_name"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AccountCredentials represents decrypted account credentials
type AccountCredentials struct {
	ID         string
	Email      string
	IMAPServer string
	IMAPPort   int
	IMAPSSL    bool
	SMTPServer string
	SMTPPort   int
	SMTPSSL    bool
	Username   string
	Password   string
}
