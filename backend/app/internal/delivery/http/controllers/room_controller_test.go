package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"drawo/internal/core/domain"
	svcports "drawo/internal/core/ports/services"
	svcerrors "drawo/pkg/errors"
)

// --- Mocks -------------------------------------------------------------------

type mockRoomService struct {
	mock.Mock
}

func (m *mockRoomService) CreateRoom(ctx context.Context, ownerID string, roomType domain.RoomType, s svcports.RoomSettings) (*domain.Room, error) {
	args := m.Called(ctx, ownerID, roomType, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}
func (m *mockRoomService) GetRoom(ctx context.Context, roomID string) (*domain.Room, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}
func (m *mockRoomService) GetRoomByInvite(ctx context.Context, inviteCode string) (*domain.Room, error) {
	args := m.Called(ctx, inviteCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}
func (m *mockRoomService) JoinRoom(ctx context.Context, inviteCode, password string) (*domain.Room, error) {
	args := m.Called(ctx, inviteCode, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}
func (m *mockRoomService) UpdateSettings(ctx context.Context, roomID, ownerID string, patch svcports.RoomSettings) (*domain.Room, error) {
	args := m.Called(ctx, roomID, ownerID, patch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}
func (m *mockRoomService) CloseRoom(ctx context.Context, roomID, ownerID string) error {
	return m.Called(ctx, roomID, ownerID).Error(0)
}
func (m *mockRoomService) StartGame(ctx context.Context, roomID, ownerID string, playerCount int) error {
	return m.Called(ctx, roomID, ownerID, playerCount).Error(0)
}
func (m *mockRoomService) LeaveRoom(ctx context.Context, roomID, userID, newOwnerID string) (*domain.Room, error) {
	args := m.Called(ctx, roomID, userID, newOwnerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}
func (m *mockRoomService) ValidateCustomCategories(cats []domain.CustomCategory) error {
	return m.Called(cats).Error(0)
}
func (m *mockRoomService) IssueGuestToken(ctx context.Context, roomID, nickname string) (*domain.GuestAuth, error) {
	args := m.Called(ctx, roomID, nickname)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GuestAuth), args.Error(1)
}
func (m *mockRoomService) ValidateGuestToken(ctx context.Context, token string) (*domain.GuestAuth, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GuestAuth), args.Error(1)
}

// --- Helpers -----------------------------------------------------------------

func setupRoomRouter(t *testing.T, setUserID string) (*gin.Engine, *mockRoomService, *mockAdminSvc) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	roomSvc := new(mockRoomService)
	adminSvc := new(mockAdminSvc)
	ctrl := NewRoomController(roomSvc, adminSvc)

	// Inject a user-id middleware that simulates OptionalAuth/RequireAuth.
	// When setUserID is empty the middleware skips setting "userID" — exactly
	// what the guest path expects (anonymous visitor).
	auth := func(c *gin.Context) {
		if setUserID != "" {
			c.Set("userID", setUserID)
		}
		c.Next()
	}

	r.POST("/api/v1/rooms", auth, ctrl.Create)
	r.GET("/api/v1/rooms/by-code/:code", ctrl.GetByCode)
	r.POST("/api/v1/rooms/by-code/:code/join", auth, ctrl.Join)
	r.GET("/api/v1/rooms/:id", auth, ctrl.Get)
	r.PATCH("/api/v1/rooms/:id", auth, ctrl.Update)
	r.POST("/api/v1/rooms/:id/start", auth, ctrl.Start)
	r.POST("/api/v1/rooms/:id/leave", auth, ctrl.Leave)
	r.POST("/api/v1/rooms/:id/close", auth, ctrl.Close)
	r.GET("/api/v1/categories", ctrl.ListCategories)
	return r, roomSvc, adminSvc
}

func sampleRoom() *domain.Room {
	now := time.Now()
	return &domain.Room{
		ID: "r-1", Name: "Test", InviteCode: "ABCDEF", OwnerID: "u-1",
		Type: domain.RoomTypePrivate, HasPassword: false,
		Language: "en", WordSource: domain.WordSourceDefault,
		State: domain.RoomStateLobby, MinPlayers: 2, MaxPlayers: 8,
		RoundTime: 80, MaxRounds: 3, CustomCategories: nil,
		CreatedAt: now, UpdatedAt: now,
	}
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), v))
}

// --- Tests -------------------------------------------------------------------

