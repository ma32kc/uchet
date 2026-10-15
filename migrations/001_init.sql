-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    telegram_id  BIGINT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS expenses (
    id         UUID PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (telegram_id),
    amount     NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    category   TEXT NOT NULL CHECK (
        category IN (
            'food',
            'transport',
            'housing',
            'entertainment',
            'health',
            'shopping',
            'other'
        )
    ),
    spent_at   DATE NOT NULL,
    comment    TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_expenses_user_spent_at ON expenses (user_id, spent_at DESC);
CREATE INDEX IF NOT EXISTS idx_expenses_spent_at ON expenses (spent_at);
CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses (category);

-- +goose Down
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS users;
