# 需求文档

## 简介

Telegram 群组游戏机器人是一个部署在 Telegram 群组中的互动系统，允许群成员通过虚拟积分参与多种概率游戏。系统通过积分经济、每日签到、游戏机制和排行榜等功能增加群组活跃度和用户粘性。

## 术语表

- **Bot**: Telegram 游戏机器人系统
- **User**: Telegram 群组成员
- **Coin**: 虚拟积分/金币
- **Account**: 用户的虚拟账户
- **Database**: 数据存储系统
- **Admin**: 群组管理员
- **Telegram_API**: Telegram Bot API 接口
- **sendDice**: Telegram 原生骰子动画接口
- **Inline_Keyboard**: Telegram 内联按钮界面

## 需求

### 需求 1: 用户账户管理

**用户故事:** 作为群组成员，我希望拥有独立的虚拟账户，以便参与群内的游戏和经济活动。

#### 验收标准

1. WHEN 用户首次在群组中发言或调用命令 THEN THE Bot SHALL 自动在 Database 中创建该用户的 Account 并赠送 1000 Coins
2. WHEN 用户调用 /start 命令 THEN THE Bot SHALL 初始化或确认用户的 Account 存在
3. WHEN 用户调用 /balance 或 /my 命令 THEN THE Bot SHALL 显示该用户当前的 Coin 余额
4. WHEN 用户的 Account 被创建 THEN THE Bot SHALL 记录用户的 Telegram ID、用户名和初始余额
5. WHEN 查询不存在的 Account THEN THE Bot SHALL 自动创建该 Account

### 需求 2: 每日签到系统

**用户故事:** 作为群组成员，我希望每天可以领取签到奖励，以便在输光积分后仍能继续参与游戏。

#### 验收标准

1. WHEN 用户调用 /daily 命令且距离上次签到已超过 24 小时 THEN THE Bot SHALL 向用户的 Account 增加 500 Coins
2. WHEN 用户调用 /daily 命令且距离上次签到不足 24 小时 THEN THE Bot SHALL 拒绝签到并显示剩余等待时间
3. WHEN 用户首次调用 /daily 命令 THEN THE Bot SHALL 允许立即签到
4. WHEN 签到成功 THEN THE Bot SHALL 更新用户的最后签到时间戳
5. WHEN 签到成功 THEN THE Bot SHALL 向用户发送确认消息显示获得的 Coins 数量

### 需求 3: 用户间转账

**用户故事:** 作为群组成员，我希望能够向其他成员转账积分，以便进行社交互动和交易。

#### 验收标准

1. WHEN 用户调用 /pay @target_user amount 命令且余额充足 THEN THE Bot SHALL 从发送者 Account 扣除 amount 并向接收者 Account 增加 amount * 0.95
2. WHEN 转账发生 THEN THE Bot SHALL 扣除 5% 作为手续费回收到系统
3. WHEN 用户余额不足 THEN THE Bot SHALL 拒绝转账并提示余额不足
4. WHEN 转账金额小于或等于 0 THEN THE Bot SHALL 拒绝转账并提示无效金额
5. WHEN 目标用户不存在 THEN THE Bot SHALL 拒绝转账并提示用户不存在
6. WHEN 用户尝试向自己转账 THEN THE Bot SHALL 拒绝转账并提示不能自己转给自己
7. WHEN 转账成功 THEN THE Bot SHALL 向双方发送确认消息

### 需求 4: 财富排行榜

**用户故事:** 作为群组成员，我希望查看群内最富有的用户排名，以便了解自己的相对位置并激发竞争意识。

#### 验收标准

1. WHEN 用户调用 /top 命令 THEN THE Bot SHALL 显示群内 Coin 余额最高的前 10 名 Users
2. WHEN 显示排行榜 THEN THE Bot SHALL 包含每个用户的排名、用户名和 Coin 余额
3. WHEN 群内用户少于 10 人 THEN THE Bot SHALL 显示所有用户的排名
4. WHEN 多个用户余额相同 THEN THE Bot SHALL 按账户创建时间排序

### 需求 5: 骰子游戏

**用户故事:** 作为群组成员，我希望通过掷骰子游戏下注积分，以便体验简单的概率游戏。

#### 验收标准

1. WHEN 用户调用 /dice amount 命令且余额充足 THEN THE Bot SHALL 使用 Telegram_API 的 sendDice 接口发送骰子动画
2. WHEN 骰子结果为 1-3 THEN THE Bot SHALL 从用户 Account 扣除 amount
3. WHEN 骰子结果为 4-5 THEN THE Bot SHALL 向用户 Account 增加 amount (返还本金 + 1倍奖励)
4. WHEN 骰子结果为 6 THEN THE Bot SHALL 向用户 Account 增加 amount * 2 (返还本金 + 2倍奖励)
5. WHEN 用户余额不足 amount THEN THE Bot SHALL 拒绝游戏并提示余额不足
6. WHEN amount 小于或等于 0 THEN THE Bot SHALL 拒绝游戏并提示无效金额
7. WHEN 游戏结束 THEN THE Bot SHALL 显示结果和用户当前余额

### 需求 6: 老虎机游戏

**用户故事:** 作为群组成员，我希望通过老虎机游戏下注积分，以便体验高赔率的概率游戏。

#### 验收标准

