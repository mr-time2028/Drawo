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
	room := NewRoom(&domain.Room{ID: "report-room", State: domain.RoomStatePlaying, Language: "en"}, func(string, string) {}, nil, nil, nil, reportRepo)
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
	room := NewRoom(&domain.Room{ID: "report-room", State: domain.RoomStatePlaying, Language: "en", CurrentRound: 1}, func(string, string) {}, nil, profiles, nil, reportRepo)
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
