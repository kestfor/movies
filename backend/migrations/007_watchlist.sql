CREATE TABLE watchlist_items (
    user_id    BIGINT NOT NULL REFERENCES users(id),
    title_id   BIGINT NOT NULL REFERENCES titles(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, title_id)
);

CREATE INDEX idx_watchlist_user_created
    ON watchlist_items (user_id, created_at DESC, title_id DESC);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_watchlist_user_created;
DROP TABLE IF EXISTS watchlist_items;
