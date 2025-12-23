-- 002_create_transactions.up.sql
-- Create transactions table for recording all balance changes

CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for user transaction history queries
CREATE INDEX IF NOT EXISTS idx_transactions_user_time ON transactions(user_id, created_at DESC);

-- Index for transaction type queries (used for daily stats)
CREATE INDEX IF NOT EXISTS idx_transactions_type_time ON transactions(type, created_at DESC);
