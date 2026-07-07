CREATE TABLE player_statistics (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_games BIGINT DEFAULT 0,
    total_wins BIGINT DEFAULT 0,
    total_drawings BIGINT DEFAULT 0,
    correct_guesses BIGINT DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
