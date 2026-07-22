package realtime

import (
	"strings"
	"time"

	"drawo/internal/core/domain"
)

const (
	GameStateWaitingForPlayers  = "waiting_for_players"
	GameStateCountdown          = "countdown"
	GameStateWordSelection      = "word_selection"
	GameStateDrawing            = "drawing"
	GameStateDrawerDisconnected = "drawer_disconnected"
	GameStateRoundEnd           = "round_end"
	GameStateLeaderboard        = "leaderboard"
	GameStateGameEnd            = "game_end"
)

var (
	countdownDuration     = 5 * time.Second
	wordSelectionDuration = 10 * time.Second
	roundEndDuration      = 5 * time.Second
	leaderboardDuration   = 5 * time.Second
	reconnectGrace        = 60 * time.Second
	drawerReconnectGrace  = 15 * time.Second
)

const (
	defaultSuggestedWords = 3
	defaultWordPoints     = 1
	completionRepBonus    = int64(20)
	noReportRepBonus      = int64(10)
	correctGuessRepBonus  = int64(2)
	drawerSuccessRepBonus = int64(5)
	abandonRepPenalty     = int64(-50)
	drawerAbandonPenalty  = int64(-80)
	maxPositiveRepPerGame = int64(50)
)

type PlayerState struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username,omitempty"`
	Score             int64  `json:"score"`
	IsDrawer          bool   `json:"is_drawer"`
	IsOnline          bool   `json:"is_online"`
	GuessedWord       bool   `json:"guessed_word"`
	ClientID          string `json:"-"`
	JoinedAt          int64  `json:"joined_at"`
	DisconnectedAt    int64  `json:"disconnected_at,omitempty"`
	ReconnectDeadline int64  `json:"reconnect_deadline,omitempty"`
	Abandoned         bool   `json:"abandoned"`
}

type PlayerEventPayload struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username,omitempty"`
	ReconnectDeadline int64  `json:"reconnect_deadline,omitempty"`
}

type GameStatePayload struct {
	State        string        `json:"state"`
	RoomID       string        `json:"room_id"`
	Language     string        `json:"language"`
	Round        int           `json:"round"`
	MaxRounds    int           `json:"max_rounds"`
	DrawerID     string        `json:"drawer_id,omitempty"`
	Players      []PlayerState `json:"players"`
	MinPlayers   int           `json:"min_players"`
	MaxPlayers   int           `json:"max_players"`
	EndsAt       int64         `json:"ends_at,omitempty"`
	WordRevealed string        `json:"word,omitempty"`
}

type GameEventPayload struct {
	Event    string          `json:"event"`
	Words    []WordCandidate `json:"words,omitempty"`
	Word     string          `json:"word,omitempty"`
	Points   int             `json:"points,omitempty"`
	GroupID  string          `json:"group_id,omitempty"`
	Language string          `json:"language,omitempty"`
}

type ChooseWordPayload struct {
	Event   string `json:"event"`
	GroupID string `json:"group_id"`
}

type ChatPayload struct {
	Text    string `json:"text,omitempty"`
	System  bool   `json:"system,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	Message string `json:"message,omitempty"`
}

type WordCandidate struct {
	GroupID string `json:"group_id"`
	Text    string `json:"text"`
	Points  int    `json:"points"`
}

func fallbackWords(lang string) []WordCandidate {
	if strings.EqualFold(lang, "fa") {
		return []WordCandidate{
			{GroupID: "apple", Text: "سیب", Points: 1},
			{GroupID: "airplane", Text: "هواپیما", Points: 2},
			{GroupID: "democracy", Text: "دموکراسی", Points: 3},
		}
	}
	return []WordCandidate{
		{GroupID: "apple", Text: "apple", Points: 1},
		{GroupID: "airplane", Text: "airplane", Points: 2},
		{GroupID: "democracy", Text: "democracy", Points: 3},
	}
}

func wordCandidatesFromDomain(words []domain.Word) []WordCandidate {
	out := make([]WordCandidate, 0, len(words))
	for _, w := range words {
		points := w.Points
		if points <= 0 {
			points = defaultWordPoints
		}
		out = append(out, WordCandidate{GroupID: w.GroupID, Text: w.Text, Points: points})
	}
	return out
}