1. WHEN 用户调用 /slot amount 命令且余额充足 THEN THE Bot SHALL 使用 Telegram_API 的 sendDice 接口发送老虎机动画
2. WHEN 三个图案完全一致 THEN THE Bot SHALL 向用户 Account 增加 amount * 10 (返还本金 + 10倍奖励)
3. WHEN 两个图案一致 THEN THE Bot SHALL 向用户 Account 增加 amount * 2 (返还本金 + 2倍奖励)
4. WHEN 三个图案不一致 THEN THE Bot SHALL 从用户 Account 扣除 amount
5. WHEN 用户余额不足 amount THEN THE Bot SHALL 拒绝游戏并提示余额不足
6. WHEN amount 小于或等于 0 THEN THE Bot SHALL 拒绝游戏并提示无效金额
7. WHEN 游戏结束 THEN THE Bot SHALL 显示结果和用户当前余额

### 需求 7: 21点游戏

**用户故事:** 作为群组成员，我希望通过21点游戏进行策略性下注，以便体验更复杂的互动游戏。

#### 验收标准

1. WHEN 用户调用 /bj amount 命令且余额充足 THEN THE Bot SHALL 创建新的21点游戏会话并发送 Inline_Keyboard
2. WHEN 游戏开始 THEN THE Bot SHALL 向用户发两张明牌并向庄家发一张明牌一张暗牌
3. WHEN 用户点击"要牌"按钮 THEN THE Bot SHALL 向用户发一张新牌并更新消息显示
4. WHEN 用户点击"停牌"按钮 THEN THE Bot SHALL 结束用户回合并执行庄家逻辑
5. WHEN 用户点击"加倍"按钮且余额充足 THEN THE Bot SHALL 将下注金额翻倍、发一张牌后自动停牌
6. WHEN 用户点数超过 21 THEN THE Bot SHALL 判定用户爆牌并从 Account 扣除 amount
7. WHEN 庄家点数小于 17 THEN THE Bot SHALL 继续要牌直到点数大于等于 17
8. WHEN 用户点数大于庄家且未爆牌 THEN THE Bot SHALL 向用户 Account 增加 amount
9. WHEN 用户点数等于庄家 THEN THE Bot SHALL 返还用户本金
10. WHEN 用户点数小于庄家或庄家未爆牌 THEN THE Bot SHALL 从用户 Account 扣除 amount
11. WHEN 用户获得 Blackjack (首两张牌点数为21) THEN THE Bot SHALL 向用户 Account 增加 amount * 1.5

### 需求 8: 管理员功能

**用户故事:** 作为群组管理员，我希望能够管理用户积分和系统状态，以便维护游戏公平性和处理特殊情况。

#### 验收标准

1. WHEN Admin 调用 /admin_add @user amount 命令 THEN THE Bot SHALL 向指定用户 Account 增加 amount Coins
2. WHEN Admin 调用 /admin_remove @user amount 命令 THEN THE Bot SHALL 从指定用户 Account 扣除 amount Coins
3. WHEN 非 Admin 用户调用管理员命令 THEN THE Bot SHALL 拒绝执行并提示权限不足
4. WHEN Admin 调用 /admin_reset @user 命令 THEN THE Bot SHALL 将指定用户 Account 重置为初始状态 (1000 Coins)
5. WHEN Admin 调用管理员命令 THEN THE Bot SHALL 记录操作日志包含操作者、目标用户、操作类型和时间戳

### 需求 9: 数据持久化

**用户故事:** 作为系统，我需要可靠地存储用户数据，以便在机器人重启后保持数据完整性。

#### 验收标准

1. WHEN 用户数据发生变化 THEN THE Bot SHALL 立即将变化持久化到 Database
2. WHEN Bot 启动 THEN THE Bot SHALL 从 Database 加载所有用户 Accounts
3. WHEN 多个操作同时修改同一 Account THEN THE Bot SHALL 使用事务机制确保数据一致性
4. WHEN Database 操作失败 THEN THE Bot SHALL 回滚事务并向用户返回错误消息
5. WHEN 游戏结算发生 THEN THE Bot SHALL 在单个事务中完成所有余额变更

### 需求 10: 并发控制

**用户故事:** 作为系统，我需要处理并发请求，以便防止用户通过同时发送多个命令来作弊或导致数据不一致。

#### 验收标准

1. WHEN 用户同时发送多个游戏命令 THEN THE Bot SHALL 按顺序处理每个命令
2. WHEN 用户在游戏进行中发送新游戏命令 THEN THE Bot SHALL 拒绝新命令并提示当前游戏未结束
3. WHEN 多个用户同时操作 THEN THE Bot SHALL 确保每个用户的操作独立处理不互相影响
4. WHEN 转账操作进行中 THEN THE Bot SHALL 锁定相关 Accounts 直到操作完成
5. WHEN 余额检查和扣除之间 THEN THE Bot SHALL 使用原子操作防止余额被其他操作修改

### 需求 11: 错误处理和用户反馈

**用户故事:** 作为群组成员，我希望在操作失败时收到清晰的错误提示，以便了解问题并采取正确的行动。

#### 验收标准

1. WHEN 命令格式错误 THEN THE Bot SHALL 返回使用说明和正确的命令格式示例
2. WHEN 系统发生内部错误 THEN THE Bot SHALL 向用户发送友好的错误消息而不是技术细节
3. WHEN 用户输入无效参数 THEN THE Bot SHALL 指出具体哪个参数无效及原因
4. WHEN Telegram_API 调用失败 THEN THE Bot SHALL 重试最多 3 次后向用户报告失败
5. WHEN Database 连接失败 THEN THE Bot SHALL 拒绝所有操作并提示系统维护中
