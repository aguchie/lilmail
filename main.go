package main

import (
	"fmt"
	"lilmail/config"
	"lilmail/handlers/api"
	"lilmail/handlers/web"
	"lilmail/middleware"
	"lilmail/storage"
	"lilmail/utils"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var store *session.Store

func init() {
	// Initialize logger
	utils.Log.Info("Initializing LilMail...")

	// Create file storage
	storage, err := storage.NewFileStorage("./sessions")
	if err != nil {
		utils.Log.Error("Failed to initialize session storage: %v", err)
	}

	store = session.New(session.Config{
		Storage:        storage,
		Expiration:     24 * time.Hour,
		CookieSecure:   false, // Set to true in production with HTTPS
		CookieHTTPOnly: true,
	})
}

// Helper function to determine if request is an API request
func isAPIRequest(c *fiber.Ctx) bool {
	if c == nil {
		return false
	}

	// Check for HTMX request first
	if c.Get("HX-Request") != "" {
		return true
	}

	// Safely check if path starts with /api
	path := c.Path()
	return len(path) >= 4 && path[:4] == "/api"
}

func main() {
	// Load configuration
	config, err := config.LoadConfig("config.toml")
	if err != nil {
		utils.Log.Error("Failed to load config: %v", err)
	}

	// Initialize i18n system
	if err := utils.InitI18n(); err != nil {
		utils.Log.Error("Failed to initialize i18n: %v", err)
	}

	// Initialize template engine with custom functions
	engine := html.New("./templates", ".html")

	// String manipulation functions
	engine.AddFunc("split", strings.Split)
	engine.AddFunc("join", strings.Join)
	engine.AddFunc("lower", strings.ToLower)
	engine.AddFunc("upper", strings.ToUpper)
	engine.AddFunc("title", strings.Title)
	engine.AddFunc("trim", strings.TrimSpace)
	engine.AddFunc("hasPrefix", strings.HasPrefix)

	// i18n template functions
	engine.AddFunc("t", func(messageID string) string {
		// This will be overridden per-request with the correct localizer
		return utils.T(utils.Localizer, messageID)
	})

	engine.AddFunc("tWithData", func(messageID string, data map[string]interface{}) string {
		return utils.TWithData(utils.Localizer, messageID, data)
	})

	engine.AddFunc("tPlural", func(messageID string, count int) string {
		return utils.TPlural(utils.Localizer, messageID, count)
	})

	// Date formatting function
	engine.AddFunc("formatDate", func(t time.Time) string {
		return t.Format("Jan 02, 2006 15:04")
	})

	// File size formatting function
	engine.AddFunc("formatSize", func(size int64) string {
		const unit = 1024
		if size < unit {
			return fmt.Sprintf("%d B", size)
		}
		div, exp := int64(unit), 0
		for n := size / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
	})

	engine.Reload(true)

	// Initialize Fiber with template engine
	app := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layouts/main", // Default layout
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			
			// Check for AppError
			if appErr, ok := err.(*utils.AppError); ok {
				code = appErr.Code
				utils.Log.Error("Application error: %v", appErr)
			} else if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			// Handle API requests differently
			if isAPIRequest(c) {
				return c.Status(code).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			// Render error page for regular requests
			return c.Status(code).Render("error", fiber.Map{
				"Error": err.Error(),
				"Code":  code,
			})
		},
	})

	// Add global middleware
	app.Use(recover.New()) // Recover from panics
	app.Use(logger.New())  // Request logging
	app.Use(compress.New()) // Response compression
	app.Use(helmet.New(helmet.Config{ // Security headers
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		ReferrerPolicy:        "no-referrer",
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://cdn.quilljs.com https://unpkg.com; style-src 'self' 'unsafe-inline' https://cdn.quilljs.com; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';",
	}))
	
	// Add locale middleware
	app.Use(middleware.LocaleMiddleware())

	// Add rate limiting (100 requests per minute per IP)
	app.Use(middleware.RateLimiter(100, time.Minute))

	// Serve static files
	app.Static("/assets", "./assets", fiber.Static{
		Compress:      true,
		CacheDuration: 24 * time.Hour,
	})

	// Initialize storage layers
	accountStorage, err := storage.NewAccountStorage("./data")
	if err != nil {
		utils.Log.Error("Failed to initialize account storage: %v", err)
	}

	userStorage, err := storage.NewUserStorage("./data")
	if err != nil {
		utils.Log.Error("Failed to initialize user storage: %v", err)
	}

	// Web handlers initialized later with NotificationHandler

	_, err = storage.NewThreadStorage("./data")
	if err != nil {
		utils.Log.Error("Failed to initialize thread storage: %v", err)
	}

	draftStorage := storage.NewDraftStorage("./data")

	labelStorage, err := storage.NewLabelStorage("./data")
	if err != nil {
		utils.Log.Error("Failed to initialize label storage: %v", err)
	}
	defer labelStorage.Close()

	// Initialize Notification Handler
	notificationHandler := api.NewNotificationHandler(store)

	// Initialize API handlers
	searchHandler := api.NewSearchHandler(store, config)
	folderHandler := api.NewFolderHandler(store, config)
	accountHandler := api.NewAccountHandler(store, config, accountStorage)
	labelHandler := api.NewLabelHandler(store, labelStorage)
	i18nHandler := &api.I18nHandler{}

	// Initialize web handlers
	webAuthHandler := web.NewAuthHandler(store, config, userStorage, accountStorage)
	webEmailHandler := web.NewEmailHandler(store, config, webAuthHandler, notificationHandler)

	// Public routes
	app.Get("/login", webAuthHandler.ShowLogin)
	app.Post("/login", webAuthHandler.HandleLogin)
	app.Get("/logout", webAuthHandler.HandleLogout)

	// Protected routes group
	protected := app.Group("", api.SessionMiddleware(store))
	
	// Add CSRF Middleware to protected routes
	app.Use(csrf.New(csrf.Config{
		KeyLookup:      "header:X-CSRF-Token,form:csrf_",
		CookieName:     "csrf_",
		CookieSameSite: "Strict",
		Expiration:     1 * time.Hour,
		ContextKey:     "csrf",
	}))

	// Notification Routes
	protected.Get("/events", notificationHandler.HandleSSE)
	protected.Get("/ws", websocket.New(notificationHandler.HandleWebSocket))

	// Main web routes
	protected.Get("/", webEmailHandler.HandleInbox)      // Default to inbox
	protected.Get("/inbox", webEmailHandler.HandleInbox) // Explicit inbox route
	protected.Get("/folder/:name", webEmailHandler.HandleFolder)

	// Settings page
	webSettingsHandler := web.NewSettingsHandler(store, config, userStorage, accountStorage, labelStorage)
	protected.Get("/settings", webSettingsHandler.ShowSettings)

	// API routes
	apiRoutes := protected.Group("/api")
	{
		// Email routes
		apiRoutes.Get("/email/:id", webEmailHandler.HandleEmailView)
		apiRoutes.Delete("/email/:id", webEmailHandler.HandleDeleteEmail)
		apiRoutes.Put("/email/:id/read", webEmailHandler.HandleMarkRead)
		apiRoutes.Put("/email/:id/unread", webEmailHandler.HandleMarkUnread)
		apiRoutes.Post("/email/:id/move", webEmailHandler.HandleMoveEmail)

		// Reply and forward routes
		replyHandler := web.NewReplyHandler(store, config, webAuthHandler)
		apiRoutes.Get("/reply/:id", replyHandler.HandleReply)
		apiRoutes.Get("/replyall/:id", replyHandler.HandleReplyAll)
		apiRoutes.Get("/forward/:id", replyHandler.HandleForward)

		// Folder routes
		apiRoutes.Get("/folder/:name/emails", webEmailHandler.HandleFolderEmails)
		apiRoutes.Post("/folder", folderHandler.CreateFolder)
		apiRoutes.Delete("/folder/:name", folderHandler.DeleteFolder)
		apiRoutes.Put("/folder", folderHandler.RenameFolder)

		// Composition routes
		apiRoutes.Post("/compose", webEmailHandler.HandleComposeEmail)

		// Search routes
		apiRoutes.Post("/search", searchHandler.HandleSearch)

		// Account management routes
		apiRoutes.Get("/accounts", accountHandler.GetAccounts)
		apiRoutes.Post("/accounts", accountHandler.CreateAccount)
		apiRoutes.Get("/accounts/:id", accountHandler.GetAccount)
		apiRoutes.Put("/accounts/:id", accountHandler.UpdateAccount)
		apiRoutes.Delete("/accounts/:id", accountHandler.DeleteAccount)
		apiRoutes.Post("/accounts/:id/default", accountHandler.SetDefaultAccount)
		apiRoutes.Post("/accounts/:id/switch", accountHandler.SwitchAccount)

		// Label routes
		apiRoutes.Get("/labels", labelHandler.GetLabels)
		apiRoutes.Post("/labels", labelHandler.CreateLabel)
		apiRoutes.Delete("/labels/:id", labelHandler.DeleteLabel)
		apiRoutes.Post("/emails/:emailId/labels/:labelId", labelHandler.AssignLabel)
		apiRoutes.Delete("/emails/:emailId/labels/:labelId", labelHandler.RemoveLabel)
		apiRoutes.Get("/emails/:emailId/labels", labelHandler.GetEmailLabels)

		// i18n routes
		apiRoutes.Get("/i18n/:lang", i18nHandler.GetTranslations)

		// Draft routes
		draftHandler := api.NewDraftHandler(store, draftStorage)
		apiRoutes.Get("/drafts", draftHandler.GetDrafts)
		apiRoutes.Get("/drafts/:id", draftHandler.GetDraft)
		apiRoutes.Post("/drafts", draftHandler.SaveDraft)
		apiRoutes.Post("/drafts/autosave", draftHandler.AutoSave)
		apiRoutes.Delete("/drafts/:id", draftHandler.DeleteDraft)

		// Settings routes
		apiRoutes.Post("/settings/general", webSettingsHandler.UpdateGeneralSettings)
	}

	// HTMX routes (partial template renders)
	htmx := protected.Group("/htmx")
	{
		htmx.Get("/email/:id", webEmailHandler.HandleEmailView)
		htmx.Get("/folder/:name/emails", webEmailHandler.HandleFolderEmails)
	}

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 404 Handler for undefined routes
	app.Use(func(c *fiber.Ctx) error {
		localizer := c.Locals("localizer").(*i18n.Localizer)
		
		if isAPIRequest(c) {
			return c.Status(404).JSON(fiber.Map{
				"error": utils.T(localizer, "error_404"),
			})
		}
		return c.Status(404).Render("error", fiber.Map{
			"Error": utils.T(localizer, "error_404"),
			"Code":  404,
		})
	})

	// Start server
	utils.Log.Info("Starting server on port %d...", config.Server.Port)
	if err := app.Listen(fmt.Sprintf(":%d", config.Server.Port)); err != nil {
		utils.Log.Error("Error starting server: %v", err)
	}
}
