package web

import (
	"lilmail/config"
	"lilmail/models"
	"lilmail/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type SettingsHandler struct {
	store          *session.Store
	config         *config.Config
	userStorage    *storage.UserStorage
	accountStorage *storage.AccountStorage
	labelStorage   *storage.LabelStorage
}

func NewSettingsHandler(store *session.Store, cfg *config.Config, userStorage *storage.UserStorage, accountStorage *storage.AccountStorage, labelStorage *storage.LabelStorage) *SettingsHandler {
	return &SettingsHandler{
		store:          store,
		config:         cfg,
		userStorage:    userStorage,
		accountStorage: accountStorage,
		labelStorage:   labelStorage,
	}
}

// ShowSettings renders the settings page
func (h *SettingsHandler) ShowSettings(c *fiber.Ctx) error {
	username := c.Locals("username")
	if username == nil {
		return c.Redirect("/login")
	}

	userStr, ok := username.(string)
	if !ok {
		return c.Redirect("/login")
	}

	// Load user settings
	user, err := h.userStorage.GetUserByUsername(userStr)
	if err != nil {
		return c.Status(500).SendString("Error loading user settings")
	}

	// Default values if not set
	if user.Language == "" {
		user.Language = "en"
	}
	if user.Theme == "" {
		user.Theme = "light"
	}

	// Load user accounts - using empty encryption key for now
	accounts, err := h.accountStorage.GetAccountsByUser(user.ID, []byte(h.config.Encryption.Key))
	if err != nil {
		accounts = []*models.Account{} // Empty array if error
	}

	// Load user labels
	labels, err := h.labelStorage.GetLabelsByUser(user.ID)
	if err != nil {
		labels = []models.Label{}
	}

	// Get session to retrieve current account ID
	sess, err := h.store.Get(c)
	var currentAccountID string
	if err == nil {
		if accID := sess.Get("accountId"); accID != nil {
			currentAccountID, _ = accID.(string)
		}
	}

	return c.Render("settings", fiber.Map{
		"Username": userStr,
		"User":     user,
		"Accounts": accounts,
		"Labels":   labels,
		"NotificationSettings": fiber.Map{
			"Desktop":  false,
			"NewEmail": false,
		},
		"CurrentAccountID": currentAccountID,
		"CSRFToken":        c.Locals("csrf"),
	})
}

// UpdateGeneralSettings updates general user settings
func (h *SettingsHandler) UpdateGeneralSettings(c *fiber.Ctx) error {
	username := c.Locals("username")
	if username == nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	userStr, ok := username.(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Get form values
	language := c.FormValue("language")
	theme := c.FormValue("theme")

	// Load user
	user, err := h.userStorage.GetUserByUsername(userStr)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error loading user"})
	}

	// Update user settings
	user.Language = language
	user.Theme = theme

	// Save updated user
	if err := h.userStorage.UpdateUser(user); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error saving settings"})
	}

	// Update language cookie
	c.Cookie(&fiber.Cookie{
		Name:  "lang",
		Value: language,
		Path:  "/",
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Settings updated successfully",
	})
}
