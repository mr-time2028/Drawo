-- 000018_user_add_ban_tracking.up.sql
ALTER TABLE users ADD COLUMN ban_count INT DEFAULT 0;
ALTER TABLE users ADD COLUMN banned_at TIMESTAMP WITH TIME ZONE;
