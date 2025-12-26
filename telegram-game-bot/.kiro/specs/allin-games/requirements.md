# Requirements Document

## Introduction

梭哈游戏系统 - 为玩家提供高风险高回报的赌博玩法，包括梭哈打劫、梭哈对决和梭哈骰子三种模式。

## Glossary

- **All_In_System**: 梭哈游戏系统
- **All_In_Rob**: 梭哈打劫 - 押上全部金币打劫对方
- **All_In_Duel**: 梭哈对决 - 双方各押全部金币对决
- **All_In_Dice**: 梭哈骰子 - 押全部金币掷骰子
- **User**: 游戏玩家
- **Balance**: 用户金币余额

## Requirements

### Requirement 1: 梭哈打劫

**User Story:** As a user, I want to all-in rob another player, so that I can risk everything for a big reward.

#### Acceptance Criteria

1. WHEN a user initiates all-in rob, THE All_In_System SHALL require the user to have at least 100 coins
2. WHEN all-in rob succeeds (50% chance), THE All_In_System SHALL transfer the lesser of (attacker's balance, victim's balance) from victim to attacker
3. WHEN all-in rob fails (50% chance), THE All_In_System SHALL transfer attacker's entire balance to victim
4. THE All_In_System SHALL have a separate cooldown (60 seconds) for all-in rob
5. WHEN victim has Emperor Clothes, THE All_In_System SHALL block the all-in rob attempt

### Requirement 2: 梭哈对决

**User Story:** As a user, I want to challenge another player to an all-in duel, so that we can have a winner-takes-all battle.

#### Acceptance Criteria

1. WHEN a user initiates a duel challenge, THE All_In_System SHALL create a pending duel request
2. WHEN the challenged user accepts, THE All_In_System SHALL execute the duel with 50/50 odds
3. WHEN duel executes, THE All_In_System SHALL transfer the lesser of both balances from loser to winner
4. WHEN the challenged user declines or timeout (60 seconds), THE All_In_System SHALL cancel the duel
5. THE All_In_System SHALL require both users to have at least 100 coins to participate
6. WHEN a user has a pending duel, THE All_In_System SHALL prevent them from starting another duel

### Requirement 3: 梭哈骰子

**User Story:** As a user, I want to gamble all my coins on a dice roll, so that I can double or lose everything.

#### Acceptance Criteria

1. WHEN a user plays all-in dice, THE All_In_System SHALL require at least 100 coins
2. THE All_In_System SHALL roll two dice (2-12 total)
3. WHEN dice total is 7 or higher, THE All_In_System SHALL double the user's balance
4. WHEN dice total is 6 or lower, THE All_In_System SHALL set user's balance to 0
5. THE All_In_System SHALL have a cooldown (30 seconds) for all-in dice
6. THE All_In_System SHALL record all dice results in transaction history

### Requirement 4: 命令接口

**User Story:** As a user, I want simple commands to access all-in features.

#### Acceptance Criteria

1. WHEN user sends /shdj @target, THE Bot SHALL initiate all-in rob against target
2. WHEN user sends /duijue @target, THE Bot SHALL send duel challenge to target with inline buttons
3. WHEN target clicks accept button, THE Bot SHALL execute the duel
4. WHEN target clicks decline button or 60 seconds timeout, THE Bot SHALL cancel the duel
5. WHEN user sends /shdice, THE Bot SHALL play all-in dice game
