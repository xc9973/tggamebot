-- 001_create_users.up.sql
-- Create users table for storing Telegram user accounts

CREATE TABLE IF NOT EXISTS users (
    telegram_id BIGINT PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 1000,
    last_daily_claim BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for ranking queries (top users by balance)
CREATE INDEX IF NOT EXISTS idx_users_balance ON users(balance DESC);
