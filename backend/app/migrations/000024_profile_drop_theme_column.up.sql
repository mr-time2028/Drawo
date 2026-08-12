-- Theme is owned by the frontend (localStorage) and is no longer persisted on
-- the profile. Drop the column from the database.
ALTER TABLE profiles DROP COLUMN IF EXISTS theme;
