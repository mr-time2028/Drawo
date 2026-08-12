package services

import (
	"context"
	"crypto/rand"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	svcerrors "drawo/pkg/errors"
	"drawo/pkg/security"
)

// How long a guest token lives. Guests only exist for one room session — 24h
// is plenty for a game night and keeps Redis clean.
const guestTokenTTL = 24 * time.Hour

const (
	guestTokenLen    = 32
	guestTokenAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"
	minGuestNickname = 2
	maxGuestNickname = 20
)

func generateGuestToken() string {
	b := make([]byte, guestTokenLen)
	if _, err := rand.Read(b); err != nil {
		return uuid.NewString()
	}
	out := make([]byte, guestTokenLen)
	for i, v := range b {
		out[i] = guestTokenAlphabet[int(v)%len(guestTokenAlphabet)]
	}
	return string(out)
}

func sanitizeNickname(raw string) string {
	s := strings.TrimSpace(raw)
	// Collapse internal whitespace and strip control characters.
	var b strings.Builder
	var prevSpace bool
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// RoomSettings carries owner-configurable gameplay options. Zero/empty
// values are interpreted as "use Drawo default" except where explicitly set.
// Password uses a pointer: nil = leave unchanged, pointer to "" = clear,
// pointer to non-empty string = set new password.
type RoomSettings struct {
	Name             string
	Password         *string
	Language         string
	MinPlayers       int
	MaxPlayers       int
	MaxRounds        int
	RoundTime        int
	WordSource       domain.WordSource
	CustomCategories []domain.CustomCategory
}

// RoomService creates/joins ephemeral drawing rooms (private+public) and
// validates owner-supplied gameplay settings. Room state itself is stored
// ephemerally through RoomRepository (Redis) and mutated at runtime by the
// realtime hub; the service is the authoritative gate for settings.
type RoomService interface {
	CreateRoom(ctx context.Context, ownerID string, roomType domain.RoomType, s RoomSettings) (*domain.Room, error)
	GetRoom(ctx context.Context, roomID string) (*domain.Room, error)
	GetRoomByInvite(ctx context.Context, inviteCode string) (*domain.Room, error)
	JoinRoom(ctx context.Context, inviteCode, password string) (*domain.Room, error)
	IssueGuestToken(ctx context.Context, roomID, nickname string) (*domain.GuestAuth, error)
	ValidateGuestToken(ctx context.Context, token string) (*domain.GuestAuth, error)
	UpdateSettings(ctx context.Context, roomID, ownerID string, patch RoomSettings) (*domain.Room, error)
	CloseRoom(ctx context.Context, roomID, ownerID string) error
	StartGame(ctx context.Context, roomID, ownerID string, playerCount int) error
	LeaveRoom(ctx context.Context, roomID, userID string, newOwnerID string) (*domain.Room, error)
	ValidateCustomCategories(cats []domain.CustomCategory) error
}

type roomService struct {
	roomRepo repositories.RoomRepository
}

func NewRoomService(roomRepo repositories.RoomRepository) RoomService {
	return &roomService{roomRepo: roomRepo}
}
const inviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateInviteCode() string {
	b := make([]byte, domain.RoomInviteCodeLen)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand Read failing is catastrophic — fall back to a UUID snippet.
		return strings.ToUpper(strings.ReplaceAll(uuid.New().String()[:domain.RoomInviteCodeLen], "-", "X"))
	}
	for i := range b {
		b[i] = inviteAlphabet[int(b[i])%len(inviteAlphabet)]
	}
	return string(b)
}

