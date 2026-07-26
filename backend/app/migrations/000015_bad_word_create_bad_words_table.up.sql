CREATE TABLE bad_words (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    text VARCHAR(100) NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'fa',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(text, language)
);
