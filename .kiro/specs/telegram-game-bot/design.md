# 设计文档

## 概述

Telegram 游戏机器人是一个基于 Python 的异步系统，使用 python-telegram-bot 库与 Telegram Bot API 交互，使用 SQLite 数据库存储用户数据。系统采用命令处理器模式，每个命令对应一个处理器函数，通过事务机制确保数据一致性，通过用户级锁防止并发问题。

**技术栈:**
- Python 3.10+
- python-telegram-bot v20+ (async/await)
- SQLite 3 with WAL mode
- asyncio for concurrency

**设计原则:**
- 简单优先：使用 SQLite 而非复杂数据库
- 安全第一：所有金币操作使用事务
- 用户体验：快速响应，清晰的错误提示
- 可扩展性：模块化设计便于添加新游戏

## 架构

系统采用三层架构：

```
┌─────────────────────────────────────┐
│     Telegram Bot API (外部)         │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Handler Layer (处理层)          │
│  - Command Handlers                 │
│  - Callback Query Handlers          │
│  - Error Handlers                   │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│     Business Logic Layer (业务层)    │
│  - Game Logic                       │
│  - Account Management               │
│  - Transaction Management           │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Data Layer (数据层)             │
│  - Database Manager                 │
│  - User Repository                  │
│  - Transaction Repository           │
└─────────────────────────────────────┘
```

**数据流:**
1. 用户发送命令到 Telegram
2. Handler Layer 接收并解析命令
3. Business Logic Layer 执行游戏逻辑
4. Data Layer 持久化数据变更
5. Handler Layer 返回结果给用户

## 组件和接口

### 1. Database Manager

负责数据库连接、初始化和事务管理。

```python
class DatabaseManager:
    def __init__(self, db_path: str):
        """初始化数据库连接"""
        pass
    
    async def initialize(self) -> None:
        """创建表结构，启用 WAL 模式"""
        pass
    
    async def execute(self, query: str, params: tuple) -> None:
        """执行单个查询"""
        pass
    
    async def fetch_one(self, query: str, params: tuple) -> dict:
        """查询单行数据"""
        pass
    
    async def fetch_all(self, query: str, params: tuple) -> list[dict]:
        """查询多行数据"""
        pass
    
    async def transaction(self, operations: list[callable]) -> bool:
        """执行事务，全部成功或全部回滚"""
        pass
```

### 2. User Repository

管理用户账户的 CRUD 操作。

```python
class UserRepository:
    def __init__(self, db: DatabaseManager):
        pass
    
    async def create_user(self, telegram_id: int, username: str) -> User:
        """创建新用户，初始 1000 金币"""
        pass
    
    async def get_user(self, telegram_id: int) -> User | None:
        """获取用户信息"""
        pass
    
    async def update_balance(self, telegram_id: int, amount: int) -> bool:
        """更新用户余额（可正可负）"""
        pass
    
    async def get_top_users(self, limit: int = 10) -> list[User]:
        """获取财富榜"""
        pass
    
    async def update_daily_claim(self, telegram_id: int) -> bool:
        """更新每日签到时间"""
        pass
    
    async def can_claim_daily(self, telegram_id: int) -> bool:
        """检查是否可以签到"""
        pass
```

### 3. Transaction Repository

记录所有金币变动历史。

```python
class TransactionRepository:
    def __init__(self, db: DatabaseManager):
        pass
    
    async def log_transaction(
        self, 
        user_id: int, 
        amount: int, 
        transaction_type: str, 
        description: str
    ) -> None:
        """记录交易日志"""
        pass
    
    async def get_user_history(
        self, 
        user_id: int, 
        limit: int = 50
    ) -> list[Transaction]:
        """获取用户交易历史"""
        pass
```

### 4. Account Manager

处理账户相关业务逻辑。

