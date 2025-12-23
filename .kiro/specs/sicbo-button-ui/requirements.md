# Requirements Document

## Introduction

将骰宝游戏从命令行交互改为按钮交互，提升用户体验。采用固定金额模式，用户点击一次按钮即可完成下注，无需多步选择。暂不支持两数组合下注。

## Glossary

- **SicBo_Game**: 骰宝游戏会话，管理一局游戏的状态和押注
- **Inline_Keyboard**: Telegram 的内联键盘，显示在消息下方的可点击按钮
- **Bet_Panel**: 下注面板，显示所有可用的下注选项
- **Fixed_Bet_Amount**: 固定下注金额，每次点击按钮下注的金额（默认 100 金币）

## Requirements

### Requirement 1: 游戏启动与主面板

**User Story:** As a player, I want to start a SicBo game and see all betting options at once, so that I can quickly place bets with one click.

#### Acceptance Criteria

1. WHEN a user sends /sicbo command, THE SicBo_Game SHALL display an Inline_Keyboard with all betting options
2. THE Bet_Panel SHALL display single number buttons (1-6) in one row
3. THE Bet_Panel SHALL display big/small buttons ("大", "小") in one row
4. THE Bet_Panel SHALL display sum buttons grouped by odds in rows
5. THE Bet_Panel SHALL display a "开骰子" button and "我的押注" button at the bottom
6. WHEN a game is already in progress, THE SicBo_Game SHALL show the existing game panel instead of starting a new one

### Requirement 2: 一键下注

**User Story:** As a player, I want to place a bet with a single button click, so that betting is fast and simple.

#### Acceptance Criteria

1. WHEN a user clicks any betting button, THE SicBo_Game SHALL immediately place a bet with the Fixed_Bet_Amount (100 gold coins)
2. WHEN a bet is placed, THE SicBo_Game SHALL show a popup confirmation (callback query answer)
3. WHEN a user's balance is insufficient, THE SicBo_Game SHALL show an error popup and not place the bet
4. THE same user SHALL be able to click the same button multiple times to increase their bet on that option
5. WHEN a bet is placed, THE SicBo_Game SHALL update the panel message to reflect new totals

### Requirement 3: 单一数字下注按钮

**User Story:** As a player, I want to bet on single numbers with one click, so that I can quickly bet on my lucky numbers.

#### Acceptance Criteria

1. THE Bet_Panel SHALL display buttons labeled "1", "2", "3", "4", "5", "6" for single number bets
2. WHEN a user clicks a number button, THE SicBo_Game SHALL place a single number bet with Fixed_Bet_Amount
3. THE payout for single number bets SHALL be: 1 match = 1:1, 2 matches = 2:1, 3 matches = 3:1

### Requirement 4: 大小下注按钮

**User Story:** As a player, I want to bet on big/small with one click, so that I can place the most common bets instantly.

#### Acceptance Criteria

1. THE Bet_Panel SHALL display "大" and "小" buttons with odds info
2. WHEN a user clicks "大" button, THE SicBo_Game SHALL place a big bet (sum 11-17) with Fixed_Bet_Amount
3. WHEN a user clicks "小" button, THE SicBo_Game SHALL place a small bet (sum 4-10) with Fixed_Bet_Amount
4. THE payout for big/small bets SHALL be 1:1, except when triple (house wins)

### Requirement 5: 总和下注按钮

**User Story:** As a player, I want to bet on sums with one click, so that I can chase high payouts easily.

#### Acceptance Criteria

1. THE Bet_Panel SHALL display sum buttons for values 4 through 17
2. THE sum buttons SHALL show the payout odds: 4/17=60:1, 5/16=30:1, 6/15=17:1, 7/14=12:1, 8/13=8:1, 9-12=6:1
3. WHEN a user clicks a sum button, THE SicBo_Game SHALL place a sum bet with Fixed_Bet_Amount
4. THE sum buttons SHALL be arranged in logical groups (high odds, medium odds, low odds)

### Requirement 6: 下注状态显示

**User Story:** As a player, I want to see the current game status on the panel, so that I know the game state.

#### Acceptance Criteria

1. THE Bet_Panel message SHALL display the game status header with remaining betting time
2. THE Bet_Panel message SHALL display total number of players and total bet amount
3. WHEN any bet is placed, THE SicBo_Game SHALL update the panel message with new statistics
4. THE "我的押注" button SHALL show the user's own bets when clicked

### Requirement 7: 开骰子与结算

**User Story:** As a player, I want to roll the dice and see results clearly, so that I know if I won or lost.

#### Acceptance Criteria

1. THE Bet_Panel SHALL display a "🎲 开骰子" button
2. WHEN any user clicks "开骰子" button, THE SicBo_Game SHALL end betting phase and start rolling
3. WHEN the betting time (60 seconds) expires, THE SicBo_Game SHALL automatically roll the dice
4. WHEN rolling starts, THE SicBo_Game SHALL send three dice animations using Telegram dice API
5. WHEN dice animations complete, THE SicBo_Game SHALL display the results and settle all bets
6. THE settlement message SHALL show each player's username and their net win/loss amount (e.g., "@zhangsan +500", "@lisi -200")
7. THE settlement message SHALL clearly indicate winners with 🎉 emoji and losers with 😢 emoji

### Requirement 8: 按钮交互安全

**User Story:** As a player, I want my button clicks to be processed correctly, so that my bets are recorded accurately.

#### Acceptance Criteria

1. WHEN a user clicks a button, THE SicBo_Game SHALL verify the game is still in betting phase
2. IF the game is not in betting phase, THE SicBo_Game SHALL show an error popup
3. THE SicBo_Game SHALL handle concurrent button clicks from multiple users correctly
4. WHEN a button click fails, THE SicBo_Game SHALL not deduct the user's balance
