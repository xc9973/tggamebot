# 需求文档

## 简介

骰宝(Sic Bo)是一个多人参与的骰子游戏，使用三个骰子进行游戏。玩家可以在多种下注区域进行押注，包括单一数字、两个数字组合、数字总和、大小等。游戏在群组中进行，一次只能有一场游戏，必须等当前游戏结束才能开始新的一场。

## 术语表

- **Bot**: Telegram 游戏机器人系统
- **User**: Telegram 群组成员/玩家
- **Coin**: 虚拟积分/金币
- **Account**: 用户的虚拟账户
- **Database**: 数据存储系统
- **Sic_Bo_Game**: 骰宝游戏会话
- **Dice**: 骰子（三个）
- **Bet**: 下注
- **Single_Number_Bet**: 单一数字押注（押1-6中的某个数字）
- **Pair_Combination_Bet**: 两个数字组合押注（押两个不同数字的组合）
- **Sum_Bet**: 数字总和押注（押三个骰子的总和）
- **Big_Small_Bet**: 大小押注（大11-17，小4-10）
- **Triple**: 围骰/三同号（三个骰子显示相同数字）
- **Payout**: 赔率/赢钱倍数
- **Game_Session**: 游戏会话（从开场到结算的完整周期）
- **Betting_Phase**: 下注阶段
- **Result_Phase**: 开奖阶段

## 需求

### 需求 1: 游戏会话管理

**用户故事:** 作为群组成员，我希望在群里开启骰宝游戏，以便多人可以参与下注。

#### 验收标准

1. WHEN 用户调用 /sicbo 命令且当前群组没有进行中的游戏 THEN THE Bot SHALL 创建新的 Sic_Bo_Game 会话并进入 Betting_Phase
2. WHEN 用户调用 /sicbo 命令且当前群组已有进行中的游戏 THEN THE Bot SHALL 拒绝创建并提示当前游戏未结束
3. WHEN Sic_Bo_Game 创建成功 THEN THE Bot SHALL 发送游戏开始消息，显示下注区域和赔率说明
4. WHEN Betting_Phase 开始 THEN THE Bot SHALL 设置 60 秒的下注时间限制
5. WHEN 下注时间结束或庄家调用 /roll 命令 THEN THE Bot SHALL 结束 Betting_Phase 并进入 Result_Phase

### 需求 2: 单一数字押注

**用户故事:** 作为玩家，我希望押注骰子会显示哪个数字，以便通过预测获得奖励。

#### 验收标准

1. WHEN 玩家在 Betting_Phase 调用 /bet single <数字> <金额> 命令且余额充足 THEN THE Bot SHALL 记录该押注并从 Account 扣除金额
2. WHEN 数字参数不在 1-6 范围内 THEN THE Bot SHALL 拒绝押注并提示有效数字范围
3. WHEN 三个骰子中有一个显示玩家押的数字 THEN THE Bot SHALL 向玩家 Account 增加 bet * 2（1:1 赔率，返还本金+1倍奖励）
4. WHEN 三个骰子中有两个显示玩家押的数字 THEN THE Bot SHALL 向玩家 Account 增加 bet * 3（2:1 赔率，返还本金+2倍奖励）
5. WHEN 三个骰子中有三个显示玩家押的数字 THEN THE Bot SHALL 向玩家 Account 增加 bet * 4（3:1 赔率，返还本金+3倍奖励）
6. WHEN 三个骰子中没有显示玩家押的数字 THEN THE Bot SHALL 不返还任何金额（玩家输掉本金）

### 需求 3: 两个数字组合押注

**用户故事:** 作为玩家，我希望押注两个骰子的数字组合，以便获得更高的赔率。

#### 验收标准

1. WHEN 玩家在 Betting_Phase 调用 /bet pair <数字1> <数字2> <金额> 命令且余额充足 THEN THE Bot SHALL 记录该押注并从 Account 扣除金额
2. WHEN 数字1 等于 数字2 THEN THE Bot SHALL 拒绝押注并提示两个数字必须不同
3. WHEN 数字参数不在 1-6 范围内 THEN THE Bot SHALL 拒绝押注并提示有效数字范围
4. WHEN 三个骰子中至少有两个分别显示数字1和数字2 THEN THE Bot SHALL 向玩家 Account 增加 bet * 6（5:1 赔率，返还本金+5倍奖励）
5. WHEN 三个骰子未能同时包含数字1和数字2 THEN THE Bot SHALL 不返还任何金额
6. WHEN 骰子显示如 3,3,5 而玩家押注 3-5 组合 THEN THE Bot SHALL 只计算一次组合赢钱（不因重复数字多次计算）

