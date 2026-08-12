package errors

import (
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAppError(t *testing.T) {
	err := New(ErrNotFound, "not found")
	assert.Contains(t, err.Error(), "not found")

	errf := Newf(ErrInternalServer, "error %d", 500)
	assert.Equal(t, "internal server error: error 500", errf.Error())

	errField := err.WithField("id")
	assert.Equal(t, "id", errField.Field)

	errCode := err.WithCode("account_banned")
	assert.Equal(t, "account_banned", errCode.Code)
}

func TestAppErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		appErr     *AppError
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "bad request",
			appErr:     New(ErrBadRequest, "invalid input"),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid input",
		},
		{
			name:       "unauthorized",
			appErr:     New(ErrUnauthorized, "unauthorized"),
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "unauthorized",
		},
		{
			name:       "forbidden",
			appErr:     New(ErrForbidden, "forbidden"),
			wantStatus: http.StatusForbidden,
			wantMsg:    "forbidden",
		},
		{
			name:       "not found",
			appErr:     New(ErrNotFound, "not found"),
			wantStatus: http.StatusNotFound,
			wantMsg:    "not found",
		},
		{
			name:       "conflict",
			appErr:     New(ErrConflict, "conflict"),
			wantStatus: http.StatusConflict,
			wantMsg:    "conflict",
		},
		{
			name:       "too many requests",
			appErr:     New(ErrTooManyRequests, "slow down"),
			wantStatus: http.StatusTooManyRequests,
			wantMsg:    "slow down",
		},
		{
			name:       "validation error with field",
			appErr:     New(ErrValidationFailed, "too short").WithField("username"),
			wantStatus: http.StatusUnprocessableEntity,
			wantMsg:    "too short",
		},
		{
			name:       "internal error hides details",
			appErr:     Newf(ErrInternalServer, "database timeout: %v", "postgres"),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "internal server error",
		},
		{
			name:       "unknown code returns internal",
			appErr:     &AppError{Err: stderrors.New("unknown"), Message: "oops"},
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := tt.appErr.Response()
			assert.Equal(t, tt.wantStatus, status)
			if status == http.StatusInternalServerError {
				assert.Equal(t, tt.wantMsg, body["message"])
			} else if tt.appErr.Field != "" {
				msg := body["message"].(gin.H)
				fieldMsgs := msg[tt.appErr.Field].([]string)
				assert.Contains(t, fieldMsgs[0], tt.wantMsg)
			} else {
				assert.Equal(t, tt.wantMsg, body["message"])
			}
		})
	}
}

func TestAppErrorResponseWithCode(t *testing.T) {
	status, body := New(ErrForbidden, "banned").WithCode("account_banned").Response()
	assert.Equal(t, http.StatusForbidden, status)
	assert.Equal(t, "banned", body["message"])
	assert.Equal(t, "account_banned", body["code"])
}

func TestValidationError(t *testing.T) {
	fields := map[string][]string{"username": {"required"}}
	status, body := ValidationError(fields)
	assert.Equal(t, http.StatusUnprocessableEntity, status)
	assert.Equal(t, fields, body["message"])
}

// TestRespondError_AppError verifies that RespondError returns the correct
// status and body for expected (AppError) failures.
func TestRespondError_AppError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/test", nil)

	RespondError(c, New(ErrNotFound, "user not found"))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "user not found")
}

// TestRespondError_RawError verifies that unexpected (non-AppError) errors
// produce a 500 with a generic message (no leakage of internal details) AND
// do not panic.
func TestRespondError_RawError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/test", nil)

	// Simulate an infrastructure failure that wasn't wrapped in AppError —
	// e.g. a database driver error.
	RespondError(c, stderrors.New("dial tcp 10.0.0.1:5432: connect: connection refused"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// The body must NOT contain the real error text.
	assert.NotContains(t, w.Body.String(), "connection refused")
	assert.Contains(t, w.Body.String(), "internal server error")
}

// TestRespondError_Nil is a no-op safety check.
func TestRespondError_Nil(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/test", nil)

	// Should not panic or write anything.
	RespondError(c, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, w.Body.Len())
}
