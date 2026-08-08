CREATE TYPE achievement_award_source AS ENUM ('live', 'backfill', 'reconcile');

CREATE TABLE achievement_catalog_state (
    achievement_code       TEXT PRIMARY KEY,
    definition_fingerprint TEXT NOT NULL,
    introduced_at          TIMESTAMPTZ NOT NULL
);

CREATE TABLE user_achievement_metrics (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_code TEXT NOT NULL,
    value       BIGINT NOT NULL CHECK (value >= 0),
    reached_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, metric_code)
);

CREATE TABLE user_achievements (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_code TEXT NOT NULL REFERENCES achievement_catalog_state(achievement_code),
    xp               INT NOT NULL CHECK (xp > 0),
    earned_at        TIMESTAMPTZ NOT NULL,
    awarded_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    source           achievement_award_source NOT NULL,
    seen_at          TIMESTAMPTZ,
    UNIQUE (user_id, achievement_code)
);

CREATE INDEX idx_user_achievements_user_earned
    ON user_achievements (user_id, earned_at DESC, id DESC);

CREATE INDEX idx_user_achievements_unseen
    ON user_achievements (user_id, awarded_at, id)
    WHERE seen_at IS NULL;

CREATE TABLE achievement_backfill_runs (
    id                  BIGSERIAL PRIMARY KEY,
    catalog_fingerprint TEXT NOT NULL,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    last_user_id        BIGINT,
    status              TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed')),
    processed_users     BIGINT NOT NULL DEFAULT 0,
    awarded_count       BIGINT NOT NULL DEFAULT 0,
    error               TEXT
);

CREATE INDEX idx_achievement_backfill_runs_status
    ON achievement_backfill_runs (catalog_fingerprint, status, started_at DESC);

ALTER TYPE activity_event_kind RENAME TO activity_event_kind_old;
CREATE TYPE activity_event_kind AS ENUM (
    'rating_created',
    'comment_created',
    'achievement_unlocked'
);
ALTER TABLE activity_events
    ALTER COLUMN kind TYPE activity_event_kind
    USING kind::text::activity_event_kind;
DROP TYPE activity_event_kind_old;

ALTER TABLE activity_events
    ALTER COLUMN title_id DROP NOT NULL,
    ADD COLUMN achievement_id UUID REFERENCES user_achievements(id);

ALTER TABLE activity_events
    ADD CONSTRAINT activity_events_payload_check CHECK (
        (
            kind = 'achievement_unlocked'
            AND achievement_id IS NOT NULL
            AND title_id IS NULL
        )
        OR
        (
            kind IN ('rating_created', 'comment_created')
            AND title_id IS NOT NULL
            AND achievement_id IS NULL
        )
    );

---- create above / drop below ----

DELETE FROM activity_events WHERE kind = 'achievement_unlocked';

ALTER TABLE activity_events
    DROP CONSTRAINT IF EXISTS activity_events_payload_check,
    DROP COLUMN IF EXISTS achievement_id,
    ALTER COLUMN title_id SET NOT NULL;

ALTER TABLE activity_events ALTER COLUMN kind TYPE TEXT;
DROP TYPE activity_event_kind;
CREATE TYPE activity_event_kind AS ENUM ('rating_created', 'comment_created');
ALTER TABLE activity_events
    ALTER COLUMN kind TYPE activity_event_kind
    USING kind::activity_event_kind;

DROP INDEX IF EXISTS idx_achievement_backfill_runs_status;
DROP TABLE IF EXISTS achievement_backfill_runs;
DROP INDEX IF EXISTS idx_user_achievements_unseen;
DROP INDEX IF EXISTS idx_user_achievements_user_earned;
DROP TABLE IF EXISTS user_achievements;
DROP TABLE IF EXISTS user_achievement_metrics;
DROP TABLE IF EXISTS achievement_catalog_state;
DROP TYPE IF EXISTS achievement_award_source;
