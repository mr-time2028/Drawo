package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/services"
	"drawo/internal/delivery/http/middlewares"
	"drawo/pkg/errors"
)

// RoomController owns HTTP endpoints for ephemeral public/private rooms.
//
// Invite URLs are NOT built on the backend — the frontend knows its own origin
// and constructs `${origin}/r/${invite_code}` so we stay deployment-agnostic.
type RoomController struct {
	roomSvc  services.RoomService
	adminSvc services.AdminService // for public category listing
}

// NewRoomController wires the controller.
func NewRoomController(roomSvc services.RoomService, adminSvc services.AdminService) *RoomController {
	return &RoomController{roomSvc: roomSvc, adminSvc: adminSvc}
}

// CreateRoomRequest is the JSON body for POST /rooms.
type CreateRoomRequest struct {
	Name             string                   `json:"name"`
	Password         *string                  `json:"password"`
	Language         string                   `json:"language"`
	MinPlayers       int                      `json:"min_players"`
	MaxPlayers       int                      `json:"max_players"`
	MaxRounds        int                      `json:"max_rounds"`
	RoundTime        int                      `json:"round_time"`
	WordSource       string                   `json:"word_source"`
	RoomType         string                   `json:"room_type"` // "private" (default) or "public"
	CustomCategories []createCategoryInput    `json:"custom_categories"`
}

type createCategoryInput struct {
	Name  string         `json:"name"`
	Words map[int][]string `json:"words"`
}

type updateRoomRequest struct {
	Name             *string                 `json:"name"`
	Password         *string                 `json:"password"` // pointer to distinguish "clear" (empty) from "unchanged" (nil)
	Language         *string                 `json:"language"`
	MinPlayers       *int                    `json:"min_players"`
	MaxPlayers       *int                    `json:"max_players"`
	MaxRounds        *int                    `json:"max_rounds"`
	RoundTime        *int                    `json:"round_time"`
	WordSource       *string                 `json:"word_source"`
	CustomCategories *[]createCategoryInput  `json:"custom_categories"`
}

type joinRequest struct {
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

func (ctrl *RoomController) Create(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid request body").Response())
		return
	}
	roomType := domain.RoomTypePrivate
	if strings.EqualFold(req.RoomType, "public") {
		roomType = domain.RoomTypePublic
	}
	sets := services.RoomSettings{
		Name:       req.Name,
		Password:   req.Password,
		Language:   strings.ToLower(strings.TrimSpace(req.Language)),
		MinPlayers: req.MinPlayers,
		MaxPlayers: req.MaxPlayers,
		MaxRounds:  req.MaxRounds,
		RoundTime:  req.RoundTime,
		WordSource: domain.WordSource(strings.ToLower(strings.TrimSpace(req.WordSource))),
	}
	if len(req.CustomCategories) > 0 {
		sets.CustomCategories = toDomainCategories(req.CustomCategories)
	}
	room, err := ctrl.roomSvc.CreateRoom(c.Request.Context(), userID, roomType, sets)
	if err != nil {
		errors.RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, ctrl.roomWithMeta(room))
}

// GetByCode is public (no auth) and returns only metadata safe to expose to
// anonymous visitors — no password hash, no raw custom word lists.
func (ctrl *RoomController) GetByCode(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	room, err := ctrl.roomSvc.GetRoomByInvite(c.Request.Context(), code)
	if err != nil {
		errors.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"room_id":     room.ID,
		"name":        room.Name,
		"state":       room.State,
		"language":    room.Language,
		"word_source": room.WordSource,
		"has_password": room.HasPassword,
		"max_players": room.MaxPlayers,
		"min_players": room.MinPlayers,
		"round_time":  room.RoundTime,
		"max_rounds":  room.MaxRounds,
		"custom_category_count": len(room.CustomCategories),
		"custom_word_count":     countCustomWords(room.CustomCategories),
		"invite_code": room.InviteCode,
	})
}

// Join works for both registered users (Bearer token present) and anonymous
// guests (no Authorization header, nickname supplied). Registered users get
// back the room; anonymous users additionally get a short-lived guest_token
// they can use to authenticate the WebSocket connection.
func (ctrl *RoomController) Join(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	var body joinRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(errors.New(errors.ErrBadRequest, "invalid request body").Response())
			return
		}
	}
	room, err := ctrl.roomSvc.JoinRoom(c.Request.Context(), code, body.Password)
	if err != nil {
		errors.RespondError(c, err)
		return
	}
	out := ctrl.roomWithMeta(room)
	userID, hasUser := c.Get(middlewares.ContextUserID)
	if !hasUser || userID == nil || userID.(string) == "" {
		// Guest path — issue a room-bound guest token.
		guest, err := ctrl.roomSvc.IssueGuestToken(c.Request.Context(), room.ID, body.Nickname)
		if err != nil {
			errors.RespondError(c, err)
			return
		}
		out["guest_token"] = guest.Token
		out["guest_id"] = guest.GuestID
		out["nickname"] = guest.Nickname
		out["is_guest"] = true
	} else {
		out["is_guest"] = false
	}
	c.JSON(http.StatusOK, out)
}

// Get returns the room for members. We don't enforce membership here since
// the hub does; but we do return the full settings (including custom words)
// so the lobby UI can render them. The password hash is stripped by JSON tag.
func (ctrl *RoomController) Get(c *gin.Context) {
	id := c.Param("id")
	room, err := ctrl.roomSvc.GetRoom(c.Request.Context(), id)
	if err != nil {
		errors.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ctrl.roomWithMeta(room))
}

