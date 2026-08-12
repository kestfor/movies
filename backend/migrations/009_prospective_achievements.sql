CREATE TYPE achievement_fact_kind AS ENUM (
    'rating_created',
    'rating_updated',
    'comment_created',
    'watchlist_added'
);

CREATE TABLE achievement_facts (
    id                   BIGSERIAL PRIMARY KEY,
    kind                 achievement_fact_kind NOT NULL,
    actor_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title_id             BIGINT NOT NULL REFERENCES titles(id) ON DELETE CASCADE,
    entity_id            BIGINT,
    parent_entity_id     BIGINT,
    avg_tenths           SMALLINT,
    previous_avg_tenths  SMALLINT,
    scores               JSONB NOT NULL DEFAULT '{}'::jsonb,
    previous_scores      JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (avg_tenths IS NULL OR avg_tenths BETWEEN 10 AND 100),
    CHECK (previous_avg_tenths IS NULL OR previous_avg_tenths BETWEEN 10 AND 100),
    CHECK (jsonb_typeof(scores) = 'object'),
    CHECK (jsonb_typeof(previous_scores) = 'object')
);

CREATE INDEX idx_achievement_facts_actor_time
    ON achievement_facts (actor_id, occurred_at, id);

CREATE INDEX idx_achievement_facts_title_time
    ON achievement_facts (title_id, occurred_at, id);

CREATE UNIQUE INDEX idx_achievement_facts_comment
    ON achievement_facts (entity_id)
    WHERE kind = 'comment_created';

---- create above / drop below ----

DROP INDEX IF EXISTS idx_achievement_facts_comment;
DROP INDEX IF EXISTS idx_achievement_facts_title_time;
DROP INDEX IF EXISTS idx_achievement_facts_actor_time;
DROP TABLE IF EXISTS achievement_facts;
DROP TYPE IF EXISTS achievement_fact_kind;
