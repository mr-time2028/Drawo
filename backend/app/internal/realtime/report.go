package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"drawo/internal/core/domain"
	"drawo/pkg/errors"
)

const (
	maxReportDetailsLength  = 500
	reportPenaltyTwoUsers   = int64(-50)
	reportPenaltyThreeUsers = int64(-150)
	maxEvidenceCanvasOps    = 300
)

type ReportPayload struct {
	Event          string              `json:"event"`
	ReportedUserID string              `json:"reported_user_id"`
	Reason         domain.ReportReason `json:"reason"`
	Details        string              `json:"details,omitempty"`
}

type ReportEvidence struct {
	RoomID      string          `json:"room_id"`
	Round       int             `json:"round"`
	GameState   string          `json:"game_state"`
	ReporterID  string          `json:"reporter_id"`
	ReportedID  string          `json:"reported_id"`
	DrawerID    string          `json:"drawer_id,omitempty"`
	WordGroupID string          `json:"word_group_id,omitempty"`
	WordText    string          `json:"word_text,omitempty"`
	CanvasOps   []DrawOperation `json:"canvas_ops"`
	ChatHistory []ChatPayload   `json:"chat_history"`
	CreatedAt   int64           `json:"created_at"`
}

func (r *Room) handleReportEvent(client *Client, payload ReportPayload) {
	if r.reportRepo == nil {
		r.sendError(client, errors.WSErrReportsUnavailable, "reporting is not configured")
		return
	}
	payload.ReportedUserID = strings.TrimSpace(payload.ReportedUserID)
	payload.Details = strings.TrimSpace(payload.Details)
	if len(payload.Details) > maxReportDetailsLength {
		r.sendError(client, errors.WSErrInvalidReport, "report details are too long")
		return
	}
	if payload.ReportedUserID == "" || r.players[payload.ReportedUserID] == nil {
		r.sendError(client, errors.WSErrInvalidReport, "reported player is not in this room")
		return
	}
	if payload.ReportedUserID == client.UserID {
		r.sendError(client, errors.WSErrInvalidReport, "you cannot report yourself")
		return
	}
	if !validReportReason(payload.Reason) {
		r.sendError(client, errors.WSErrInvalidReport, "invalid report reason")
		return
	}
	key := fmt.Sprintf("%d:%s:%s:%s", r.state.CurrentRound, client.UserID, payload.ReportedUserID, payload.Reason)
	if _, exists := r.reportKeys[key]; exists {
		r.sendError(client, errors.WSErrDuplicateReport, "you already reported this player for this reason")
		return
	}
	r.reportKeys[key] = struct{}{}

	evidenceJSON, _ := json.Marshal(r.buildReportEvidence(client.UserID, payload.ReportedUserID))
	report := &domain.Report{
		ID:         uuid.New().String(),
		ReporterID: client.UserID,
		ReportedID: payload.ReportedUserID,
		RoomID:     r.state.ID,
		Round:      r.state.CurrentRound,
		Reason:     payload.Reason,
		Details:    payload.Details,
		Status:     domain.ReportStatusPending,
		Evidence:   string(evidenceJSON),
		CreatedAt:  time.Now(),
	}
	if err := r.reportRepo.InsertReport(context.Background(), report); err != nil {
		r.sendError(client, errors.WSErrReportFailed, "could not store report")
		return
	}
	r.applyAggregatedReportPenalty(payload.ReportedUserID, payload.Reason, client.UserID)
	r.sendSystem(client, EventGame, ReportSubmittedPayload{Event: "report_submitted", ReportID: report.ID})
}

func (r *Room) buildReportEvidence(reporterID, reportedID string) ReportEvidence {
	canvasOps := r.canvasOps
	if len(canvasOps) > maxEvidenceCanvasOps {
		canvasOps = canvasOps[len(canvasOps)-maxEvidenceCanvasOps:]
	}
	wordText := ""
	wordGroupID := ""
	if r.currentWord != nil {
		wordText = r.currentWord.Text
		wordGroupID = r.currentWord.GroupID
	}
	return ReportEvidence{
		RoomID:      r.state.ID,
		Round:       r.state.CurrentRound,
		GameState:   r.gameState,
		ReporterID:  reporterID,
		ReportedID:  reportedID,
		DrawerID:    r.state.CurrentDrawerID,
		WordText:    wordText,
		WordGroupID: wordGroupID,
		CanvasOps:   append([]DrawOperation(nil), canvasOps...),
		ChatHistory: append([]ChatPayload(nil), r.chatHistory...),
		CreatedAt:   time.Now().Unix(),
	}
}

func (r *Room) applyAggregatedReportPenalty(reportedID string, reason domain.ReportReason, reporterID string) {
	key := fmt.Sprintf("%d:%s:%s", r.state.CurrentRound, reportedID, reason)
	reporters := r.roundReports[key]
	if reporters == nil {
		reporters = make(map[string]struct{})
		r.roundReports[key] = reporters
	}
	reporters[reporterID] = struct{}{}
	switch len(reporters) {
	case 2:
		r.reputation.add(reportedID, reportPenaltyTwoUsers, "multiple_reports")
	case 3:
		r.reputation.add(reportedID, reportPenaltyThreeUsers, "multiple_reports")
	}
}

func validReportReason(reason domain.ReportReason) bool {
	switch reason {
	case domain.ReportReasonInappropriateDrawing, domain.ReportReasonAbusiveChat, domain.ReportReasonCheating, domain.ReportReasonGriefing:
		return true
	default:
		return false
	}
}
