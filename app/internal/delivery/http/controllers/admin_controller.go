// Package controllers handles administrative operations.
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/services"
	"drawo/pkg/errors"
)

// AdminController implements handlers for site configuration and moderation.
type AdminController struct {
	adminSvc services.AdminService
}

// NewAdminController creates a new admin controller.
func NewAdminController(adminSvc services.AdminService) *AdminController {
	return &AdminController{adminSvc: adminSvc}
}

// UploadSongRequest defines the data for song uploading.
// Since Gin handles multipart/form-data differently, we extract fields manually in the handler.
type UploadSongRequest struct {
	Title string          `form:"title" validate:"required"`
	Type  domain.SongType `form:"type" validate:"required,oneof=landing game"`
}

// UploadSong handles the uploading of MP3 files to MinIO.
func (ctrl *AdminController) UploadSong(c *gin.Context) {
	// 1. Parse Form Fields
	var req UploadSongRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid query parameters").Response())
		return
	}

	// 2. Get the file from request
	file, err := c.FormFile("song")
	if err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "song file is required").Response())
		return
	}

	// 3. Open the file to pass a reader to the service
	src, err := file.Open()
	if err != nil {
		c.JSON(errors.New(errors.ErrInternalServer, "failed to open uploaded file").Response())
		return
	}
	defer src.Close()

	// 4. Call service to store in MinIO and DB
	song, err := ctrl.adminSvc.UploadSong(c.Request.Context(), req.Title, req.Type, src, file.Size)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusCreated, song)
}

// ListSongs returns all songs of a specific type.
func (ctrl *AdminController) ListSongs(c *gin.Context) {
	songType := domain.SongType(c.Query("type"))
	if songType == "" {
		songType = domain.SongTypeLanding
	}

	songs, err := ctrl.adminSvc.ListSongs(c.Request.Context(), songType)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, songs)
}

// ToggleSongRequest defines the body to enable/disable a song.
type ToggleSongRequest struct {
	Active bool `json:"active"`
}

// ToggleSong enables or disables a song in the playlist.
func (ctrl *AdminController) ToggleSong(c *gin.Context) {
	id := c.Param("id")
	var req ToggleSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid body").Response())
		return
	}

	if err := ctrl.adminSvc.ToggleSong(c.Request.Context(), id, req.Active); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "song status updated"})
}

// DeleteSong removes the song from storage and DB.
func (ctrl *AdminController) DeleteSong(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.adminSvc.DeleteSong(c.Request.Context(), id); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "song deleted successfully"})
}

// SearchUsers searches for users by username, email, or phone.
func (ctrl *AdminController) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(errors.New(errors.ErrBadRequest, "search query 'q' is required").Response())
		return
	}

	results, err := ctrl.adminSvc.SearchUsers(c.Request.Context(), query)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, results)
}

// BanUser deactivates an account and kicks the user.
func (ctrl *AdminController) BanUser(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.adminSvc.BanUser(c.Request.Context(), id); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user banned and logged out"})
}

// UnbanUser reactivates an account.
func (ctrl *AdminController) UnbanUser(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.adminSvc.UnbanUser(c.Request.Context(), id); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user unbanned successfully"})
}

// UpdateSettingRequest for global configs.
type UpdateSettingRequest struct {
	Value string `json:"value" validate:"required"`
}

// UpdateSetting updates a global game configuration.
func (ctrl *AdminController) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid body").Response())
		return
	}

	if err := ctrl.adminSvc.UpdateGlobalSetting(c.Request.Context(), key, req.Value); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "setting updated"})
}

// CreateBadWordRequest defines an admin-managed prohibited word.
type CreateBadWordRequest struct {
	Text     string `json:"text" validate:"required"`
	Language string `json:"language" validate:"required,oneof=en fa"`
}

// CreateBadWord adds a prohibited word to the moderation dictionary.
func (ctrl *AdminController) CreateBadWord(c *gin.Context) {
	var req CreateBadWordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid body").Response())
		return
	}
	badWord, err := ctrl.adminSvc.CreateBadWord(c.Request.Context(), req.Text, req.Language)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusCreated, badWord)
}

// ListBadWords returns the prohibited word list for a language.
func (ctrl *AdminController) ListBadWords(c *gin.Context) {
	badWords, err := ctrl.adminSvc.ListBadWords(c.Request.Context(), c.Query("language"))
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, badWords)
}

// DeleteBadWord removes a prohibited word from the moderation dictionary.
func (ctrl *AdminController) DeleteBadWord(c *gin.Context) {
	if err := ctrl.adminSvc.DeleteBadWord(c.Request.Context(), c.Param("id")); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bad word deleted"})
}

// ReviewReportRequest defines the admin note for a report review decision.
type ReviewReportRequest struct {
	Note string `json:"note"`
}

// ListReports returns moderation reports, optionally filtered by status.
func (ctrl *AdminController) ListReports(c *gin.Context) {
	paging := domain.Paging{Limit: 50, Offset: 0}
	reports, err := ctrl.adminSvc.ListReports(c.Request.Context(), domain.ReportStatus(c.Query("status")), paging)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, reports)
}

// GetReport returns a single moderation report with its stored evidence.
func (ctrl *AdminController) GetReport(c *gin.Context) {
	report, err := ctrl.adminSvc.GetReport(c.Request.Context(), c.Param("id"))
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, report)
}

// ConfirmReport marks a report as valid and applies a reputation penalty.
func (ctrl *AdminController) ConfirmReport(c *gin.Context) {
	var req ReviewReportRequest
	_ = c.ShouldBindJSON(&req)
	adminID := c.GetString("userID")
	if err := ctrl.adminSvc.ConfirmReport(c.Request.Context(), c.Param("id"), adminID, req.Note); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "report confirmed"})
}

// RejectReport marks a report as invalid without penalizing the reported user.
func (ctrl *AdminController) RejectReport(c *gin.Context) {
	var req ReviewReportRequest
	_ = c.ShouldBindJSON(&req)
	adminID := c.GetString("userID")
	if err := ctrl.adminSvc.RejectReport(c.Request.Context(), c.Param("id"), adminID, req.Note); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "report rejected"})
}
