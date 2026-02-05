package api

import (
	"lilmail/models"
	"lilmail/storage"
	"lilmail/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/google/uuid"
)

// LabelHandler handles label management requests
type LabelHandler struct {
	store   *session.Store
	storage *storage.LabelStorage
}

// NewLabelHandler creates a new label handler
func NewLabelHandler(store *session.Store, labelStorage *storage.LabelStorage) *LabelHandler {
	return &LabelHandler{
		store:   store,
		storage: labelStorage,
	}
}

// CreateLabel creates a new label
func (h *LabelHandler) CreateLabel(c *fiber.Ctx) error {
	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	var req models.Label
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}

	if req.Name == "" {
		return utils.BadRequestError("Label name required", nil)
	}

	// Set ID and UserID
	req.ID = uuid.New().String()
	req.UserID = userID
	if req.Color == "" {
		req.Color = "#808080" // Default grey
	}

	if err := h.storage.CreateLabel(&req); err != nil {
		return utils.InternalServerError("Failed to create label", err)
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"label":   req,
	})
}

// GetLabels retrieves all labels for the current user
func (h *LabelHandler) GetLabels(c *fiber.Ctx) error {
	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	labels, err := h.storage.GetLabelsByUser(userID)
	if err != nil {
		return utils.InternalServerError("Failed to retrieve labels", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"labels":  labels,
	})
}

// DeleteLabel deletes a label
func (h *LabelHandler) DeleteLabel(c *fiber.Ctx) error {
	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	id := c.Params("id")
	if id == "" {
		return utils.BadRequestError("Label ID required", nil)
	}

	// Verify ownership
	label, err := h.storage.GetLabel(id)
	if err != nil {
		return utils.NotFoundError("Label not found", nil)
	}
	if label.UserID != userID {
		return utils.UnauthorizedError("Access denied", nil)
	}

	if err := h.storage.DeleteLabel(id); err != nil {
		return utils.InternalServerError("Failed to delete label", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Label deleted",
	})
}

// AssignLabel adds a label to an email
func (h *LabelHandler) AssignLabel(c *fiber.Ctx) error {
	// ... Authentication check ...
	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	emailID := c.Params("emailId")
	labelID := c.Params("labelId")

	// Verify label ownership
	label, err := h.storage.GetLabel(labelID)
	if err != nil {
		return utils.NotFoundError("Label not found", nil)
	}
	if label.UserID != userID {
		return utils.UnauthorizedError("Access denied", nil)
	}

	if err := h.storage.AssignLabel(emailID, labelID); err != nil {
		return utils.InternalServerError("Failed to assign label", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Label assigned",
	})
}

// RemoveLabel removes a label from an email
func (h *LabelHandler) RemoveLabel(c *fiber.Ctx) error {
	// ... Authentication check ...
	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	emailID := c.Params("emailId")
	labelID := c.Params("labelId")

	// Verify label ownership
	label, err := h.storage.GetLabel(labelID)
	if err != nil {
		return utils.NotFoundError("Label not found", nil)
	}
	if label.UserID != userID {
		return utils.UnauthorizedError("Access denied", nil)
	}

	if err := h.storage.RemoveLabel(emailID, labelID); err != nil {
		return utils.InternalServerError("Failed to remove label", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Label removed",
	})
}

// GetEmailLabels retrieves labels for a specific email
func (h *LabelHandler) GetEmailLabels(c *fiber.Ctx) error {
	emailID := c.Params("emailId")
	
	// Note: Ideally we should verify if the user owns this email, 
	// but email ownership check requires IMAP access or checking cache.
	// For now, checks are loose or assumed handled by upstream middleware/check.
	
	labels, err := h.storage.GetLabelsForEmail(emailID)
	if err != nil {
		return utils.InternalServerError("Failed to get email labels", err)
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"labels": labels,
	})
}
