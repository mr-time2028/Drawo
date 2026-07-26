package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
	"drawo/pkg/errors"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 * 1024 // 64 KiB hard limit to prevent memory abuse.
)

type Room struct {
	state   *domain.Room
	clients map[string]*Client
	inbox   chan *RoomEvent
	onClose func(roomID, inviteCode string)

	contentRepo  repositories.ContentRepository
	reputation   *reputationLedger
	reportRepo   repositories.ReportRepository
	reportKeys   map[string]struct{}
	roundReports map[string]map[string]struct{}

	gameState             string
	players               map[string]*PlayerState
	playerOrder           []string
	currentDrawerIndex    int
	suggestedWords        []WordCandidate
	currentWord           *WordCandidate
	roundDrawingSucceeded bool
	roundEndsAt           time.Time
	pausedRoundRemaining  time.Duration
	timer                 *time.Timer
	timerC                <-chan time.Time
	reconnectTimer        *time.Timer
	reconnectTimerC       <-chan time.Time

	// Canvas operation log. This is the authoritative in-memory canvas
	// state for the current round. New joiners receive this log via canvas_sync
	// and reconstruct the canvas client-side.
	canvasOps []DrawOperation
	redoOps   map[string][]DrawOperation
	drawSeq   int64

	// Per-client drawing rate-limit state. It is owned by Room.Run, so it needs
	// no mutex. This catches canvas griefing such as fill spam and clear spam.
	drawLimits  map[string]*drawLimitState
	chatLimits  map[string]*chatLimitState
	badWords    badWordCache
	chatHistory []ChatPayload
}

func NewRoom(state *domain.Room, onClose func(string, string), contentRepo repositories.ContentRepository, profileRepo repositories.ProfileRepository, reputationRepo repositories.ReputationRepository, reportRepo repositories.ReportRepository, extraRepos ...interface{}) *Room {
	return &Room{
		state:        state,
		clients:      make(map[string]*Client),
		inbox:        make(chan *RoomEvent, 512),
		onClose:      onClose,
		contentRepo:  contentRepo,
		reputation:   newReputationLedger(profileRepo, reputationRepo, state.ID, extraRepos...),
		reportRepo:   reportRepo,
		reportKeys:   make(map[string]struct{}),
		roundReports: make(map[string]map[string]struct{}),
		gameState:    GameStateWaitingForPlayers,
		players:      make(map[string]*PlayerState),
		canvasOps:    make([]DrawOperation, 0, 256),
		redoOps:      make(map[string][]DrawOperation),
		drawLimits:   make(map[string]*drawLimitState),
		chatLimits:   make(map[string]*chatLimitState),
		chatHistory:  make([]ChatPayload, 0, 100),
	}
}

// Dispatch queues an event for this room's single goroutine.
// It is intentionally non-blocking: if a room cannot keep up, callers receive
// false and may disconnect/back-pressure the client rather than deadlocking the
// WebSocket read pump.
func (r *Room) Dispatch(e *RoomEvent) bool {
	select {
	case r.inbox <- e:
		return true
	default:
		return false
	}
}

func (r *Room) Run(ctx context.Context) {
	defer func() {
		for _, client := range r.clients {
			closeClientSend(client)
		}
		r.onClose(r.state.ID, r.state.InviteCode)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-r.inbox:
			if !r.handleEvent(event) {
				return
			}
		case <-r.timerC:
			r.timerC = nil
			if !r.handleTimer() {
				return
			}
		case <-r.reconnectTimerC:
			r.reconnectTimerC = nil
			if !r.handleReconnectTimer(false) {
				return
			}
		}
	}
}

