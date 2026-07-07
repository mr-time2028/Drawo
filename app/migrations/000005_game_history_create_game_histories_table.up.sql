CREATE TABLE game_histories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id VARCHAR(255) NOT NULL,
    room_name VARCHAR(255) NOT NULL,
    language VARCHAR(10) NOT NULL,
    winner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ended_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX idx_game_histories_winner_id ON game_histories(winner_id);
