package models

// PaginatedEmails represents a paginated list of emails
type PaginatedEmails struct {
	Emails      []Email `json:"emails"`
	Page        uint32  `json:"page"`
	PageSize    uint32  `json:"page_size"`
	TotalPages  uint32  `json:"total_pages"`
	TotalEmails uint32  `json:"total_emails"`
	HasNext     bool    `json:"has_next"`
	HasPrev     bool    `json:"has_prev"`
}

// NewPaginatedEmails creates a new paginated emails response
func NewPaginatedEmails(emails []Email, page, pageSize, totalEmails uint32) *PaginatedEmails {
	totalPages := (totalEmails + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginatedEmails{
		Emails:      emails,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		TotalEmails: totalEmails,
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}
}
