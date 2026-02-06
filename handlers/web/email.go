// handlers/web/email.go
package web

import (
	"fmt"
	"io"
	"lilmail/config"
	"lilmail/handlers/api"
	"lilmail/storage"
	"lilmail/utils"
	"log"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type EmailHandler struct {
	store         *session.Store
	config        *config.Config
	auth          *AuthHandler
	notify        *api.NotificationHandler
	threadStorage *storage.ThreadStorage
}

func NewEmailHandler(store *session.Store, config *config.Config, auth *AuthHandler, notify *api.NotificationHandler, threadStorage *storage.ThreadStorage) *EmailHandler {
	return &EmailHandler{
		store:         store,
		config:        config,
		auth:          auth,
		notify:        notify,
		threadStorage: threadStorage,
	}
}

// HandleInbox renders the main inbox page
func (h *EmailHandler) HandleInbox(c *fiber.Ctx) error {
	username := c.Locals("username")
	if username == nil {
		return c.Redirect("/login")
	}

	userStr, ok := username.(string)
	if !ok {
		return c.Redirect("/login")
	}

	// Load folders from cache
	userCacheFolder := filepath.Join(h.config.Cache.Folder, userStr)
	var folders []*api.MailboxInfo
	if err := utils.LoadCache(filepath.Join(userCacheFolder, "folders.json"), &folders); err != nil {
		return c.Status(500).SendString("Error loading folders")
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).SendString("Error connecting to email server")
	}
	defer client.Close()

	// Check if thread view is requested
	viewMode := c.Query("view", "flat")
	isThreaded := viewMode == "threaded"

	// Get JWT token for API requests
	token, err := api.GetSessionToken(c, h.store)
	if err != nil {
		return c.Redirect("/login")
	}

	// Get email from session for UI
	sess, _ := h.store.Get(c)
	email := sess.Get("email")
	
	// Get UserID from session for storage
	var userID string
	if uid := sess.Get("userId"); uid != nil {
		userID = uid.(string)
	} else {
		// Fallback to username if userId not set (should be set in auth)
		userID = userStr
	}

	// Parse page number
	page := 1
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	pageSize := 50

	if isThreaded {
		// Fetch threaded messages
		// 1. Try to get from storage first
		threads, err := h.threadStorage.GetThreadsByFolder(userID, "INBOX")
		
		// If cache miss or empty, fetch from IMAP
		if err != nil || len(threads) == 0 {
			apiThreads, err := client.FetchThreads("INBOX", 100) // Threading currently fetches recent 100
			if err != nil {
				return c.Status(500).SendString("Error fetching threads")
			}
			
			// Save to storage
			for _, t := range apiThreads {
				t.UserID = userID
				t.Folder = "INBOX"
				h.threadStorage.SaveThread(t)
			}
			threads = apiThreads
		}

		return c.Render("inbox", fiber.Map{
			"Username":      userStr,
			"Email":         email,
			"Folders":       folders,
			"Threads":       threads,
			"CurrentFolder": "INBOX",
			"Token":         token,
			"ViewMode":      "threaded",
			"CSRFToken":     c.Locals("csrf"),
		})
	} else {
		// Fetch paginated messages
		paginated, err := client.FetchMessagesPaginated("INBOX", uint32(page), uint32(pageSize))
		if err != nil {
			return c.Status(500).SendString("Error fetching emails")
		}

		return c.Render("inbox", fiber.Map{
			"Username":      userStr,
			"Email":         email,
			"Folders":       folders,
			"Emails":        paginated.Emails,
			"Pagination":    paginated,
			"CurrentFolder": "INBOX",
			"Token":         token,
			"ViewMode":      "flat",
			"CSRFToken":     c.Locals("csrf"),
		})
	}
}