func (r *Room) handleEvent(e *RoomEvent) bool {
	if e == nil || e.Client == nil {
		return true
	}

	switch e.Type {
	case EventJoin:
		r.handleJoin(e.Client)
	case EventLeave:
		r.handleLeave(e.Client)
		if len(r.clients) == 0 && r.state.State == domain.RoomStateFinished {
			return false
		}
	case EventDraw:
		if _, ok := r.clients[e.Client.ID]; ok && r.gameState == GameStateDrawing {
			if payload, ok := r.applyDrawingEvent(e); ok {
				e.Payload = payload
				r.broadcast(e, "")
			}
		} else {
			r.sendError(e.Client, errors.WSErrDrawNotAllowed, "drawing is not active")
		}
	case EventChat:
		r.handleChat(e)
	case EventGame:
		r.handleGameEvent(e)
	case EventClearCanvas, EventGameState:
		if _, ok := r.clients[e.Client.ID]; ok {
			r.broadcast(e, "")
		}
	}
	return true
}

func (r *Room) applyDrawingEvent(e *RoomEvent) (json.RawMessage, bool) {
	if !r.canClientDraw(e.Client) {
		r.sendError(e.Client, errors.WSErrDrawForbidden, "only the current drawer can draw")
		return nil, false
	}

	op, err := ValidateDrawingPayload(e.Payload)
	if err != nil {
		r.sendError(e.Client, errors.WSErrInvalidDraw, err.Error())
		return nil, false
	}
	if err := r.allowDrawingOperation(e.Client.ID, op); err != nil {
		r.sendError(e.Client, errors.WSErrDrawRateLimited, err.Error())
		return nil, false
	}

	r.drawSeq++
	op.ServerSeq = r.drawSeq
	op.Timestamp = e.Timestamp.UnixMilli()
	op.UserID = e.Client.UserID

	switch op.Op {
	case DrawOpUndo:
		return r.applyUndo(op, e.Client.UserID)
	case DrawOpRedo:
		return r.applyRedo(op, e.Client.UserID)
	case DrawOpClear:
		r.canvasOps = r.canvasOps[:0]
		r.redoOps = make(map[string][]DrawOperation)
		return mustMarshalRaw(op), true
	default:
		if op.ID == "" {
			op.ID = fmt.Sprintf("%s-%d", r.state.ID, op.ServerSeq)
		}
		r.canvasOps = append(r.canvasOps, op)
		if len(r.canvasOps) > maxCanvasHistoryOps {
			r.canvasOps = append([]DrawOperation(nil), r.canvasOps[len(r.canvasOps)-maxCanvasHistoryOps:]...)
		}
		delete(r.redoOps, e.Client.UserID)
		return mustMarshalRaw(op), true
	}
}

func (r *Room) applyUndo(op DrawOperation, userID string) (json.RawMessage, bool) {
	for i := len(r.canvasOps) - 1; i >= 0; i-- {
		candidate := r.canvasOps[i]
		if candidate.UserID != userID {
			continue
		}
		r.canvasOps = append(r.canvasOps[:i], r.canvasOps[i+1:]...)
		r.redoOps[userID] = append(r.redoOps[userID], candidate)
		if len(r.redoOps[userID]) > maxRedoOpsPerClient {
			r.redoOps[userID] = r.redoOps[userID][len(r.redoOps[userID])-maxRedoOpsPerClient:]
		}
		op.TargetID = candidate.ID
		return mustMarshalRaw(op), true
	}
	return nil, false
}

func (r *Room) applyRedo(op DrawOperation, userID string) (json.RawMessage, bool) {
	stack := r.redoOps[userID]
	if len(stack) == 0 {
		return nil, false
	}
	restored := stack[len(stack)-1]
	r.redoOps[userID] = stack[:len(stack)-1]
	r.canvasOps = append(r.canvasOps, restored)
	op.TargetID = restored.ID
	op.Target = &restored
	return mustMarshalRaw(op), true
}