func defaultSettings(s RoomSettings, ownerLocale string) RoomSettings {
	if strings.TrimSpace(s.Name) == "" {
		s.Name = "Drawo Room"
	}
	if s.Language == "" {
		s.Language = ownerLocale
	}
	if s.Language != "en" && s.Language != "fa" {
		s.Language = "en"
	}
	// Clamp to allowed ranges.
	if s.MinPlayers <= 0 {
		s.MinPlayers = domain.RoomMinPlayers
	}
	if s.MinPlayers < domain.RoomMinPlayers {
		s.MinPlayers = domain.RoomMinPlayers
	}
	if s.MinPlayers > domain.RoomMaxPlayers {
		s.MinPlayers = domain.RoomMaxPlayers
	}
	if s.MaxPlayers <= 0 {
		s.MaxPlayers = domain.RoomDefaultPlayers
	}
	if s.MaxPlayers < s.MinPlayers {
		s.MaxPlayers = s.MinPlayers
	}
	if s.MaxPlayers > domain.RoomMaxPlayers {
		s.MaxPlayers = domain.RoomMaxPlayers
	}
	if s.MaxRounds <= 0 {
		s.MaxRounds = domain.RoomDefaultRounds
	}
	if s.MaxRounds < domain.RoomMinRounds {
		s.MaxRounds = domain.RoomMinRounds
	}
	if s.MaxRounds > domain.RoomMaxRounds {
		s.MaxRounds = domain.RoomMaxRounds
	}
	if s.RoundTime <= 0 {
		s.RoundTime = domain.RoomDefaultRoundTime
	}
	if s.RoundTime < domain.RoomMinRoundTime {
		s.RoundTime = domain.RoomMinRoundTime
	}
	if s.RoundTime > domain.RoomMaxRoundTime {
		s.RoundTime = domain.RoomMaxRoundTime
	}
	// Snap round time to nearest step.
	step := domain.RoomRoundTimeStep
	if rem := s.RoundTime % step; rem != 0 {
		s.RoundTime -= rem
	}
	if s.WordSource == "" {
		s.WordSource = domain.WordSourceDefault
	}
	return s
}

func normalizeCustomCategories(cats []domain.CustomCategory) []domain.CustomCategory {
	if len(cats) == 0 {
		return nil
	}
	out := make([]domain.CustomCategory, 0, len(cats))
	seen := map[string]bool{}
	seenWords := map[string]bool{}
	for _, c := range cats {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		nc := domain.CustomCategory{Name: name, Words: map[int][]string{}}
		for tier, words := range c.Words {
			if tier != domain.WordPointsEasy && tier != domain.WordPointsMedium && tier != domain.WordPointsHard {
				continue
			}
			var clean []string
			for _, w := range words {
				w = strings.TrimSpace(w)
				if w == "" {
					continue
				}
				// Collapse internal whitespace.
				var b strings.Builder
				prevSpace := false
				for _, r := range w {
					if unicode.IsSpace(r) {
						if !prevSpace {
							b.WriteRune(' ')
						}
						prevSpace = true
						continue
					}
					prevSpace = false
					b.WriteRune(r)
				}
				norm := strings.ToLower(strings.TrimSpace(b.String()))
				if norm == "" {
					continue
				}
				if len([]rune(norm)) < 2 || len([]rune(norm)) > 30 {
					continue
				}
				if seenWords[norm] {
					continue
				}
				seenWords[norm] = true
				clean = append(clean, norm)
			}
			if len(clean) > 0 {
				nc.Words[tier] = clean
			}
		}
		if len(nc.Words) > 0 {
			out = append(out, nc)
		}
	}
	return out
}

// totalCustomWords counts all words across categories and tiers.
func totalCustomWords(cats []domain.CustomCategory) int {
	n := 0
	for _, c := range cats {
		for _, w := range c.Words {
			n += len(w)
		}
	}
	return n
}

// ValidateCustomCategories verifies owner-supplied custom content: at least
// one named category, at least RoomMinCustomWords total words, each word
// non-empty, no duplicates within the room.
func (s *roomService) ValidateCustomCategories(cats []domain.CustomCategory) error {
	normalized := normalizeCustomCategories(cats)
	if len(normalized) == 0 {
		return svcerrors.New(svcerrors.ErrBadRequest, "add at least one category with words")
	}
	if totalCustomWords(normalized) < domain.RoomMinCustomWords {
		return svcerrors.Newf(svcerrors.ErrBadRequest, "add at least %d words to start (you have %d)", domain.RoomMinCustomWords, totalCustomWords(normalized))
	}
	if len(normalized) > 20 {
		return svcerrors.New(svcerrors.ErrBadRequest, "too many categories (max 20)")
	}
	if totalCustomWords(normalized) > domain.RoomMaxCustomWords {
		return svcerrors.Newf(svcerrors.ErrBadRequest, "too many words (max %d)", domain.RoomMaxCustomWords)
	}
	return nil
}