```python
class AccountManager:
    def __init__(self, user_repo: UserRepository, tx_repo: TransactionRepository):
        pass
    
    async def ensure_user_exists(self, telegram_id: int, username: str) -> User:
        """确保用户存在，不存在则创建"""
        pass
    
    async def get_balance(self, telegram_id: int) -> int:
        """获取余额"""
        pass
    
    async def claim_daily_reward(self, telegram_id: int) -> tuple[bool, str]:
        """领取每日奖励，返回 (成功, 消息)"""
        pass
    
    async def transfer(
        self, 
        from_id: int, 
        to_id: int, 
        amount: int
    ) -> tuple[bool, str]:
        """转账，返回 (成功, 消息)"""
        pass
```

### 5. Game Engine

处理游戏逻辑和结算。

```python
class GameEngine:
    def __init__(self, account_mgr: AccountManager, tx_repo: TransactionRepository):
        pass
    
    async def play_dice(self, user_id: int, bet: int) -> tuple[bool, int, int]:
        """
        玩骰子游戏
        返回: (成功, 骰子点数, 奖金)
        """
        pass
    
    async def play_slot(self, user_id: int, bet: int) -> tuple[bool, int, int]:
        """
        玩老虎机
        返回: (成功, 老虎机值, 奖金)
        """
        pass
    
    async def calculate_dice_payout(self, dice_value: int, bet: int) -> int:
        """计算骰子奖金"""
        pass
    
    async def calculate_slot_payout(self, slot_value: int, bet: int) -> int:
        """计算老虎机奖金"""
        pass
```

### 6. Blackjack Manager

管理 21 点游戏会话。

```python
class BlackjackManager:
    def __init__(self, account_mgr: AccountManager, tx_repo: TransactionRepository):
        self.active_games: dict[int, BlackjackGame] = {}
    
    async def start_game(self, user_id: int, bet: int) -> tuple[bool, BlackjackGame]:
        """开始新游戏"""
        pass
    
    async def hit(self, user_id: int) -> tuple[bool, BlackjackGame]:
        """要牌"""
        pass
    
    async def stand(self, user_id: int) -> tuple[bool, BlackjackGame, int]:
        """停牌并结算"""
        pass
    
    async def double_down(self, user_id: int) -> tuple[bool, BlackjackGame, int]:
        """加倍"""
        pass
    
    def get_game(self, user_id: int) -> BlackjackGame | None:
        """获取当前游戏"""
        pass
```

### 7. Command Handlers

处理 Telegram 命令。

```python
async def start_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /start 命令"""
    pass

async def balance_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /balance 命令"""
    pass

async def daily_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /daily 命令"""
    pass

async def pay_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /pay 命令"""
    pass

async def top_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /top 命令"""
    pass

async def dice_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /dice 命令"""
    pass

async def slot_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /slot 命令"""
    pass

async def blackjack_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /bj 命令"""
    pass

async def blackjack_callback_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 21 点按钮回调"""
    pass

async def admin_add_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /admin_add 命令"""
    pass

async def admin_remove_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /admin_remove 命令"""
    pass

async def admin_reset_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /admin_reset 命令"""
    pass
```

## 数据模型

### Users 表

```sql
CREATE TABLE users (
    telegram_id INTEGER PRIMARY KEY,
    username TEXT NOT NULL,
    balance INTEGER NOT NULL DEFAULT 1000,
    last_daily_claim INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX idx_balance ON users(balance DESC);
```

**字段说明:**
- `telegram_id`: Telegram 用户 ID（主键）
- `username`: Telegram 用户名
- `balance`: 当前金币余额
- `last_daily_claim`: 上次签到的 Unix 时间戳
- `created_at`: 账户创建时间
- `updated_at`: 最后更新时间

### Transactions 表

```sql
CREATE TABLE transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    type TEXT NOT NULL,
    description TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(telegram_id)
);

CREATE INDEX idx_user_transactions ON transactions(user_id, created_at DESC);
```

**字段说明:**
- `id`: 交易 ID（自增主键）
- `user_id`: 用户 ID
- `amount`: 金币变动量（正数为增加，负数为减少）
- `type`: 交易类型（`daily`, `dice`, `slot`, `blackjack`, `transfer_send`, `transfer_receive`, `admin_add`, `admin_remove`）
- `description`: 交易描述
- `created_at`: 交易时间

### Blackjack Games (内存存储)

21 点游戏会话存储在内存中，不持久化。

