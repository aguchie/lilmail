package middleware

import (
	"lilmail/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// LocaleMiddleware detects and sets the user's locale
func LocaleMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		lang := ""

		// 1. Try to get language from query parameter
		lang = c.Query("lang")

		// 2. Try to get language from cookie
		if lang == "" {
			lang = c.Cookies("lang")
		}

		// 3. Try to get language from Accept-Language header
		if lang == "" {
			acceptLang := c.Get("Accept-Language")
			if strings.HasPrefix(acceptLang, "ja") {
				lang = "ja"
			} else {
				lang = "en"
			}
		}

		// Default to English
		if lang == "" {
			lang = "en"
		}

		// Only allow supported languages
		if lang != "en" && lang != "ja" {
			lang = "en"
		}

		// Get localizer for this language
		localizer := utils.GetLocalizer(lang)

		// Store in context
		c.Locals("localizer", localizer)
		c.Locals("lang", lang)

		// Log the detected language
		utils.Log.Debug("Locale detected: %s for path: %s", lang, c.Path())

		return c.Next()
	}
}
