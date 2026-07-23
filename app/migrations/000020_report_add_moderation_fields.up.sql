ALTER TABLE reports
    ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS evidence JSONB,
    ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS resolution_note TEXT,
    ADD COLUMN IF NOT EXISTS round INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_reports_status_created_at
    ON reports(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reports_room_round_reported_reason
    ON reports(room_id, round, reported_id, reason);