```python
@dataclass
class BlackjackGame:
    user_id: int
    bet: int
    player_cards: list[int]
    dealer_cards: list[int]
    is_finished: bool
    created_at: float
```

## 正确性属性

*属性是一个特征或行为，应该在系统的所有有效执行中保持为真——本质上是关于系统应该做什么的形式化陈述。属性作为人类可读规范和机器可验证正确性保证之间的桥梁。*

### 账户管理属性

**属性 1: 新用户自动创建**
*对于任何* 新的 telegram_id，当首次与系统交互时，应该自动创建账户且初始余额为 1000 金币
**验证需求: 1.1, 1.4**

**属性 2: 账户查询幂等性**
*对于任何* 用户，多次调用账户初始化或查询操作应该返回相同的账户状态
**验证需求: 1.2, 1.5**

**属性 3: 余额查询准确性**
*对于任何* 用户，查询余额应该返回数据库中存储的当前准确值
**验证需求: 1.3**

### 每日签到属性

**属性 4: 签到时间间隔验证**
*对于任何* 用户，如果距离上次签到超过 24 小时，应该允许签到并增加 500 金币；如果不足 24 小时，应该拒绝并显示剩余时间
**验证需求: 2.1, 2.2**

**属性 5: 签到时间戳更新**
*对于任何* 成功的签到操作，用户的 last_daily_claim 时间戳应该更新为当前时间
**验证需求: 2.4**

**属性 6: 签到反馈完整性**
*对于任何* 签到操作，返回的消息应该包含操作结果和获得的金币数量
**验证需求: 2.5**

### 转账系统属性

**属性 7: 转账余额变化正确性**
*对于任何* 有效的转账操作（发送者余额充足，金额为正，目标用户存在），发送者余额应该减少 amount，接收者余额应该增加 amount * 0.95
**验证需求: 3.1**

**属性 8: 转账手续费回收**
*对于任何* 转账操作，系统总金币量应该减少 amount * 0.05（手续费）
**验证需求: 3.2**

**属性 9: 转账输入验证**
*对于任何* 转账请求，如果余额不足、金额非正、目标用户不存在或目标是自己，应该拒绝并返回相应错误消息
**验证需求: 3.3, 3.4, 3.5**

**属性 10: 转账确认消息**
*对于任何* 成功的转账，应该向发送者和接收者都发送包含金额和对方信息的确认消息
**验证需求: 3.7**

### 排行榜属性

**属性 11: 排行榜排序正确性**
*对于任何* 排行榜查询，返回的用户列表应该按余额降序排列，余额相同时按创建时间升序排列
**验证需求: 4.1, 4.4**

**属性 12: 排行榜数据完整性**
*对于任何* 排行榜中的用户条目，应该包含排名、用户名和余额信息
**验证需求: 4.2**

### 骰子游戏属性

**属性 13: 骰子游戏赔率正确性**
*对于任何* 骰子游戏结果，余额变化应该符合赔率表：1-3 点输掉本金，4-5 点赢得 1 倍本金，6 点赢得 2 倍本金
**验证需求: 5.2, 5.3, 5.4**

**属性 14: 骰子游戏前置条件验证**
*对于任何* 骰子游戏请求，如果余额不足或金额非正，应该拒绝并返回错误消息
**验证需求: 5.5, 5.6**

**属性 15: 骰子游戏结果反馈**
*对于任何* 骰子游戏结束，应该显示骰子点数、输赢结果和当前余额
**验证需求: 5.7**

### 老虎机游戏属性

**属性 16: 老虎机赔率正确性**
*对于任何* 老虎机结果，余额变化应该符合赔率表：三个图案一致赢得 10 倍本金，两个图案一致赢得 2 倍本金，三个图案不一致输掉本金
**验证需求: 6.2, 6.3, 6.4**

**属性 17: 老虎机前置条件验证**
*对于任何* 老虎机游戏请求，如果余额不足或金额非正，应该拒绝并返回错误消息
**验证需求: 6.5, 6.6**

**属性 18: 老虎机结果反馈**
*对于任何* 老虎机游戏结束，应该显示老虎机结果、输赢情况和当前余额
**验证需求: 6.7**

