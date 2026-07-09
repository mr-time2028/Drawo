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

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) GetProfile(ctx context.Context, id string) (*domain.UserWithProfile, error) {
	args := m.Called(ctx, id)
    if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.UserWithProfile), args.Error(1)
}
func (m *mockUserService) UpdateProfile(ctx context.Context, id string, p domain.Profile) (*domain.Profile, error) {
	args := m.Called(ctx, id, p)
    if args.Get(0) == nil { return nil, args.Error(1) }
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
	
	router.Use(func(c *gin.Context) {
		c.Set("userID", "u1")
		c.Next()
	})

	router.GET("/profile", ctrl.GetProfile)
	router.PATCH("/profile", ctrl.UpdateProfile)
    router.POST("/username", ctrl.ChangeUsername)
    router.POST("/verify/request", ctrl.RequestVerification)
    router.POST("/verify/confirm", ctrl.ConfirmVerification)

	t.Run("GetProfile_Success", func(t *testing.T) {
		svc.On("GetProfile", mock.Anything, "u1").Return(&domain.UserWithProfile{}, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/profile", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

    t.Run("GetProfile_Fail", func(t *testing.T) {
		svc.On("GetProfile", mock.Anything, "u1").Return(nil, errors.New(errors.ErrNotFound, "x")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/profile", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("UpdateProfile_Success", func(t *testing.T) {
		svc.On("UpdateProfile", mock.Anything, "u1", mock.Anything).Return(&domain.Profile{}, nil).Once()
		w := httptest.NewRecorder()
		body := `{"theme":"dark"}`
		req, _ := http.NewRequest("PATCH", "/profile", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

    t.Run("UpdateProfile_InvalidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"theme":`
		req, _ := http.NewRequest("PATCH", "/profile", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

    t.Run("UpdateProfile_ValidationFail", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"locale":"too_long"}`
		req, _ := http.NewRequest("PATCH", "/profile", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

    t.Run("UpdateProfile_ServiceFail", func(t *testing.T) {
        svc.On("UpdateProfile", mock.Anything, "u1", mock.Anything).Return(nil, errors.New(errors.ErrInternalServer, "fail")).Once()
		w := httptest.NewRecorder()
		body := `{"theme":"dark"}`
		req, _ := http.NewRequest("PATCH", "/profile", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
    
    t.Run("ChangeUsername_Success", func(t *testing.T) {
		svc.On("ChangeUsername", mock.Anything, "u1", "newname").Return(nil).Once()
		w := httptest.NewRecorder()
		body := `{"username":"newname"}`
		req, _ := http.NewRequest("POST", "/username", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

    t.Run("ChangeUsername_InvalidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"username":`
		req, _ := http.NewRequest("POST", "/username", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

    t.Run("ChangeUsername_ValidationFail", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"username":"a"}` // too short
		req, _ := http.NewRequest("POST", "/username", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

    t.Run("ChangeUsername_ServiceFail", func(t *testing.T) {
        svc.On("ChangeUsername", mock.Anything, "u1", "fail").Return(errors.New(errors.ErrConflict, "taken")).Once()
		w := httptest.NewRecorder()
		body := `{"username":"fail"}`
		req, _ := http.NewRequest("POST", "/username", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

    t.Run("VerifyRequest_Success", func(t *testing.T) {
		svc.On("RequestVerification", mock.Anything, "u1", domain.OTPEmail).Return(nil).Once()
		w := httptest.NewRecorder()
		body := `{"type":"email"}`
		req, _ := http.NewRequest("POST", "/verify/request", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

    t.Run("VerifyRequest_InvalidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"type":`
		req, _ := http.NewRequest("POST", "/verify/request", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

    t.Run("VerifyRequest_ServiceFail", func(t *testing.T) {
        svc.On("RequestVerification", mock.Anything, "u1", domain.OTPEmail).Return(errors.New(errors.ErrInternalServer, "fail")).Once()
		w := httptest.NewRecorder()
		body := `{"type":"email"}`
		req, _ := http.NewRequest("POST", "/verify/request", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

    t.Run("VerifyConfirm_Success", func(t *testing.T) {
		svc.On("ConfirmVerification", mock.Anything, "u1", "123456", domain.OTPEmail).Return(nil).Once()
		w := httptest.NewRecorder()
		body := `{"type":"email", "code":"123456"}`
		req, _ := http.NewRequest("POST", "/verify/confirm", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

    t.Run("VerifyConfirm_InvalidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"type":`
		req, _ := http.NewRequest("POST", "/verify/confirm", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

    t.Run("VerifyConfirm_ServiceFail", func(t *testing.T) {
        svc.On("ConfirmVerification", mock.Anything, "u1", "123456", domain.OTPEmail).Return(errors.New(errors.ErrUnauthorized, "bad code")).Once()
		w := httptest.NewRecorder()
		body := `{"type":"email", "code":"123456"}`
		req, _ := http.NewRequest("POST", "/verify/confirm", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
