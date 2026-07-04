package errors

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		appErr     *AppError
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "bad request without field",
			appErr:     New(ErrBadRequest, "invalid input"),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid input",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := tt.appErr.Response()
			assert.Equal(t, tt.wantStatus, status)
			assert.Contains(t, body, "message")
		})
	}
}
