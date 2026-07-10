CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    language VARCHAR(10) NOT NULL DEFAULT 'fa',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, language)
);
CREATE INDEX idx_categories_group_id ON categories(group_id);
