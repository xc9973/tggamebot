# Requirements Document

## Introduction

商店系统允许玩家使用游戏金币购买道具，这些道具可以影响打劫游戏的结果。玩家通过私聊 bot 访问商店，使用按钮界面浏览和购买道具。所有道具都是次数限制型，用完即失效。

## Glossary

- **Shop_System**: 商店系统，管理道具展示、购买和库存
- **Item**: 道具，可购买的虚拟物品
- **User_Inventory**: 用户背包，存储用户拥有的道具
- **Handcuff**: 手铐道具，锁定目标玩家使其无法打劫
- **Shield**: 保护罩道具，使自己无法被打劫
- **Thorn_Armor**: 荆棘刺甲道具，被打劫成功时攻击方扣双倍金币
- **Bloodthirst_Sword**: 饮血剑道具，提升打劫成功率到80%
- **Blunt_Knife**: 钝刀道具，无视防御但打劫金额随机1-100
- **Great_Sword**: 大宝剑道具，无视防御且有极小概率打劫90%金币
- **Golden_Cassock**: 紫金袈裟道具，攻击拥有者会导致攻击方防御道具掉落
- **Emperor_Clothes**: 皇帝的新衣道具，免疫所有攻击（包括无视防御的攻击）

## Requirements

### Requirement 1: 商店界面

**User Story:** As a player, I want to access a shop interface via private chat, so that I can browse and purchase items.

#### Acceptance Criteria

1. WHEN a user sends /start in private chat with the bot, THE Shop_System SHALL display a shop panel with inline buttons for each item (8 items)
2. WHEN displaying the shop panel, THE Shop_System SHALL show item name, price, use count, and brief description for each item
3. WHEN a user clicks an item button, THE Shop_System SHALL display a confirmation dialog with "购买" and "取消" buttons
4. WHEN a user clicks "购买", THE Shop_System SHALL deduct the item price from user balance and add item to inventory
5. WHEN a user clicks "取消", THE Shop_System SHALL close the confirmation dialog without any changes
6. IF a user has insufficient balance, THEN THE Shop_System SHALL display an error message and not complete the purchase

### Requirement 2: 手铐道具 (Handcuff)

**User Story:** As a player, I want to use handcuffs to lock another player, so that they cannot rob anyone for a period of time.

#### Acceptance Criteria

1. THE Handcuff item SHALL cost 500 coins
2. THE Handcuff item SHALL be a one-time use item (1 use per purchase)
3. THE Handcuff item SHALL have a daily purchase limit of 5 per user
4. WHEN a user with Handcuff uses /handcuff command (reply to message or @username), THE Shop_System SHALL lock the target for 30 minutes
5. WHILE a player is locked by Handcuff, THE Rob_Game SHALL prevent them from initiating any robbery
6. WHEN a locked player attempts to rob, THE Rob_Game SHALL display "你被手铐锁定，无法打劫" message
7. IF a user without Handcuff uses /handcuff command, THEN THE Shop_System SHALL not respond (silent ignore)
8. WHEN Handcuff is used successfully, THE Shop_System SHALL remove one Handcuff from user inventory
9. IF a user has already purchased 5 Handcuffs today, THEN THE Shop_System SHALL reject the purchase with "今日购买次数已达上限" message

### Requirement 3: 保护罩道具 (Shield)

**User Story:** As a player, I want to use a shield to protect myself from being robbed.

#### Acceptance Criteria

1. THE Shield item SHALL cost 500 coins
2. THE Shield item SHALL have 10 uses per purchase
3. THE Shield item SHALL have a daily purchase limit of 2 per user
4. WHEN a user purchases Shield, THE Shop_System SHALL activate protection immediately
5. WHILE Shield use count is greater than 0, THE Rob_Game SHALL prevent any robbery attempts against the protected user
6. WHEN someone attempts to rob a shielded player, THE Rob_Game SHALL display "目标有保护罩，无法打劫" message and decrement use count by 1
7. WHEN Shield use count reaches 0, THE Shop_System SHALL remove the protection effect
8. IF a user has already purchased 2 Shields today, THEN THE Shop_System SHALL reject the purchase with "今日购买次数已达上限" message

### Requirement 4: 荆棘刺甲道具 (Thorn Armor)

**User Story:** As a player, I want to use thorn armor so that attackers lose double coins when they successfully rob me.

#### Acceptance Criteria

1. THE Thorn_Armor item SHALL cost 500 coins
2. THE Thorn_Armor item SHALL have 5 uses per purchase
3. WHEN a user purchases Thorn_Armor, THE Shop_System SHALL activate the effect immediately
4. WHILE Thorn_Armor use count is greater than 0 AND a robbery against the user succeeds, THE Rob_Game SHALL deduct double the robbery amount from the attacker and decrement use count by 1
5. WHEN Thorn_Armor use count reaches 0, THE Shop_System SHALL remove the effect

### Requirement 5: 饮血剑道具 (Bloodthirst Sword)

**User Story:** As a player, I want to use a bloodthirst sword to increase my robbery success rate.

#### Acceptance Criteria

1. THE Bloodthirst_Sword item SHALL cost 1000 coins
2. THE Bloodthirst_Sword item SHALL have 10 uses per purchase
3. WHEN a user purchases Bloodthirst_Sword, THE Shop_System SHALL activate the effect immediately
4. WHILE Bloodthirst_Sword use count is greater than 0, THE Rob_Game SHALL increase the user's robbery success rate to 80% and decrement use count by 1 on each robbery attempt
5. WHEN Bloodthirst_Sword use count reaches 0, THE Shop_System SHALL remove the effect

