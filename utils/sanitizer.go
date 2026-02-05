package utils

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var (
	// StrictPolicy for user-generated content
	StrictPolicy *bluemonday.Policy
	// UGCPolicy for rich text content
	UGCPolicy *bluemonday.Policy
)

func init() {
	// Initialize strict policy
	StrictPolicy = bluemonday.StrictPolicy()

	// Initialize UGC (User Generated Content) policy
	UGCPolicy = bluemonday.UGCPolicy()

	// Allow additional safe elements for email content
	UGCPolicy.AllowElements("p", "br", "div", "span", "h1", "h2", "h3", "h4", "h5", "h6")
	UGCPolicy.AllowElements("strong", "em", "u", "s", "code", "pre")
	UGCPolicy.AllowElements("ul", "ol", "li")
	UGCPolicy.AllowElements("blockquote")
	UGCPolicy.AllowElements("a", "img")
	UGCPolicy.AllowElements("table", "thead", "tbody", "tr", "th", "td")

	// Allow safe attributes
	UGCPolicy.AllowAttrs("href").OnElements("a")
	UGCPolicy.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")
	UGCPolicy.AllowAttrs("class", "id").Globally()
	UGCPolicy.AllowAttrs("style").OnElements("span", "div", "p")

	// Require URLs to be safe
	UGCPolicy.RequireParseableURLs(true)
	UGCPolicy.AllowURLSchemes("http", "https", "mailto")
}

// SanitizeHTML sanitizes HTML content using the UGC policy
func SanitizeHTML(html string) string {
	return UGCPolicy.Sanitize(html)
}

// SanitizeHTMLStrict sanitizes HTML content using the strict policy (removes all HTML)
func SanitizeHTMLStrict(html string) string {
	return StrictPolicy.Sanitize(html)
}

// StripHTML removes all HTML tags from content
func StripHTML(html string) string {
	return StrictPolicy.Sanitize(html)
}

// NormalizeSubject normalizes email subject for threading
func NormalizeSubject(subject string) string {
	// Convert to lowercase
	subject = strings.ToLower(strings.TrimSpace(subject))

	// Remove common prefixes
	prefixes := []string{"re:", "fwd:", "fw:", "aw:", "wg:"}
	for {
		trimmed := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(subject, prefix) {
				subject = strings.TrimSpace(strings.TrimPrefix(subject, prefix))
				trimmed = true
				break
			}
		}
		if !trimmed {
			break
		}
	}

	return subject
}

// GenerateThreadID generates a unique thread ID from the normalized subject
func GenerateThreadID(normalizedSubject string) string {
	hash := sha256.Sum256([]byte(normalizedSubject))
	return fmt.Sprintf("%x", hash[:16])
}
