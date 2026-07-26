DROP INDEX IF EXISTS idx_profiles_email;
DROP INDEX IF EXISTS idx_profiles_email_unique_not_empty;

UPDATE profiles SET email = NULL WHERE email = '';

ALTER TABLE profiles ADD CONSTRAINT profiles_email_key UNIQUE (email);
CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles(email);
