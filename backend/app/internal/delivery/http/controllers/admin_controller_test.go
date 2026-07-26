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
	args := m.Called(ctx, t, st, r, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Song), args.Error(1)
}
func (m *mockAdminSvc) ListSongs(ctx context.Context, st domain.SongType) ([]domain.Song, error) {
	args := m.Called(ctx, st)
	return args.Get(0).([]domain.Song), args.Error(1)
}
func (m *mockAdminSvc) ToggleSong(ctx context.Context, id string, a bool) error {
	return m.Called(ctx, id, a).Error(0)
}
func (m *mockAdminSvc) DeleteSong(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockAdminSvc) SearchUsers(ctx context.Context, q string) ([]domain.UserWithProfile, error) {
	args := m.Called(ctx, q)
	return args.Get(0).([]domain.UserWithProfile), args.Error(1)
}
func (m *mockAdminSvc) BanUser(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockAdminSvc) UnbanUser(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockAdminSvc) CreateCategory(ctx context.Context, name, language, groupID string) (*domain.Category, error) {
	args := m.Called(ctx, name, language, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Category), args.Error(1)
}
func (m *mockAdminSvc) ListCategories(ctx context.Context, language string) ([]domain.Category, error) {
	args := m.Called(ctx, language)
	return args.Get(0).([]domain.Category), args.Error(1)
}
func (m *mockAdminSvc) DeleteCategory(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockAdminSvc) CreateWord(ctx context.Context, categoryID, groupID, text, language string, points int) (*domain.Word, error) {
	args := m.Called(ctx, categoryID, groupID, text, language, points)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Word), args.Error(1)
}
func (m *mockAdminSvc) ListWords(ctx context.Context, categoryID, language string) ([]domain.Word, error) {
	args := m.Called(ctx, categoryID, language)
	return args.Get(0).([]domain.Word), args.Error(1)
}
func (m *mockAdminSvc) DeleteWord(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockAdminSvc) ListReports(ctx context.Context, status domain.ReportStatus, paging domain.Paging) (*domain.PageOf[domain.Report], error) {
	args := m.Called(ctx, status, paging)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PageOf[domain.Report]), args.Error(1)
}
func (m *mockAdminSvc) GetReport(ctx context.Context, id string) (*domain.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Report), args.Error(1)
}
func (m *mockAdminSvc) ConfirmReport(ctx context.Context, id, adminID, note string) error {
	return m.Called(ctx, id, adminID, note).Error(0)
}
func (m *mockAdminSvc) RejectReport(ctx context.Context, id, adminID, note string) error {
	return m.Called(ctx, id, adminID, note).Error(0)
}

func (m *mockAdminSvc) UpdateGlobalSetting(ctx context.Context, k, v string) error {
	return m.Called(ctx, k, v).Error(0)
}
func (m *mockAdminSvc) CreateBadWord(ctx context.Context, text, language string) (*domain.BadWord, error) {
	args := m.Called(ctx, text, language)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.BadWord), args.Error(1)
}
func (m *mockAdminSvc) ListBadWords(ctx context.Context, language string) ([]domain.BadWord, error) {
	args := m.Called(ctx, language)
	return args.Get(0).([]domain.BadWord), args.Error(1)
}
func (m *mockAdminSvc) DeleteBadWord(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

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
	router.POST("/bad-words", ctrl.CreateBadWord)
	router.GET("/bad-words", ctrl.ListBadWords)
	router.DELETE("/bad-words/:id", ctrl.DeleteBadWord)

	t.Run("UploadSong", func(t *testing.T) {
		svc.On("UploadSong", mock.Anything, "T", domain.SongTypeLanding, mock.Anything, mock.Anything).Return(&domain.Song{}, nil).Once()
		body := &bytes.Buffer{}
		mw := multipart.NewWriter(body)
		part, _ := mw.CreateFormFile("song", "a.mp3")
		part.Write([]byte("data"))
		mw.Close()
		req, _ := http.NewRequest("POST", "/songs?title=T&type=landing", body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
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

func TestAdminControllerBadWords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAdminSvc)
	ctrl := NewAdminController(svc)
	router := gin.New()
	router.POST("/bad-words", ctrl.CreateBadWord)
	router.GET("/bad-words", ctrl.ListBadWords)
	router.DELETE("/bad-words/:id", ctrl.DeleteBadWord)

	svc.On("CreateBadWord", mock.Anything, "bad", "en").Return(&domain.BadWord{ID: "b1", Text: "bad", Language: "en"}, nil).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/bad-words", bytes.NewBufferString(`{"text":"bad","language":"en"}`))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	svc.On("ListBadWords", mock.Anything, "en").Return([]domain.BadWord{{ID: "b1", Text: "bad", Language: "en"}}, nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/bad-words?language=en", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	svc.On("DeleteBadWord", mock.Anything, "b1").Return(nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/bad-words/b1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminControllerReports(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAdminSvc)
	ctrl := NewAdminController(svc)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userID", "admin1"); c.Next() })
	router.GET("/reports", ctrl.ListReports)
	router.GET("/reports/:id", ctrl.GetReport)
	router.POST("/reports/:id/confirm", ctrl.ConfirmReport)
	router.POST("/reports/:id/reject", ctrl.RejectReport)

	svc.On("ListReports", mock.Anything, domain.ReportStatusPending, mock.Anything).Return(&domain.PageOf[domain.Report]{Items: []domain.Report{}}, nil).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/reports?status=pending", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	svc.On("GetReport", mock.Anything, "r1").Return(&domain.Report{ID: "r1"}, nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/reports/r1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	svc.On("ConfirmReport", mock.Anything, "r1", "admin1", "ok").Return(nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/reports/r1/confirm", bytes.NewBufferString(`{"note":"ok"}`))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	svc.On("RejectReport", mock.Anything, "r1", "admin1", "no").Return(nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/reports/r1/reject", bytes.NewBufferString(`{"note":"no"}`))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminControllerDictionary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockAdminSvc)
	ctrl := NewAdminController(svc)
	router := gin.New()
	router.POST("/categories", ctrl.CreateCategory)
	router.GET("/categories", ctrl.ListCategories)
	router.DELETE("/categories/:id", ctrl.DeleteCategory)
	router.POST("/words", ctrl.CreateWord)
	router.GET("/words", ctrl.ListWords)
	router.DELETE("/words/:id", ctrl.DeleteWord)

	svc.On("CreateCategory", mock.Anything, "Animals", "en", "cg1").Return(&domain.Category{ID: "c1", Name: "Animals", Language: "en", GroupID: "cg1"}, nil).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/categories", bytes.NewBufferString(`{"name":"Animals","language":"en","group_id":"cg1"}`))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	svc.On("ListCategories", mock.Anything, "en").Return([]domain.Category{{ID: "c1"}}, nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/categories?language=en", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	svc.On("DeleteCategory", mock.Anything, "c1").Return(nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/categories/c1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	svc.On("CreateWord", mock.Anything, "c1", "wg1", "cat", "en", 1).Return(&domain.Word{ID: "w1"}, nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/words", bytes.NewBufferString(`{"category_id":"c1","group_id":"wg1","text":"cat","language":"en","points":1}`))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	svc.On("ListWords", mock.Anything, "c1", "en").Return([]domain.Word{{ID: "w1"}}, nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/words?category_id=c1&language=en", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	svc.On("DeleteWord", mock.Anything, "w1").Return(nil).Once()
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/words/w1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
