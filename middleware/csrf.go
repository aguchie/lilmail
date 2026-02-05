package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"

	"github.com/gofiber/fiber/v2"
)

// CSRFConfig holds CSRF protection configuration
type CSRFConfig struct {
	TokenLength  int
	CookieName   string
	HeaderName   string
	ContextKey   string
	CookieMaxAge int
	Skipper      func(*fiber.Ctx) bool
}

// DefaultCSRFConfig returns default CSRF configuration
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		TokenLength:  32,
		CookieName:   "csrf_token",
		HeaderName:   "X-CSRF-Token",
		ContextKey:   "csrf",
		CookieMaxAge: 3600, // 1 hour
		Skipper:      nil,
	}
}

// CSRFProtection creates CSRF protection middleware
func CSRFProtection(config ...CSRFConfig) fiber.Handler {
	cfg := DefaultCSRFConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Skip if skipper function returns true
		if cfg.Skipper != nil && cfg.Skipper(c) {
			return c.Next()
		}

		// Skip GET, HEAD, OPTIONS requests
		if c.Method() == fiber.MethodGet ||
			c.Method() == fiber.MethodHead ||
			c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		// Get token from cookie
		cookieToken := c.Cookies(cfg.CookieName)

		// Get token from header
		headerToken := c.Get(cfg.HeaderName)

		// Validate token
		if cookieToken == "" || headerToken == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token missing",
			})
		}

		if !tokensEqual(cookieToken, headerToken) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token mismatch",
			})
		}

		return c.Next()
	}
}

// GenerateCSRFToken generates a new CSRF token and sets it in a cookie
func GenerateCSRFToken(c *fiber.Ctx, config ...CSRFConfig) string {
	cfg := DefaultCSRFConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Generate random token
	token := generateToken(cfg.TokenLength)

	// Set cookie
	c.Cookie(&fiber.Cookie{
		Name:     cfg.CookieName,
		Value:    token,
		MaxAge:   cfg.CookieMaxAge,
		HTTPOnly: true,
		SameSite: "Strict",
		Secure:   false, // Set to true in production with HTTPS
	})

	// Store in context
	c.Locals(cfg.ContextKey, token)

	return token
}

// generateToken generates a random token
func generateToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// tokensEqual performs constant-time comparison of tokens
func tokensEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
