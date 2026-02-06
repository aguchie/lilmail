package api

import (
	"io"
	"strings"
	"lilmail/config"
	"lilmail/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// SendHandler handles email sending
type SendHandler struct {
	store  *session.Store
	config *config.Config
}

// NewSendHandler creates a new send handler
func NewSendHandler(store *session.Store, cfg *config.Config) *SendHandler {
	return &SendHandler{
		store:  store,
		config: cfg,
	}
}

// SendRequest represents an email send request
type SendRequest struct {
	To      string `json:"to"`
	Cc      string `json:"cc"`
	Bcc     string `json:"bcc"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	IsHTML  bool   `json:"is_html"`
}

// HandleSend handles the email send request
// HandleSend handles the email send request
func (h *SendHandler) HandleSend(c *fiber.Ctx) error {
	var to, cc, bcc, subject, body string
	var isHTML bool
	var attachments []AttachmentData

	contentType := c.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		// Handle Multipart Form (for attachments)
		form, err := c.MultipartForm()
		if err != nil {
			return utils.BadRequestError("Invalid form data", err)
		}

		// Get standard fields
		if v, ok := form.Value["to"]; ok && len(v) > 0 { to = v[0] }
		if v, ok := form.Value["cc"]; ok && len(v) > 0 { cc = v[0] }
		if v, ok := form.Value["bcc"]; ok && len(v) > 0 { bcc = v[0] }
		if v, ok := form.Value["subject"]; ok && len(v) > 0 { subject = v[0] }
		if v, ok := form.Value["body"]; ok && len(v) > 0 { body = v[0] }
		if v, ok := form.Value["is_html"]; ok && len(v) > 0 { isHTML = v[0] == "true" }

		// Process attachments
		for _, files := range form.File {
			for _, file := range files {
				// Open file
				f, err := file.Open()
				if err != nil {
					utils.Log.Error("Failed to open attachment: %v", err)
					continue
				}
				defer f.Close()

				// Read content
				data, err := io.ReadAll(f)
				if err != nil {
					utils.Log.Error("Failed to read attachment: %v", err)
					continue
				}

				// Create attachment data
				att := AttachmentData{
					Filename:    file.Filename,
					ContentType: file.Header.Get("Content-Type"),
					Data:        data,
				}
				if att.ContentType == "" {
					att.ContentType = DetectContentType(file.Filename)
				}

				attachments = append(attachments, att)
			}
		}

	} else {
		// Handle JSON (legacy/no-attachment)
		var req SendRequest
		if err := c.BodyParser(&req); err != nil {
			return utils.BadRequestError("Invalid request", err)
		}
		to = req.To
		cc = req.Cc
		bcc = req.Bcc
		subject = req.Subject
		body = req.Body
		isHTML = req.IsHTML
	}

	// Validate required fields
	if to == "" || subject == "" {
		return utils.BadRequestError("Missing required fields (to, subject)", nil)
	}

	// Get session credentials
	credentials, err := GetCredentials(c, h.store, h.config.Encryption.Key)
	if err != nil {
		return utils.UnauthorizedError("Invalid session", err)
	}

	// Create SMTP client
	smtpClient := NewSMTPClient(
		h.config.SMTP.Server,
		h.config.SMTP.Port,
		credentials.Email,
		credentials.Password,
	)

	// Send email
	err = smtpClient.SendMail(to, cc, bcc, subject, body, isHTML, attachments)
	if err != nil {
		return utils.InternalServerError("Failed to send email", err)
	}

	utils.Log.Info("Email sent successfully: to=%s subject=%s attachments=%d", to, subject, len(attachments))

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email sent successfully",
	})
}

