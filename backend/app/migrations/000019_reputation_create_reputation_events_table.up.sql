CREATE TABLE IF NOT EXISTS reputation_events (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delta BIGINT NOT NULL,
    reason VARCHAR(100) NOT NULL,
    room_id VARCHAR(100),
    round INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reputation_events_user_id_created_at
    ON reputation_events(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reputation_events_room_id
    ON reputation_events(room_id);
