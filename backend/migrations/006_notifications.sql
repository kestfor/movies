CREATE TYPE activity_event_kind AS ENUM (
    'rating_created',
    'comment_created'
);

CREATE TABLE activity_events (
    id         BIGSERIAL PRIMARY KEY,
    actor_id   BIGINT NOT NULL REFERENCES users(id),
    title_id   BIGINT NOT NULL REFERENCES titles(id),
    kind       activity_event_kind NOT NULL,
    rating_id  BIGINT REFERENCES ratings(id) ON DELETE SET NULL,
    comment_id BIGINT REFERENCES comments(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_activity_events_created ON activity_events (created_at DESC, id DESC);

CREATE TABLE notification_deliveries (
    user_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES activity_events(id) ON DELETE CASCADE,
    read_at  TIMESTAMPTZ,
    PRIMARY KEY (user_id, event_id)
);

CREATE INDEX idx_notification_deliveries_user_event
    ON notification_deliveries (user_id, event_id DESC);

CREATE INDEX idx_notification_deliveries_unread
    ON notification_deliveries (user_id, read_at, event_id DESC);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_notification_deliveries_unread;
DROP INDEX IF EXISTS idx_notification_deliveries_user_event;
DROP TABLE IF EXISTS notification_deliveries;
DROP INDEX IF EXISTS idx_activity_events_created;
DROP TABLE IF EXISTS activity_events;
DROP TYPE IF EXISTS activity_event_kind;