func TestRoomController_Create(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "owner-1")

	t.Run("success private", func(t *testing.T) {
		svc.On("CreateRoom", mock.Anything, "owner-1", domain.RoomTypePrivate, mock.MatchedBy(func(s svcports.RoomSettings) bool {
			return s.Name == "My Room" && s.Language == "en" && s.MaxPlayers == 6 && s.MaxRounds == 3 &&
				s.RoundTime == 80 && s.WordSource == domain.WordSourceDefault && s.Password == nil
		})).Return(sampleRoom(), nil).Once()

		body := `{"name":"My Room","language":"EN","max_players":6,"max_rounds":3,"round_time":80,"word_source":"default"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var out map[string]any
		decodeJSON(t, w, &out)
		assert.Equal(t, "Test", out["name"])
		svc.AssertExpectations(t)
	})

	t.Run("success public", func(t *testing.T) {
		svc.On("CreateRoom", mock.Anything, "owner-1", domain.RoomTypePublic, mock.Anything).
			Return(sampleRoom(), nil).Once()
		body := `{"name":"Pub","room_type":"PUBLIC","word_source":"default"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("with password and custom categories", func(t *testing.T) {
		svc.On("CreateRoom", mock.Anything, "owner-1", domain.RoomTypePrivate, mock.MatchedBy(func(s svcports.RoomSettings) bool {
			return s.Password != nil && *s.Password == "pass1234" && len(s.CustomCategories) == 1
		})).Return(sampleRoom(), nil).Once()

		body := `{"name":"C","password":"pass1234","word_source":"custom","custom_categories":[{"name":"A","words":{"1":["a","b"],"2":["c"],"3":["d"]}}]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms", bytes.NewBufferString(`{bad`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc.On("CreateRoom", mock.Anything, "owner-1", mock.Anything, mock.Anything).
			Return(nil, svcerrors.New(svcerrors.ErrBadRequest, "bad")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms", bytes.NewBufferString(`{"name":"x","word_source":"default"}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRoomController_GetByCode(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "")

	t.Run("success", func(t *testing.T) {
		r := sampleRoom()
		r.CustomCategories = []domain.CustomCategory{
			{Name: "Animals", Words: map[int][]string{1: {"cat"}}},
		}
		svc.On("GetRoomByInvite", mock.Anything, "ABCDEF").Return(r, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/rooms/by-code/abcdef", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var out map[string]any
		decodeJSON(t, w, &out)
		assert.Equal(t, "ABCDEF", out["invite_code"])
		assert.Equal(t, float64(1), out["custom_word_count"])
		// Public endpoint must NOT leak password hash.
		assert.NotContains(t, w.Body.String(), "PasswordHash")
	})

	t.Run("not found", func(t *testing.T) {
		svc.On("GetRoomByInvite", mock.Anything, "NOPE").
			Return(nil, svcerrors.New(svcerrors.ErrNotFound, "nope")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/rooms/by-code/NOPE", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRoomController_Join(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "u-2")

	t.Run("with password body", func(t *testing.T) {
		svc.On("JoinRoom", mock.Anything, "ABCDEF", "pw").Return(sampleRoom(), nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/by-code/abcdef/join",
			bytes.NewBufferString(`{"password":"pw"}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("no body works", func(t *testing.T) {
		svc.On("JoinRoom", mock.Anything, "ABCDEF", "").Return(sampleRoom(), nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/by-code/abcdef/join", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("bad password", func(t *testing.T) {
		svc.On("JoinRoom", mock.Anything, "ABCDEF", "wrong").
			Return(nil, svcerrors.New(svcerrors.ErrForbidden, "wrong")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/by-code/abcdef/join",
			bytes.NewBufferString(`{"password":"wrong"}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/by-code/abcdef/join", bytes.NewBufferString(`{`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRoomController_Get(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "u-1")

	t.Run("success", func(t *testing.T) {
		svc.On("GetRoom", mock.Anything, "r-1").Return(sampleRoom(), nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/rooms/r-1", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc.On("GetRoom", mock.Anything, "missing").
			Return(nil, svcerrors.New(svcerrors.ErrNotFound, "no")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/rooms/missing", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRoomController_Update(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "owner-1")

	t.Run("full patch", func(t *testing.T) {
		svc.On("UpdateSettings", mock.Anything, "r-1", "owner-1", mock.MatchedBy(func(p svcports.RoomSettings) bool {
			return p.Name == "New" && p.MaxPlayers == 10 && p.Password != nil && *p.Password == "newpw"
		})).Return(sampleRoom(), nil).Once()
		body := `{"name":"New","max_players":10,"password":"newpw"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/api/v1/rooms/r-1", bytes.NewBufferString(body))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("clear password", func(t *testing.T) {
		svc.On("UpdateSettings", mock.Anything, "r-1", "owner-1", mock.MatchedBy(func(p svcports.RoomSettings) bool {
			return p.Password != nil && *p.Password == ""
		})).Return(sampleRoom(), nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/api/v1/rooms/r-1", bytes.NewBufferString(`{"password":""}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/api/v1/rooms/r-1", bytes.NewBufferString(`{bad`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRoomController_Start(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "owner-1")

	t.Run("success with player_count", func(t *testing.T) {
		svc.On("StartGame", mock.Anything, "r-1", "owner-1", 4).Return(nil).Once()
		svc.On("GetRoom", mock.Anything, "r-1").Return(sampleRoom(), nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/start",
			bytes.NewBufferString(`{"player_count":4}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("defaults player_count to min", func(t *testing.T) {
		svc.On("StartGame", mock.Anything, "r-1", "owner-1", domain.RoomMinPlayers).Return(nil).Once()
		svc.On("GetRoom", mock.Anything, "r-1").Return(sampleRoom(), nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/start", bytes.NewBufferString(`{}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc.On("StartGame", mock.Anything, "r-1", "owner-1", domain.RoomMinPlayers).
			Return(svcerrors.New(svcerrors.ErrConflict, "already started")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/start", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestRoomController_Leave(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "u-2")
	svc.On("LeaveRoom", mock.Anything, "r-1", "u-2", "").Return(sampleRoom(), nil).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/leave", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoomController_Close(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "owner-1")
	t.Run("success", func(t *testing.T) {
		svc.On("CloseRoom", mock.Anything, "r-1", "owner-1").Return(nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/close", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "room closed")
	})
	t.Run("forbidden", func(t *testing.T) {
		svc.On("CloseRoom", mock.Anything, "r-1", "owner-1").
			Return(svcerrors.New(svcerrors.ErrForbidden, "not owner")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/close", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRoomController_ListCategories(t *testing.T) {
	router, _, admin := setupRoomRouter(t, "")

	t.Run("default lang", func(t *testing.T) {
		admin.On("ListCategories", mock.Anything, "en").Return([]domain.Category{
			{ID: "c1", Name: "Animals", Language: "en"},
		}, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/categories", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var out map[string]any
		decodeJSON(t, w, &out)
		arr := out["categories"].([]any)
		assert.Len(t, arr, 1)
	})

	t.Run("explicit fa", func(t *testing.T) {
		admin.On("ListCategories", mock.Anything, "fa").Return([]domain.Category{}, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/categories?language=fa", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("unknown lang falls back en", func(t *testing.T) {
		admin.On("ListCategories", mock.Anything, "en").Return([]domain.Category{}, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/categories?language=xx", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		admin.On("ListCategories", mock.Anything, "en").Return([]domain.Category(nil), errors.New("db down")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/categories", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRoomController_ToDomainCategories(t *testing.T) {
	in := []createCategoryInput{
		{Name: "  Animals  ", Words: map[int][]string{
			1: {"cat", " dog ", ""},     // "" stripped
			2: {"tiger"},                // ok
			9: {"bad"},                  // passes through (validation filters later)
		}},
		{Name: "", Words: map[int][]string{1: {"x"}}},         // dropped (empty name)
		{Name: "Empty", Words: map[int][]string{1: {"", " "}}}, // dropped (no words after trim)
	}
	out := toDomainCategories(in)
	require.Len(t, out, 1)
	assert.Equal(t, "Animals", out[0].Name)
	assert.Equal(t, []string{"cat", "dog"}, out[0].Words[1])
	assert.Equal(t, []string{"tiger"}, out[0].Words[2])
}

func TestRoomController_RoomWithMeta_NilSafe(t *testing.T) {
	// This covers the nil branch in roomWithMeta.
	gin.SetMode(gin.TestMode)
	ctrl := &RoomController{}
	assert.Nil(t, ctrl.roomWithMeta(nil))
}

func TestRoomController_JoinGuestPath(t *testing.T) {
	// Guest = no userID in context; nickname in body triggers IssueGuestToken.
	router, svc, _ := setupRoomRouter(t, "")
	room := sampleRoom()

	svc.On("JoinRoom", mock.Anything, "ABCDEF", "pw").Return(room, nil).Once()
	svc.On("IssueGuestToken", mock.Anything, room.ID, "Alice").
		Return(&domain.GuestAuth{
			Token: "guest-tok", GuestID: "guest:g1", RoomID: room.ID, Nickname: "Alice",
			ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
		}, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rooms/by-code/abcdef/join",
		bytes.NewBufferString(`{"password":"pw","nickname":"Alice"}`))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["is_guest"])
	assert.Equal(t, "guest-tok", body["guest_token"])
	assert.Equal(t, "guest:g1", body["guest_id"])
	assert.Equal(t, "Alice", body["nickname"])
	svc.AssertExpectations(t)
}

func TestRoomController_JoinGuestPath_IssueTokenError(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "")
	room := sampleRoom()
	svc.On("JoinRoom", mock.Anything, "ABCDEF", "").Return(room, nil).Once()
	svc.On("IssueGuestToken", mock.Anything, room.ID, "  ").
		Return(nil, svcerrors.New(svcerrors.ErrBadRequest, "nickname too short")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rooms/by-code/abcdef/join",
		bytes.NewBufferString(`{"nickname":"  "}`))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoomController_GetByCode_FullMeta(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "")
	room := sampleRoom()
	room.HasPassword = true
	room.CustomCategories = []domain.CustomCategory{
		{Name: "A", Words: map[int][]string{1: {"x"}, 2: {"y"}, 3: {"z"}}},
	}
	svc.On("GetRoomByInvite", mock.Anything, "ABCDEF").Return(room, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/rooms/by-code/abcdef", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ABCDEF", body["invite_code"])
	assert.Equal(t, true, body["has_password"])
	assert.Equal(t, float64(3), body["custom_word_count"])
	// password_hash must never leak to anonymous visitors.
	_, hasHash := body["password_hash"]
	assert.False(t, hasHash)
}

func TestRoomController_GetByCode_Errors(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "")
	svc.On("GetRoomByInvite", mock.Anything, "ZZZZZZ").
		Return(nil, svcerrors.New(svcerrors.ErrNotFound, "nope")).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/rooms/by-code/ZZZZZZ", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRoomController_CreateFullCoverage(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "u-1")

	t.Run("invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms", bytes.NewBufferString(`{`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("explicit public room type", func(t *testing.T) {
		room := sampleRoom()
		room.Type = domain.RoomTypePublic
		room.InviteCode = ""
		svc.On("CreateRoom", mock.Anything, "u-1", domain.RoomTypePublic, mock.Anything).
			Return(room, nil).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms",
			bytes.NewBufferString(`{"name":"Pub","room_type":"public","language":"en"}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("service error propagated", func(t *testing.T) {
		svc.On("CreateRoom", mock.Anything, "u-1", domain.RoomTypePrivate, mock.Anything).
			Return(nil, svcerrors.New(svcerrors.ErrBadRequest, "bad")).Once()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/rooms", bytes.NewBufferString(`{"name":"X"}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRoomController_Start_DefaultsPlayerCount(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "u-1")
	room := sampleRoom()

	// Body with too-low player_count should be clamped up to RoomMinPlayers,
	// and on success we return the refreshed room state.
	svc.On("StartGame", mock.Anything, "r-1", "u-1", domain.RoomMinPlayers).Return(nil).Once()
	svc.On("GetRoom", mock.Anything, "r-1").Return(room, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/start",
		bytes.NewBufferString(`{"player_count":0}`))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoomController_Start_Error(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "u-1")
	svc.On("StartGame", mock.Anything, "r-1", "u-1", domain.RoomMinPlayers).
		Return(svcerrors.New(svcerrors.ErrBadRequest, "need more")).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/start", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoomController_Update_InvalidJSON(t *testing.T) {
	router, _, _ := setupRoomRouter(t, "u-1")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/rooms/r-1", bytes.NewBufferString(`{`))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoomController_Leave_Error(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "u-1")
	svc.On("LeaveRoom", mock.Anything, "r-1", "u-1", "").
		Return(nil, svcerrors.New(svcerrors.ErrInternalServer, "boom")).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/leave", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoomController_Close_Error(t *testing.T) {
	router, svc, _ := setupRoomRouter(t, "u-1")
	svc.On("CloseRoom", mock.Anything, "r-1", "u-1").
		Return(svcerrors.New(svcerrors.ErrForbidden, "nope")).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rooms/r-1/close", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRoomController_ListCategories_UnknownLanguageFallsBackToEn(t *testing.T) {
	router, _, adminSvc := setupRoomRouter(t, "")
	adminSvc.On("ListCategories", mock.Anything, "en").
		Return([]domain.Category{{ID: "c1", Name: "Animals", Language: "en"}}, nil).Once()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/categories?language=xx", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	cats := body["categories"].([]any)
	assert.Len(t, cats, 1)
}

func TestCountCustomWords(t *testing.T) {
	assert.Equal(t, 0, countCustomWords(nil))
	assert.Equal(t, 5, countCustomWords([]domain.CustomCategory{
		{Words: map[int][]string{1: {"a", "b"}, 2: {"c", "d", "e"}}},
	}))
}
