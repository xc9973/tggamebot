-- Shop System Tables
-- Requirements: 3.7, 4.5, 5.5, 6.6, 7.7, 8.5, 9.6 - Use count based items

-- 用户道具表（存储道具剩余使用次数）
CREATE TABLE IF NOT EXISTS user_items (
    user_id BIGINT NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    use_count INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, item_type)
);

-- 手铐锁定表（存储被锁定的用户）
CREATE TABLE IF NOT EXISTS handcuff_locks (
    target_id BIGINT PRIMARY KEY,
    locked_by BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_handcuff_locks_expires ON handcuff_locks(expires_at);

-- 每日购买记录表
-- Requirements: 12.1, 12.2 - Daily purchase tracking
CREATE TABLE IF NOT EXISTS daily_purchases (
    user_id BIGINT NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    purchase_count INT NOT NULL DEFAULT 0,
    purchase_date DATE NOT NULL DEFAULT CURRENT_DATE,
    PRIMARY KEY (user_id, item_type, purchase_date)
);
CREATE INDEX IF NOT EXISTS idx_daily_purchases_date ON daily_purchases(purchase_date);
