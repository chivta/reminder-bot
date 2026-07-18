CREATE TABLE users (
    id BIGINT PRIMARY KEY
);

CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    message_id INTEGER NOT NULL,
    saved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ,
    UNIQUE (chat_id, message_id)
);

CREATE INDEX idx_messages_pending ON messages (user_id) WHERE acknowledged_at IS NULL;
---- create above / drop below ----
DROP TABLE messages;
DROP TABLE users;
