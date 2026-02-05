package api

import (
	"lilmail/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// FolderHandler handles folder management requests
type FolderHandler struct {
	store  *session.Store
	config *config.Config
}

// NewFolderHandler creates a new folder handler
func NewFolderHandler(store *session.Store, cfg *config.Config) *FolderHandler {
	return &FolderHandler{
		store:  store,
		config: cfg,
	}
}

// CreateFolderRequest represents a folder creation request
type CreateFolderRequest struct {
	Name string `json:"name"`
}

// RenameFolderRequest represents a folder rename request
type RenameFolderRequest struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// CreateFolder creates a new IMAP folder
func (h *FolderHandler) CreateFolder(c *fiber.Ctx) error {
	var req CreateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Folder name is required",
		})
	}

	// NOTE: Folder creation requires IMAP client support which is not yet implemented
	// This is a placeholder for future implementation
	return c.Status(501).JSON(fiber.Map{
		"error": "Folder creation is not yet supported",
	})
}

// DeleteFolder deletes an IMAP folder
func (h *FolderHandler) DeleteFolder(c *fiber.Ctx) error {
	folderName := c.Params("name")
	if folderName == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Folder name is required",
		})
	}

	// Prevent deletion of system folders
	systemFolders := []string{"INBOX", "Sent", "Drafts", "Trash", "Spam"}
	for _, sf := range systemFolders {
		if folderName == sf {
			return c.Status(400).JSON(fiber.Map{
				"error": "Cannot delete system folder",
			})
		}
	}

	// NOTE: Folder deletion requires IMAP client support which is not yet implemented
	// This is a placeholder for future implementation
	return c.Status(501).JSON(fiber.Map{
		"error": "Folder deletion is not yet supported",
	})
}

// RenameFolder renames an IMAP folder
func (h *FolderHandler) RenameFolder(c *fiber.Ctx) error {
	var req RenameFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	if req.OldName == "" || req.NewName == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Both old name and new name are required",
		})
	}

	// Prevent renaming system folders
	systemFolders := []string{"INBOX", "Sent", "Drafts", "Trash", "Spam"}
	for _, sf := range systemFolders {
		if req.OldName == sf {
			return c.Status(400).JSON(fiber.Map{
				"error": "Cannot rename system folder",
			})
		}
	}

	// NOTE: Folder renaming requires IMAP client support which is not yet implemented
	// This is a placeholder for future implementation
	return c.Status(501).JSON(fiber.Map{
		"error": "Folder renaming is not yet supported",
	})
}
