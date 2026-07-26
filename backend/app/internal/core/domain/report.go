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

// ReportStatus represents review state for a moderation report.
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusConfirmed ReportStatus = "confirmed"
	ReportStatusRejected  ReportStatus = "rejected"
)

// Report records user-submitted moderation alerts with enough context for admin
// review. Evidence is JSON encoded by the realtime room and kept as a string so
// both SQLite tests and PostgreSQL deployments can persist it safely.
type Report struct {
	ID             string
	ReporterID     string
	ReportedID     string
	RoomID         string
	Round          int
	Reason         ReportReason
	Details        string
	Status         ReportStatus
	Evidence       string
	ReviewedBy     string
	ReviewedAt     *time.Time
	ResolutionNote string
	CreatedAt      time.Time
}
