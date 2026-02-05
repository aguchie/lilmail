package utils

import (
	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	// Bundle is the global translation bundle
	Bundle *i18n.Bundle
	// Localizer is the default localizer
	Localizer *i18n.Localizer
)

// InitI18n initializes the i18n system
func InitI18n() error {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// Load English locale file
	_, err := Bundle.LoadMessageFile("locales/active.en.toml")
	if err != nil {
		Log.Warn("Failed to load English locale: %v", err)
	}

	// Load Japanese locale file
	_, err = Bundle.LoadMessageFile("locales/active.ja.toml")
	if err != nil {
		Log.Warn("Failed to load Japanese locale: %v", err)
	}

	// Set default localizer to English
	Localizer = i18n.NewLocalizer(Bundle, language.English.String())

	Log.Info("i18n system initialized successfully")
	return nil
}

// GetLocalizer returns a localizer for the specified language
func GetLocalizer(lang string) *i18n.Localizer {
	if lang == "" {
		lang = "en"
	}
	return i18n.NewLocalizer(Bundle, lang)
}

// T translates a message ID
func T(localizer *i18n.Localizer, messageID string) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err != nil {
		Log.Debug("Translation error for '%s': %v", messageID, err)
		return messageID
	}
	return msg
}

// TWithData translates a message ID with template data
func TWithData(localizer *i18n.Localizer, messageID string, data map[string]interface{}) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		Log.Debug("Translation error for '%s': %v", messageID, err)
		return messageID
	}
	return msg
}

// TPlural translates a message ID with plural support
func TPlural(localizer *i18n.Localizer, messageID string, count int) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:   messageID,
		PluralCount: count,
		TemplateData: map[string]interface{}{
			"Count": count,
		},
	})
	if err != nil {
		Log.Debug("Translation error for '%s': %v", messageID, err)
		return messageID
	}
	return msg
}
