CREATE TABLE if not exists users (
    id         BIGSERIAL PRIMARY KEY,
    tg_id      BIGINT UNIQUE NOT NULL,
    username   TEXT,
    first_name TEXT NOT NULL,
    photo_url  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE media_type AS ENUM ('movie', 'tv');

CREATE TABLE if not exists titles (
    id             BIGSERIAL PRIMARY KEY,
    tmdb_id        BIGINT NOT NULL,
    media_type     media_type NOT NULL,
    title          TEXT NOT NULL,
    original_title TEXT,
    poster_path    TEXT,
    release_year   INT,
    genres         JSONB,
    overview       TEXT,
    cached_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tmdb_id, media_type)
);

CREATE TYPE friendship_status AS ENUM ('pending', 'accepted');

CREATE TABLE if not exists friendships (
    requester_id BIGINT NOT NULL REFERENCES users(id),
    addressee_id BIGINT NOT NULL REFERENCES users(id),
    status       friendship_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at TIMESTAMPTZ,
    PRIMARY KEY (requester_id, addressee_id),
    CHECK (requester_id <> addressee_id)
);

CREATE INDEX if not exists idx_friendships_addressee ON friendships (addressee_id, status);

CREATE TABLE if not exists criteria (
    id         SMALLSERIAL PRIMARY KEY,
    code       TEXT UNIQUE NOT NULL,
    name       TEXT NOT NULL,
    sort_order SMALLINT NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true
);

INSERT INTO criteria (code, name, sort_order) VALUES
    ('plot',       'Сюжет',          1),
    ('directing',  'Режиссура',      2),
    ('acting',     'Актёрская игра', 3),
    ('music',      'Музыка',         4),
    ('visuals',    'Визуал',         5),
    ('atmosphere', 'Атмосфера',      6)
on conflict do nothing;

CREATE TABLE if not exists ratings (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id),
    title_id   BIGINT NOT NULL REFERENCES titles(id),
    avg_score  NUMERIC(3,1) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, title_id)
);

CREATE INDEX if not exists idx_ratings_user_created ON ratings (user_id, created_at DESC, id DESC);
CREATE INDEX if not exists idx_ratings_title ON ratings (title_id);

CREATE TABLE if not exists rating_scores (
    rating_id    BIGINT NOT NULL REFERENCES ratings(id) ON DELETE CASCADE,
    criterion_id SMALLINT NOT NULL REFERENCES criteria(id),
    score        SMALLINT NOT NULL CHECK (score BETWEEN 1 AND 10),
    PRIMARY KEY (rating_id, criterion_id)
);

CREATE TABLE if not exists comments (
    id         BIGSERIAL PRIMARY KEY,
    title_id   BIGINT NOT NULL REFERENCES titles(id),
    user_id    BIGINT NOT NULL REFERENCES users(id),
    parent_id  BIGINT REFERENCES comments(id),
    body       TEXT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX if not exists idx_comments_title ON comments (title_id, created_at, id);
CREATE INDEX if not exists idx_comments_parent ON comments (parent_id);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_comments_parent;
DROP INDEX IF EXISTS idx_comments_title;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS rating_scores;
DROP INDEX IF EXISTS idx_ratings_title;
DROP INDEX IF EXISTS idx_ratings_user_created;
DROP TABLE IF EXISTS ratings;
DROP TABLE IF EXISTS criteria;
DROP INDEX IF EXISTS idx_friendships_addressee;
DROP TABLE IF EXISTS friendships;
DROP TYPE IF EXISTS friendship_status;
DROP TABLE IF EXISTS titles;
DROP TYPE IF EXISTS media_type;
DROP INDEX IF EXISTS idx_users_username_lower;
DROP INDEX IF EXISTS idx_users_uuid;
DROP TABLE IF EXISTS users;
