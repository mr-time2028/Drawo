CREATE TABLE profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    avatar_url TEXT,
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(20),
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    locale VARCHAR(10) DEFAULT 'en',
    theme VARCHAR(10) DEFAULT 'light',
    background_sound BOOLEAN DEFAULT TRUE,
    tool_sound BOOLEAN DEFAULT TRUE,
    word_score BIGINT DEFAULT 0,
    reputation_score BIGINT DEFAULT 10000,
    games_played BIGINT DEFAULT 0,
    mvps BIGINT DEFAULT 0,
    rank VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_profiles_email ON profiles(email);
