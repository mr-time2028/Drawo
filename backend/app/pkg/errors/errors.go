// Package errors defines application-wide error types and HTTP response helpers.
//
// Responsibility:
//   - Provide sentinel errors that business logic can return.
//   - Map those errors to HTTP status codes and JSON responses.
//   - Log UNEXPECTED (non-AppError) errors — real bugs, DB failures, panics
//     caught by Recovery, etc. — to stderr using the standard library logger
//     (plain log.Printf output, no structured logger required).
//   - Keep HTTP concerns out of domain/application packages.
//
// Why not use the standard errors package everywhere?
//
//	Sentinel errors make it easy for controllers to decide status codes without
//	parsing strings. The application layer stays framework-agnostic.
package errors

import (
	stdErrors "errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestIDKey is the Gin context key where RequestID middleware stores the
// request ID. We pull it into error logs so terminal output can be cross-
// referenced with the access log line (which already prints request_id).
const RequestIDKey = "X-Request-ID"

// Sentinel errors returned by the application layer.
var (
	ErrInternalServer   = stdErrors.New("internal server error")
	ErrBadRequest       = stdErrors.New("bad request")
	ErrUnauthorized     = stdErrors.New("unauthorized")
	ErrForbidden        = stdErrors.New("forbidden")
	ErrNotFound         = stdErrors.New("not found")
	ErrConflict         = stdErrors.New("conflict")
	ErrGone             = stdErrors.New("gone")
	ErrTooManyRequests  = stdErrors.New("too many requests")
	ErrValidationFailed = stdErrors.New("validation failed")
)

// AppError pairs a sentinel error with a user-facing message and optional field.
//
// This is the only error type HTTP controllers should receive from services
// for expected failure modes (validation, auth, not found, conflict, etc.).
// Unexpected/unknown errors (DB down, nil pointer, network blip, redis.Nil
// bubbling up, etc.) come back as plain `error` values; RespondError logs
// them and answers with a generic 500 so internal details never leak.
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
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// Response builds a Gin response from an AppError.
//
// NOTE: Prefer calling RespondError(c, err) instead of calling Response()
// directly — RespondError also handles the unexpected-error / logging path.
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
	case stdErrors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	case stdErrors.Is(err, ErrValidationFailed):
		return http.StatusUnprocessableEntity
	case stdErrors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case stdErrors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case stdErrors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case stdErrors.Is(err, ErrConflict):
		return http.StatusConflict
	case stdErrors.Is(err, ErrGone):
		return http.StatusGone
	case stdErrors.Is(err, ErrTooManyRequests):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// ValidationError builds a 422-style validation response from a field map.
func ValidationError(fieldErrors map[string][]string) (int, gin.H) {
	return http.StatusUnprocessableEntity, gin.H{"message": fieldErrors}
}

// RespondError writes an error response for `err` to the Gin context.
//
// This is the single entry point controllers should use when a service call
// returns an error. It handles two cases:
//
//  1. `err` is an *AppError — expected/handled failure (validation, 401, 404,
//     conflict, etc.). Responds with the correct HTTP status and the safe
//     user-facing message from the AppError.
//  2. `err` is any other error — unexpected/bug-condition failure (database
//     error, network error, an unhandled nil deref, etc.). Logs the real
//     error to stderr via the standard library `log` package (plain,
//     timestamped lines — exactly what you asked for) with request ID and
//     path, then responds 500 with a generic message. Internal details are
//     NEVER sent to the client.
//
// Usage in controllers:
//
//	profile, err := ctrl.userSvc.GetProfile(c.Request.Context(), userID)
//	if err != nil {
//	    errors.RespondError(c, err)
//	    return
//	}
func RespondError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Use errors.As (not a direct type assertion) so wrapped AppErrors
	// (e.g. fmt.Errorf("load profile: %w", appErr)) are still recognised.
	var appErr *AppError
	if stdErrors.As(err, &appErr) {
		status, body := appErr.Response()
		c.JSON(status, body)
		return
	}

	// --- Unexpected / internal error ---
	requestID, _ := c.Get(RequestIDKey)
	log.Printf(
		"[INTERNAL SERVER ERROR] method=%s path=%s request_id=%v client_ip=%s error=%v",
		c.Request.Method,
		c.Request.URL.Path,
		requestID,
		c.ClientIP(),
		err,
	)

	c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
}
