package errors

// WSErrorCode is the stable machine-readable error code used inside WebSocket
// error frames. Keep these values stable because frontend code switches on them.
type WSErrorCode string

const (
	WSErrAuthFailed           WSErrorCode = "auth_failed"
	WSErrSendFailed           WSErrorCode = "send_failed"
	WSErrJoinFailed           WSErrorCode = "join_failed"
	WSErrRateLimited          WSErrorCode = "rate_limited"
	WSErrSessionRevoked       WSErrorCode = "session_revoked"
	WSErrAuthExpired          WSErrorCode = "auth_expired"
	WSErrBadMessage           WSErrorCode = "bad_message"
	WSErrRoomError            WSErrorCode = "room_error"
	WSErrDrawNotAllowed       WSErrorCode = "draw_not_allowed"
	WSErrDrawForbidden        WSErrorCode = "draw_forbidden"
	WSErrInvalidDraw          WSErrorCode = "invalid_draw"
	WSErrDrawRateLimited      WSErrorCode = "draw_rate_limited"
	WSErrReconnectExpired     WSErrorCode = "reconnect_expired"
	WSErrDrawerChatBlocked    WSErrorCode = "drawer_chat_blocked"
	WSErrInvalidChat          WSErrorCode = "invalid_chat"
	WSErrChatRateLimited      WSErrorCode = "chat_rate_limited"
	WSErrBadWord              WSErrorCode = "bad_word"
	WSErrAlreadyGuessed       WSErrorCode = "already_guessed"
	WSErrInvalidGameEvent     WSErrorCode = "invalid_game_event"
	WSErrWordChoiceForbidden  WSErrorCode = "word_choice_forbidden"
	WSErrUnsupportedGameEvent WSErrorCode = "unsupported_game_event"
	WSErrReportsUnavailable   WSErrorCode = "reports_unavailable"
	WSErrInvalidReport        WSErrorCode = "invalid_report"
	WSErrDuplicateReport      WSErrorCode = "duplicate_report"
	WSErrReportFailed         WSErrorCode = "report_failed"
)

// String returns the JSON-safe representation of the WebSocket error code.
func (c WSErrorCode) String() string { return string(c) }

// WSDefaultMessage returns the default frontend-safe message for a WebSocket
// error code. Callers can override it when they need a more specific validation
// message, but should keep the code constant.
func WSDefaultMessage(code WSErrorCode) string {
	switch code {
	case WSErrAuthFailed:
		return "authentication failed"
	case WSErrSendFailed:
		return "could not send websocket message"
	case WSErrJoinFailed:
		return "could not join room"
	case WSErrRateLimited:
		return "too many websocket messages"
	case WSErrSessionRevoked:
		return "session no longer active"
	case WSErrAuthExpired:
		return "websocket access token expired; reconnect with a fresh access token"
	case WSErrBadMessage:
		return "invalid websocket message"
	case WSErrRoomError:
		return "room error"
	case WSErrDrawNotAllowed:
		return "drawing is not active"
	case WSErrDrawForbidden:
		return "only the current drawer can draw"
	case WSErrInvalidDraw:
		return "invalid drawing operation"
	case WSErrDrawRateLimited:
		return "too many drawing operations"
	case WSErrReconnectExpired:
		return "reconnect window expired"
	case WSErrDrawerChatBlocked:
		return "drawer cannot chat during drawing"
	case WSErrInvalidChat:
		return "chat text is required"
	case WSErrChatRateLimited:
		return "too many chat messages"
	case WSErrBadWord:
		return "message contains prohibited words"
	case WSErrAlreadyGuessed:
		return "you already guessed this word"
	case WSErrInvalidGameEvent:
		return "invalid game event"
	case WSErrWordChoiceForbidden:
		return "only the current drawer can choose a word"
	case WSErrUnsupportedGameEvent:
		return "unsupported game event"
	case WSErrReportsUnavailable:
		return "reporting is not configured"
	case WSErrInvalidReport:
		return "invalid report"
	case WSErrDuplicateReport:
		return "you already reported this player for this reason"
	case WSErrReportFailed:
		return "could not store report"
	default:
		return "websocket error"
	}
}