### 需求 4: 数字总和押注

**用户故事:** 作为玩家，我希望押注三个骰子的数字总和，以便根据不同总和获得不同赔率。

#### 验收标准

1. WHEN 玩家在 Betting_Phase 调用 /bet sum <总和> <金额> 命令且余额充足 THEN THE Bot SHALL 记录该押注并从 Account 扣除金额
2. WHEN 总和参数不在 4-17 范围内 THEN THE Bot SHALL 拒绝押注并提示有效总和范围（3和18不可押注）
3. WHEN 三个骰子的总和等于玩家押的总和且不是 Triple THEN THE Bot SHALL 按对应赔率向玩家 Account 增加奖金
4. WHEN 三个骰子形成 Triple THEN THE Bot SHALL 判定所有总和押注为输（庄家通吃）
5. THE Bot SHALL 使用以下赔率表计算总和押注奖金:
   - 总和 4 或 17: 60:1 赔率
   - 总和 5 或 16: 30:1 赔率
   - 总和 6 或 15: 17:1 赔率
   - 总和 7 或 14: 12:1 赔率
   - 总和 8 或 13: 8:1 赔率
   - 总和 9 或 12: 6:1 赔率
   - 总和 10 或 11: 6:1 赔率

### 需求 5: 大小押注

**用户故事:** 作为玩家，我希望押注大或小，以便进行简单的 1:1 赔率游戏。

#### 验收标准

1. WHEN 玩家在 Betting_Phase 调用 /bet big <金额> 命令且余额充足 THEN THE Bot SHALL 记录大押注并从 Account 扣除金额
2. WHEN 玩家在 Betting_Phase 调用 /bet small <金额> 命令且余额充足 THEN THE Bot SHALL 记录小押注并从 Account 扣除金额
3. WHEN 三个骰子总和为 11-17 且不是 Triple THEN THE Bot SHALL 向押大的玩家 Account 增加 bet * 2（1:1 赔率）
4. WHEN 三个骰子总和为 4-10 且不是 Triple THEN THE Bot SHALL 向押小的玩家 Account 增加 bet * 2（1:1 赔率）
5. WHEN 三个骰子形成 Triple THEN THE Bot SHALL 判定所有大小押注为输（庄家通吃）

### 需求 6: 游戏结算

**用户故事:** 作为玩家，我希望在骰子开出后立即看到结果和奖金，以便了解我的输赢情况。

#### 验收标准

1. WHEN Result_Phase 开始 THEN THE Bot SHALL 使用 Telegram sendDice API 发送三个骰子动画
2. WHEN 骰子结果确定 THEN THE Bot SHALL 显示三个骰子的点数和总和
3. WHEN 骰子结果确定 THEN THE Bot SHALL 计算所有玩家的所有押注并进行结算
4. WHEN 结算完成 THEN THE Bot SHALL 显示每位玩家的输赢情况和当前余额
5. WHEN 结算完成 THEN THE Bot SHALL 结束当前 Sic_Bo_Game 会话，允许开始新游戏
6. WHEN 玩家有多个押注 THEN THE Bot SHALL 分别计算每个押注的输赢

### 需求 7: 下注验证

**用户故事:** 作为系统，我需要验证所有下注请求，以确保游戏公平性和数据完整性。

#### 验收标准

1. WHEN 玩家余额不足下注金额 THEN THE Bot SHALL 拒绝下注并提示余额不足
2. WHEN 下注金额小于或等于 0 THEN THE Bot SHALL 拒绝下注并提示无效金额
3. WHEN 玩家在非 Betting_Phase 尝试下注 THEN THE Bot SHALL 拒绝下注并提示当前不在下注阶段
4. WHEN 下注成功 THEN THE Bot SHALL 立即从玩家 Account 扣除下注金额
5. WHEN 下注成功 THEN THE Bot SHALL 向玩家发送确认消息显示下注类型和金额

### 需求 8: 游戏状态查询

**用户故事:** 作为玩家，我希望查看当前游戏状态和我的下注情况，以便做出更好的决策。

#### 验收标准

1. WHEN 玩家调用 /sicbo_status 命令且有进行中的游戏 THEN THE Bot SHALL 显示当前游戏状态和剩余下注时间
2. WHEN 玩家调用 /sicbo_status 命令且没有进行中的游戏 THEN THE Bot SHALL 提示当前没有进行中的游戏
3. WHEN 玩家调用 /mybets 命令且有进行中的游戏 THEN THE Bot SHALL 显示该玩家在当前游戏中的所有押注
4. WHEN 显示游戏状态 THEN THE Bot SHALL 包含参与人数和总下注金额

