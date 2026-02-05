package api

import (
	"fmt"
	"lilmail/config"
	"lilmail/models"
	"lilmail/utils"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// SearchHandler handles email search requests
type SearchHandler struct {
	store  *session.Store
	config *config.Config
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(store *session.Store, cfg *config.Config) *SearchHandler {
	return &SearchHandler{
		store:  store,
		config: cfg,
	}
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query       string `json:"query"`
	Folder      string `json:"folder"`
	SearchIn    string `json:"search_in"`    // "all", "from", "to", "subject", "body"
	HasAttachment bool   `json:"has_attachment"`
	DateFrom    string `json:"date_from"`
	DateTo      string `json:"date_to"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
}

// HandleSearch processes an email search request
func (h *SearchHandler) HandleSearch(c *fiber.Ctx) error {
	// Parse request
	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	// Default values
	if req.Folder == "" {
		req.Folder = "INBOX"
	}
	if req.SearchIn == "" {
		req.SearchIn = "all"
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 50
	}

	// Get session credentials
	credentials, err := GetCredentials(c, h.store, h.config.Encryption.Key)
	if err != nil {
		return utils.UnauthorizedError("Invalid session", err)
	}

	// Create IMAP client
	client, err := createIMAPClientFromCredentials(credentials, h.config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to connect to email server",
		})
	}
	defer client.Close()

	// Fetch all messages from folder
	allEmails, err := client.FetchMessages(req.Folder, 1000) // Limit to 1000 messages
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to fetch messages: %v", err),
		})
	}

	// Filter emails based on search criteria
	filteredEmails := h.filterEmails(allEmails, req)

	// Parse date filters if provided
	var dateFrom, dateTo time.Time
	if req.DateFrom != "" {
		dateFrom, _ = time.Parse("2006-01-02", req.DateFrom)
	}
	if req.DateTo != "" {
		dateTo, _ = time.Parse("2006-01-02", req.DateTo)
	}

	// Apply date filters
	if !dateFrom.IsZero() || !dateTo.IsZero() {
		filtered := []models.Email{}
		for _, email := range filteredEmails {
			if !dateFrom.IsZero() && email.Date.Before(dateFrom) {
				continue
			}
			if !dateTo.IsZero() && email.Date.After(dateTo.Add(24*time.Hour)) {
				continue
			}
			filtered = append(filtered, email)
		}
		filteredEmails = filtered
	}

	// Calculate pagination
	totalEmails := uint32(len(filteredEmails))
	
	// Get page of results
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(filteredEmails) {
		start = len(filteredEmails)
	}
	if end > len(filteredEmails) {
		end = len(filteredEmails)
	}

	pageEmails := filteredEmails[start:end]

	// Create response
	result := models.NewPaginatedEmails(
		pageEmails,
		uint32(req.Page),
		uint32(req.PageSize),
		totalEmails,
	)

	utils.Log.Info("Search completed: query='%s' folder='%s' results=%d", req.Query, req.Folder, totalEmails)

	return c.JSON(result)
}

// filterEmails filters emails based on search criteria
func (h *SearchHandler) filterEmails(emails []models.Email, req SearchRequest) []models.Email {
	if req.Query == "" && !req.HasAttachment {
		return emails
	}

	query := strings.ToLower(req.Query)
	filtered := []models.Email{}

	for _, email := range emails {
		match := false

		// Check attachment filter
		if req.HasAttachment && !email.HasAttachments {
			continue
		}

		// If no query, just check attachment filter
		if req.Query == "" {
			filtered = append(filtered, email)
			continue
		}

		// Search based on searchIn parameter
		switch req.SearchIn {
		case "from":
			match = strings.Contains(strings.ToLower(email.From), query) ||
				strings.Contains(strings.ToLower(email.FromName), query)
		case "to":
			match = strings.Contains(strings.ToLower(email.To), query)
		case "subject":
			match = strings.Contains(strings.ToLower(email.Subject), query)
		case "body":
			match = strings.Contains(strings.ToLower(email.Body), query)
		default: // "all"
			match = strings.Contains(strings.ToLower(email.From), query) ||
				strings.Contains(strings.ToLower(email.FromName), query) ||
				strings.Contains(strings.ToLower(email.To), query) ||
				strings.Contains(strings.ToLower(email.Subject), query) ||
				strings.Contains(strings.ToLower(email.Body), query)
		}

		if match {
			filtered = append(filtered, email)
		}
	}

	return filtered
}
