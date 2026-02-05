package api

import (
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
func (h *SendHandler) HandleSend(c *fiber.Ctx) error {
	var req SendRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}

	// Validate required fields
	if req.To == "" || req.Subject == "" || req.Body == "" {
		return utils.BadRequestError("Missing required fields", nil)
	}

	// Get session credentials
	credentials, err := GetCredentials(c, h.store, h.config.Encryption.Key)
	if err != nil {
		return utils.UnauthorizedError("Invalid session", err)
	}

	// Create SMTP client using the existing implementation
	// Note: Credentials has IMAPServer/IMAPPort, we need to use config or get from account
	smtpClient := NewSMTPClient(
		h.config.SMTP.Server,
		h.config.SMTP.Port,
		credentials.Email,
		credentials.Password,
	)

	// Send email using the existing SendMail method
	err = smtpClient.SendMail(req.To, req.Subject, req.Body, req.IsHTML, nil)
	if err != nil {
		return utils.InternalServerError("Failed to send email", err)
	}

	utils.Log.Info("Email sent successfully: to=%s subject=%s", req.To, req.Subject)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email sent successfully",
	})
}

