CREATE TABLE global_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial defaults
INSERT INTO global_settings (key, value) VALUES 
('bad_word_penalty', '100'),
('suggested_words_count', '3'),
('round_time', '60'),
('max_rounds', '10');
