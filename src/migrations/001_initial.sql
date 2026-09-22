CREATE TABLE users (
 id BIGSERIAL PRIMARY KEY,
 username TEXT NOT NULL UNIQUE CHECK (char_length(username) BETWEEN 3 AND 32),
 password_hash TEXT NOT NULL,
 xp INTEGER NOT NULL DEFAULT 0 CHECK (xp >= 0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE sessions (
 id BIGSERIAL PRIMARY KEY,
 user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 token_hash TEXT NOT NULL UNIQUE,
 expires_at TIMESTAMPTZ NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX sessions_expiry ON sessions(expires_at);
CREATE TABLE quests (
 id BIGSERIAL PRIMARY KEY,
 user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
 description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 4000),
 difficulty SMALLINT NOT NULL CHECK (difficulty BETWEEN 1 AND 5),
 estimated_minutes INTEGER NOT NULL CHECK (estimated_minutes BETWEEN 1 AND 10080),
 reward_xp INTEGER NOT NULL CHECK (reward_xp > 0),
 status TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available','active','completed','abandoned')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 completed_at TIMESTAMPTZ,
 CHECK ((status = 'completed') = (completed_at IS NOT NULL))
);
CREATE INDEX quests_user_status ON quests(user_id, status, created_at DESC);
