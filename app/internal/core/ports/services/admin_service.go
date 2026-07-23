// Package services implements the business logic.
package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"drawo/config"
	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/errors"
)

// AdminService defines the administrative operations for Drawo.
type AdminService interface {
	// Songs
	UploadSong(ctx context.Context, title string, songType domain.SongType, reader io.Reader, size int64) (*domain.Song, error)
	ListSongs(ctx context.Context, songType domain.SongType) ([]domain.Song, error)
	ToggleSong(ctx context.Context, songID string, active bool) error
	DeleteSong(ctx context.Context, songID string) error

	// Moderation
	SearchUsers(ctx context.Context, query string) ([]domain.UserWithProfile, error)
	BanUser(ctx context.Context, userID string) error
	UnbanUser(ctx context.Context, userID string) error

	// Bad words
	CreateBadWord(ctx context.Context, text, language string) (*domain.BadWord, error)
	ListBadWords(ctx context.Context, language string) ([]domain.BadWord, error)
	DeleteBadWord(ctx context.Context, id string) error

	// Settings
	UpdateGlobalSetting(ctx context.Context, key, value string) error
}

type adminService struct {
	cfg         config.Config
	adminRepo   repositories.AdminRepository
	contentRepo repositories.ContentRepository
	userRepo    repositories.UserRepository
	profileRepo repositories.ProfileRepository
	sessionRepo repositories.SessionRepository
	storage     repositories.FileStorage
}

// NewAdminService creates the service responsible for site configuration and moderation.
func NewAdminService(
	cfg config.Config,
	adminRepo repositories.AdminRepository,
	userRepo repositories.UserRepository,
	profileRepo repositories.ProfileRepository,
	sessionRepo repositories.SessionRepository,
	storage repositories.FileStorage,
	contentRepos ...repositories.ContentRepository,
) AdminService {
	var contentRepo repositories.ContentRepository
	if len(contentRepos) > 0 {
		contentRepo = contentRepos[0]
	}
	return &adminService{
		cfg:         cfg,
		adminRepo:   adminRepo,
		contentRepo: contentRepo,
		userRepo:    userRepo,
		profileRepo: profileRepo,
		sessionRepo: sessionRepo,
		storage:     storage,
	}
}

// UploadSong saves an MP3 file to MinIO and records its metadata in Postgres.
func (s *adminService) UploadSong(ctx context.Context, title string, songType domain.SongType, reader io.Reader, size int64) (*domain.Song, error) {
	// 1. Generate a unique key for MinIO
	objectName := fmt.Sprintf("music/%s/%s.mp3", songType, uuid.New().String())

	// 2. Upload to the storage provider (MinIO)
	key, err := s.storage.Upload(ctx, s.cfg.App.Storage.BucketName, objectName, reader, size, "audio/mpeg")
	if err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to upload file to storage")
	}

	// 3. Save metadata to DB
	song := &domain.Song{
		ID:        uuid.New().String(),
		Title:     title,
		FileKey:   key,
		Type:      songType,
		IsActive:  true, // Default to active
		CreatedAt: time.Now(),
	}

	if err := s.adminRepo.SaveSong(ctx, song); err != nil {
		return nil, errors.New(errors.ErrInternalServer, "failed to save song metadata")
	}

	return song, nil
}

func (s *adminService) ListSongs(ctx context.Context, songType domain.SongType) ([]domain.Song, error) {
	return s.adminRepo.ListSongs(ctx, songType)
}

func (s *adminService) ToggleSong(ctx context.Context, songID string, active bool) error {
	song, err := s.adminRepo.GetSongByID(ctx, songID)
	if err != nil {
		return errors.New(errors.ErrNotFound, "song not found")
	}
	song.IsActive = active
	return s.adminRepo.SaveSong(ctx, song)
}

func (s *adminService) DeleteSong(ctx context.Context, songID string) error {
	song, err := s.adminRepo.GetSongByID(ctx, songID)
	if err != nil {
		return errors.New(errors.ErrNotFound, "song not found")
	}

	// 1. Delete from MinIO
	_ = s.storage.Delete(ctx, s.cfg.App.Storage.BucketName, song.FileKey)

	// 2. Delete from DB
	return s.adminRepo.DeleteSong(ctx, songID)
}

// SearchUsers allows Admin to find users by partial username, email, or phone.
func (s *adminService) SearchUsers(ctx context.Context, query string) ([]domain.UserWithProfile, error) {
	return s.userRepo.SearchUsers(query)
}

// BanUser disables the account AND instantly kicks the user from any active connection.
func (s *adminService) BanUser(ctx context.Context, userID string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return errors.New(errors.ErrNotFound, "user not found")
	}

	// 1. Disable account and update ban tracking metadata.
	user.IsActive = false
	user.BanCount++
	now := time.Now()
	user.BannedAt = &now
	user.UpdatedAt = now

	if err := s.userRepo.Update(user); err != nil {
		return errors.New(errors.ErrInternalServer, "failed to update user status")
	}

	// 2. THE NUCLEAR OPTION: Single Device Policy Integration
	// By deleting the active session from Redis, the user is kicked instantly from the game.
	return s.sessionRepo.DeleteAllForUser(ctx, userID)
}

func (s *adminService) UnbanUser(ctx context.Context, userID string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return errors.New(errors.ErrNotFound, "user not found")
	}
	user.IsActive = true
	return s.userRepo.Update(user)
}

func (s *adminService) CreateBadWord(ctx context.Context, text, language string) (*domain.BadWord, error) {
	if s.contentRepo == nil {
		return nil, errors.New(errors.ErrInternalServer, "content repository is not configured")
	}
	text = strings.TrimSpace(text)
	language = strings.ToLower(strings.TrimSpace(language))
	if text == "" {
		return nil, errors.New(errors.ErrBadRequest, "bad word text is required")
	}
	if language != "en" && language != "fa" {
		return nil, errors.New(errors.ErrBadRequest, "language must be en or fa")
	}
	badWord := &domain.BadWord{ID: uuid.New().String(), Text: text, Language: language, CreatedAt: time.Now()}
	if err := s.contentRepo.InsertBadWord(ctx, badWord); err != nil {
		return nil, errors.New(errors.ErrConflict, "bad word already exists or could not be saved")
	}
	return badWord, nil
}

func (s *adminService) ListBadWords(ctx context.Context, language string) ([]domain.BadWord, error) {
	if s.contentRepo == nil {
		return nil, errors.New(errors.ErrInternalServer, "content repository is not configured")
	}
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" {
		language = "fa"
	}
	if language != "en" && language != "fa" {
		return nil, errors.New(errors.ErrBadRequest, "language must be en or fa")
	}
	return s.contentRepo.ListBadWords(ctx, language)
}

func (s *adminService) DeleteBadWord(ctx context.Context, id string) error {
	if s.contentRepo == nil {
		return errors.New(errors.ErrInternalServer, "content repository is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New(errors.ErrBadRequest, "bad word id is required")
	}
	return s.contentRepo.DeleteBadWord(ctx, id)
}

func (s *adminService) UpdateGlobalSetting(ctx context.Context, key, value string) error {
	return s.adminRepo.UpdateSetting(ctx, key, value)
}