### 21点游戏属性

**属性 19: 21点游戏初始化**
*对于任何* 有效的 21 点游戏开始请求，应该创建游戏会话，玩家获得 2 张牌，庄家获得 2 张牌（1 明 1 暗）
**验证需求: 7.1, 7.2**

**属性 20: 21点要牌操作**
*对于任何* 要牌操作，玩家手牌数量应该增加 1
**验证需求: 7.3**

**属性 21: 21点加倍操作**
*对于任何* 加倍操作（余额充足），下注金额应该翻倍，玩家获得 1 张牌后自动停牌
**验证需求: 7.5**

**属性 22: 21点爆牌判定**
*对于任何* 玩家手牌，如果点数超过 21，应该立即判定为输并扣除下注金额
**验证需求: 7.6**

**属性 23: 21点庄家逻辑**
*对于任何* 庄家回合，如果点数小于 17，应该继续要牌直到点数 >= 17
**验证需求: 7.7**

**属性 24: 21点结算正确性**
*对于任何* 21 点游戏结束，结算应该符合规则：玩家点数大于庄家且未爆牌赢得 1 倍本金，平局返还本金，其他情况输掉本金，Blackjack 赢得 1.5 倍本金
**验证需求: 7.8, 7.9, 7.10, 7.11**

### 管理员功能属性

**属性 25: 管理员金币操作**
*对于任何* 管理员的添加/扣除金币命令，目标用户余额应该相应增加或减少指定金额
**验证需求: 8.1, 8.2**

**属性 26: 管理员权限验证**
*对于任何* 管理员命令，如果调用者不是管理员，应该拒绝并提示权限不足
**验证需求: 8.3**

**属性 27: 管理员重置操作**
*对于任何* 管理员重置命令，目标用户余额应该重置为 1000，签到时间应该重置为 0
**验证需求: 8.4**

**属性 28: 管理员操作审计**
*对于任何* 管理员操作，应该在 transactions 表中记录日志，包含操作者、目标用户、操作类型和时间戳
**验证需求: 8.5**

### 数据持久化属性

**属性 29: 数据变更即时持久化**
*对于任何* 用户数据变更（余额、签到时间等），应该立即写入数据库
**验证需求: 9.1**

**属性 30: 事务原子性**
*对于任何* 涉及多个数据库操作的业务逻辑（如转账、游戏结算），要么全部成功要么全部回滚
**验证需求: 9.3, 9.5**

**属性 31: 数据库错误处理**
*对于任何* 数据库操作失败，应该回滚事务并向用户返回错误消息
**验证需求: 9.4**

### 并发控制属性

**属性 32: 用户操作串行化**
*对于任何* 单个用户的多个并发请求，应该按顺序处理，确保状态一致性
**验证需求: 10.1**

**属性 33: 游戏会话互斥**
*对于任何* 用户，如果已有进行中的游戏会话，应该拒绝新的游戏请求
**验证需求: 10.2**

**属性 34: 用户操作隔离性**
*对于任何* 多个不同用户的并发操作，应该互不影响，各自独立处理
**验证需求: 10.3**

**属性 35: 余额操作原子性**
*对于任何* 余额检查和扣除操作，应该在同一事务中完成，防止竞态条件
**验证需求: 10.4, 10.5**

### 错误处理属性

**属性 36: 命令格式错误反馈**
*对于任何* 格式错误的命令，应该返回使用说明和正确的命令格式示例
**验证需求: 11.1**

**属性 37: 参数验证反馈**
*对于任何* 无效参数，应该明确指出哪个参数无效及原因
**验证需求: 11.3**

**属性 38: API 调用重试机制**
*对于任何* Telegram API 调用失败，应该自动重试最多 3 次
**验证需求: 11.4**

## 错误处理

### 用户输入错误

**命令格式错误:**
- 捕获异常并解析命令参数
- 返回友好的错误消息和使用示例
- 示例: "/dice 需要一个数字参数。用法: /dice 100"

**参数验证错误:**
- 金额必须为正整数
- 用户名必须存在
- 余额必须充足
- 返回具体的错误原因