// HandleFolder displays emails from a specific folder
func (h *EmailHandler) HandleFolder(c *fiber.Ctx) error {
	username := c.Locals("username")
	if username == nil {
		return c.Redirect("/login")
	}

	userStr, ok := username.(string)
	if !ok {
		return c.Redirect("/login")
	}

	folderName, err := url.QueryUnescape(c.Params("name"))
	if folderName == "" {
		return c.Redirect("/inbox")
	}

	// Load folders for sidebar
	userCacheFolder := filepath.Join(h.config.Cache.Folder, userStr)
	var folders []*api.MailboxInfo
	if err := utils.LoadCache(filepath.Join(userCacheFolder, "folders.json"), &folders); err != nil {
		return c.Status(500).SendString("Error loading folders")
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).SendString("Error connecting to email server")
	}
	defer client.Close()

	// Check if thread view is requested
	viewMode := c.Query("view", "flat")
	isThreaded := viewMode == "threaded"

	// Get JWT token for API requests
	token, err := api.GetSessionToken(c, h.store)
	if err != nil {
		return c.Redirect("/login")
	}

	// Get email from session for UI
	sess, _ := h.store.Get(c)
	email := sess.Get("email")
	
	// Get UserID from session for storage
	var userID string
	if uid := sess.Get("userId"); uid != nil {
		userID = uid.(string)
	} else {
		// Fallback to username if userId not set
		userID = userStr
	}

	// Parse page number
	page := 1
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	pageSize := 50

	if isThreaded {
		// Fetch threaded messages
		// 1. Try to get from storage first
		threads, err := h.threadStorage.GetThreadsByFolder(userID, folderName)
		
		// If cache miss or empty, fetch from IMAP
		if err != nil || len(threads) == 0 {
			apiThreads, err := client.FetchThreads(folderName, 100)
			if err != nil {
				return c.Status(500).SendString("Error fetching threads")
			}
			
			// Save to storage
			for _, t := range apiThreads {
				t.UserID = userID
				t.Folder = folderName
				h.threadStorage.SaveThread(t)
			}
			threads = apiThreads
		}

		return c.Render("inbox", fiber.Map{
			"Username":      userStr,
			"Email":         email,
			"Folders":       folders,
			"Threads":       threads,
			"CurrentFolder": folderName,
			"Token":         token,
			"ViewMode":      "threaded",
			"CSRFToken":     c.Locals("csrf"),
		})
	} else {
		// Fetch paginated messages
		paginated, err := client.FetchMessagesPaginated(folderName, uint32(page), uint32(pageSize))
		if err != nil {
			return c.Status(500).SendString("Error fetching emails")
		}

		return c.Render("inbox", fiber.Map{
			"Username":      userStr,
			"Email":         email,
			"Folders":       folders,
			"Emails":        paginated.Emails,
			"Pagination":    paginated,
			"CurrentFolder": folderName,
			"Token":         token,
			"ViewMode":      "flat",
			"CSRFToken":     c.Locals("csrf"),
		})
	}
}

// HandleEmailView handles the HTMX request for viewing a single email
func (h *EmailHandler) HandleEmailView(c *fiber.Ctx) error {
	// Validate Authorization header
	token := c.Get("Authorization")
	if token == "" || len(token) < 8 || token[:7] != "Bearer " {
		return c.Status(401).SendString("Unauthorized")
	}

	// Get folder and email ID
	folderName := c.Get("X-Folder")
	if folderName == "" {
		folderName = c.Query("folder")
		if folderName == "" {
			folderName = "INBOX"
		}
	}

	emailID := c.Params("id")
	if emailID == "" {
		return c.Status(400).SendString("Email ID required")
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error connecting to email server",
		})
	}
	defer client.Close()

	// Fetch the email
	email, err := client.FetchSingleMessage(folderName, emailID)
	if err != nil {
		log.Printf("Error fetching email %s from folder %s: %v", emailID, folderName, err)
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Error fetching email: %v", err),
		})
	}
	// Important: Set empty layout and only render the partial
	return c.Render("partials/email-viewer", fiber.Map{
		"Email":         email,
		"CurrentFolder": folderName,
		"Layout":        "", // This is crucial to prevent full HTML rendering
	}, "") // Add empty string as second argument to explicitly disable layout
}

// HandleDeleteEmail handles the email deletion request
func (h *EmailHandler) HandleDeleteEmail(c *fiber.Ctx) error {
	// Validate Authorization header
	token := c.Get("Authorization")
	if token == "" || len(token) < 8 || token[:7] != "Bearer " {
		return c.Status(401).SendString("Unauthorized")
	}

	// Validate JWT token
	_, err := api.ValidateToken(token[7:], h.config.JWT.Secret)
	if err != nil {
		return c.Status(401).SendString("Invalid token")
	}

	// Get folder and email ID
	folderName := c.Get("X-Folder")
	if folderName == "" {
		folderName = c.Query("folder")
		if folderName == "" {
			folderName = "INBOX"
		}
	}

	emailID := c.Params("id")
	if emailID == "" {
		return c.Status(400).SendString("Email ID required")
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error connecting to email server",
		})
	}
	defer client.Close()

	// Delete the email
	err = client.DeleteMessage(folderName, emailID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Error deleting email: %v", err),
		})
	}

	// Notify
	if userID, ok := c.Locals("username").(string); ok {
		h.notify.NotifyEmailDeleted(userID, emailID)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email deleted successfully",
	})
}

