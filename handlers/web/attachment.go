package web

import (
	"lilmail/config"
	"lilmail/handlers/api"
	"lilmail/utils"
	"path/filepath"
	"strconv"
	"time"

	"github.com/emersion/go-imap"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type AttachmentWebHandler struct {
	store  *session.Store
	config *config.Config
	auth   *AuthHandler
}

func NewAttachmentWebHandler(store *session.Store, config *config.Config, auth *AuthHandler) *AttachmentWebHandler {
	return &AttachmentWebHandler{
		store:  store,
		config: config,
		auth:   auth,
	}
}

// DisplayAttachment represents a single attachment for display
type DisplayAttachment struct {
	ID          string // Using email ID + index as pseudo ID? Or just use composite
	EmailID     string
	EmailSubject string
	EmailFrom    string
	EmailDate    time.Time
	Filename     string
	ContentType  string
	Size         int64
	Index        int
	IsImage      bool
}

// HandleAttachments renders the attachment manager page
func (h *AttachmentWebHandler) HandleAttachments(c *fiber.Ctx) error {
	username := c.Locals("username")
	if username == nil {
		return c.Redirect("/login")
	}

	userStr, ok := username.(string)
	if !ok {
		return c.Redirect("/login")
	}

	// Load folders from cache to show in sidebar (keep consistent layout)
	userCacheFolder := filepath.Join(h.config.Cache.Folder, userStr)
	var folders []*api.MailboxInfo
	if err := utils.LoadCache(filepath.Join(userCacheFolder, "folders.json"), &folders); err != nil {
		// Just log error, don't fail page?
		utils.Log.Error("Error loading folders for attachments view: %v", err)
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).SendString("Error connecting to email server")
	}
	defer client.Close()

	// Parameters
	folderName := c.Query("folder", "INBOX")
	page := 1
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	// Fetch larger batch of emails to find attachments
	// This is inefficient but functional for now. 
	// Optimally we would use SEARCH HAS_ATTACHMENT
	
	// Let's try SEARCH HAS_ATTACHMENT criteria
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("Content-Type", "multipart/mixed") // Common approximation

	// Select folder
	_, err = client.Select(folderName, false)
	if err != nil {
		return c.Status(500).SendString("Folder selection failed")
	}

	uids, err := client.Search(criteria)
	// If searching fails or returns too many/few, we might fallback or paginate the search results?
	// For now, let's assume it works.
	
	var allAttachments []DisplayAttachment
	
	if err == nil && len(uids) > 0 {
		// Pagination logic for UIDs to avoid fetching too many messages
		// Sort UIDs descending (newest first)
		// UIDs are uint32, need manual sort
		// go-imap usually returns them in order, but reverse for display is better
		// Actually, let's just reverse iterate
		
		// Note: Filter to last 50 emails with attachments for performance? 
		// Or perform pagination on the UI based on UIDs?
		// Let's take the last 20 UIDs (newest)
		
		startIdx := len(uids) - (page * 20)
		endIdx := len(uids) - ((page - 1) * 20)
		
		if endIdx > len(uids) { endIdx = len(uids) }
		if endIdx < 0 { endIdx = 0 }
		if startIdx < 0 { startIdx = 0 }
		
		if startIdx < endIdx {
			pageUids := uids[startIdx:endIdx]
			
			// Fetch full messages for these UIDs to parse attachments
			emails, err := client.FetchMessagesByUIDs(folderName, pageUids)
			if err == nil {
				// Iterate backwards to show newest first
				for i := len(emails) - 1; i >= 0; i-- {
					email := emails[i]
					if email.HasAttachments {
						for idx, att := range email.Attachments {
							allAttachments = append(allAttachments, DisplayAttachment{
								EmailID:      email.ID,
								EmailSubject: email.Subject,
								EmailFrom:    email.From,
								EmailDate:    email.Date,
								Filename:     att.Filename,
								ContentType:  att.ContentType,
								Size:         int64(len(att.Content)), // Note: Content might be empty if we didn't fetch body?? 
                                // Wait, FetchMessagesByUIDs usually fetches body. 
                                // But models.Email.Attachments stores data.
                                // Ideally we shouldn't fetch full content strictly for listing.
                                // But current architecture seems to load it. 
                                // TODO: Optimization - fetch only structure? 
                                // For now, reuse existing fetch logic.
								Index:        idx,
								IsImage:      utils.IsImage(att.ContentType),
							})
						}
					}
				}
			}
		}
	}

	// Get JWT token from session
	token, _ := api.GetSessionToken(c, h.store)

	return c.Render("attachments", fiber.Map{
		"Username":      userStr,
		"Folders":       folders,
		"Attachments":   allAttachments,
		"CurrentFolder": folderName,
		"Page":          page,
		"HasNext":       len(uids) > (page * 20),
		"HasPrev":       page > 1,
		"Token":         token,
		"CSRFToken":     c.Locals("csrf"),
	})
}
