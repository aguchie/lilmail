package web

import (
	"lilmail/config"
	"lilmail/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type AdminHandler struct {
	store       *session.Store
	config      *config.Config
	userStorage *storage.UserStorage
}

func NewAdminHandler(store *session.Store, cfg *config.Config, userStorage *storage.UserStorage) *AdminHandler {
	return &AdminHandler{
		store:       store,
		config:      cfg,
		userStorage: userStorage,
	}
}

// ShowUsers renders the user management page
func (h *AdminHandler) ShowUsers(c *fiber.Ctx) error {
	// Verify Admin Role
	if !h.isAdmin(c) {
		return c.Redirect("/settings")
	}

	username := c.Locals("username").(string)
    token := ""
    // Get token for API calls
    sess, _ := h.store.Get(c)
    if sess != nil {
        if t := sess.Get("token"); t != nil {
            token = t.(string)
        }
    }

	return c.Render("admin/users", fiber.Map{
		"Username":  username,
		"Token":     token,
		"CSRFToken": c.Locals("csrf"),
	})
}

// Helper to check admin role
func (h *AdminHandler) isAdmin(c *fiber.Ctx) bool {
    userID, ok := c.Locals("userId").(string)
    if !ok || userID == "" {
        // Fallback: try to load user by username from session
        username, ok := c.Locals("username").(string)
        if !ok || username == "" {
            return false
        }
        user, err := h.userStorage.GetUserByUsername(username)
        if err != nil {
            return false
        }
        return user.Role == "admin"
    }
    
    user, err := h.userStorage.GetUser(userID)
    if err != nil {
        return false
    }
    
    return user.Role == "admin"
}
