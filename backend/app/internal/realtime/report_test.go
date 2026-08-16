package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"drawo/internal/core/domain"
)

type fakeReportRepo struct {
	reports []*domain.Report
}

func (f *fakeReportRepo) InsertReport(ctx context.Context, report *domain.Report) error {
	copy := *report
	f.reports = append(f.reports, &copy)
	return nil
}
func (f *fakeReportRepo) GetReportByID(ctx context.Context, id string) (*domain.Report, error) {
	return nil, nil
}
func (f *fakeReportRepo) UpdateReport(ctx context.Context, report *domain.Report) error { return nil }
func (f *fakeReportRepo) ListReports(ctx context.Context, paging domain.Paging) (*domain.PageOf[domain.Report], error) {
	return nil, nil
}
func (f *fakeReportRepo) ListReportsByStatus(ctx context.Context, status domain.ReportStatus, paging domain.Paging) (*domain.PageOf[domain.Report], error) {
	return nil, nil
}
func (f *fakeReportRepo) CountRoundReports(ctx context.Context, roomID string, round int, reportedID string, reason domain.ReportReason) (int64, error) {
	return int64(len(f.reports)), nil
}

func TestRoomReportStoresEvidenceAndRejectsInvalidReports(t *testing.T) {
	reportRepo := &fakeReportRepo{}
	room := NewRoom(&domain.Room{ID: "report-room", State: domain.RoomStatePlaying, Language: "en"}, func(string, string) {}, nil, nil, nil, nil, reportRepo)
	reporter := &Client{ID: "c1", UserID: "reporter", Send: make(chan []byte, 20), Done: make(chan struct{})}
	reported := &Client{ID: "c2", UserID: "reported", Send: make(chan []byte, 20), Done: make(chan struct{})}
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: reporter, Timestamp: time.Now()})
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: reported, Timestamp: time.Now()})
	drainClient(reporter)
	drainClient(reported)
	room.canvasOps = append(room.canvasOps, DrawOperation{Op: DrawOpStroke, UserID: "reported", ID: "op1"})
	room.recordChat(ChatPayload{UserID: "reported", Text: "bad behavior"})
	room.state.CurrentRound = 2
	room.gameState = GameStateDrawing
	room.currentWord = &WordCandidate{GroupID: "apple", Text: "apple", Points: 1}

	room.handleReportEvent(reporter, ReportPayload{ReportedUserID: "reported", Reason: domain.ReportReasonCheating, Details: "wrote answer"})
	assert.Len(t, reportRepo.reports, 1)
	assert.Contains(t, reportRepo.reports[0].Evidence, "canvas_ops")
	assert.Contains(t, reportRepo.reports[0].Evidence, "chat_history")
	assert.Contains(t, reportRepo.reports[0].Evidence, "apple")
	drainClient(reporter)

	room.handleReportEvent(reporter, ReportPayload{ReportedUserID: "reported", Reason: domain.ReportReasonCheating})
	msg := nextEnvelope(t, reporter)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "duplicate_report")
	drainClient(reporter)

	room.handleReportEvent(reporter, ReportPayload{ReportedUserID: "reporter", Reason: domain.ReportReasonCheating})
	msg = nextEnvelope(t, reporter)
	assert.Equal(t, EventError, msg.Type)
	assert.Contains(t, string(msg.Payload), "yourself")
}

func TestMultipleReportsRecordPenaltyEvent(t *testing.T) {
	reportRepo := &fakeReportRepo{}
	profiles := &fakeProfileRepo{profiles: map[string]*domain.Profile{"reported": {UserID: "reported", ReputationScore: 10000}}}
	room := NewRoom(&domain.Room{ID: "report-room", State: domain.RoomStatePlaying, Language: "en", CurrentRound: 1}, func(string, string) {}, nil, nil, profiles, nil, reportRepo)
	reported := &Client{ID: "reported-c", UserID: "reported", Send: make(chan []byte, 20), Done: make(chan struct{})}
	reporter1 := &Client{ID: "r1-c", UserID: "r1", Send: make(chan []byte, 20), Done: make(chan struct{})}
	reporter2 := &Client{ID: "r2-c", UserID: "r2", Send: make(chan []byte, 20), Done: make(chan struct{})}
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: reported, Timestamp: time.Now()})
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: reporter1, Timestamp: time.Now()})
	room.handleEvent(&RoomEvent{Type: EventJoin, Client: reporter2, Timestamp: time.Now()})
	drainClient(reporter1)
	drainClient(reporter2)

	room.handleReportEvent(reporter1, ReportPayload{ReportedUserID: "reported", Reason: domain.ReportReasonGriefing})
	room.handleReportEvent(reporter2, ReportPayload{ReportedUserID: "reported", Reason: domain.ReportReasonGriefing})
	assert.Len(t, reportRepo.reports, 2)
	assert.NotEmpty(t, room.reputation.events)
}

func TestRoomReportSkipsPersistenceForGuests(t *testing.T) {
	reportRepo := &fakeReportRepo{}
	room := NewRoom(&domain.Room{ID: "guest-report-room", State: domain.RoomStatePlaying, Language: "en"}, func(string, string) {}, nil, nil, nil, nil, reportRepo)
	guest := &Client{ID: "c1", UserID: "guest:11111111-1111-1111-1111-111111111111", Send: make(chan []byte, 20), Done: make(chan struct{})}
	registered := &Client{ID: "c2", UserID: "22222222-2222-2222-2222-222222222222", Send: make(chan []byte, 20), Done: make(chan struct{})}
	room.clients[guest.ID] = guest
	room.clients[registered.ID] = registered
	room.players[guest.UserID] = &PlayerState{UserID: guest.UserID, IsOnline: true}
	room.players[registered.UserID] = &PlayerState{UserID: registered.UserID, IsOnline: true}

	// Guest reporting a registered user: accepted in-room, NOT persisted.
	room.handleReportEvent(guest, ReportPayload{Event: "report", ReportedUserID: registered.UserID, Reason: domain.ReportReasonAbusiveChat})
	assert.Empty(t, reportRepo.reports, "guest reporter must not hit the DB (UUID FK would fail)")

	// Registered user reporting a guest: same — no DB row.
	room.handleReportEvent(registered, ReportPayload{Event: "report", ReportedUserID: guest.UserID, Reason: domain.ReportReasonCheating})
	assert.Empty(t, reportRepo.reports, "guest reported must not hit the DB (UUID FK would fail)")
}

func TestReputationLedgerIgnoresGuests(t *testing.T) {
	ledger := newReputationLedger(nil, nil, "room-1")
	ledger.add("guest:33333333-3333-3333-3333-333333333333", -50, "abandoned_active_game")
	assert.Empty(t, ledger.events, "guest reputation events must never be recorded")

	ledger.addPositiveCapped("guest:33333333-3333-3333-3333-333333333333", 20, "completed_game")
	assert.Empty(t, ledger.events)

	ledger.add("44444444-4444-4444-4444-444444444444", -50, "abandoned_active_game")
	assert.Len(t, ledger.events, 1, "registered users still accrue reputation")
}
