// Package errors defines application-wide error types and HTTP response helpers.
//
// Responsibility:
//   - Provide sentinel errors that business logic can return.
//   - Map those errors to HTTP status codes and JSON responses.
//   - Keep HTTP concerns out of domain/application packages.
//
// Why not use the standard errors package everywhere?
//
//	Sentinel errors make it easy for controllers to decide status codes without
//	parsing strings. The application layer stays framework-agnostic.
package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Sentinel errors returned by the application layer.
var (
	ErrInternalServer   = errors.New("internal server error")
	ErrBadRequest       = errors.New("bad request")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrTooManyRequests  = errors.New("too many requests")
	ErrValidationFailed = errors.New("validation failed")
)

// AppError pairs a sentinel error with a user-facing message and optional field.
//
// This is the only error type HTTP controllers should receive from
// Keeping the original sentinel error lets the controller map it to a status code,
// while the message is safe to show to the user.
type AppError struct {
	Err     error
	Field   string
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Err.Error(), e.Message)
}

// New creates a new AppError.
func New(err error, message string) *AppError {
	return &AppError{Err: err, Message: message}
}

// Newf creates a new AppError with a formatted message.
func Newf(err error, format string, args ...interface{}) *AppError {
	return &AppError{Err: err, Message: fmt.Sprintf(format, args...)}
}

// WithField adds a field name to an AppError (useful for validation errors).
func (e *AppError) WithField(field string) *AppError {
	e.Field = field
	return e
}

// WithCode adds a stable machine-readable code to an AppError.
//
// The human-facing message can be translated, but the code should stay stable so
// frontend code can branch on it without parsing text.
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// Response builds a Gin response from an AppError.
func (e *AppError) Response() (int, gin.H) {
	status := mapStatus(e.Err)

	if status == http.StatusInternalServerError {
		// Never leak internal details on 500 responses.
		return status, gin.H{"message": "internal server error"}
	}

	if e.Field != "" {
		body := gin.H{"message": gin.H{e.Field: []string{e.Message}}}
		if e.Code != "" {
			body["code"] = e.Code
		}
		return status, body
	}

	body := gin.H{"message": e.Message}
	if e.Code != "" {
		body["code"] = e.Code
	}
	return status, body
}

// mapStatus converts sentinel errors to HTTP status codes.
func mapStatus(err error) int {
	switch {
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrValidationFailed):
		return http.StatusUnprocessableEntity
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrTooManyRequests):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// ValidationError builds a 422-style validation response from a field map.
func ValidationError(fieldErrors map[string][]string) (int, gin.H) {
	return http.StatusUnprocessableEntity, gin.H{"message": fieldErrors}
}
