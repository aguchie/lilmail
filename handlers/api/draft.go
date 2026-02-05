package api

import (
	"lilmail/models"
	"lilmail/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// DraftHandler handles draft operations
type DraftHandler struct {
	store        *session.Store
	draftStorage *storage.DraftStorage
}

// NewDraftHandler creates a new draft handler
func NewDraftHandler(store *session.Store, draftStorage *storage.DraftStorage) *DraftHandler {
	return &DraftHandler{
		store:        store,
		draftStorage: draftStorage,
	}
}

// SaveDraft saves or updates a draft
func (h *DraftHandler) SaveDraft(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Session error"})
	}

	userID := sess.Get("user_id")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Parse request
	var req struct {
		ID      string `json:"id"`
		To      string `json:"to"`
		Cc      string `json:"cc"`
		Bcc     string `json:"bcc"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
		IsHTML  bool   `json:"is_html"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Create draft model
	draft := &models.Draft{
		To:      req.To,
		Cc:      req.Cc,
		Bcc:     req.Bcc,
		Subject: req.Subject,
		Body:    req.Body,
		IsHTML:  req.IsHTML,
	}

	// Save draft
	if err := h.draftStorage.SaveDraft(userID.(string), req.ID, draft); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save draft"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"draft":   draft,
	})
}

// AutoSave handles auto-save requests
func (h *DraftHandler) AutoSave(c *fiber.Ctx) error {
	// Reuse SaveDraft logic
	return h.SaveDraft(c)
}

// GetDrafts retrieves all drafts for the current user
func (h *DraftHandler) GetDrafts(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Session error"})
	}

	userID := sess.Get("user_id")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	drafts, err := h.draftStorage.GetDrafts(userID.(string))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get drafts"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"drafts":  drafts,
	})
}

// GetDraft retrieves a specific draft
func (h *DraftHandler) GetDraft(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Session error"})
	}

	userID := sess.Get("user_id")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	draftID := c.Params("id")
	draft, err := h.draftStorage.GetDraft(userID.(string), draftID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Draft not found"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"draft":   draft,
	})
}

// DeleteDraft deletes a draft
func (h *DraftHandler) DeleteDraft(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Session error"})
	}

	userID := sess.Get("user_id")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	draftID := c.Params("id")
	if err := h.draftStorage.DeleteDraft(userID.(string), draftID); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Draft not found"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Draft deleted",
	})
}
