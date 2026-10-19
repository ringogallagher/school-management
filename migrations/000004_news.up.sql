-- School news shown on the home page
CREATE TABLE news (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    body TEXT NOT NULL,
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    author_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- The feed lists pinned items first, then newest first
CREATE INDEX idx_news_feed ON news(pinned DESC, created_at DESC);
