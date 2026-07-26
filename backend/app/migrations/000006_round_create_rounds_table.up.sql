CREATE TABLE rounds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_history_id UUID REFERENCES game_histories(id) ON DELETE CASCADE,
    round_number INT NOT NULL,
    drawer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    word VARCHAR(255) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ended_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX idx_rounds_game_history_id ON rounds(game_history_id);