func mustMarshalRaw(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

type drawLimitState struct {
	windowStart      time.Time
	count            int
	fillWindowStart  time.Time
	fillCount        int
	clearWindowStart time.Time
	clearCount       int
}

func (r *Room) canClientDraw(client *Client) bool {
	if client == nil {
		return false
	}
	// The game loop will set CurrentDrawerID each turn. Until then, empty means the
	// room is not enforcing drawer ownership yet (useful for lobby/prototype tests).
	if r.state.CurrentDrawerID == "" {
		return true
	}
	return client.UserID == r.state.CurrentDrawerID
}

func (r *Room) allowDrawingOperation(clientID string, op DrawOperation) error {
	now := time.Now()
	state := r.drawLimits[clientID]
	if state == nil {
		state = &drawLimitState{windowStart: now, fillWindowStart: now, clearWindowStart: now}
		r.drawLimits[clientID] = state
	}

	if now.Sub(state.windowStart) >= time.Second {
		state.windowStart = now
		state.count = 0
	}
	state.count++
	if state.count > maxDrawOpsPerSecond {
		return fmt.Errorf("too many drawing operations per second")
	}

	if op.Op == DrawOpFill {
		if now.Sub(state.fillWindowStart) >= time.Second {
			state.fillWindowStart = now
			state.fillCount = 0
		}
		state.fillCount++
		if state.fillCount > maxFillOpsPerSecond {
			return fmt.Errorf("too many fill operations per second")
		}
	}

	if op.Op == DrawOpClear {
		if now.Sub(state.clearWindowStart) >= time.Minute {
			state.clearWindowStart = now
			state.clearCount = 0
		}
		state.clearCount++
		if state.clearCount > maxClearOpsPerMinute {
			return fmt.Errorf("too many canvas clear operations per minute")
		}
	}

	return nil
}

func (r *Room) sendError(client *Client, code errors.WSErrorCode, details string) {
	message := errors.WSTranslatedMessage(r.state.Language, code)
	payload := ErrorPayload{Code: code.String(), Message: message}
	if details != "" && details != message && details != errors.WSDefaultMessage(code) {
		payload.Details = details
	}
	r.sendSystem(client, EventError, payload)
}

func (r *Room) broadcast(e *RoomEvent, excludeClientID string) {
	data, err := json.Marshal(MessageEnvelope{
		Type:      e.Type,
		Payload:   e.Payload,
		Seq:       e.Seq,
		Timestamp: e.Timestamp.Unix(),
	})
	if err != nil {
		return
	}

	for _, client := range r.clients {
		if client.ID == excludeClientID {
			continue
		}
		if !safeSend(client, data) {
			delete(r.clients, client.ID)
			closeClientSend(client)
		}
	}
}

func (r *Room) sendSystem(client *Client, eventType EventType, payload any) {
	payloadJSON, _ := json.Marshal(payload)
	data, _ := json.Marshal(MessageEnvelope{
		Type:      eventType,
		Payload:   payloadJSON,
		Timestamp: time.Now().Unix(),
	})
	if !safeSend(client, data) {
		delete(r.clients, client.ID)
		closeClientSend(client)
	}
}

func safeSend(client *Client, data []byte) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	if client == nil || client.Send == nil {
		return false
	}
	select {
	case <-client.Done:
		return false
	default:
	}
	select {
	case client.Send <- data:
		return true
	default:
		return false
	}
}

func closeClientSend(client *Client) {
	defer func() { _ = recover() }()
	if client == nil {
		return
	}
	if client.Done != nil {
		select {
		case <-client.Done:
			return
		default:
			close(client.Done)
		}
	}
	if client.Send != nil {
		close(client.Send)
	}
}

