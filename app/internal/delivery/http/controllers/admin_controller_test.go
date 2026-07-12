package controllers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"drawo/internal/core/domain"
)

type mockAdminSvc struct {
	mock.Mock
}
func (m *mockAdminSvc) UploadSong(ctx context.Context, t string, st domain.SongType, r io.Reader, s int64) (*domain.Song, error) {
	args := m.Called(ctx, t, st, r, s); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Song), args.Error(1)
}
func (m *mockAdminSvc) ListSongs(ctx context.Context, st domain.SongType) ([]domain.Song, error) {
	args := m.Called(ctx, st); return args.Get(0).([]domain.Song), args.Error(1)
}
func (m *mockAdminSvc) ToggleSong(ctx context.Context, id string, a bool) error { return m.Called(ctx, id, a).Error(0) }
func (m *mockAdminSvc) DeleteSong(ctx context.Context, id string) error { return m.Called(ctx, id).Error(0) }
func (m *mockAdminSvc) SearchUsers(ctx context.Context, q string) ([]domain.UserWithProfile, error) {
	args := m.Called(ctx, q); return args.Get(0).([]domain.UserWithProfile), args.Error(1)
}
func (m *mockAdminSvc) BanUser(ctx context.Context, id string) error { return m.Called(ctx, id).Error(0) }
func (m *mockAdminSvc) UnbanUser(ctx context.Context, id string) error { return m.Called(ctx, id).Error(0) }
func (m *mockAdminSvc) UpdateGlobalSetting(ctx context.Context, k, v string) error { return m.Called(ctx, k, v).Error(0) }

func TestAdminController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAdminSvc)
	ctrl := NewAdminController(svc)
	router := gin.New()

	router.POST("/songs", ctrl.UploadSong)
	router.GET("/songs", ctrl.ListSongs)
	router.PATCH("/songs/:id/toggle", ctrl.ToggleSong)
	router.DELETE("/songs/:id", ctrl.DeleteSong)
	router.GET("/users/search", ctrl.SearchUsers)
	router.POST("/users/:id/ban", ctrl.BanUser)
	router.POST("/users/:id/unban", ctrl.UnbanUser)
	router.PATCH("/settings/:key", ctrl.UpdateSetting)

	t.Run("UploadSong", func(t *testing.T) {
		svc.On("UploadSong", mock.Anything, "T", domain.SongTypeLanding, mock.Anything, mock.Anything).Return(&domain.Song{}, nil).Once()
		body := &bytes.Buffer{}; mw := multipart.NewWriter(body)
		part, _ := mw.CreateFormFile("song", "a.mp3"); part.Write([]byte("data")); mw.Close()
		req, _ := http.NewRequest("POST", "/songs?title=T&type=landing", body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		resp := httptest.NewRecorder(); router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusCreated, resp.Code)
	})

	t.Run("ListSongs", func(t *testing.T) {
		svc.On("ListSongs", mock.Anything, domain.SongTypeLanding).Return([]domain.Song{}, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/songs?type=landing", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
    
    t.Run("ToggleSong", func(t *testing.T) {
		svc.On("ToggleSong", mock.Anything, "id", true).Return(nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/songs/id/toggle", bytes.NewBufferString(`{"active":true}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("BanUser", func(t *testing.T) {
		svc.On("BanUser", mock.Anything, "u1").Return(nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/users/u1/ban", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