func (s *roomService) CreateRoom(ctx context.Context, ownerID string, roomType domain.RoomType, in RoomSettings) (*domain.Room, error) {
	if ownerID == "" {
		return nil, svcerrors.New(svcerrors.ErrUnauthorized, "login required")
	}

	// Validate name BEFORE defaults replace an empty name with "Drawo Room".
	name := strings.TrimSpace(in.Name)
	if len([]rune(name)) < domain.RoomMinNameLength {
		return nil, svcerrors.New(svcerrors.ErrBadRequest, "room name is too short")
	}
	if len([]rune(name)) > domain.RoomMaxNameLength {
		return nil, svcerrors.New(svcerrors.ErrBadRequest, "room name is too long")
	}

	// Reject invalid language early (defaultSettings silently falls back to
	// "en" which would hide user mistakes in test and production).
	if in.Language != "" {
		lang := strings.ToLower(strings.TrimSpace(in.Language))
		if lang != "en" && lang != "fa" {
			return nil, svcerrors.New(svcerrors.ErrBadRequest, "language must be en or fa")
		}
	}

	sets := defaultSettings(in, "") // Language fixed up below against actual locale if caller didn't set.
	if in.Language == "" {
		sets.Language = "en"
	}

	var hasPassword bool
	var passwordHash string
	if in.Password != nil {
		pw := strings.TrimSpace(*in.Password)
		if pw != "" {
			if len(pw) < 4 || len(pw) > 32 {
				return nil, svcerrors.New(svcerrors.ErrBadRequest, "password must be 4–32 characters")
			}
			h, err := security.HashPassword(pw)
			if err != nil {
				return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to hash password")
			}
			passwordHash = h
			hasPassword = true
		}
	}

	if sets.WordSource != domain.WordSourceDefault &&
		sets.WordSource != domain.WordSourceCustom &&
		sets.WordSource != domain.WordSourceCategory {
		return nil, svcerrors.New(svcerrors.ErrBadRequest, "invalid word source")
	}
	var customCats []domain.CustomCategory
	if sets.WordSource == domain.WordSourceCustom {
		if err := s.ValidateCustomCategories(sets.CustomCategories); err != nil {
			return nil, err
		}
		customCats = normalizeCustomCategories(sets.CustomCategories)
	}

	var code string
	if roomType == domain.RoomTypePrivate {
		// Generate a code that doesn't collide with any live room. In practice
		// collisions are astronomically unlikely for 6 chars from a 32-char
		// alphabet, but we check anyway.
		for i := 0; i < 5; i++ {
			candidate := generateInviteCode()
			existing, _ := s.roomRepo.GetByInviteCode(ctx, candidate)
			if existing == nil {
				code = candidate
				break
			}
		}
		if code == "" {
			return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to generate invite code")
		}
	}

	now := time.Now()
	room := &domain.Room{
		ID:               uuid.New().String(),
		Name:             name,
		InviteCode:       code,
		OwnerID:          ownerID,
		Type:             roomType,
		PasswordHash:     passwordHash,
		HasPassword:      hasPassword,
		Language:         sets.Language,
		WordSource:       sets.WordSource,
		State:            domain.RoomStateLobby,
		MinPlayers:       sets.MinPlayers,
		MaxPlayers:       sets.MaxPlayers,
		RoundTime:        sets.RoundTime,
		MaxRounds:        sets.MaxRounds,
		CustomCategories: customCats,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.roomRepo.Save(ctx, room); err != nil {
		return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to save room")
	}
	return room, nil
}

func (s *roomService) GetRoom(ctx context.Context, roomID string) (*domain.Room, error) {
	if roomID == "" {
		return nil, svcerrors.New(svcerrors.ErrBadRequest, "room id required")
	}
	r, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to load room")
	}
	if r == nil {
		return nil, svcerrors.New(svcerrors.ErrNotFound, "room not found")
	}
	return r, nil
}

func (s *roomService) GetRoomByInvite(ctx context.Context, inviteCode string) (*domain.Room, error) {
	code := strings.ToUpper(strings.TrimSpace(inviteCode))
	if code == "" {
		return nil, svcerrors.New(svcerrors.ErrBadRequest, "invite code required")
	}
	r, err := s.roomRepo.GetByInviteCode(ctx, code)
	if err != nil {
		return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to look up room")
	}
	if r == nil {
		return nil, svcerrors.New(svcerrors.ErrNotFound, "invite link is invalid or the room has closed")
	}
	return r, nil
}

func (s *roomService) JoinRoom(ctx context.Context, inviteCode, password string) (*domain.Room, error) {
	r, err := s.GetRoomByInvite(ctx, inviteCode)
	if err != nil {
		return nil, err
	}
	if r.State == domain.RoomStateClosed || r.State == domain.RoomStateFinished {
		return nil, svcerrors.New(svcerrors.ErrGone, "this room is closed")
	}
	if r.HasPassword {
		if err := security.VerifyPassword(r.PasswordHash, password); err != nil {
			return nil, svcerrors.New(svcerrors.ErrForbidden, "wrong room password")
		}
	}
	// Capacity + state checks are enforced by the realtime hub at join time.
	return r, nil
}

