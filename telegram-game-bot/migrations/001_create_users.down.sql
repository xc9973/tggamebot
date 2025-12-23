-- 001_create_users.down.sql
-- Drop users table and related indexes

DROP INDEX IF EXISTS idx_users_balance;
DROP TABLE IF EXISTS users;
