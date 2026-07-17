package controllers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"drawo/internal/core/domain"
)

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) GetProfile(ctx context.Context, id string) (*domain.UserWithProfile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserWithProfile), args.Error(1)
}
func (m *mockUserService) UpdateProfile(ctx context.Context, id string, p domain.Profile) (*domain.Profile, error) {
	args := m.Called(ctx, id, p)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Profile), args.Error(1)
}
func (m *mockUserService) ChangeUsername(ctx context.Context, id, name string) error {
	return m.Called(ctx, id, name).Error(0)
}
func (m *mockUserService) RequestVerification(ctx context.Context, id string, t domain.OTPType) error {
	return m.Called(ctx, id, t).Error(0)
}
func (m *mockUserService) ConfirmVerification(ctx context.Context, id, code string, t domain.OTPType) error {
	return m.Called(ctx, id, code, t).Error(0)
}

func TestUserController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockUserService)
	ctrl := NewUserController(svc)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userID", "u1"); c.Next() })

	router.GET("/profile", ctrl.GetProfile)
	router.POST("/verify/request", ctrl.RequestVerification)
	router.POST("/verify/confirm", ctrl.ConfirmVerification)

	t.Run("GetProfile", func(t *testing.T) {
		svc.On("GetProfile", mock.Anything, "u1").Return(&domain.UserWithProfile{}, nil).Once()
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/profile", nil)
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("RequestVerification", func(t *testing.T) {
		svc.On("RequestVerification", mock.Anything, "u1", domain.OTPEmail).Return(nil).Once()
		body := `{"type":"email"}`
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/verify/request", bytes.NewBufferString(body))
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("ConfirmVerification", func(t *testing.T) {
		svc.On("ConfirmVerification", mock.Anything, "u1", "123456", domain.OTPEmail).Return(nil).Once()
		body := `{"type":"email", "code":"123456"}`
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/verify/confirm", bytes.NewBufferString(body))
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
	})
}
