# Requirements Document

## Introduction

商店系统允许玩家使用游戏金币购买道具，这些道具可以影响打劫游戏的结果。玩家通过私聊 bot 访问商店，使用按钮界面浏览和购买道具。

## Glossary

- **Shop_System**: 商店系统，管理道具展示、购买和库存
- **Item**: 道具，可购买的虚拟物品
- **User_Inventory**: 用户背包，存储用户拥有的道具
- **Handcuff**: 手铐道具，锁定目标玩家使其无法打劫
- **Shield**: 保护罩道具，使自己无法被打劫
- **Thorn_Armor**: 荆棘刺甲道具，被打劫成功时攻击方扣双倍金币
- **Bloodthirst_Sword**: 饮血剑道具，提升打劫成功率到80%

## Requirements

### Requirement 1: 商店界面

**User Story:** As a player, I want to access a shop interface via private chat, so that I can browse and purchase items.

#### Acceptance Criteria

1. WHEN a user sends /start in private chat with the bot, THE Shop_System SHALL display a shop panel with inline buttons for each item
2. WHEN displaying the shop panel, THE Shop_System SHALL show item name, price, and brief description for each item
3. WHEN a user clicks an item button, THE Shop_System SHALL display a confirmation dialog with "购买" and "取消" buttons
4. WHEN a user clicks "购买", THE Shop_System SHALL deduct the item price from user balance and add item to inventory
5. WHEN a user clicks "取消", THE Shop_System SHALL close the confirmation dialog without any changes
6. IF a user has insufficient balance, THEN THE Shop_System SHALL display an error message and not complete the purchase

### Requirement 2: 手铐道具 (Handcuff)

**User Story:** As a player, I want to use handcuffs to lock another player, so that they cannot rob anyone for a period of time.

#### Acceptance Criteria

1. THE Handcuff item SHALL cost 500 coins
2. THE Handcuff item SHALL be a one-time use item (consumed on use)
3. WHEN a user with Handcuff uses /handcuff command (reply to message or @username), THE Shop_System SHALL lock the target for 30 minutes
4. WHILE a player is locked by Handcuff, THE Rob_Game SHALL prevent them from initiating any robbery
5. WHEN a locked player attempts to rob, THE Rob_Game SHALL display "你被手铐锁定，无法打劫" message
6. IF a user without Handcuff uses /handcuff command, THEN THE Shop_System SHALL not respond (silent ignore)
7. WHEN Handcuff is used successfully, THE Shop_System SHALL remove one Handcuff from user inventory

### Requirement 3: 保护罩道具 (Shield)

**User Story:** As a player, I want to use a shield to protect myself from being robbed.

#### Acceptance Criteria

1. THE Shield item SHALL cost 500 coins
2. THE Shield item SHALL have a duration of 6 hours after purchase
3. WHEN a user purchases Shield, THE Shop_System SHALL activate protection immediately
4. WHILE Shield is active, THE Rob_Game SHALL prevent any robbery attempts against the protected user
5. WHEN someone attempts to rob a shielded player, THE Rob_Game SHALL display "目标有保护罩，无法打劫" message
6. WHEN Shield duration expires, THE Shop_System SHALL remove the protection effect

### Requirement 4: 荆棘刺甲道具 (Thorn Armor)

**User Story:** As a player, I want to use thorn armor so that attackers lose double coins when they successfully rob me.

#### Acceptance Criteria

1. THE Thorn_Armor item SHALL cost 500 coins
2. THE Thorn_Armor item SHALL have a duration of 3 hours after purchase
3. WHEN a user purchases Thorn_Armor, THE Shop_System SHALL activate the effect immediately
4. WHILE Thorn_Armor is active AND a robbery against the user succeeds, THE Rob_Game SHALL deduct double the robbery amount from the attacker
5. WHEN Thorn_Armor duration expires, THE Shop_System SHALL remove the effect

### Requirement 5: 饮血剑道具 (Bloodthirst Sword)

**User Story:** As a player, I want to use a bloodthirst sword to increase my robbery success rate.

#### Acceptance Criteria

1. THE Bloodthirst_Sword item SHALL cost 1000 coins
2. THE Bloodthirst_Sword item SHALL have a duration of 30 minutes after purchase
3. WHEN a user purchases Bloodthirst_Sword, THE Shop_System SHALL activate the effect immediately
4. WHILE Bloodthirst_Sword is active, THE Rob_Game SHALL increase the user's robbery success rate to 80%
5. WHEN Bloodthirst_Sword duration expires, THE Shop_System SHALL remove the effect

### Requirement 6: 道具叠加

**User Story:** As a player, I want to own multiple items simultaneously, so that I can combine their effects.

#### Acceptance Criteria

1. THE User_Inventory SHALL allow storing multiple different item types simultaneously
2. THE User_Inventory SHALL allow storing multiple Handcuffs (stackable quantity)
3. WHEN a user has multiple active time-based items, THE Shop_System SHALL apply all their effects simultaneously
4. WHEN checking item effects, THE Rob_Game SHALL check all active items for the relevant user

### Requirement 7: 用户背包查看

**User Story:** As a player, I want to view my inventory, so that I can see what items I own and their remaining duration.

#### Acceptance Criteria

1. WHEN a user sends /bag or /inventory command, THE Shop_System SHALL display user's current items
2. WHEN displaying inventory, THE Shop_System SHALL show item name, quantity (for Handcuffs), and remaining duration (for time-based items)
3. IF user has no items, THEN THE Shop_System SHALL display "背包为空" message