func (r *Room) handleJoin(client *Client) {
	player := r.players[client.UserID]
	isReconnect := false
	if player != nil && player.IsOnline && player.ClientID != "" && player.ClientID != client.ID {
		// One active socket per user per room. A reconnect or duplicate browser tab
		// replaces the older socket so the same player cannot draw/guess twice.
		if oldClient, ok := r.clients[player.ClientID]; ok {
			delete(r.clients, oldClient.ID)
			delete(r.drawLimits, oldClient.ID)
			closeClientSend(oldClient)
		}
	}
	r.clients[client.ID] = client
	if player == nil {
		player = &PlayerState{UserID: client.UserID, Username: client.Username, Score: 0, IsOnline: true, ClientID: client.ID, JoinedAt: time.Now().Unix()}
		r.players[client.UserID] = player
		r.playerOrder = append(r.playerOrder, client.UserID)
	} else {
		if player.Abandoned {
			r.sendError(client, errors.WSErrReconnectExpired, "reconnect window expired")
			return
		}
		isReconnect = !player.IsOnline
		player.IsOnline = true
		player.ClientID = client.ID
		player.DisconnectedAt = 0
		player.ReconnectDeadline = 0
	}
	if isReconnect {
		r.broadcastSystemExcept(client.ID, EventPlayerReconnected, PlayerEventPayload{UserID: player.UserID, Username: player.Username})
	} else {
		r.broadcastSystemExcept(client.ID, EventPlayerJoined, PlayerEventPayload{UserID: player.UserID, Username: player.Username})
	}
	if client.UserID == r.state.CurrentDrawerID && r.gameState == GameStateDrawerDisconnected {
		r.resumeDrawerAfterReconnect()
	}
	r.sendSystem(client, EventCanvasSync, CanvasSyncPayload{Operations: append([]DrawOperation(nil), r.canvasOps...), ServerSeq: r.drawSeq})
	r.sendSystem(client, EventJoined, JoinedPayload{RoomID: r.state.ID, State: r.gameState})
	r.broadcastGameState()
	if r.gameState == GameStateWaitingForPlayers && r.onlinePlayerCount() >= r.minPlayers() {
		r.startCountdown()
	}
}

func (r *Room) handleLeave(client *Client) {
	if existing, ok := r.clients[client.ID]; ok {
		delete(r.clients, client.ID)
		delete(r.drawLimits, client.ID)
		closeClientSend(existing)
	}
	player := r.players[client.UserID]
	if player != nil {
		player.IsOnline = false
		player.ClientID = ""
		player.DisconnectedAt = time.Now().Unix()
		player.ReconnectDeadline = time.Now().Add(reconnectGrace).Unix()
	}
	if player != nil && r.isActiveGameState() {
		r.scheduleReconnectCheck(time.Until(time.Unix(player.ReconnectDeadline, 0)))
		if player.IsDrawer && r.gameState == GameStateDrawing {
			r.pauseForDrawerDisconnect()
		}
	}
	r.broadcastSystem(EventPlayerLeft, PlayerEventPayload{UserID: client.UserID, Username: displayName(player), ReconnectDeadline: playerReconnectDeadline(player)})
	r.broadcastGameState()
	if len(r.clients) == 0 {
		r.clearTimer()
		r.scheduleReconnectCheck(reconnectGrace)
	}
}

func playerReconnectDeadline(player *PlayerState) int64 {
	if player == nil {
		return 0
	}
	return player.ReconnectDeadline
}

func (r *Room) handleChat(e *RoomEvent) {
	if r.gameState != GameStateDrawing {
		r.broadcast(e, "")
		return
	}
	if e.Client.UserID == r.state.CurrentDrawerID {
		r.sendError(e.Client, errors.WSErrDrawerChatBlocked, "drawer cannot chat during drawing")
		return
	}
	var payload ChatPayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil || strings.TrimSpace(payload.Text) == "" {
		r.sendError(e.Client, errors.WSErrInvalidChat, "chat text is required")
		return
	}
	if !r.allowChat(e.Client.ID) {
		r.sendError(e.Client, errors.WSErrChatRateLimited, "too many chat messages")
		return
	}
	if r.containsBadWord(payload.Text) {
		r.sendError(e.Client, errors.WSErrBadWord, "message contains prohibited words")
		r.reputation.add(e.Client.UserID, -20, "chat_bad_word")
		return
	}
	player := r.players[e.Client.UserID]
	if player != nil && player.GuessedWord {
		r.sendError(e.Client, errors.WSErrAlreadyGuessed, "you already guessed this word")
		return
	}
	if r.currentWord != nil && NormalizeGuess(payload.Text, r.state.Language) == NormalizeGuess(r.currentWord.Text, r.state.Language) {
		r.handleCorrectGuess(e.Client)
		return
	}
	payload.UserID = e.Client.UserID
	r.recordChat(payload)
	e.Payload = mustMarshalRaw(payload)
	r.broadcast(e, "")
}

