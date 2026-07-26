ALTER TABLE users
    ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'active';

UPDATE users
SET status = CASE
    WHEN is_active = FALSE THEN 'banned'
    ELSE 'active'
END
WHERE status IS NULL OR status = '';

CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
