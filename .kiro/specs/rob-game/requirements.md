# Requirements Document

## Introduction

为 Go Telegram 游戏机器人添加打劫游戏功能。玩家可以通过回复其他用户的消息或 @用户名 来打劫对方的金币。

## Glossary

- **Robber**: 发起打劫的用户
- **Victim**: 被打劫的用户
- **Rob_Game**: 打劫游戏模块
- **Protection_Period**: 保护期，被连续打劫后的免疫时间

## Requirements

### Requirement 1: 打劫触发方式

**User Story:** As a user, I want to rob other users by replying to their message or mentioning them, so that I can steal their coins.

#### Acceptance Criteria

1. WHEN a user sends /dajie and replies to another user's message, THE Bot SHALL initiate a robbery against that user
2. WHEN a user sends /dajie @username, THE Bot SHALL initiate a robbery against the mentioned user
3. THE Bot SHALL prevent users from robbing themselves
4. THE Bot SHALL prevent users from robbing users who are not registered

### Requirement 2: 打劫金额计算

**User Story:** As a user, I want the robbery amount to be random, so that the game is exciting and unpredictable.

#### Acceptance Criteria

1. THE Rob_Game SHALL generate a random robbery amount between 10 and 1000 coins
2. IF the Victim's balance is less than the robbery amount, THEN THE Rob_Game SHALL only take the Victim's entire balance
3. THE Rob_Game SHALL transfer the robbery amount from Victim to Robber
4. THE Bot SHALL display the robbery result with amount stolen

### Requirement 3: 保护期机制

**User Story:** As a user, I want protection after being robbed multiple times, so that I don't lose all my coins.

#### Acceptance Criteria

1. THE Bot SHALL track consecutive robbery count for each user
2. WHEN a user is robbed 3 times consecutively, THE Bot SHALL activate a 30-minute protection period
3. WHILE a user is in protection period, THE Bot SHALL reject all robbery attempts against them
4. THE Bot SHALL display remaining protection time when robbery is rejected
5. WHEN protection period ends, THE Bot SHALL reset the consecutive robbery count

### Requirement 4: 打劫冷却

**User Story:** As a user, I want a cooldown between robberies, so that the game is balanced.

#### Acceptance Criteria

1. THE Bot SHALL enforce a 21-second cooldown between robbery attempts for each Robber
2. IF a user attempts to rob during cooldown, THEN THE Bot SHALL show remaining cooldown time
3. THE cooldown SHALL only apply to the Robber, not the Victim

### Requirement 5: 打劫记录

**User Story:** As a developer, I want robbery transactions recorded, so that they appear in daily rankings.

#### Acceptance Criteria

1. THE Bot SHALL record robbery as transaction type "rob" for the Robber (positive amount)
2. THE Bot SHALL record robbery as transaction type "robbed" for the Victim (negative amount)
3. THE robbery transactions SHALL be included in daily game rankings