// HandleMarkRead marks an email as read
func (h *EmailHandler) HandleMarkRead(c *fiber.Ctx) error {
	// Validate Authorization header
	token := c.Get("Authorization")
	if token == "" || len(token) < 8 || token[:7] != "Bearer " {
		return c.Status(401).SendString("Unauthorized")
	}

	// Get folder and email ID
	folderName := c.Get("X-Folder")
	if folderName == "" {
		folderName = c.Query("folder")
		if folderName == "" {
			folderName = "INBOX"
		}
	}

	emailID := c.Params("id")
	if emailID == "" {
		return c.Status(400).SendString("Email ID required")
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error connecting to email server",
		})
	}
	defer client.Close()

	// Mark as read
	err = client.MarkMessageAsRead(folderName, emailID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Error marking email as read: %v", err),
		})
	}

	// Notify
	if userID, ok := c.Locals("username").(string); ok {
		h.notify.NotifyStatusChange(userID, emailID, "read")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email marked as read",
	})
}

// HandleMarkUnread marks an email as unread
func (h *EmailHandler) HandleMarkUnread(c *fiber.Ctx) error {
	// Validate Authorization header
	token := c.Get("Authorization")
	if token == "" || len(token) < 8 || token[:7] != "Bearer " {
		return c.Status(401).SendString("Unauthorized")
	}

	// Get folder and email ID
	folderName := c.Get("X-Folder")
	if folderName == "" {
		folderName = c.Query("folder")
		if folderName == "" {
			folderName = "INBOX"
		}
	}

	emailID := c.Params("id")
	if emailID == "" {
		return c.Status(400).SendString("Email ID required")
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error connecting to email server",
		})
	}
	defer client.Close()

	// Mark as unread
	err = client.MarkMessageAsUnread(folderName, emailID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Error marking email as unread: %v", err),
		})
	}

	// Notify
	if userID, ok := c.Locals("username").(string); ok {
		h.notify.NotifyStatusChange(userID, emailID, "unread")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email marked as unread",
	})
}

// handlers/web/email.go
// HandleFolderEmails handles template rendering for folder contents
func (h *EmailHandler) HandleFolderEmails(c *fiber.Ctx) error {
	folderName, err := url.QueryUnescape(c.Params("name"))
	if err != nil || folderName == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid folder name",
		})
	}

	username := c.Locals("username")
	if username == nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get JWT token for API requests
	token, err := api.GetSessionToken(c, h.store)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid session",
		})
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error connecting to email server",
		})
	}
	defer client.Close()

	// Parse page number
	page := 1
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	pageSize := 50

	// Fetch emails from the folder
	paginated, err := client.FetchMessagesPaginated(folderName, uint32(page), uint32(pageSize))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Error fetching emails: %v", err),
		})
	}

	// Add debug logging
	log.Printf("Folder: %s, Emails count: %d, Page: %d", folderName, len(paginated.Emails), page)

	return c.Render("partials/email-list", fiber.Map{
		"Emails":        paginated.Emails,
		"Pagination":    paginated,
		"CurrentFolder": folderName,
		"Token":         token,
	}, "") // Explicitly set no layout
}

