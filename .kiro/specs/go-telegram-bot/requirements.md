# Requirements Document

## Introduction

使用 Go 语言完全重写现有的 Python Telegram 游戏机器人，包括所有游戏逻辑、用户管理和 Telegram Bot 功能。数据库从 SQLite 迁移到 PostgreSQL。

## Glossary

- **Bot**: Go 语言编写的 Telegram 游戏机器人
- **PostgreSQL_DB**: PostgreSQL 数据库实例
- **User**: 机器人用户，通过 Telegram ID 标识
- **Account_Manager**: 用户账户管理模块
- **Game_Engine**: 游戏引擎，处理骰子和老虎机游戏
- **SicBo_Manager**: 骰宝游戏管理器

## Requirements

### Requirement 1: 用户账户管理

**User Story:** As a user, I want to create and manage my game account, so that I can track my balance and play games.

#### Acceptance Criteria

1. WHEN a new user sends /start, THE Bot SHALL create an account with 1000 initial coins
2. WHEN a user sends /balance, THE Bot SHALL display current balance
3. WHEN a user sends /daily, THE Bot SHALL grant 500 coins if 24 hours passed since last claim
4. IF a user claims daily within 24 hours, THEN THE Bot SHALL show remaining time
5. WHEN a user sends /top, THE Bot SHALL display top 10 users by balance

### Requirement 2: 转账功能

**User Story:** As a user, I want to transfer coins to other users, so that I can share my winnings.

#### Acceptance Criteria

1. WHEN a user sends /pay @username amount, THE Bot SHALL transfer coins to target user
2. IF sender balance is insufficient, THEN THE Bot SHALL reject transfer with error message
3. IF transfer amount is <= 0, THEN THE Bot SHALL reject transfer
4. THE Bot SHALL prevent self-transfer
5. THE Bot SHALL record all transfers in transaction history

### Requirement 3: 骰子游戏

**User Story:** As a user, I want to play dice games, so that I can win coins based on luck.

#### Acceptance Criteria

1. WHEN a user sends /dice amount, THE Bot SHALL send two Telegram dice
2. THE Bot SHALL calculate payout based on dice total (2-6: lose, 7: push, 8-11: win, 12: jackpot)
3. IF bet amount exceeds 1000, THEN THE Bot SHALL reject the bet
4. THE Bot SHALL enforce 3-second cooldown between dice games
5. THE Bot SHALL deduct bet before rolling and credit winnings after

### Requirement 4: 老虎机游戏

**User Story:** As a user, I want to play slot machine games, so that I can try to win big prizes.

#### Acceptance Criteria

1. WHEN a user sends /slot amount, THE Bot SHALL send Telegram slot machine animation
2. THE Bot SHALL calculate payout: 3 matches = tiered win, 2 matches = push, 0 matches = lose
3. THE Bot SHALL enforce 5-second cooldown between slot games
4. THE Bot SHALL decode slot value (1-64) into three symbols correctly

### Requirement 5: 骰宝游戏

**User Story:** As a user, I want to play sic bo with other users in group chat, so that we can gamble together.

#### Acceptance Criteria

1. WHEN admin sends /sicbo, THE Bot SHALL start a 60-second betting phase
2. THE Bot SHALL support bet types: single number (1-6), big, small (no pair bet)
3. THE Bot SHALL use fixed bet amount of 100 coins per button click
4. THE SicBo_Manager SHALL calculate payouts according to standard sic bo odds
5. WHEN triple occurs, THE Bot SHALL make big/small bets lose
6. THE Bot SHALL display betting panel with inline keyboard buttons (1-6, 大, 小)
7. THE Bot SHALL show settlement results with each player's net win/loss
8. THE Bot SHALL support multiple clicks on same button (accumulate bets)

### Requirement 6: 管理员功能

**User Story:** As an admin, I want to manage user accounts, so that I can handle special situations.

#### Acceptance Criteria

1. WHEN admin sends /admin_add @user amount, THE Bot SHALL add coins to user
2. WHEN admin sends /admin_sub @user amount, THE Bot SHALL subtract coins from user
3. WHEN admin sends /admin_set @user amount, THE Bot SHALL set user balance
4. THE Bot SHALL verify admin permission before executing admin commands
5. THE Bot SHALL log all admin operations

### Requirement 7: 群组白名单

**User Story:** As an admin, I want to restrict bot usage to specific groups, so that I can control access.

#### Acceptance Criteria

1. THE Bot SHALL only respond to commands in whitelisted groups
2. THE Bot SHALL allow private chat only for users who used bot in whitelisted group
3. THE Bot SHALL load whitelist from configuration file

### Requirement 8: PostgreSQL 数据库

**User Story:** As a developer, I want to use PostgreSQL, so that I have better concurrency and reliability.

#### Acceptance Criteria

1. THE PostgreSQL_DB SHALL store users table with: telegram_id, username, balance, last_daily_claim, created_at, updated_at
2. THE PostgreSQL_DB SHALL store transactions table with: id, user_id, amount, type, description, created_at
3. THE Bot SHALL use connection pooling for database access
4. THE Bot SHALL implement database migrations for schema management

### Requirement 9: 并发控制

**User Story:** As a developer, I want proper concurrency control, so that race conditions don't corrupt data.

#### Acceptance Criteria

1. THE Bot SHALL use per-user locks for balance operations
2. THE Bot SHALL prevent concurrent game sessions for same user
3. THE Bot SHALL handle database transaction isolation correctly

### Requirement 10: 可扩展游戏架构

**User Story:** As a developer, I want a modular game architecture, so that I can easily add new games in the future.

#### Acceptance Criteria

1. THE Bot SHALL define a common Game interface for all games
2. THE Bot SHALL use plugin-style game registration
3. WHEN adding a new game, THE developer SHALL only need to implement the Game interface
4. THE Bot SHALL separate game logic from Telegram handler logic
5. THE Bot SHALL support future integration with Telegram Mini Apps

### Requirement 11: 每日游戏榜单

**User Story:** As a user, I want to see daily win/loss rankings, so that I can compare my performance with others.

#### Acceptance Criteria

1. WHEN a user sends /daily_top, THE Bot SHALL display today's top winners and losers
2. THE Bot SHALL track daily net profit/loss for each user from game transactions
3. THE Bot SHALL show top 10 winners (most profit) and top 10 losers (most loss)
4. THE Bot SHALL reset daily statistics at midnight (configurable timezone)
5. THE Bot SHALL only count game-related transactions (exclude transfers, daily rewards)
