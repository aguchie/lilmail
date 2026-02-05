package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/time/rate"
)

// RateLimiter creates a rate limiting middleware
func RateLimiter(requests int, duration time.Duration) fiber.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		clients = make(map[string]*client)
		mu      sync.Mutex
	)

	// Cleanup old clients every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *fiber.Ctx) error {
		ip := c.IP()

		mu.Lock()
		cl, exists := clients[ip]
		if !exists {
			// Create new limiter: requests per duration
			limiter := rate.NewLimiter(rate.Every(duration/time.Duration(requests)), requests)
			cl = &client{limiter: limiter}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		mu.Unlock()

		if !cl.limiter.Allow() {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		}

		return c.Next()
	}
}
