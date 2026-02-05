package api

import (
	"fmt"
	"lilmail/models"
	"lilmail/storage"
	"lilmail/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/google/uuid"
)

// LabelHandler handles label management requests
type LabelHandler struct {
	store         *session.Store
	threadStorage *storage.ThreadStorage
}

// NewLabelHandler creates a new label handler
func NewLabelHandler(store *session.Store, threadStorage *storage.ThreadStorage) *LabelHandler {
	return &LabelHandler{
		store:         store,
		threadStorage: threadStorage,
	}
}

// CreateLabelRequest represents a label creation request
type CreateLabelRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// CreateLabel creates a new label
func (h *LabelHandler) CreateLabel(c *fiber.Ctx) error {
	var req CreateLabelRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}
	
	if req.Name == "" {
		return utils.BadRequestError("Label name is required", nil)
	}
	
	// Default color if not provided
	if req.Color == "" {
		req.Color = "#3B82F6" // Blue
	}
	
	label := &models.Label{
		ID:    uuid.New().String(),
		Name:  req.Name,
		Color: req.Color,
	}
	
	if err := h.threadStorage.SaveLabel(label); err != nil {
		return utils.InternalServerError("Failed to create label", err)
	}
	
	utils.Log.Info("Label created: %s (ID: %s)", label.Name, label.ID)
	
	return c.JSON(fiber.Map{
		"success": true,
		"label":   label,
	})
}

// GetLabels retrieves all labels
func (h *LabelHandler) GetLabels(c *fiber.Ctx) error {
	labels, err := h.threadStorage.GetAllLabels()
	if err != nil {
		return utils.InternalServerError("Failed to retrieve labels", err)
	}
	
	return c.JSON(labels)
}

// DeleteLabel deletes a label
func (h *LabelHandler) DeleteLabel(c *fiber.Ctx) error {
	labelID := c.Params("id")
	if labelID == "" {
		return utils.BadRequestError("Label ID is required", nil)
	}
	
	if err := h.threadStorage.DeleteLabel(labelID); err != nil {
		return utils.InternalServerError("Failed to delete label", err)
	}
	
	utils.Log.Info("Label deleted: ID %s", labelID)
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Label deleted successfully",
	})
}

// AddLabelToEmailRequest represents a request to add a label to an email
type AddLabelToEmailRequest struct {
	EmailID string `json:"email_id"`
	LabelID string `json:"label_id"`
}

// AddLabelToEmail adds a label to an email
func (h *LabelHandler) AddLabelToEmail(c *fiber.Ctx) error {
	var req AddLabelToEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}
	
	if req.EmailID == "" || req.LabelID == "" {
		return utils.BadRequestError("Email ID and Label ID are required", nil)
	}
	
	if err := h.threadStorage.AddLabelToEmail(req.EmailID, req.LabelID); err != nil {
		return utils.InternalServerError("Failed to add label to email", err)
	}
	
	utils.Log.Info("Label %s added to email %s", req.LabelID, req.EmailID)
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Label added to email successfully",
	})
}

// RemoveLabelFromEmail removes a label from an email
func (h *LabelHandler) RemoveLabelFromEmail(c *fiber.Ctx) error {
	emailID := c.Params("email_id")
	labelID := c.Params("label_id")
	
	if emailID == "" || labelID == "" {
		return utils.BadRequestError("Email ID and Label ID are required", nil)
	}
	
	if err := h.threadStorage.RemoveLabelFromEmail(emailID, labelID); err != nil {
		return utils.InternalServerError("Failed to remove label from email", err)
	}
	
	utils.Log.Info("Label %s removed from email %s", labelID, emailID)
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Label removed from email successfully",
	})
}

// GetEmailLabels retrieves all labels for an email
func (h *LabelHandler) GetEmailLabels(c *fiber.Ctx) error {
	emailID := c.Params("email_id")
	if emailID == "" {
		return utils.BadRequestError("Email ID is required", nil)
	}
	
	labels, err := h.threadStorage.GetLabelsByEmail(emailID)
	if err != nil {
		return utils.InternalServerError(fmt.Sprintf("Failed to retrieve labels: %v", err), err)
	}
	
	return c.JSON(labels)
}
