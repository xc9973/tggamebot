-- 003_create_daily_stats_view.up.sql
-- Create view for daily game statistics (used for daily rankings)

CREATE OR REPLACE VIEW daily_game_stats AS
SELECT 
    user_id,
    SUM(amount) as net_profit,
    DATE(created_at) as game_date
FROM transactions
WHERE type IN ('dice', 'slot', 'sicbo_win', 'sicbo_bet')
GROUP BY user_id, DATE(created_at);
