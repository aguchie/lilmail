package api

import (
	"lilmail/utils"

	"github.com/gofiber/fiber/v2"
)

// I18nHandler handles i18n-related requests
type I18nHandler struct{}

// GetTranslations returns translations for the client-side JavaScript
func (h *I18nHandler) GetTranslations(c *fiber.Ctx) error {
	lang := c.Params("lang")
	if lang == "" {
		lang = "en"
	}

	// Only allow supported languages
	if lang != "en" && lang != "ja" {
		lang = "en"
	}

	localizer := utils.GetLocalizer(lang)

	// Create a map of common translation keys for client-side use
	translations := map[string]string{
		"message_sent_success":   utils.T(localizer, "message_sent_success"),
		"message_saved_draft":    utils.T(localizer, "message_saved_draft"),
		"message_deleted":        utils.T(localizer, "message_deleted"),
		"message_error":          utils.T(localizer, "message_error"),
		"message_connection_error": utils.T(localizer, "message_connection_error"),
		"confirm_delete_email":   utils.T(localizer, "confirm_delete_email"),
		"confirm_yes":            utils.T(localizer, "confirm_yes"),
		"confirm_no":             utils.T(localizer, "confirm_no"),
		"email_loading":          utils.T(localizer, "email_loading"),
		"email_no_messages":      utils.T(localizer, "email_no_messages"),
		"error_network":          utils.T(localizer, "error_network"),
		"error_404":              utils.T(localizer, "error_404"),
		"error_500":              utils.T(localizer, "error_500"),
	}

	return c.JSON(translations)
}
