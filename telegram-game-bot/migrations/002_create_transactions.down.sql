-- 002_create_transactions.down.sql
-- Drop transactions table and related indexes

DROP INDEX IF EXISTS idx_transactions_type_time;
DROP INDEX IF EXISTS idx_transactions_user_time;
DROP TABLE IF EXISTS transactions;
