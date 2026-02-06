package api

import (
	"lilmail/config"
	"lilmail/models"
	"lilmail/storage"
	"lilmail/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// UserHandler handles user management
type UserHandler struct {
	store   *session.Store
	config  *config.Config
	storage *storage.UserStorage
}

// NewUserHandler creates a new user handler
func NewUserHandler(store *session.Store, cfg *config.Config, userStorage *storage.UserStorage) *UserHandler {
	return &UserHandler{
		store:   store,
		config:  cfg,
		storage: userStorage,
	}
}

// GetUsers retrieves all users (Admin only)
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	// Verify Admin Role
	if !h.isAdmin(c) {
		return utils.ForbiddenError("Access denied", nil)
	}

	users, err := h.storage.ListUsers()
	if err != nil {
		return utils.InternalServerError("Failed to retrieve users", err)
	}

	// Remove sensitive data
	for _, u := range users {
		u.PasswordHash = ""
	}

	return c.JSON(fiber.Map{
		"success": true,
		"users":   users,
	})
}

// UpdateUser updates a user (Admin only)
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
    // Verify Admin Role
	if !h.isAdmin(c) {
		return utils.ForbiddenError("Access denied", nil)
	}

	userID := c.Params("id")
	if userID == "" {
		return utils.BadRequestError("User ID required", nil)
	}

	var req models.User
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}

	// Retrieve existing user
	user, err := h.storage.GetUser(userID)
	if err != nil {
		return utils.NotFoundError("User not found", err)
	}

	// Update allowed fields
    if req.Role != "" {
         user.Role = req.Role
    }
    if req.DisplayName != "" {
        user.DisplayName = req.DisplayName
    }

	if err := h.storage.UpdateUser(user); err != nil {
		return utils.InternalServerError("Failed to update user", err)
	}

    user.PasswordHash = ""

	return c.JSON(fiber.Map{
		"success": true,
		"user":    user,
	})
}

// DeleteUser deletes a user (Admin only)
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
    // Verify Admin Role
	if !h.isAdmin(c) {
		return utils.ForbiddenError("Access denied", nil)
	}

	userID := c.Params("id")
	if userID == "" {
		return utils.BadRequestError("User ID required", nil)
	}

    // Prevent deleting self?
    currentUserID := c.Locals("userId")
    if currentUserID == userID {
        return utils.BadRequestError("Cannot delete yourself", nil)
    }

	if err := h.storage.DeleteUser(userID); err != nil {
		return utils.InternalServerError("Failed to delete user", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deleted successfully",
	})
}

// Helper to check admin role
func (h *UserHandler) isAdmin(c *fiber.Ctx) bool {
    userID, ok := c.Locals("userId").(string) // Ensure userId is set in Locals by middleware
    if !ok || userID == "" {
        // Fallback or check storage?
        // Ideally middleware sets user object or role.
        // Let's assume we need to fetch user to check role if not in session/locals.
        // But for efficiency, role should be in session?
        // Let's check storage.
        return false
    }
    
    user, err := h.storage.GetUser(userID)
    if err != nil {
        return false
    }
    
    return user.Role == "admin"
}

// CreateUser creates a new user (Admin only)
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	// Verify Admin Role
	if !h.isAdmin(c) {
		return utils.ForbiddenError("Access denied", nil)
	}

	var req struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"` // Plain password
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}

	// Validate
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return utils.BadRequestError("Username, Email and Password are required", nil)
	}

	user := &models.User{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Role:        req.Role,
	}

	if err := h.storage.CreateUser(user, req.Password); err != nil {
		return utils.InternalServerError("Failed to create user", err)
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"user":    user,
	})
}

// UpdatePassword updates a user's password
func (h *UserHandler) UpdatePassword(c *fiber.Ctx) error {
	targetUserID := c.Params("id")
	if targetUserID == "" {
		return utils.BadRequestError("User ID required", nil)
	}

	var req struct {
		CurrentPassword string `json:"current_password"` // Optional if admin
		NewPassword     string `json:"new_password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}

	if req.NewPassword == "" {
		return utils.BadRequestError("New password is required", nil)
	}

	// Check permissions
	currentUserID, ok := c.Locals("userId").(string)
	if !ok {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	// If changing own password, verify current password
	if currentUserID == targetUserID {
		if req.CurrentPassword == "" {
			return utils.BadRequestError("Current password is required", nil)
		}
		if err := h.storage.VerifyPassword(currentUserID, req.CurrentPassword); err != nil {
			return utils.UnauthorizedError("Invalid current password", err)
		}
	} else {
		// If changing someone else's password, must be admin
		if !h.isAdmin(c) {
			return utils.ForbiddenError("Access denied", nil)
		}
	}

	if err := h.storage.UpdatePassword(targetUserID, req.NewPassword); err != nil {
		return utils.InternalServerError("Failed to update password", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Password updated successfully",
	})
}
