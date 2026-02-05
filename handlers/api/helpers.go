package api

import (
	"fmt"
	"lilmail/config"
)

// createIMAPClientFromCredentials creates an IMAP client from credentials
func createIMAPClientFromCredentials(creds *Credentials, cfg *config.Config) (*Client, error) {
	if creds == nil {
		return nil, fmt.Errorf("credentials cannot be nil")
	}

	var username string
	if cfg.Server.UsernameIsEmail {
		username = creds.Email
	} else {
		username = GetUsernameFromEmail(creds.Email)
	}

	if username == "" {
		return nil, fmt.Errorf("invalid email format")
	}

	return NewClient(
		cfg.IMAP.Server,
		cfg.IMAP.Port,
		username,
		creds.Password,
	)
}