// Update modifies settings. Owner + lobby only (enforced in service).
func (ctrl *RoomController) Update(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)
	id := c.Param("id")
	var body updateRoomRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(errors.New(errors.ErrBadRequest, "invalid request body").Response())
		return
	}
	patch := services.RoomSettings{}
	if body.Name != nil {
		patch.Name = *body.Name
	}
	if body.Language != nil {
		patch.Language = *body.Language
	}
	if body.MinPlayers != nil {
		patch.MinPlayers = *body.MinPlayers
	}
	if body.MaxPlayers != nil {
		patch.MaxPlayers = *body.MaxPlayers
	}
	if body.MaxRounds != nil {
		patch.MaxRounds = *body.MaxRounds
	}
	if body.RoundTime != nil {
		patch.RoundTime = *body.RoundTime
	}
	if body.WordSource != nil {
		patch.WordSource = domain.WordSource(*body.WordSource)
	}
	// Password uses pointer semantics: nil = unchanged, "" = clear, "abc" = set.
	patch.Password = body.Password
	if body.CustomCategories != nil {
		patch.CustomCategories = toDomainCategories(*body.CustomCategories)
	}
	room, err := ctrl.roomSvc.UpdateSettings(c.Request.Context(), id, userID, patch)
	if err != nil {
		errors.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ctrl.roomWithMeta(room))
}

func (ctrl *RoomController) Start(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)
	id := c.Param("id")
	// The service validates player count; we default to min (it will enforce).
	// Once the hub tracks player counts per room we'll pass the real count —
	// for now clients send `player_count` explicitly (best effort).
	var body struct {
		PlayerCount int `json:"player_count"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.PlayerCount < domain.RoomMinPlayers {
		body.PlayerCount = domain.RoomMinPlayers
	}
	if err := ctrl.roomSvc.StartGame(c.Request.Context(), id, userID, body.PlayerCount); err != nil {
		errors.RespondError(c, err)
		return
	}
	room, _ := ctrl.roomSvc.GetRoom(c.Request.Context(), id)
	c.JSON(http.StatusOK, ctrl.roomWithMeta(room))
}

func (ctrl *RoomController) Leave(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)
	id := c.Param("id")
	room, err := ctrl.roomSvc.LeaveRoom(c.Request.Context(), id, userID, "")
	if err != nil {
		errors.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ctrl.roomWithMeta(room))
}

func (ctrl *RoomController) Close(c *gin.Context) {
	userID := c.GetString(middlewares.ContextUserID)
	id := c.Param("id")
	if err := ctrl.roomSvc.CloseRoom(c.Request.Context(), id, userID); err != nil {
		errors.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "room closed"})
}

// ListCategories returns all dictionary categories in the requested language
// so the UI can show helpers (not strictly required for custom words but we
// keep it for future "copy from Drawo" functionality and power users).
func (ctrl *RoomController) ListCategories(c *gin.Context) {
	lang := strings.ToLower(strings.TrimSpace(c.Query("language")))
	if lang != "en" && lang != "fa" {
		lang = "en"
	}
	cats, err := ctrl.adminSvc.ListCategories(c.Request.Context(), lang)
	if err != nil {
		errors.RespondError(c, err)
		return
	}
	out := make([]gin.H, 0, len(cats))
	for _, cat := range cats {
		out = append(out, gin.H{"id": cat.ID, "name": cat.Name, "language": cat.Language})
	}
	c.JSON(http.StatusOK, gin.H{"categories": out})
}

// ---------- helpers ----------

func toDomainCategories(in []createCategoryInput) []domain.CustomCategory {
	out := make([]domain.CustomCategory, 0, len(in))
	for _, c := range in {
		cc := domain.CustomCategory{
			Name:  strings.TrimSpace(c.Name),
			Words: map[int][]string{},
		}
		for tier, words := range c.Words {
			clean := make([]string, 0, len(words))
			for _, w := range words {
				if trimmed := strings.TrimSpace(w); trimmed != "" {
					clean = append(clean, trimmed)
				}
			}
			if len(clean) > 0 {
				cc.Words[tier] = clean
			}
		}
		if cc.Name != "" && len(cc.Words) > 0 {
			out = append(out, cc)
		}
	}
	return out
}

func countCustomWords(cats []domain.CustomCategory) int {
	n := 0
	for _, c := range cats {
		for _, ws := range c.Words {
			n += len(ws)
		}
	}
	return n
}

func (ctrl *RoomController) roomWithMeta(r *domain.Room) gin.H {
	if r == nil {
		return nil
	}
	body := gin.H{
		"id":                r.ID,
		"name":              r.Name,
		"invite_code":       r.InviteCode,
		"owner_id":          r.OwnerID,
		"type":              r.Type,
		"has_password":      r.HasPassword,
		"language":          r.Language,
		"word_source":       r.WordSource,
		"state":             r.State,
		"min_players":       r.MinPlayers,
		"max_players":       r.MaxPlayers,
		"round_time":        r.RoundTime,
		"max_rounds":        r.MaxRounds,
		"current_round":     r.CurrentRound,
		"created_at":        r.CreatedAt,
		"updated_at":        r.UpdatedAt,
	}
	if r.WordSource == domain.WordSourceCustom {
		body["custom_categories"] = r.CustomCategories
		body["custom_word_count"] = countCustomWords(r.CustomCategories)
	}
	return body
}
