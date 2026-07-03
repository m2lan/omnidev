// Package errors provides unified error types for the application.
package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application-level error with HTTP status code.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// StatusCode returns the HTTP status code.
func (e *AppError) StatusCode() int {
	return e.Code
}

// --- Predefined errors ---

var (
	ErrNotFound = &AppError{
		Code:    http.StatusNotFound,
		Message: "resource not found",
	}

	ErrUnauthorized = &AppError{
		Code:    http.StatusUnauthorized,
		Message: "unauthorized",
	}

	ErrForbidden = &AppError{
		Code:    http.StatusForbidden,
		Message: "forbidden",
	}

	ErrValidation = &AppError{
		Code:    http.StatusBadRequest,
		Message: "validation error",
	}

	ErrConflict = &AppError{
		Code:    http.StatusConflict,
		Message: "resource already exists",
	}

	ErrInternal = &AppError{
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
	}

	ErrTooManyRequests = &AppError{
		Code:    http.StatusTooManyRequests,
		Message: "too many requests",
	}

	ErrServiceUnavailable = &AppError{
		Code:    http.StatusServiceUnavailable,
		Message: "service unavailable",
	}
)

// New creates a new AppError.
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, message string) *AppError {
	if err == nil {
		return nil
	}

	// If it's already an AppError, preserve the code
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:    appErr.Code,
			Message: message,
			Detail:  appErr.Message,
			Err:     err,
		}
	}

	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Detail:  err.Error(),
		Err:     err,
	}
}

// WrapWithCode wraps an error with a specific status code.
func WrapWithCode(err error, code int, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NotFound creates a not-found error with detail.
func NotFound(resource string) *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

// Validation creates a validation error with detail.
func Validation(detail string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: "validation error",
		Detail:  detail,
	}
}

// Conflict creates a conflict error with detail.
func Conflict(detail string) *AppError {
	return &AppError{
		Code:    http.StatusConflict,
		Message: "resource already exists",
		Detail:  detail,
	}
}

// Is checks if the target error is an AppError with the given code.
func Is(err error, target *AppError) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == target.Code
	}
	return false
}

// Code extracts the HTTP status code from an error.
// Returns 500 for non-AppError errors.
func Code(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return http.StatusInternalServerError
}
