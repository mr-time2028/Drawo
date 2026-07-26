// Package controllers handles profile management and contact verification.
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/services"
	"drawo/internal/delivery/http/middlewares"
	"drawo/pkg/errors"
	"drawo/pkg/validator"
)

// UserController implements profile and verification handlers.
type UserController struct {
	userSvc services.UserService
}

// NewUserController creates a new controller.
func NewUserController(userSvc services.UserService) *UserController {
	return &UserController{userSvc: userSvc}
}

// GetProfile returns the authenticated user's profile.
func (ctrl *UserController) GetProfile(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)

	profile, err := ctrl.userSvc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfileRequest defines editable fields.
type UpdateProfileRequest struct {
	AvatarURL       string `json:"avatar_url"`
	Locale          string `json:"locale" validate:"omitempty,len=2"`
	Theme           string `json:"theme" validate:"omitempty,oneof=light dark"`
	BackgroundSound bool   `json:"background_sound"`
	ToolSound       bool   `json:"tool_sound"`
}

// UpdateProfile handles partial updates to the user's settings.
func (ctrl *UserController) UpdateProfile(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid request body").Response())
		return
	}

	if errs := validator.Struct(req); errs != nil {
		c.JSON(errors.ValidationError(errs))
		return
	}

	updates := domain.Profile{
		AvatarURL:       req.AvatarURL,
		Locale:          req.Locale,
		Theme:           req.Theme,
		BackgroundSound: req.BackgroundSound,
		ToolSound:       req.ToolSound,
	}

	updated, err := ctrl.userSvc.UpdateProfile(c.Request.Context(), userID, updates)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, updated)
}

// ChangeUsernameRequest defines the body for name changes.
type ChangeUsernameRequest struct {
	Username string `json:"username" validate:"required,min=3,max=20"`
}

// ChangeUsername handles account renaming with uniqueness check.
func (ctrl *UserController) ChangeUsername(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)

	var req ChangeUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid request body").Response())
		return
	}

	if errs := validator.Struct(req); errs != nil {
		c.JSON(errors.ValidationError(errs))
		return
	}

	if err := ctrl.userSvc.ChangeUsername(c.Request.Context(), userID, req.Username); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "username updated successfully"})
}

// VerifyRequest body.
type VerifyRequest struct {
	Type domain.OTPType `json:"type" validate:"required,oneof=email phone"`
}

// RequestVerification sends a new OTP code.
func (ctrl *UserController) RequestVerification(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)

	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid request body").Response())
		return
	}

	if err := ctrl.userSvc.RequestVerification(c.Request.Context(), userID, req.Type); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification code sent"})
}

// ConfirmVerifyRequest body.
type ConfirmVerifyRequest struct {
	Type domain.OTPType `json:"type" validate:"required,oneof=email phone"`
	Code string         `json:"code" validate:"required,len=6"`
}

// ConfirmVerification validates the code and marks the account as verified.
func (ctrl *UserController) ConfirmVerification(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)

	var req ConfirmVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid request body").Response())
		return
	}

	if err := ctrl.userSvc.ConfirmVerification(c.Request.Context(), userID, req.Code, req.Type); err != nil {
		appErr, _ := err.(*errors.AppError)
		c.JSON(appErr.Response())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification successful"})
}