func (r *Room) handleCorrectGuess(client *Client) {
	player := r.players[client.UserID]
	if player == nil || r.currentWord == nil {
		return
	}
	player.GuessedWord = true
	player.CorrectGuesses++
	remaining := time.Until(r.roundEndsAt)
	player.Score += CalculateGuessScore(r.currentWord.Points, int64(remaining.Seconds()))
	if r.isRankedGame() {
		r.reputation.addPositiveCapped(player.UserID, correctGuessRepBonus, "correct_guess")
	}
	if drawer := r.players[r.state.CurrentDrawerID]; drawer != nil {
		drawer.Score += CalculateDrawerBonus(r.currentWord.Points)
		if !r.roundDrawingSucceeded {
			drawer.SuccessfulDrawings++
			r.roundDrawingSucceeded = true
		}
		if r.isRankedGame() {
			r.reputation.addPositiveCapped(drawer.UserID, drawerSuccessRepBonus, "successful_drawing")
		}
	}
	systemMsg := ChatPayload{System: true, Message: fmt.Sprintf("%s guessed the word!", displayName(player))}
	r.recordChat(systemMsg)
	r.broadcastSystem(EventChat, systemMsg)
	r.broadcastGameState()
	if r.allGuessersDone() {
		r.endRound()
	}
}

func (r *Room) allowChat(clientID string) bool {
	now := time.Now()
	state := r.chatLimits[clientID]
	if state == nil {
		state = &chatLimitState{windowStart: now}
		r.chatLimits[clientID] = state
	}
	if now.Sub(state.windowStart) >= chatWindowDuration {
		state.windowStart = now
		state.count = 0
	}
	state.count++
	return state.count <= maxChatMessagesPerWind
}

func (r *Room) containsBadWord(text string) bool {
	if r.contentRepo == nil {
		return false
	}
	now := time.Now()
	if now.After(r.badWords.expiresAt) {
		words, err := r.contentRepo.ListBadWords(context.Background(), r.state.Language)
		if err != nil {
			words = nil
		}
		r.badWords = badWordCache{words: words, loadedAt: now, expiresAt: now.Add(badWordCacheTTL)}
	}
	normalizedText := NormalizeModerationText(text, r.state.Language)
	if normalizedText == "" {
		return false
	}
	for _, bw := range r.badWords.words {
		normalizedBadWord := NormalizeModerationText(bw.Text, bw.Language)
		if normalizedBadWord != "" && strings.Contains(normalizedText, normalizedBadWord) {
			return true
		}
	}
	return false
}

func (r *Room) recordChat(payload ChatPayload) {
	r.chatHistory = append(r.chatHistory, payload)
	if len(r.chatHistory) > 100 {
		r.chatHistory = append([]ChatPayload(nil), r.chatHistory[len(r.chatHistory)-100:]...)
	}
}

func (r *Room) handleGameEvent(e *RoomEvent) {
	var base struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal(e.Payload, &base); err != nil {
		r.sendError(e.Client, errors.WSErrInvalidGameEvent, "invalid game event")
		return
	}
	switch base.Event {
	case "choose_word":
		var payload ChooseWordPayload
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			r.sendError(e.Client, errors.WSErrInvalidGameEvent, "invalid word choice")
			return
		}
		if r.gameState != GameStateWordSelection || e.Client.UserID != r.state.CurrentDrawerID {
			r.sendError(e.Client, errors.WSErrWordChoiceForbidden, "only the current drawer can choose a word")
			return
		}
		r.chooseWord(payload.GroupID)
	case "report":
		var payload ReportPayload
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			r.sendError(e.Client, errors.WSErrInvalidReport, "invalid report payload")
			return
		}
		r.handleReportEvent(e.Client, payload)
	default:
		r.sendError(e.Client, errors.WSErrUnsupportedGameEvent, "unsupported game event")
	}
}

