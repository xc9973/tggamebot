# 设计文档

## 概述

骰宝(Sic Bo)游戏模块是 Telegram 游戏机器人的扩展功能，实现多人参与的骰子游戏。系统采用状态机模式管理游戏会话，支持多种下注类型和复杂的赔率计算。游戏在群组级别进行管理，确保同一时间只有一场游戏进行。

**技术栈:**
- Python 3.10+
- python-telegram-bot v20+ (async/await)
- SQLite 3 with WAL mode
- asyncio for concurrency
- Hypothesis for property-based testing

**设计原则:**
- 复用现有架构：继承现有的 DatabaseManager、UserRepository 等组件
- 状态机模式：清晰的游戏状态转换
- 多人并发：支持多玩家同时下注
- 公平性：围骰规则确保庄家优势

## 架构

骰宝游戏模块集成到现有的三层架构中：

```
┌─────────────────────────────────────┐
│     Telegram Bot API (外部)         │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Handler Layer (处理层)          │
│  - SicBo Command Handlers           │
│  - Bet Command Handlers             │
│  - Callback Query Handlers          │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│     Business Logic Layer (业务层)    │
│  - SicBoManager (游戏管理)           │
│  - SicBoCalculator (赔率计算)        │
│  - Account Management (复用)         │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Data Layer (数据层)             │
│  - SicBoBetRepository (押注存储)     │
│  - Database Manager (复用)           │
│  - User Repository (复用)            │
└─────────────────────────────────────┘
```

**游戏状态机:**

```
┌─────────┐    /sicbo    ┌─────────────┐
│  IDLE   │─────────────▶│   BETTING   │
└─────────┘              └──────┬──────┘
     ▲                          │
     │                   timeout/roll
     │                          │
     │                   ┌──────▼──────┐
     │                   │   ROLLING   │
     │                   └──────┬──────┘
     │                          │
     │                    dice result
     │                          │
     │                   ┌──────▼──────┐
     └───────────────────│  SETTLING   │
         game ended      └─────────────┘
```

## 组件和接口

### 1. SicBoGame 数据类

游戏会话的数据结构。

```python
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional

class GamePhase(Enum):
    IDLE = "idle"
    BETTING = "betting"
    ROLLING = "rolling"
    SETTLING = "settling"

class BetType(Enum):
    SINGLE = "single"      # 单一数字
    PAIR = "pair"          # 两个数字组合
    SUM = "sum"            # 数字总和
    BIG = "big"            # 大
    SMALL = "small"        # 小

@dataclass
class SicBoBet:
    user_id: int
    bet_type: BetType
    amount: int
    numbers: list[int]     # 押注的数字（单一数字或组合）
    created_at: float

@dataclass
class SicBoGame:
    chat_id: int
    phase: GamePhase
    bets: list[SicBoBet] = field(default_factory=list)
    dice_results: list[int] = field(default_factory=list)  # 三个骰子结果
    created_at: float = 0.0
    betting_end_time: float = 0.0


### 2. SicBoCalculator 类

负责赔率计算的纯函数类。

```python
class SicBoCalculator:
    # 总和赔率表
    SUM_PAYOUTS = {
        4: 60, 17: 60,
        5: 30, 16: 30,
        6: 17, 15: 17,
        7: 12, 14: 12,
        8: 8, 13: 8,
        9: 6, 12: 6,
        10: 6, 11: 6,
    }
    
    @staticmethod
    def is_triple(dice: list[int]) -> bool:
        """判断是否为围骰（三个相同）"""
        pass
    
    @staticmethod
    def calculate_single_payout(
        bet_number: int, 
        dice: list[int], 
        bet_amount: int
    ) -> int:
        """
        计算单一数字押注的赔付
        返回: 总返还金额（0 表示输，包含本金的返还）
        """
        pass
    
    @staticmethod
    def calculate_pair_payout(
        numbers: list[int], 
        dice: list[int], 
        bet_amount: int
    ) -> int:
        """
        计算两个数字组合押注的赔付
        返回: 总返还金额
        """
        pass
    
    @staticmethod
    def calculate_sum_payout(
        target_sum: int, 
        dice: list[int], 
        bet_amount: int
    ) -> int:
        """
        计算总和押注的赔付
        围骰时返回 0
        返回: 总返还金额
        """
        pass
    
    @staticmethod
    def calculate_big_small_payout(
        is_big: bool, 
        dice: list[int], 
        bet_amount: int
    ) -> int:
        """
        计算大小押注的赔付
        围骰时返回 0
        返回: 总返还金额
        """
        pass
    
    @staticmethod
    def calculate_bet_payout(bet: SicBoBet, dice: list[int]) -> int:
        """
        计算任意押注的赔付
        返回: 总返还金额
        """
        pass
