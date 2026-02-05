package api

import (
	"fmt"
	"lilmail/config"
	"lilmail/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// AttachmentHandler handles attachment-related requests
type AttachmentHandler struct {
	store  *session.Store
	config *config.Config
}

// NewAttachmentHandler creates a new attachment handler
func NewAttachmentHandler(store *session.Store, cfg *config.Config) *AttachmentHandler {
	return &AttachmentHandler{
		store:  store,
		config: cfg,
	}
}

// HandleDownload serves an attachment for download
func (h *AttachmentHandler) HandleDownload(c *fiber.Ctx) error {
	emailID := c.Params("email_id")
	attachmentIndex := c.Params("index")
	
	if emailID == "" || attachmentIndex == "" {
		return utils.BadRequestError("Email ID and attachment index are required", nil)
	}
	
	folderName := c.Query("folder", "INBOX")
	
	// Get session credentials
	credentials, err := GetCredentials(c, h.store, h.config.Encryption.Key)
	if err != nil {
		return utils.UnauthorizedError("Invalid session", err)
	}
	
	// Create IMAP client
	client, err := createIMAPClientFromCredentials(credentials, h.config)
	if err != nil {
		return utils.InternalServerError("Failed to connect to server", err)
	}
	defer client.Close()
	
	// Fetch email
	email, err := client.FetchSingleMessage(folderName, emailID)
	if err != nil {
		return utils.InternalServerError(fmt.Sprintf("Failed to fetch email: %v", err), err)
	}
	
	// Get attachment by index
	var index int
	fmt.Sscanf(attachmentIndex, "%d", &index)
	
	if index < 0 || index >= len(email.Attachments) {
		return utils.NotFoundError("Attachment not found", nil)
	}
	
	attachment := email.Attachments[index]
	
	// Set headers
	c.Set("Content-Type", attachment.ContentType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename))
	c.Set("Content-Length", fmt.Sprintf("%d", len(attachment.Content)))
	
	return c.Send(attachment.Content)
}

// HandlePreview serves an attachment for preview (images, PDFs)
func (h *AttachmentHandler) HandlePreview(c *fiber.Ctx) error {
	emailID := c.Params("email_id")
	attachmentIndex := c.Params("index")
	
	if emailID == "" || attachmentIndex == "" {
		return utils.BadRequestError("Email ID and attachment index are required", nil)
	}
	
	folderName := c.Query("folder", "INBOX")
	
	// Get session credentials
	credentials, err := GetCredentials(c, h.store, h.config.Encryption.Key)
	if err != nil {
		return utils.UnauthorizedError("Invalid session", err)
	}
	
	// Create IMAP client
	client, err := createIMAPClientFromCredentials(credentials, h.config)
	if err != nil {
		return utils.InternalServerError("Failed to connect to server", err)
	}
	defer client.Close()
	
	// Fetch email
	email, err := client.FetchSingleMessage(folderName, emailID)
	if err != nil {
		return utils.InternalServerError(fmt.Sprintf("Failed to fetch email: %v", err), err)
	}
	
	// Get attachment by index
	var index int
	fmt.Sscanf(attachmentIndex, "%d", &index)
	
	if index < 0 || index >= len(email.Attachments) {
		return utils.NotFoundError("Attachment not found", nil)
	}
	
	attachment := email.Attachments[index]
	
	// Set headers for inline display
	c.Set("Content-Type", attachment.ContentType)
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", attachment.Filename))
	c.Set("Content-Length", fmt.Sprintf("%d", len(attachment.Content)))
	
	return c.Send(attachment.Content)
}
