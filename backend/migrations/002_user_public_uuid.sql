CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS uuid UUID;

UPDATE users
SET uuid = gen_random_uuid()
WHERE uuid IS NULL;

ALTER TABLE users
    ALTER COLUMN uuid SET DEFAULT gen_random_uuid(),
    ALTER COLUMN uuid SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_uuid ON users (uuid);
CREATE INDEX IF NOT EXISTS idx_users_username_lower ON users (lower(username) text_pattern_ops);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_users_username_lower;
DROP INDEX IF EXISTS idx_users_uuid;

ALTER TABLE users
    DROP COLUMN IF EXISTS uuid;
