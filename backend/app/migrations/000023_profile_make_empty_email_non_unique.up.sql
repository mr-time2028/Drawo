UPDATE profiles SET email = NULL WHERE email = '';

ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_email_key;
DROP INDEX IF EXISTS idx_profiles_email;

CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_email_unique_not_empty
    ON profiles(email)
    WHERE email IS NOT NULL AND email <> '';

CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles(email);
