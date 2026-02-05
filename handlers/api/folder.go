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

	// Get session credentials
	credentials, err := GetCredentials(c, h.store, h.config.Encryption.Key)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid session",
		})
	}

	// Create IMAP client
	client, err := createIMAPClientFromCredentials(credentials, h.config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to connect to email server",
		})
	}
	defer client.Close()

	// Create folder
	if err := client.CreateFolder(req.Name); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create folder: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Folder created successfully",
		"folder":  req.Name,
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

	// Get session credentials
	credentials, err := GetCredentials(c, h.store, h.config.Encryption.Key)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid session",
		})
	}

	// Create IMAP client
	client, err := createIMAPClientFromCredentials(credentials, h.config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to connect to email server",
		})
	}
	defer client.Close()

	// Delete folder
	if err := client.DeleteFolder(folderName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete folder: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Folder deleted successfully",
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

	// Get session credentials
	credentials, err := GetCredentials(c, h.store, h.config.Encryption.Key)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid session",
		})
	}

	// Create IMAP client
	client, err := createIMAPClientFromCredentials(credentials, h.config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to connect to email server",
		})
	}
	defer client.Close()

	// Rename folder
	if err := client.RenameFolder(req.OldName, req.NewName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to rename folder: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Folder renamed successfully",
		"oldName": req.OldName,
		"newName": req.NewName,
	})
}
