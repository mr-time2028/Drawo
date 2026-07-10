CREATE TABLE words (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id UUID NOT NULL,
    category_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    text VARCHAR(100) NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'fa',
    points INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(text, language)
);
CREATE INDEX idx_words_group_id ON words(group_id);
CREATE INDEX idx_words_category_id ON words(category_id);