func (r *Room) startCountdown() {
	r.gameState = GameStateCountdown
	r.clearTimer()
	r.setTimer(countdownDuration)
	r.broadcastGameState()
}

func (r *Room) startWordSelection() {
	r.gameState = GameStateWordSelection
	r.state.State = domain.RoomStatePlaying
	r.state.CurrentRound++
	r.reputation.setContext(r.state.ID, r.state.CurrentRound)
	if r.state.MaxRounds <= 0 {
		r.state.MaxRounds = defaultMaxRounds
	}
	r.pickDrawer()
	r.suggestedWords = r.loadWordSuggestions()
	r.currentWord = nil
	r.roundDrawingSucceeded = false
	r.canvasOps = r.canvasOps[:0]
	r.redoOps = make(map[string][]DrawOperation)
	r.drawSeq = 0
	r.clearTimer()
	r.setTimer(wordSelectionDuration)
	r.sendWordSuggestions()
	r.broadcastGameState()
}

func (r *Room) chooseWord(groupID string) {
	if len(r.suggestedWords) == 0 {
		r.suggestedWords = fallbackWords(r.state.Language)
	}
	choice := r.suggestedWords[0]
	for _, candidate := range r.suggestedWords {
		if candidate.GroupID == groupID {
			choice = candidate
			break
		}
	}
	r.currentWord = &choice
	r.gameState = GameStateDrawing
	for _, player := range r.players {
		player.GuessedWord = false
	}
	if r.state.RoundTime <= 0 {
		r.state.RoundTime = defaultRoundTime
	}
	r.roundEndsAt = time.Now().Add(time.Duration(r.state.RoundTime) * time.Second)
	r.clearTimer()
	r.setTimer(time.Duration(r.state.RoundTime) * time.Second)
	r.sendToUser(r.state.CurrentDrawerID, EventGame, GameEventPayload{Event: "word_chosen", Word: choice.Text, Points: choice.Points, GroupID: choice.GroupID, Language: r.state.Language})
	r.broadcastGameState()
}

func (r *Room) endRound() {
	if r.currentWord == nil {
		return
	}
	r.gameState = GameStateRoundEnd
	r.clearTimer()
	r.setTimer(roundEndDuration)
	r.broadcastGameStateWithWord(r.currentWord.Text)
}

func (r *Room) showLeaderboard() {
	r.gameState = GameStateLeaderboard
	r.clearTimer()
	r.setTimer(leaderboardDuration)
	r.broadcastGameState()
}

func (r *Room) endGame() {
	r.markExpiredDisconnects(true)
	r.gameState = GameStateGameEnd
	r.state.State = domain.RoomStateFinished
	r.clearTimer()
	if r.isRankedGame() {
		for _, player := range r.players {
			if !player.Abandoned {
				r.reputation.addPositiveCapped(player.UserID, completionRepBonus, "completed_game")
				r.reputation.addPositiveCapped(player.UserID, noReportRepBonus, "no_reports")
			}
		}
	}
	r.reputation.flush()
	r.broadcastGameState()
}

func (r *Room) handleTimer() bool {
	if len(r.clients) == 0 {
		r.state.State = domain.RoomStateFinished
		r.reputation.flush()
		return false
	}
	switch r.gameState {
	case GameStateCountdown:
		r.startWordSelection()
	case GameStateWordSelection:
		if len(r.suggestedWords) > 0 {
			r.chooseWord(r.suggestedWords[0].GroupID)
		}
	case GameStateDrawing:
		r.endRound()
	case GameStateRoundEnd:
		r.showLeaderboard()
	case GameStateLeaderboard:
		if r.state.CurrentRound >= r.state.MaxRounds {
			r.endGame()
		} else {
			r.startWordSelection()
		}
	}
	return true
}