// HandleComposeEmail handles the email composition and sending
func (h *EmailHandler) HandleComposeEmail(c *fiber.Ctx) error {

	// Parse multipart/form-data
	// Default max memory is 32MB
	form, err := c.MultipartForm()
	
	var to, cc, bcc, subject, body string
	var isHTML bool

	if err == nil && form != nil {
		if v, ok := form.Value["to"]; ok && len(v) > 0 { to = v[0] }
		if v, ok := form.Value["cc"]; ok && len(v) > 0 { cc = v[0] }
		if v, ok := form.Value["bcc"]; ok && len(v) > 0 { bcc = v[0] }
		if v, ok := form.Value["subject"]; ok && len(v) > 0 { subject = v[0] }
		if v, ok := form.Value["body"]; ok && len(v) > 0 { body = v[0] }
		if v, ok := form.Value["is_html"]; ok && len(v) > 0 { 
			isHTML = v[0] == "true" 
		}
	} else {
		// Fallback to JSON or FormValue if not multipart?
		// But client will send JSON or Multipart.
		// If JSON, usage of BodyParser is needed.
		// Let's support both.
		type ComposeRequest struct {
			To      string `json:"to"`
			Cc      string `json:"cc"`
			Bcc     string `json:"bcc"`
			Subject string `json:"subject"`
			Body    string `json:"body"`
			IsHTML  bool   `json:"is_html"`
		}
		var req ComposeRequest
		if err := c.BodyParser(&req); err == nil && req.To != "" {
			to = req.To
			cc = req.Cc
			bcc = req.Bcc
			subject = req.Subject
			body = req.Body
			isHTML = req.IsHTML
		} else {
			// Try FormValue fallback
			to = c.FormValue("to")
			cc = c.FormValue("cc")
			bcc = c.FormValue("bcc")
			subject = c.FormValue("subject")
			body = c.FormValue("body")
			isHTML = c.FormValue("is_html") == "true"
		}
	}

	if to == "" || subject == "" || body == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "All fields are required",
		})
	}

	// Handle Attachments
	var attachments []api.AttachmentData
	if form != nil {
		for _, fileHeaders := range form.File["attachments"] {
			file, err := fileHeaders.Open()
			if err != nil {
				log.Printf("Error opening attachment: %v", err)
				continue
			}
			data, err := io.ReadAll(file)
			file.Close()
			if err != nil {
				log.Printf("Error reading attachment: %v", err)
				continue
			}
			
			contentType := fileHeaders.Header.Get("Content-Type")
			if contentType == "" {
				contentType = api.DetectContentType(fileHeaders.Filename)
			}

			// Optimize image if needed
			if utils.IsImage(contentType) {
				// Resize to max 1920px width
				if optimizedData, err := utils.OptimizeImage(data, 1920); err == nil {
					data = optimizedData
					// Update content length if needed, though usually not strictly required for byte slice
				} else {
					log.Printf("Failed to optimize image %s: %v", fileHeaders.Filename, err)
				}
			}

			attachments = append(attachments, api.AttachmentData{
				Filename:    fileHeaders.Filename,
				ContentType: contentType,
				Data:        data,
			})
		}
	}

	// Create SMTP client
	smtpClient, err := h.auth.CreateSMTPClient(c)
	if err != nil {
		log.Printf("SMTP client creation error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to connect to email server",
		})
	}

	// Send the email
	err = smtpClient.SendMail(to, cc, bcc, subject, body, isHTML, attachments)
	if err != nil {
		log.Printf("Email sending error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to send email: %v", err),
		})
	}

	// Get IMAP client to save to Sent folder
	imapClient, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		log.Printf("IMAP client error when saving to Sent: %v", err)
		// Don't return error here since email was sent successfully
	} else {
		defer imapClient.Close()

		// Try to save to Sent folder
		if err := imapClient.SaveToSent(to, subject, body); err != nil {
			log.Printf("Error saving to Sent folder: %v", err)
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email sent successfully",
		"details": fiber.Map{
			"to":      to,
			"subject": subject,
		},
	})
}

// HandleMoveEmail moves an email to another folder
func (h *EmailHandler) HandleMoveEmail(c *fiber.Ctx) error {
	// Validate Authorization header
	token := c.Get("Authorization")
	if token == "" || len(token) < 8 || token[:7] != "Bearer " {
		return c.Status(401).SendString("Unauthorized")
	}

	// Get source folder and email ID
	sourceFolder := c.Get("X-Folder")
	if sourceFolder == "" {
		sourceFolder = c.Query("folder")
		if sourceFolder == "" {
			sourceFolder = "INBOX"
		}
	}

	emailID := c.Params("id")
	if emailID == "" {
		return c.Status(400).SendString("Email ID required")
	}

	// Get target folder from request body
	type MoveRequest struct {
		TargetFolder string `json:"target_folder"`
	}
	var req MoveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	if req.TargetFolder == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Target folder required",
		})
	}

	// Get IMAP client
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error connecting to email server",
		})
	}
	defer client.Close()

	// Move the email
	err = client.MoveMessage(sourceFolder, req.TargetFolder, emailID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Error moving email: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email moved successfully",
	})
}