func (s *roomService) UpdateSettings(ctx context.Context, roomID, ownerID string, patch RoomSettings) (*domain.Room, error) {
	r, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if r.OwnerID != ownerID {
		return nil, svcerrors.New(svcerrors.ErrForbidden, "only the room owner can change settings")
	}
	if r.State != domain.RoomStateLobby {
		return nil, svcerrors.New(svcerrors.ErrConflict, "settings are locked once the game starts")
	}
	sets := defaultSettings(patch, r.Language)
	if patch.Language == "" {
		sets.Language = r.Language
	}

	// Name: validate the raw submitted name against length bounds; defaultSettings
	// may have synthesized "Drawo Room" for zero values which would bypass the
	// minimum-length check the caller is required to satisfy.
	if patch.Name != "" {
		name := strings.TrimSpace(patch.Name)
		if len([]rune(name)) < domain.RoomMinNameLength || len([]rune(name)) > domain.RoomMaxNameLength {
			return nil, svcerrors.New(svcerrors.ErrBadRequest, "room name must be 3–50 characters")
		}
		r.Name = name
	}
	// Password: nil=unchanged, pointer-to-empty=clear, non-empty=set.
	// We intentionally validate the RAW submitted value (*patch.Password),
	// not the defaultSettings-munged value, because defaultSettings may have
	// populated MaxPlayers etc. but leaves pointer fields untouched.
	if patch.Password != nil {
		pw := strings.TrimSpace(*patch.Password)
		if pw == "" {
			r.PasswordHash = ""
			r.HasPassword = false
		} else {
			if len(pw) < 4 || len(pw) > 32 {
				return nil, svcerrors.New(svcerrors.ErrBadRequest, "password must be 4–32 characters")
			}
			h, err := security.HashPassword(pw)
			if err != nil {
				return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to hash password")
			}
			r.PasswordHash = h
			r.HasPassword = true
		}
	}
	// Language — validate the submitted value, not the default-munged one.
	if patch.Language != "" {
		lang := strings.ToLower(strings.TrimSpace(patch.Language))
		if lang != "en" && lang != "fa" {
			return nil, svcerrors.New(svcerrors.ErrBadRequest, "language must be en or fa")
		}
		r.Language = lang
	}
	// Numeric settings
	if patch.MinPlayers > 0 {
		r.MinPlayers = sets.MinPlayers
	}
	if patch.MaxPlayers > 0 {
		r.MaxPlayers = sets.MaxPlayers
	}
	// Keep invariants: min ≤ max, both within bounds.
	if r.MinPlayers < domain.RoomMinPlayers {
		r.MinPlayers = domain.RoomMinPlayers
	}
	if r.MaxPlayers < r.MinPlayers {
		r.MaxPlayers = r.MinPlayers
	}
	if r.MaxPlayers > domain.RoomMaxPlayers {
		r.MaxPlayers = domain.RoomMaxPlayers
	}
	if patch.MaxRounds > 0 {
		r.MaxRounds = sets.MaxRounds
	}
	if patch.RoundTime > 0 {
		r.RoundTime = sets.RoundTime
	}
	// Word source
	if patch.WordSource != "" {
		if patch.WordSource != domain.WordSourceDefault &&
			patch.WordSource != domain.WordSourceCustom &&
			patch.WordSource != domain.WordSourceCategory {
			return nil, svcerrors.New(svcerrors.ErrBadRequest, "invalid word source")
		}
		r.WordSource = patch.WordSource
	}
	if r.WordSource == domain.WordSourceCustom {
		// Only re-validate if custom categories were submitted.
		if patch.CustomCategories != nil {
			if err := s.ValidateCustomCategories(patch.CustomCategories); err != nil {
				return nil, err
			}
			r.CustomCategories = normalizeCustomCategories(patch.CustomCategories)
		}
		if len(r.CustomCategories) == 0 {
			return nil, svcerrors.New(svcerrors.ErrBadRequest, "add at least one category with words when using custom words")
		}
	} else {
		r.CustomCategories = nil
	}

	r.UpdatedAt = time.Now()
	if err := s.roomRepo.Save(ctx, r); err != nil {
		return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to save room")
	}
	return r, nil
}