func (r *Room) pickDrawer() {
	if len(r.playerOrder) == 0 {
		return
	}
	for i := 0; i < len(r.playerOrder); i++ {
		idx := (r.currentDrawerIndex + i) % len(r.playerOrder)
		player := r.players[r.playerOrder[idx]]
		if player != nil && player.IsOnline && !player.Abandoned {
			r.currentDrawerIndex = (idx + 1) % len(r.playerOrder)
			r.state.CurrentDrawerID = player.UserID
			for _, p := range r.players {
				p.IsDrawer = p.UserID == player.UserID
			}
			return
		}
	}
}

func (r *Room) loadWordSuggestions() []WordCandidate {
	if len(r.state.CustomWords) > 0 {
		out := make([]WordCandidate, 0, len(r.state.CustomWords))
		for i, word := range r.state.CustomWords {
			if strings.TrimSpace(word) == "" {
				continue
			}
			out = append(out, WordCandidate{GroupID: fmt.Sprintf("custom-%d", i), Text: word, Points: defaultWordPoints})
			if len(out) >= defaultSuggestedWords {
				break
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if r.contentRepo != nil {
		words, err := r.contentRepo.GetRandomWordGroups(context.Background(), r.state.CategoryID, r.state.Language, defaultSuggestedWords)
		if err == nil && len(words) > 0 {
			return wordCandidatesFromDomain(words)
		}
	}
	return fallbackWords(r.state.Language)
}

func (r *Room) sendWordSuggestions() {
	r.sendToUser(r.state.CurrentDrawerID, EventGame, GameEventPayload{Event: "word_suggestions", Words: r.suggestedWords, Language: r.state.Language})
}

func (r *Room) broadcastGameState() {
	r.broadcastGameStateWithWord("")
}

func (r *Room) broadcastGameStateWithWord(word string) {
	r.broadcastSystem(EventGameState, GameStatePayload{State: r.gameState, RoomID: r.state.ID, Language: r.state.Language, Round: r.state.CurrentRound, MaxRounds: r.state.MaxRounds, DrawerID: r.state.CurrentDrawerID, Players: r.playerSnapshot(), MinPlayers: r.minPlayers(), MaxPlayers: r.maxPlayers(), EndsAt: r.endsAtUnix(), WordRevealed: word})
}

func (r *Room) playerSnapshot() []PlayerState {
	out := make([]PlayerState, 0, len(r.playerOrder))
	for _, userID := range r.playerOrder {
		if p := r.players[userID]; p != nil {
			out = append(out, *p)
		}
	}
	return out
}

func (r *Room) broadcastSystem(eventType EventType, payload any) {
	r.broadcastSystemExcept("", eventType, payload)
}

func (r *Room) broadcastSystemExcept(excludeClientID string, eventType EventType, payload any) {
	for _, client := range r.clients {
		if client.ID == excludeClientID {
			continue
		}
		r.sendSystem(client, eventType, payload)
	}
}

func (r *Room) sendToUser(userID string, eventType EventType, payload any) {
	for _, client := range r.clients {
		if client.UserID == userID {
			r.sendSystem(client, eventType, payload)
		}
	}
}

func (r *Room) onlinePlayerCount() int {
	count := 0
	for _, player := range r.players {
		if player.IsOnline && !player.Abandoned {
			count++
		}
	}
	return count
}

func (r *Room) allGuessersDone() bool {
	activeGuessers := 0
	correct := 0
	for _, player := range r.players {
		if !player.IsOnline || player.Abandoned || player.UserID == r.state.CurrentDrawerID {
			continue
		}
		activeGuessers++
		if player.GuessedWord {
			correct++
		}
	}
	return activeGuessers > 0 && activeGuessers == correct
}

func (r *Room) minPlayers() int {
	if r.state.MinPlayers > 0 {
		return r.state.MinPlayers
	}
	return defaultMinPlayers
}

func (r *Room) maxPlayers() int {
	if r.state.MaxPlayers > 0 {
		return r.state.MaxPlayers
	}
	return defaultMaxPlayers
}

func (r *Room) endsAtUnix() int64 {
	if r.roundEndsAt.IsZero() {
		return 0
	}
	return r.roundEndsAt.Unix()
}

func (r *Room) setTimer(d time.Duration) {
	if d <= 0 {
		return
	}
	r.timer = time.NewTimer(d)
	r.timerC = r.timer.C
}

func (r *Room) pauseForDrawerDisconnect() {
	r.pausedRoundRemaining = time.Until(r.roundEndsAt)
	if r.pausedRoundRemaining < 0 {
		r.pausedRoundRemaining = 0
	}
	r.gameState = GameStateDrawerDisconnected
	r.clearTimer()
	r.scheduleReconnectCheck(drawerReconnectGrace)
}

func (r *Room) resumeDrawerAfterReconnect() {
	if r.pausedRoundRemaining <= 0 {
		r.pausedRoundRemaining = time.Duration(r.state.RoundTime) * time.Second
	}
	r.gameState = GameStateDrawing
	r.roundEndsAt = time.Now().Add(r.pausedRoundRemaining)
	r.clearTimer()
	r.setTimer(r.pausedRoundRemaining)
}

func (r *Room) isActiveGameState() bool {
	switch r.gameState {
	case GameStateCountdown, GameStateWordSelection, GameStateDrawing, GameStateDrawerDisconnected, GameStateRoundEnd, GameStateLeaderboard:
		return true
	default:
		return false
	}
}

func (r *Room) handleReconnectTimer(force bool) bool {
	r.markExpiredDisconnects(force)
	if len(r.clients) == 0 {
		r.state.State = domain.RoomStateFinished
		r.reputation.flush()
		return false
	}
	if r.gameState == GameStateDrawerDisconnected {
		r.endRound()
	}
	return true
}

func (r *Room) markExpiredDisconnects(force bool) {
	now := time.Now().Unix()
	for _, player := range r.players {
		if player == nil || player.IsOnline || player.Abandoned || player.ReconnectDeadline == 0 {
			continue
		}
		if !force && player.ReconnectDeadline > now {
			continue
		}
		delta := abandonRepPenalty
		if player.IsDrawer {
			delta = drawerAbandonPenalty
		}
		r.reputation.add(player.UserID, delta, "abandoned_active_game")
		player.Abandoned = true
	}
}

func (r *Room) scheduleReconnectCheck(d time.Duration) {
	if d <= 0 {
		d = time.Millisecond
	}
	if r.reconnectTimer != nil {
		if !r.reconnectTimer.Stop() {
			select {
			case <-r.reconnectTimer.C:
			default:
			}
		}
	}
	r.reconnectTimer = time.NewTimer(d)
	r.reconnectTimerC = r.reconnectTimer.C
}

func (r *Room) clearTimer() {
	if r.timer != nil {
		if !r.timer.Stop() {
			select {
			case <-r.timer.C:
			default:
			}
		}
	}
	r.timer = nil
	r.timerC = nil
}

func (r *Room) isRankedGame() bool {
	// Public dictionary games are ranked. Private rooms and custom-word games are
	// still playable and should be stored in history, but they must not farm global
	// score/stat/reputation rewards.
	return r.state.Type == domain.RoomTypePublic && len(r.state.CustomWords) == 0
}

func displayName(player *PlayerState) string {
	if player == nil {
		return "player"
	}
	if player.Username != "" {
		return player.Username
	}
	return player.UserID
}
