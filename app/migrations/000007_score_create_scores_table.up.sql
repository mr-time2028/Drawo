CREATE TABLE scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_history_id UUID REFERENCES game_histories(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    points BIGINT NOT NULL,
    rank INT NOT NULL
);
CREATE INDEX idx_scores_game_history_id ON scores(game_history_id);