func (s *roomService) CloseRoom(ctx context.Context, roomID, ownerID string) error {
	r, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return err
	}
	if r.OwnerID != ownerID {
		return svcerrors.New(svcerrors.ErrForbidden, "only the owner can close the room")
	}
	r.State = domain.RoomStateClosed
	r.UpdatedAt = time.Now()
	if err := s.roomRepo.Save(ctx, r); err != nil {
		return svcerrors.New(svcerrors.ErrInternalServer, "failed to close room")
	}
	return nil
}

func (s *roomService) StartGame(ctx context.Context, roomID, ownerID string, playerCount int) error {
	r, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return err
	}
	if r.OwnerID != ownerID {
		return svcerrors.New(svcerrors.ErrForbidden, "only the owner can start the game")
	}
	if r.State != domain.RoomStateLobby {
		return svcerrors.New(svcerrors.ErrConflict, "the game has already started")
	}
	if playerCount < r.MinPlayers {
		return svcerrors.Newf(svcerrors.ErrBadRequest, "need at least %d players to start", r.MinPlayers)
	}
	if r.WordSource == domain.WordSourceCustom {
		if err := s.ValidateCustomCategories(r.CustomCategories); err != nil {
			return err
		}
	}
	r.State = domain.RoomStatePlaying
	r.CurrentRound = 1
	r.UpdatedAt = time.Now()
	if err := s.roomRepo.Save(ctx, r); err != nil {
		return svcerrors.New(svcerrors.ErrInternalServer, "failed to start game")
	}
	return nil
}

func (s *roomService) LeaveRoom(ctx context.Context, roomID, userID string, newOwnerID string) (*domain.Room, error) {
	r, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if r.State == domain.RoomStateClosed || r.State == domain.RoomStateFinished {
		return r, nil
	}
	if r.OwnerID == userID {
		// Ownership transfer. If newOwnerID is empty we close the room.
		if newOwnerID == "" {
			r.State = domain.RoomStateClosed
		} else {
			r.OwnerID = newOwnerID
		}
	}
	r.UpdatedAt = time.Now()
	if err := s.roomRepo.Save(ctx, r); err != nil {
		return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to update room")
	}
	return r, nil
}

// IssueGuestToken mints a short-lived, room-bound anonymous token so an
// unauthenticated user can connect to the WebSocket and play. The guest ID
// is namespaced with domain.GuestIDPrefix so it never collides with real
// user UUIDs.
func (s *roomService) IssueGuestToken(ctx context.Context, roomID, nickname string) (*domain.GuestAuth, error) {
	if roomID == "" {
		return nil, svcerrors.New(svcerrors.ErrBadRequest, "room id required")
	}
	r, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if r.State != domain.RoomStateLobby {
		return nil, svcerrors.New(svcerrors.ErrConflict, "this room is not accepting new players")
	}
	nick := sanitizeNickname(nickname)
	if len([]rune(nick)) < minGuestNickname {
		return nil, svcerrors.New(svcerrors.ErrBadRequest, "nickname is too short")
	}
	if len([]rune(nick)) > maxGuestNickname {
		return nil, svcerrors.New(svcerrors.ErrBadRequest, "nickname is too long")
	}
	now := time.Now()
	g := &domain.GuestAuth{
		Token:     generateGuestToken(),
		GuestID:   domain.GuestIDPrefix + uuid.New().String(),
		RoomID:    roomID,
		Nickname:  nick,
		ExpiresAt: now.Add(guestTokenTTL),
		CreatedAt: now,
	}
	if err := s.roomRepo.SaveGuest(ctx, g); err != nil {
		return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to issue guest token")
	}
	return g, nil
}

func (s *roomService) ValidateGuestToken(ctx context.Context, token string) (*domain.GuestAuth, error) {
	tok := strings.TrimSpace(token)
	if tok == "" {
		return nil, svcerrors.New(svcerrors.ErrUnauthorized, "guest token required")
	}
	g, err := s.roomRepo.GetGuest(ctx, tok)
	if err != nil {
		return nil, svcerrors.New(svcerrors.ErrInternalServer, "failed to validate guest token")
	}
	if g == nil {
		return nil, svcerrors.New(svcerrors.ErrUnauthorized, "invalid or expired guest token")
	}
	// Ensure the room still exists and isn't closed; if it was closed,
	// invalidate the guest token.
	r, err := s.GetRoom(ctx, g.RoomID)
	if err != nil || r == nil || r.State == domain.RoomStateClosed {
		_ = s.roomRepo.DeleteGuest(ctx, tok)
		return nil, svcerrors.New(svcerrors.ErrGone, "the room has closed")
	}
	return g, nil
}

