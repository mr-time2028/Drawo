package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

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
			appErr:     &AppError{Err: errors.New("unknown"), Message: "oops"},
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
