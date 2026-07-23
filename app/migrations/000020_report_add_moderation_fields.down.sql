DROP INDEX IF EXISTS idx_reports_room_round_reported_reason;
DROP INDEX IF EXISTS idx_reports_status_created_at;

ALTER TABLE reports
    DROP COLUMN IF EXISTS round,
    DROP COLUMN IF EXISTS resolution_note,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS evidence,
    DROP COLUMN IF EXISTS status;
