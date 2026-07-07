package domain

import "time"

// ReportReason represents the category of a moderation report.
type ReportReason string

const (
	ReportReasonInappropriateDrawing ReportReason = "inappropriate_drawing"
	ReportReasonAbusiveChat          ReportReason = "abusive_chat"
	ReportReasonCheating             ReportReason = "cheating"
	ReportReasonGriefing             ReportReason = "griefing"
)

// Report records user-submitted moderation alerts.
type Report struct {
	ID         string
	ReporterID string
	ReportedID string
	RoomID     string
	Reason     ReportReason
	Details    string
	CreatedAt  time.Time
}