```

### 3. SicBoManager 类

管理游戏会话和业务逻辑。

```python
class SicBoManager:
    def __init__(
        self, 
        account_mgr: AccountManager, 
        tx_repo: TransactionRepository
    ):
        self.active_games: dict[int, SicBoGame] = {}  # chat_id -> game
        self.calculator = SicBoCalculator()
    
    async def start_game(self, chat_id: int) -> tuple[bool, str]:
        """
        开始新游戏
        返回: (成功, 消息)
        """
        pass
    
    async def place_bet(
        self, 
        chat_id: int, 
        user_id: int, 
        bet_type: BetType, 
        amount: int, 
        numbers: list[int] = None
    ) -> tuple[bool, str]:
        """
        下注
        返回: (成功, 消息)
        """
        pass
    
    async def roll_dice(self, chat_id: int) -> tuple[bool, list[int], str]:
        """
        开骰子
        返回: (成功, 骰子结果, 消息)
        """
        pass
    
    async def settle_game(self, chat_id: int) -> tuple[bool, dict[int, int], str]:
        """
        结算游戏
        返回: (成功, {user_id: 净收益}, 消息)
        """
        pass
    
    def get_game(self, chat_id: int) -> Optional[SicBoGame]:
        """获取当前游戏"""
        pass
    
    def get_user_bets(self, chat_id: int, user_id: int) -> list[SicBoBet]:
        """获取用户在当前游戏的所有押注"""
        pass
    
    def get_game_stats(self, chat_id: int) -> dict:
        """获取游戏统计（参与人数、总下注等）"""
        pass
```

### 4. Command Handlers

处理 Telegram 命令。

```python
async def sicbo_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /sicbo 命令 - 开始新游戏"""
    pass

async def bet_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """
    处理 /bet 命令 - 下注
    格式:
    - /bet single <数字> <金额>
    - /bet pair <数字1> <数字2> <金额>
    - /bet sum <总和> <金额>
    - /bet big <金额>
    - /bet small <金额>
    """
    pass

async def roll_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /roll 命令 - 开骰子（结束下注阶段）"""
    pass

