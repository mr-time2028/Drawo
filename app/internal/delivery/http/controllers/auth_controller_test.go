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

func TestAuthController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	ctrl := NewAuthController(svc)
	router := gin.New()
	router.POST("/register", ctrl.Register)
	router.POST("/login", ctrl.Login)
	router.POST("/refresh", ctrl.Refresh)

	t.Run("Register_Success", func(t *testing.T) {
		svc.On("Register", mock.Anything, "hamid", "pass12345").Return(&domain.User{ID: "1", Username: "hamid"}, nil).Once()
		
		body := `{"username":"hamid", "password":"pass12345"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/register", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "hamid")
	})

	t.Run("Register_InvalidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/register", bytes.NewBufferString(`{invalid`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

    t.Run("Login_Success", func(t *testing.T) {
		svc.On("Login", mock.Anything, "hamid", "pass", mock.Anything, mock.Anything).Return(&domain.TokenPair{AccessToken: "acc"}, nil).Once()
		
		body := `{"username":"hamid", "password":"pass"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "acc")
	})

    t.Run("Login_InvalidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(`{invalid`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

    t.Run("Login_ServiceFail", func(t *testing.T) {
        svc.On("Login", mock.Anything, "fail", "pass", mock.Anything, mock.Anything).Return(nil, errors.New(errors.ErrUnauthorized, "bad")).Once()
		body := `{"username":"fail", "password":"pass"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
    })

    t.Run("Refresh_Success", func(t *testing.T) {
		svc.On("Refresh", mock.Anything, "old").Return(&domain.TokenPair{AccessToken: "new"}, nil).Once()
		
		body := `{"refresh_token":"old"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/refresh", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "new")
	})

    t.Run("Refresh_InvalidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/refresh", bytes.NewBufferString(`{invalid`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

    t.Run("Refresh_ServiceFail", func(t *testing.T) {
        svc.On("Refresh", mock.Anything, "fail").Return(nil, errors.New(errors.ErrUnauthorized, "bad")).Once()
		body := `{"refresh_token":"fail"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/refresh", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
    })
}

func TestAuthController_Logout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	ctrl := NewAuthController(svc)
	router := gin.New()
	router.POST("/logout", ctrl.Logout)

	t.Run("Success", func(t *testing.T) {
		svc.On("Logout", mock.Anything, "valid-token").Return(nil).Once()
		
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/logout", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "successfully")
	})

	t.Run("NoHeader", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/logout", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "already logged out")
	})
}

