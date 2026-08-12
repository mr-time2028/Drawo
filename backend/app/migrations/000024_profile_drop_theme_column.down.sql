-- Add back the theme column when rolling back. The column had a default of
-- 'light' in the original schema.
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS theme VARCHAR(10) DEFAULT 'light';