### Requirement 6: 钝刀道具 (Blunt Knife)

**User Story:** As a player, I want to use a blunt knife to bypass enemy defenses, even though the robbery amount is limited.

#### Acceptance Criteria

1. THE Blunt_Knife item SHALL cost 1000 coins
2. THE Blunt_Knife item SHALL have 10 uses per purchase
3. WHEN a user purchases Blunt_Knife, THE Shop_System SHALL activate the effect immediately
4. WHILE Blunt_Knife use count is greater than 0, THE Rob_Game SHALL ignore target's Shield and Thorn_Armor effects (but NOT Emperor_Clothes)
5. WHILE Blunt_Knife use count is greater than 0, THE Rob_Game SHALL limit robbery amount to a random value between 1-100 coins and decrement use count by 1
6. WHEN Blunt_Knife use count reaches 0, THE Shop_System SHALL remove the effect

### Requirement 7: 大宝剑道具 (Great Sword)

**User Story:** As a player, I want to use a great sword for a chance to rob a massive amount of coins while bypassing defenses.

#### Acceptance Criteria

1. THE Great_Sword item SHALL cost 10000 coins
2. THE Great_Sword item SHALL have 3 uses per purchase
3. THE Great_Sword item SHALL have a daily purchase limit of 1 per user
4. WHEN a user purchases Great_Sword, THE Shop_System SHALL activate the effect immediately
5. WHILE Great_Sword use count is greater than 0, THE Rob_Game SHALL ignore target's Shield and Thorn_Armor effects (but NOT Emperor_Clothes)
6. WHILE Great_Sword use count is greater than 0, THE Rob_Game SHALL have a 0.01% chance to rob 90% of target's coins on successful robbery and decrement use count by 1
7. WHEN Great_Sword use count reaches 0, THE Shop_System SHALL remove the effect
8. IF a user has already purchased 1 Great_Sword today, THEN THE Shop_System SHALL reject the purchase with "今日购买次数已达上限" message

### Requirement 8: 紫金袈裟道具 (Golden Cassock)

**User Story:** As a player, I want to use a golden cassock so that attackers lose their defensive items when they attack me.

#### Acceptance Criteria

1. THE Golden_Cassock item SHALL cost 10000 coins
2. THE Golden_Cassock item SHALL have 3 uses per purchase
3. WHEN a user purchases Golden_Cassock, THE Shop_System SHALL activate the effect immediately
4. WHILE Golden_Cassock use count is greater than 0 AND someone attempts to rob the user, THE Rob_Game SHALL remove all defensive items (Shield, Thorn_Armor) from the attacker and decrement use count by 1
5. WHEN Golden_Cassock use count reaches 0, THE Shop_System SHALL remove the effect

### Requirement 9: 皇帝的新衣道具 (Emperor's Clothes)

**User Story:** As a player, I want to use emperor's clothes to be completely immune to all attacks, including those that bypass normal defenses.

#### Acceptance Criteria

1. THE Emperor_Clothes item SHALL cost 5000 coins
2. THE Emperor_Clothes item SHALL have 3 uses per purchase
3. WHEN a user purchases Emperor_Clothes, THE Shop_System SHALL activate the effect immediately
4. WHILE Emperor_Clothes use count is greater than 0, THE Rob_Game SHALL prevent ALL robbery attempts against the user, including those with Blunt_Knife or Great_Sword
5. WHEN someone attempts to rob a user with Emperor_Clothes, THE Rob_Game SHALL display "目标有皇帝的新衣，无法打劫" message and decrement use count by 1
6. WHEN Emperor_Clothes use count reaches 0, THE Shop_System SHALL remove the effect

### Requirement 10: 道具叠加

**User Story:** As a player, I want to own multiple items simultaneously, so that I can combine their effects.

#### Acceptance Criteria

1. THE User_Inventory SHALL allow storing multiple different item types simultaneously
2. THE User_Inventory SHALL allow storing multiple Handcuffs (stackable quantity)
3. WHEN a user has multiple active items, THE Shop_System SHALL apply all their effects simultaneously
4. WHEN checking item effects, THE Rob_Game SHALL check all active items for the relevant user
5. WHEN multiple defensive items are active, THE Rob_Game SHALL check Emperor_Clothes first (highest priority)

### Requirement 11: 用户背包查看

**User Story:** As a player, I want to view my inventory, so that I can see what items I own and their remaining uses.

#### Acceptance Criteria

1. WHEN a user sends /bag or /inventory command, THE Shop_System SHALL display user's current items
2. WHEN displaying inventory, THE Shop_System SHALL show item name, quantity (for Handcuffs), and remaining use count (for other items)
3. IF user has no items, THEN THE Shop_System SHALL display "背包为空" message

### Requirement 12: 每日购买限制

**User Story:** As a system administrator, I want to limit daily purchases of certain items to maintain game balance.

#### Acceptance Criteria

1. THE Shop_System SHALL track daily purchase count per user per item type
2. THE Shop_System SHALL reset daily purchase counts at midnight (server time)
3. WHEN a user attempts to purchase an item with daily limit, THE Shop_System SHALL check current daily purchase count
4. IF daily purchase limit is reached, THEN THE Shop_System SHALL reject the purchase with appropriate message
