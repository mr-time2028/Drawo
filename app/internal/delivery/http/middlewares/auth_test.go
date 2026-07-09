package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/internal/infrastructure/cache"
	"drawo/internal/infrastructure/di"
	"drawo/pkg/security"
)

type mockUserSvc struct {
	mock.Mock
}

func (m *mockUserSvc) GetProfile(ctx context.Context, id string) (*domain.UserWithProfile, error) {
	args := m.Called(ctx, id)
    if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.UserWithProfile), args.Error(1)
}
func (m *mockUserSvc) UpdateProfile(ctx context.Context, id string, p domain.Profile) (*domain.Profile, error) {
	return nil, nil
}
func (m *mockUserSvc) ChangeUsername(ctx context.Context, id, name string) error {
	return nil
}
func (m *mockUserSvc) RequestVerification(ctx context.Context, id string, t domain.OTPType) error {
	return nil
}
func (m *mockUserSvc) ConfirmVerification(ctx context.Context, id, code string, t domain.OTPType) error {
	return nil
}

func TestAuthMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)
    config.Load()
	cfg := config.Get()
	cfg.App.SecretKey = "secret"
	
	cacheClient, _ := cache.NewClient(config.CacheConfig{Driver: "memory"})
    sessionRepo := repositories.NewSessionRepo(cacheClient)
    userSvc := new(mockUserSvc)
    
	container := &di.Container{
        Config: cfg,
        Sessions: sessionRepo,
        Services: di.Services{
            User: userSvc,
        },
    }
    
	jwt := security.NewJWTManager("secret", "drawo", time.Hour, time.Hour)

	router := gin.New()
	router.Use(RequireAuth(container))
	router.GET("/auth", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("NoToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("ValidTokenNoSession", func(t *testing.T) {
		acc, _, _ := jwt.GenerateTokenPair("u1", "s1", "t1")
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth", nil)
		req.Header.Set("Authorization", "Bearer "+acc)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

    t.Run("Success", func(t *testing.T) {
        sid := "s1"
        uid := "u1"
        acc, _, _ := jwt.GenerateTokenPair(uid, sid, "t1")
        
        sessionRepo.Set(context.Background(), &domain.Session{ID: sid, UserID: uid, ExpiresAt: time.Now().Add(time.Hour)})
        userSvc.On("GetProfile", mock.Anything, uid).Return(&domain.UserWithProfile{User: domain.User{ID: uid}}, nil).Once()

        w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth", nil)
		req.Header.Set("Authorization", "Bearer "+acc)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
    })

    t.Run("UserNotFound", func(t *testing.T) {
        sid := "s2"
        uid := "u2"
        acc, _, _ := jwt.GenerateTokenPair(uid, sid, "t1")
        
        sessionRepo.Set(context.Background(), &domain.Session{ID: sid, UserID: uid, ExpiresAt: time.Now().Add(time.Hour)})
        userSvc.On("GetProfile", mock.Anything, uid).Return(nil, assert.AnError).Once()

        w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth", nil)
		req.Header.Set("Authorization", "Bearer "+acc)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
    })

    t.Run("RequireAdmin_Fail", func(t *testing.T) {
        w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
        c.Set(ContextIsSuperuser, false)
        RequireAdmin()(c)
        assert.Equal(t, http.StatusForbidden, w.Code)
    })

    t.Run("RequireAdmin_Success", func(t *testing.T) {
        w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
        c.Set(ContextIsSuperuser, true)
        RequireAdmin()(c)
        assert.Equal(t, http.StatusOK, w.Code)
    })
}
