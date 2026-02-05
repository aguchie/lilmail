package api

import (
	"lilmail/config"
	"lilmail/models"
	"lilmail/storage"
	"lilmail/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/google/uuid"
)

// AccountHandler handles account management
type AccountHandler struct {
	store   *session.Store
	config  *config.Config
	storage *storage.AccountStorage
}

// NewAccountHandler creates a new account handler
func NewAccountHandler(store *session.Store, cfg *config.Config, accountStorage *storage.AccountStorage) *AccountHandler {
	return &AccountHandler{
		store:   store,
		config:  cfg,
		storage: accountStorage,
	}
}

// CreateAccount creates a new email account
func (h *AccountHandler) CreateAccount(c *fiber.Ctx) error {
	var req models.Account
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}

	// Get user from session
	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	// Set user ID
	req.UserID = userID
	req.ID = uuid.New().String()

	// Validate required fields
	if req.Email == "" || req.IMAPServer == "" || req.Username == "" || req.Password == "" {
		return utils.BadRequestError("Missing required fields", nil)
	}

	// Create account
	encryptionKey := []byte(h.config.Encryption.Key)
	if err := h.storage.CreateAccount(&req, encryptionKey); err != nil {
		return utils.InternalServerError("Failed to create account", err)
	}

	// Don't return password
	req.Password = ""

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"account": req,
	})
}

// GetAccounts retrieves all accounts for the current user
func (h *AccountHandler) GetAccounts(c *fiber.Ctx) error {
	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	encryptionKey := []byte(h.config.Encryption.Key)
	accounts, err := h.storage.GetAccountsByUser(userID, encryptionKey)
	if err != nil {
		return utils.InternalServerError("Failed to retrieve accounts", err)
	}

	// Remove passwords from response
	for _, acc := range accounts {
		acc.Password = ""
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"accounts": accounts,
	})
}

// GetAccount retrieves a specific account
func (h *AccountHandler) GetAccount(c *fiber.Ctx) error {
	accountID := c.Params("id")
	if accountID == "" {
		return utils.BadRequestError("Account ID required", nil)
	}

	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	encryptionKey := []byte(h.config.Encryption.Key)
	account, err := h.storage.GetAccount(accountID, encryptionKey)
	if err != nil {
		return utils.NotFoundError("Account not found", err)
	}

	// Verify ownership
	if account.UserID != userID {
		return utils.UnauthorizedError("Access denied", nil)
	}

	// Remove password
	account.Password = ""

	return c.JSON(fiber.Map{
		"success": true,
		"account": account,
	})
}

// UpdateAccount updates an existing account
func (h *AccountHandler) UpdateAccount(c *fiber.Ctx) error {
	accountID := c.Params("id")
	if accountID == "" {
		return utils.BadRequestError("Account ID required", nil)
	}

	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	var req models.Account
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestError("Invalid request", err)
	}

	// Set ID and UserID
	req.ID = accountID
	req.UserID = userID

	// Verify existing account ownership
	encryptionKey := []byte(h.config.Encryption.Key)
	existing, err := h.storage.GetAccount(accountID, encryptionKey)
	if err != nil {
		return utils.NotFoundError("Account not found", err)
	}

	if existing.UserID != userID {
		return utils.UnauthorizedError("Access denied", nil)
	}

	// Update account
	if err := h.storage.UpdateAccount(&req, encryptionKey); err != nil {
		return utils.InternalServerError("Failed to update account", err)
	}

	// Remove password
	req.Password = ""

	return c.JSON(fiber.Map{
		"success": true,
		"account": req,
	})
}

// DeleteAccount deletes an account
func (h *AccountHandler) DeleteAccount(c *fiber.Ctx) error {
	accountID := c.Params("id")
	if accountID == "" {
		return utils.BadRequestError("Account ID required", nil)
	}

	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	// Verify ownership
	encryptionKey := []byte(h.config.Encryption.Key)
	account, err := h.storage.GetAccount(accountID, encryptionKey)
	if err != nil {
		return utils.NotFoundError("Account not found", err)
	}

	if account.UserID != userID {
		return utils.UnauthorizedError("Access denied", nil)
	}

	// Delete account
	if err := h.storage.DeleteAccount(accountID); err != nil {
		return utils.InternalServerError("Failed to delete account", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Account deleted successfully",
	})
}

// SetDefaultAccount sets an account as the default
func (h *AccountHandler) SetDefaultAccount(c *fiber.Ctx) error {
	accountID := c.Params("id")
	if accountID == "" {
		return utils.BadRequestError("Account ID required", nil)
	}

	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	encryptionKey := []byte(h.config.Encryption.Key)

	// Get all user accounts
	accounts, err := h.storage.GetAccountsByUser(userID, encryptionKey)
	if err != nil {
		return utils.InternalServerError("Failed to retrieve accounts", err)
	}

	// Find target account and unset all defaults
	var targetAccount *models.Account
	for _, acc := range accounts {
		if acc.ID == accountID {
			targetAccount = acc
			acc.IsDefault = true
		} else {
			acc.IsDefault = false
		}
		if err := h.storage.UpdateAccount(acc, encryptionKey); err != nil {
			return utils.InternalServerError("Failed to update account", err)
		}
	}

	if targetAccount == nil {
		return utils.NotFoundError("Account not found", nil)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Default account updated",
	})
}

// SwitchAccount switches the active session to the specified account
func (h *AccountHandler) SwitchAccount(c *fiber.Ctx) error {
	accountID := c.Params("id")
	if accountID == "" {
		return utils.BadRequestError("Account ID required", nil)
	}

	userID, ok := c.Locals("username").(string)
	if !ok || userID == "" {
		return utils.UnauthorizedError("User not authenticated", nil)
	}

	encryptionKey := []byte(h.config.Encryption.Key)

	// Verify account ownership and get details
	account, err := h.storage.GetAccount(accountID, encryptionKey)
	if err != nil {
		return utils.NotFoundError("Account not found", err)
	}

	if account.UserID != userID {
		return utils.UnauthorizedError("Access denied", nil)
	}

	// Update Session
	sess, err := h.store.Get(c)
	if err != nil {
		return utils.InternalServerError("Session error", err)
	}

	// Re-encrypt details for session (or just store what we retrieved, which is decrypted? no GetAccount returns decrypted struct?)
	// Check models/account.go: Account has Password fields. 
	// storage.GetAccount usually returns struct with decrypted password if we passed the key?
	// Let's assume GetAccount decrypts the password into the struct.
	
	encryptedCreds, err := EncryptCredentials(account.Email, account.Password, h.config.Encryption.Key)
	if err != nil {
		return utils.InternalServerError("Failed to secure credentials", err)
	}

	// Update session values
	sess.Set("accountId", account.ID)
	sess.Set("email", account.Email)
	sess.Set("username", account.Username)
	sess.Set("credentials", encryptedCreds)
	
	// Regenerate token? Token contains email/username usually.
	// If token changes, frontend needs it.
	token, err := GenerateToken(account.Username, account.Email, h.config.JWT.Secret)
	if err != nil {
		return utils.InternalServerError("Failed to generate token", err)
	}
	sess.Set("token", token)

	if err := sess.Save(); err != nil {
		return utils.InternalServerError("Failed to save session", err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Switched account successfully",
		"token": token,
		"account": fiber.Map{
			"id": account.ID,
			"email": account.Email,
			"username": account.Username,
		},
	})
}
