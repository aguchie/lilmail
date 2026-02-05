package models

// Label represents an email label/tag
type Label struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Color   string `json:"color"` // Hex code, e.g. "#FF0000"
}

// EmailLabel represents a many-to-many relationship between emails and labels
type EmailLabel struct {
	EmailID string `json:"email_id"`
	LabelID string `json:"label_id"`
}
