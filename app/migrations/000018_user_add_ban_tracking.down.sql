-- 000018_user_add_ban_tracking.down.sql
ALTER TABLE users DROP COLUMN IF EXISTS ban_count;
ALTER TABLE users DROP COLUMN IF EXISTS banned_at;