async def sicbo_status_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /sicbo_status 命令 - 查看游戏状态"""
    pass

async def mybets_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """处理 /mybets 命令 - 查看我的押注"""
    pass
```

## 数据模型

### SicBo Bets 表（可选持久化）

游戏押注可以只存储在内存中，但为了审计可以持久化。

```sql
CREATE TABLE sicbo_bets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    bet_type TEXT NOT NULL,
    amount INTEGER NOT NULL,
    numbers TEXT,  -- JSON array
    created_at INTEGER NOT NULL,
    settled_at INTEGER,
    payout INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(telegram_id)
);

CREATE INDEX idx_sicbo_chat ON sicbo_bets(chat_id, created_at DESC);
CREATE INDEX idx_sicbo_user ON sicbo_bets(user_id, created_at DESC);
```

### 内存数据结构

游戏会话存储在内存中，使用 `dict[int, SicBoGame]` 按 chat_id 索引。

## 正确性属性

*属性是一个特征或行为，应该在系统的所有有效执行中保持为真——本质上是关于系统应该做什么的形式化陈述。属性作为人类可读规范和机器可验证正确性保证之间的桥梁。*

### 游戏会话属性

**属性 1: 游戏会话互斥性**
*对于任何* 群组，在任意时刻最多只能有一个进行中的骰宝游戏会话；当已有游戏进行时，创建新游戏的请求应该被拒绝
**验证需求: 1.1, 1.2**

**属性 2: 游戏状态转换正确性**
*对于任何* 骰宝游戏会话，状态转换必须遵循 IDLE → BETTING → ROLLING → SETTLING → IDLE 的顺序，不允许跳过或逆向转换
**验证需求: 1.5, 6.5**

### 下注验证属性

**属性 3: 下注输入验证**
*对于任何* 下注请求，如果数字参数超出有效范围（单一数字1-6，组合数字1-6且不相等，总和4-17），应该被拒绝并返回错误消息
**验证需求: 2.2, 3.2, 3.3, 4.2**

**属性 4: 下注前置条件验证**
*对于任何* 下注请求，如果余额不足、金额非正或不在下注阶段，应该被拒绝并返回相应错误消息
**验证需求: 7.1, 7.2, 7.3**

**属性 5: 下注余额扣除原子性**
*对于任何* 成功的下注，玩家账户余额应该立即减少下注金额，且下注记录应该被正确保存
**验证需求: 2.1, 3.1, 4.1, 5.1, 5.2, 7.4**

### 赔率计算属性

**属性 6: 单一数字赔率正确性**
*对于任何* 单一数字押注和骰子结果组合，赔付应该符合规则：0个匹配返回0，1个匹配返回bet*2，2个匹配返回bet*3，3个匹配返回bet*4
**验证需求: 2.3, 2.4, 2.5, 2.6**

**属性 7: 两个数字组合赔率正确性**
*对于任何* 两个数字组合押注和骰子结果，如果骰子包含两个押注数字则返回bet*6，否则返回0；重复数字不多次计算
**验证需求: 3.4, 3.5, 3.6**

**属性 8: 总和赔率正确性**
*对于任何* 总和押注和骰子结果，如果是围骰则返回0；否则如果总和匹配则按赔率表返回奖金，不匹配返回0
**验证需求: 4.3, 4.4, 4.5**

**属性 9: 大小赔率正确性**
*对于任何* 大小押注和骰子结果，如果是围骰则返回0；否则大(11-17)或小(4-10)匹配时返回bet*2，不匹配返回0
**验证需求: 5.3, 5.4, 5.5**

### 结算属性

**属性 10: 多押注结算正确性**
*对于任何* 游戏结算，每个玩家的每个押注应该独立计算赔付，玩家的总收益等于所有押注赔付之和减去所有押注金额
**验证需求: 6.3, 6.6**

**属性 11: 围骰通吃规则**
*对于任何* 围骰结果（三个骰子相同），所有总和押注和大小押注应该返回0（庄家通吃）
**验证需求: 4.4, 5.5**

### 辅助函数属性

**属性 12: 围骰判定正确性**
*对于任何* 三个骰子结果，当且仅当三个骰子点数完全相同时，is_triple 应该返回 True
**验证需求: 4.4, 5.5**

## 错误处理

### 用户输入错误

**命令格式错误:**
- 解析失败时返回使用说明
- 示例: "/bet 需要指定类型和金额。用法: /bet single 3 100"

**参数验证错误:**
- 数字超出范围: "数字必须在 1-6 之间"
- 总和超出范围: "总和必须在 4-17 之间（3和18不可押注）"
- 组合数字相同: "两个数字必须不同"
- 金额非正: "下注金额必须大于 0"
- 余额不足: "余额不足，当前余额: X"

### 游戏状态错误

**非下注阶段下注:**
- "当前不在下注阶段，请等待新游戏开始"

**重复开始游戏:**
- "当前已有进行中的游戏，请等待游戏结束"

**无游戏时查询:**
- "当前没有进行中的骰宝游戏"

### 系统错误

**数据库错误:**
- 回滚事务
- 返回: "系统暂时不可用，请稍后再试"

**并发冲突:**
- 使用群组级锁防止并发问题
- 重试失败后返回错误消息

## 测试策略

### 单元测试

单元测试用于验证特定示例、边界情况和错误条件：

**赔率计算测试:**
- 测试 is_triple 函数的各种输入
- 测试单一数字赔率的边界情况（0/1/2/3个匹配）
- 测试组合赔率的边界情况（包含/不包含，重复数字）
- 测试总和赔率表的所有值
- 测试大小判定的边界（10/11）

**游戏状态测试:**
- 测试状态转换的正确性
- 测试非法状态转换的拒绝
- 测试游戏超时处理

**输入验证测试:**
- 测试各种无效输入的拒绝
- 测试边界值（0, 负数, 超大数）

### 属性测试

属性测试用于验证通用属性在所有输入下都成立，使用 **Hypothesis** 库：

**配置:**
- 每个属性测试运行最少 100 次迭代
- 使用随机生成的测试数据
- 每个测试标注对应的设计属性编号

**测试标注格式:**
```python
# Feature: sic-bo-game, Property 6: 单一数字赔率正确性
@given(
    bet_number=st.integers(min_value=1, max_value=6),
    dice=st.lists(st.integers(min_value=1, max_value=6), min_size=3, max_size=3),
    bet_amount=st.integers(min_value=1, max_value=10000)
)
def test_single_number_payout_correctness(bet_number, dice, bet_amount):
    # 测试实现
    pass
```

**生成器策略:**
- `dice`: 三个 1-6 的随机整数列表
- `bet_number`: 1-6 的随机整数
- `pair_numbers`: 两个不同的 1-6 随机整数
- `target_sum`: 4-17 的随机整数
- `bet_amount`: 1-10000 的随机正整数
- `is_big`: 随机布尔值

**关键属性测试:**
1. 游戏会话互斥性
2. 下注输入验证
3. 单一数字赔率正确性
4. 组合赔率正确性
5. 总和赔率正确性
6. 大小赔率正确性
7. 围骰通吃规则
8. 多押注结算正确性

### 集成测试

**端到端流程测试:**
- 开始游戏 → 多人下注 → 开骰子 → 结算 → 验证余额
- 测试围骰情况下的结算
- 测试同一玩家多种押注的结算

**并发测试:**
- 多人同时下注
- 下注和开骰子的竞态条件

### 测试覆盖率目标

- 代码覆盖率: > 85%
- 分支覆盖率: > 80%
- 所有正确性属性都有对应的属性测试
- 所有边界情况都有单元测试覆盖