### 系统错误

**数据库错误:**
- 捕获所有数据库异常
- 回滚事务
- 记录错误日志
- 向用户返回: "系统暂时不可用，请稍后再试"

**Telegram API 错误:**
- 实现指数退避重试（最多 3 次）
- 记录失败的 API 调用
- 超过重试次数后向用户报告失败

**并发冲突:**
- 使用数据库锁和事务隔离
- 检测到冲突时自动重试
- 重试失败后返回错误消息

### 边界情况

**余额为 0:**
- 允许查询和签到
- 拒绝所有需要下注的操作
- 提示使用 /daily 领取奖励

**游戏会话超时:**
- 21 点游戏会话 10 分钟后自动清理
- 返还下注金额
- 通知用户游戏已取消

**数据库连接丢失:**
- 实现连接池和自动重连
- 重连失败时拒绝所有操作
- 记录错误并通知管理员

## 测试策略

### 单元测试

单元测试用于验证特定示例、边界情况和错误条件：

**数据层测试:**
- 测试数据库初始化和表创建
- 测试 CRUD 操作的正确性
- 测试事务回滚机制
- 测试并发写入冲突处理

**业务逻辑测试:**
- 测试游戏赔率计算函数
- 测试 21 点牌点数计算
- 测试转账手续费计算
- 测试权限验证逻辑

**边界情况测试:**
- 余额为 0 时的各种操作
- 金额为负数或 0 的输入
- 不存在的用户 ID
- 向自己转账
- 首次签到（last_daily_claim = 0）
- 骰子点数为 6 的特殊奖励
- 21 点 Blackjack 的特殊赔率

**错误处理测试:**
- 数据库连接失败
- Telegram API 调用失败
- 无效的命令格式
- 权限不足的操作

### 属性测试

属性测试用于验证通用属性在所有输入下都成立，使用 **Hypothesis** 库进行属性测试：

**配置:**
- 每个属性测试运行最少 100 次迭代
- 使用随机生成的测试数据
- 每个测试标注对应的设计属性编号

**测试标注格式:**
```python
# Feature: telegram-game-bot, Property 1: 新用户自动创建
@given(telegram_id=st.integers(min_value=1, max_value=999999999))
async def test_new_user_auto_creation(telegram_id):
    # 测试实现
    pass
```

**生成器策略:**
- `telegram_id`: 1 到 999999999 的随机整数
- `username`: 1-32 个字符的随机字符串
- `amount`: 1 到 1000000 的随机正整数
- `balance`: 0 到 1000000 的随机非负整数
- `timestamp`: 过去 30 天内的随机时间戳
- `dice_value`: 1-6 的随机整数
- `slot_value`: 1-64 的随机整数
- `cards`: 1-11 的随机整数列表（21 点牌值）

**关键属性测试:**
1. 账户创建和查询的幂等性
2. 转账前后总金币量的变化（手续费）
3. 游戏赔率的正确性（所有可能的结果）
4. 并发操作的数据一致性
5. 事务的原子性（全部成功或全部失败）
6. 排行榜的排序正确性
7. 签到时间间隔的验证
8. 余额操作的原子性（检查-扣除）

**并发测试:**
- 使用 asyncio 模拟多个用户同时操作
- 验证最终数据一致性
- 测试锁机制的有效性

### 集成测试

**端到端流程测试:**
- 新用户注册 → 签到 → 玩游戏 → 转账 → 查看排行榜
- 21 点完整游戏流程（开始 → 要牌 → 停牌 → 结算）
- 管理员操作流程（添加金币 → 扣除金币 → 重置账户）

**数据库集成测试:**
- 使用临时数据库进行测试
- 测试 WAL 模式的并发读写
- 测试数据库重启后的数据恢复

**Telegram API 集成测试:**
- 使用 Mock 对象模拟 Telegram API
- 验证 sendDice 调用的参数正确性
- 验证 Inline Keyboard 的构建正确性

### 测试覆盖率目标

- 代码覆盖率: > 85%
- 分支覆盖率: > 80%
- 所有正确性属性都有对应的属性测试
- 所有边界情况都有单元测试覆盖

