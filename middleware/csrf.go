package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CSRFToken represents a CSRF token with its expiration
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

var (
	csrfTokens = make(map[string]*CSRFToken)
	csrfMutex  sync.RWMutex
)

// CSRFProtection creates a CSRF protection middleware
func CSRFProtection() fiber.Handler {
	// Cleanup expired tokens every 10 minutes
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			csrfMutex.Lock()
			for sessionID, token := range csrfTokens {
				if time.Now().After(token.ExpiresAt) {
					delete(csrfTokens, sessionID)
				}
			}
			csrfMutex.Unlock()
		}
	}()

	return func(c *fiber.Ctx) error {
		// Skip CSRF check for GET, HEAD, OPTIONS
		if c.Method() == "GET" || c.Method() == "HEAD" || c.Method() == "OPTIONS" {
			return c.Next()
		}

		// Get session ID from cookies
		sessionID := c.Cookies("session_id")
		if sessionID == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token validation failed: no session",
			})
		}

		// Get CSRF token from header
		csrfToken := c.Get("X-CSRF-Token")
		if csrfToken == "" {
			// Try to get from form data
			csrfToken = c.FormValue("csrf_token")
		}

		if csrfToken == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token validation failed: token not found",
			})
		}

		// Validate token
		csrfMutex.RLock()
		storedToken, exists := csrfTokens[sessionID]
		csrfMutex.RUnlock()

		if !exists || storedToken.Token != csrfToken || time.Now().After(storedToken.ExpiresAt) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token validation failed: invalid or expired token",
			})
		}

		return c.Next()
	}
}

// GenerateCSRFToken generates a new CSRF token for a session
func GenerateCSRFToken(sessionID string) string {
	// Generate random token
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)

	// Store token with expiration
	csrfMutex.Lock()
	csrfTokens[sessionID] = &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	csrfMutex.Unlock()

	return token
}

// GetCSRFToken retrieves an existing CSRF token for a session or generates a new one
func GetCSRFToken(sessionID string) string {
	csrfMutex.RLock()
	token, exists := csrfTokens[sessionID]
	csrfMutex.RUnlock()

	if exists && time.Now().Before(token.ExpiresAt) {
		return token.Token
	}

	return GenerateCSRFToken(sessionID)
}
