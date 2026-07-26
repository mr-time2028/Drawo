ALTER TABLE player_statistics
    DROP COLUMN IF EXISTS reports_confirmed,
    DROP COLUMN IF EXISTS reports_received,
    DROP COLUMN IF EXISTS games_abandoned,
    DROP COLUMN IF EXISTS successful_drawings,
    DROP COLUMN IF EXISTS mvps,
    DROP COLUMN IF EXISTS best_game_score,
    DROP COLUMN IF EXISTS total_score,
    DROP COLUMN IF EXISTS private_games,
    DROP COLUMN IF EXISTS ranked_games;
