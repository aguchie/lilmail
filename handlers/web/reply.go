package web

import (
	"fmt"
	"lilmail/config"
	"lilmail/models"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// ReplyHandler handles email reply, reply-all, and forward operations
type ReplyHandler struct {
	store  *session.Store
	config *config.Config
	auth   *AuthHandler
}

// NewReplyHandler creates a new reply handler
func NewReplyHandler(store *session.Store, config *config.Config, auth *AuthHandler) *ReplyHandler {
	return &ReplyHandler{
		store:  store,
		config: config,
		auth:   auth,
	}
}

// HandleReply prepares the compose modal with reply data
func (h *ReplyHandler) HandleReply(c *fiber.Ctx) error {
	emailID := c.Params("id")
	folder := c.Get("X-Folder", "INBOX")

	// Get IMAP client using auth handler
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to connect to IMAP server"})
	}
	defer client.Close()

	// Fetch the original email
	email, err := client.FetchSingleMessage(folder, emailID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Email not found"})
	}

	// Prepare reply data
	replyData := prepareReplyData(&email, "reply")

	return c.JSON(fiber.Map{
		"success": true,
		"data":    replyData,
	})
}

// HandleReplyAll prepares the compose modal with reply-all data
func (h *ReplyHandler) HandleReplyAll(c *fiber.Ctx) error {
	emailID := c.Params("id")
	folder := c.Get("X-Folder", "INBOX")

	// Get IMAP client using auth handler
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to connect to IMAP server"})
	}
	defer client.Close()

	// Fetch the original email
	email, err := client.FetchSingleMessage(folder, emailID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Email not found"})
	}

	// Prepare reply-all data
	replyData := prepareReplyData(&email, "replyall")

	return c.JSON(fiber.Map{
		"success": true,
		"data":    replyData,
	})
}

// HandleForward prepares the compose modal with forward data
func (h *ReplyHandler) HandleForward(c *fiber.Ctx) error {
	emailID := c.Params("id")
	folder := c.Get("X-Folder", "INBOX")

	// Get IMAP client using auth handler
	client, err := h.auth.CreateIMAPClient(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to connect to IMAP server"})
	}
	defer client.Close()

	// Fetch the original email
	email, err := client.FetchSingleMessage(folder, emailID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Email not found"})
	}

	// Prepare forward data
	forwardData := prepareForwardData(&email)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    forwardData,
	})
}

// prepareReplyData prepares the reply/reply-all email data
func prepareReplyData(email *models.Email, replyType string) map[string]interface{} {
	to := email.From
	cc := ""
	
	// For reply-all, include all original recipients
	if replyType == "replyall" {
		// Parse To addresses and add to CC
		toAddrs := strings.Split(email.To, ",")
		ccAddrs := []string{}
		
		if email.Cc != "" {
			ccAddrs = append(ccAddrs, strings.Split(email.Cc, ",")...)
		}
		
		// Add all To addresses to CC (except the current user)
		for _, addr := range toAddrs {
			trimmed := strings.TrimSpace(addr)
			if trimmed != "" {
				ccAddrs = append(ccAddrs, trimmed)
			}
		}
		
		cc = strings.Join(ccAddrs, ", ")
	}

	// Add "Re:" prefix to subject if not already present
	subject := email.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	// Create quoted body
	quotedBody := formatQuotedBody(email)

	return map[string]interface{}{
		"to":      to,
		"cc":      cc,
		"subject": subject,
		"body":    quotedBody,
		"mode":    replyType,
	}
}

// prepareForwardData prepares the forward email data
func prepareForwardData(email *models.Email) map[string]interface{} {
	// Add "Fwd:" prefix to subject if not already present
	subject := email.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "fwd:") && 
	   !strings.HasPrefix(strings.ToLower(subject), "fw:") {
		subject = "Fwd: " + subject
	}

	// Create forwarded message body
	forwardedBody := formatForwardedBody(email)

	return map[string]interface{}{
		"to":      "",
		"cc":      "",
		"subject": subject,
		"body":    forwardedBody,
		"mode":    "forward",
	}
}

// formatQuotedBody formats the email body with quote marks
func formatQuotedBody(email *models.Email) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("\n\n\nOn %s, %s wrote:\n", 
		email.Date.Format(time.RFC1123), email.From))
	
	// Get the text body
	body := email.Body
	if body == "" && email.HTML != "" {
		// Strip HTML tags if only HTML is available
		body = stripHTML(string(email.HTML))
	}
	
	// Add quote marks to each line
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		sb.WriteString("> " + line + "\n")
	}
	
	return sb.String()
}

// formatForwardedBody formats the email body for forwarding
func formatForwardedBody(email *models.Email) string {
	var sb strings.Builder
	
	sb.WriteString("\n\n\n---------- Forwarded message ---------\n")
	sb.WriteString(fmt.Sprintf("From: %s\n", email.From))
	sb.WriteString(fmt.Sprintf("Date: %s\n", email.Date.Format(time.RFC1123)))
	sb.WriteString(fmt.Sprintf("Subject: %s\n", email.Subject))
	sb.WriteString(fmt.Sprintf("To: %s\n", email.To))
	
	if email.Cc != "" {
		sb.WriteString(fmt.Sprintf("Cc: %s\n", email.Cc))
	}
	
	sb.WriteString("\n\n")
	
	// Get the text body
	body := email.Body
	if body == "" && email.HTML != "" {
		body = stripHTML(string(email.HTML))
	}
	
	sb.WriteString(body)
	
	return sb.String()
}

// stripHTML removes HTML tags from a string (basic implementation)
func stripHTML(html string) string {
	// Simple tag removal - for production, use a proper HTML parser
	result := html
	result = strings.ReplaceAll(result, "<br>", "\n")
	result = strings.ReplaceAll(result, "<br/>", "\n")
	result = strings.ReplaceAll(result, "<br />", "\n")
	result = strings.ReplaceAll(result, "</p>", "\n\n")
	result = strings.ReplaceAll(result, "</div>", "\n")
	
	// Remove all remaining tags
	inTag := false
	var sb strings.Builder
	for _, r := range result {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			sb.WriteRune(r)
		}
	}
	
	return strings.TrimSpace(sb.String())
}
