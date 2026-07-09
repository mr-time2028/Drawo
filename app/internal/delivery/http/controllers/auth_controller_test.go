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
	"drawo/internal/core/ports/services"
	"drawo/pkg/errors"
)

// mockAuthService is a mock implementation of services.AuthService.
type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) Register(ctx context.Context, u, p string) (*domain.User, error) {
	args := m.Called(ctx, u, p)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockAuthService) Login(ctx context.Context, u, p, ip, ua string) (*domain.TokenPair, error) {
	args := m.Called(ctx, u, p, ip, ua)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.TokenPair), args.Error(1)
}

func (m *mockAuthService) Refresh(ctx context.Context, rt string) (*domain.TokenPair, error) {
	args := m.Called(ctx, rt)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.TokenPair), args.Error(1)
}

func (m *mockAuthService) Logout(ctx context.Context, at string) error {
	return m.Called(ctx, at).Error(0)
}

func TestAuthController_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	// We cast to services.AuthService to ensure it implements it
	var _ services.AuthService = svc
	ctrl := NewAuthController(svc)
	router := gin.New()
	router.POST("/register", ctrl.Register)

	t.Run("Success", func(t *testing.T) {
		svc.On("Register", mock.Anything, "hamid", "pass12345").Return(&domain.User{ID: "1", Username: "hamid"}, nil).Once()
		
		body := `{"username":"hamid", "password":"pass12345"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/register", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "hamid")
	})

	t.Run("ValidationError", func(t *testing.T) {
		body := `{"username":"hi"}` // too short
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/register", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
    
    t.Run("ServiceError", func(t *testing.T) {
		svc.On("Register", mock.Anything, "fail", "pass12345").Return(nil, errors.New(errors.ErrConflict, "exists")).Once()
		
		body := `{"username":"fail", "password":"pass12345"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/register", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestAuthController_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	ctrl := NewAuthController(svc)
	router := gin.New()
	router.POST("/login", ctrl.Login)

	t.Run("Success", func(t *testing.T) {
		svc.On("Login", mock.Anything, "hamid", "pass", mock.Anything, mock.Anything).Return(&domain.TokenPair{AccessToken: "acc"}, nil).Once()
		
		body := `{"username":"hamid", "password":"pass"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "acc")
	})
}

func TestAuthController_Refresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	ctrl := NewAuthController(svc)
	router := gin.New()
	router.POST("/refresh", ctrl.Refresh)

	t.Run("Success", func(t *testing.T) {
		svc.On("Refresh", mock.Anything, "old").Return(&domain.TokenPair{AccessToken: "new"}, nil).Once()
		
		body := `{"refresh_token":"old"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/refresh", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "new")
	})
}
