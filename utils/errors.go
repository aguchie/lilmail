package utils

import (
	"fmt"
)

// AppError represents a custom application error with context
type AppError struct {
	Code    int                    // HTTP status code
	Message string                 // User-friendly message
	Err     error                  // Underlying error
	Context map[string]interface{} // Additional context
}

// NewAppError creates a new AppError
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]interface{}),
	}
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	e.Context[key] = value
	return e
}

// Common error constructors
func BadRequestError(message string, err error) *AppError {
	return NewAppError(400, message, err)
}

func UnauthorizedError(message string, err error) *AppError {
	return NewAppError(401, message, err)
}

func ForbiddenError(message string, err error) *AppError {
	return NewAppError(403, message, err)
}

func NotFoundError(message string, err error) *AppError {
	return NewAppError(404, message, err)
}

func InternalServerError(message string, err error) *AppError {
	return NewAppError(500, message, err)
}
