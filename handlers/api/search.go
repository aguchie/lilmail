package api

import (
	"fmt"
	"time"
	"lilmail/config"
	"lilmail/models"

	"github.com/emersion/go-imap"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type SearchHandler struct {
	store  *session.Store
	config *config.Config
}

func NewSearchHandler(store *session.Store, config *config.Config) *SearchHandler {
	return &SearchHandler{
		store:  store,
		config: config,
	}
}

	// HandleSearch performs search on IMAP server
	func (h *SearchHandler) HandleSearch(c *fiber.Ctx) error {
		// Parse search parameters
		query := c.FormValue("query")
		folder := c.Query("folder", "INBOX")
		scope := c.FormValue("scope", "all")
		dateFromStr := c.FormValue("dateFrom")
		dateToStr := c.FormValue("dateTo")
		hasAttachment := c.FormValue("hasAttachment") == "on" // HTML checkbox sends "on"

		// Create IMAP Client from session credentials
		creds, err := GetCredentials(c, h.store, h.config.Encryption.Key)
		if err != nil {
			return c.Status(401).SendString("Unauthorized")
		}

		client, err := createIMAPClientFromCredentials(creds, h.config)
		if err != nil {
			return c.Status(500).SendString("Failed to connect to mail server")
		}
		defer client.Close()

		// Perform Search
		criteria := imap.NewSearchCriteria()
		
		if query != "" {
			switch scope {
			case "from":
				criteria.Header.Add("From", query)
			case "to":
				criteria.Header.Add("To", query)
			case "subject":
				criteria.Header.Add("Subject", query)
			case "body":
				criteria.Body = []string{query}
			default:
				// Search all reasonable fields
				// Note: Text criteria usually searches Subject, From, To, Cc, Bcc, and Body
				criteria.Text = []string{query}
			}
		}

		// Date Filters
		if dateFromStr != "" {
			if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
				criteria.Since = dateFrom
			}
		}
		if dateToStr != "" {
			if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
				// Search Before is strictly before, so we add 1 day to include the end date
				criteria.Before = dateTo.AddDate(0, 0, 1)
			}
		}

		// Attachment Filter
		// Note: IMAP doesn't have a standard HAS_ATTACHMENT flag.
		// Common workaround is checking Content-Type or Body structure.
		// Checking Header "Content-Type" for "multipart/mixed" is a common approximation.
		if hasAttachment {
			criteria.Header.Add("Content-Type", "multipart/mixed")
		}

		// Select folder
		_, err = client.client.Select(folder, false)
		if err != nil {
			return c.Status(500).SendString("Folder selection failed")
		}

		// Execute Search
		uids, err := client.client.Search(criteria)
		if err != nil {
			return c.Status(500).SendString("Search failed")
		}

		if len(uids) == 0 {
			// Return empty list partial
			return c.Render("partials/email-list", fiber.Map{
				"Emails":        []models.Email{},
				"CurrentFolder": folder,
				"Pagination":    nil,
			})
		}

		// Fetch messages for UIDs
		messages, err := client.FetchMessagesByUIDs(folder, uids)
		if err != nil {
			return c.Status(500).SendString(fmt.Sprintf("Failed to fetch search results: %v", err))
		}

		return c.Render("partials/email-list", fiber.Map{
			"Emails":        messages,
			"CurrentFolder": folder,
			"Pagination":    nil, // Search results are not paginated yet
		}, "")
	}
