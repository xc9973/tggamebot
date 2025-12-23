# Design Document: SicBo Button UI

## Overview

将骰宝游戏从命令交互改为 Telegram Inline Keyboard 按钮交互。用户点击按钮即可下注，每次下注固定金额（100 金币），可多次点击累加。

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Telegram Bot API                      │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                     BotHandlers                          │
│  ┌─────────────────┐  ┌─────────────────────────────┐   │
│  │ sicbo_handler   │  │ sicbo_callback_handler      │   │
│  │ (启动游戏)       │  │ (处理按钮点击)               │   │
│  └─────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                   SicBoManager                           │
│  - start_game()      - place_bet()                      │
│  - roll_dice()       - settle_game()                    │
│  - get_game_stats()  - get_user_bets()                  │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                  SicBoCalculator                         │
│  - calculate_bet_payout()                               │
│  - is_triple()                                          │
└─────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. SicBoKeyboardBuilder (新增)

负责构建骰宝游戏的 Inline Keyboard。

```python
class SicBoKeyboardBuilder:
    """骰宝键盘构建器"""
    
    FIXED_BET_AMOUNT = 100  # 固定下注金额
    
    @staticmethod
    def build_main_panel() -> InlineKeyboardMarkup:
        """
        构建主下注面板
        
        Returns:
            InlineKeyboardMarkup 对象
        """
        pass
    
    @staticmethod
    def build_my_bets_panel(bets: List[SicBoBet]) -> str:
        """
        构建用户押注详情文本
        
        Args:
            bets: 用户的押注列表
            
        Returns:
            格式化的押注详情文本
        """
        pass
```

### 2. Callback Data 格式

按钮回调数据采用简洁格式：

```
sicbo_{action}_{param}

示例:
- sicbo_single_3     # 押单一数字 3
- sicbo_big          # 押大
- sicbo_small        # 押小
- sicbo_sum_10       # 押总和 10
- sicbo_roll         # 开骰子
- sicbo_mybets       # 查看我的押注
```

### 3. BotHandlers 扩展

新增 `sicbo_callback_handler` 处理按钮回调：

```python
async def sicbo_callback_handler(
    self, 
    update: Update, 
    context: ContextTypes.DEFAULT_TYPE
) -> None:
    """
    处理骰宝游戏的按钮回调
    
    回调数据格式: sicbo_{action}_{param}
    """
    pass
```

### 4. 面板消息格式

```
🎲 骰宝 - 下注中
⏰ 剩余 45 秒 | 👥 3 人 | 💰 1500

点击按钮下注 (每次 100 金币)
```

### 5. 结算消息格式

```
🎰 骰宝结算
━━━━━━━━━━━━━━━
骰子: 🎲3 🎲3 🎲5 = 11 (大)
━━━━━━━━━━━━━━━
🎉 @zhangsan +500
🎉 @lisi +200
😢 @wangwu -300
━━━━━━━━━━━━━━━
游戏结束
```

## Data Models

### Callback Action 枚举

```python
class SicBoAction(Enum):
    """骰宝按钮动作类型"""
    SINGLE = "single"   # 单一数字
    BIG = "big"         # 大
    SMALL = "small"     # 小
    SUM = "sum"         # 总和
    ROLL = "roll"       # 开骰子
    MYBETS = "mybets"   # 我的押注
```

### 键盘布局常量

```python
# 单一数字行
SINGLE_NUMBERS = [1, 2, 3, 4, 5, 6]

# 大小行
BIG_SMALL = [("大", "big"), ("小", "small")]

# 总和按钮 (按赔率分组)
SUM_HIGH_ODDS = [(4, "60:1"), (5, "30:1"), (6, "17:1"), (15, "17:1"), (16, "30:1"), (17, "60:1")]
SUM_MED_ODDS = [(7, "12:1"), (8, "8:1"), (13, "8:1"), (14, "12:1")]
SUM_LOW_ODDS = [(9, "6:1"), (10, "6:1"), (11, "6:1"), (12, "6:1")]
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: 固定下注金额一致性

*For any* valid betting button click (single number, big, small, or sum) when the user has sufficient balance, the bet amount recorded SHALL equal exactly 100 gold coins.

**Validates: Requirements 2.1, 3.2, 4.2, 4.3, 5.3**

### Property 2: 余额不足时不扣款

*For any* user with balance less than 100 gold coins, clicking any betting button SHALL NOT change their balance and SHALL NOT record a bet.

**Validates: Requirements 2.3, 8.4**

### Property 3: 累加下注正确性

*For any* user clicking the same betting button N times (where N ≥ 1), their total bet amount on that option SHALL equal N × 100 gold coins.

**Validates: Requirements 2.4**

### Property 4: 游戏阶段验证

*For any* button click when the game is not in BETTING phase, the click SHALL be rejected, no bet SHALL be placed, and the user's balance SHALL remain unchanged.

**Validates: Requirements 7.2, 8.1**

### Property 5: 回调数据解析往返

*For any* valid SicBoAction and parameter combination, encoding to callback data string then parsing back SHALL produce the original action and parameter.

**Validates: Requirements 2.1, 3.1, 4.1, 5.1**

## Error Handling

| 错误场景 | 处理方式 |
|---------|---------|
| 游戏不存在 | 显示 "当前没有进行中的游戏" |
| 不在下注阶段 | 显示 "下注已结束" |
| 余额不足 | 显示 "余额不足，当前余额: X" |
| 无效回调数据 | 忽略，记录日志 |
| 并发冲突 | 使用用户锁保护 |

## Testing Strategy

### Unit Tests

- 测试键盘构建器生成正确的按钮布局
- 测试回调数据解析和格式化
- 测试各种错误场景的处理

### Property-Based Tests

使用 Hypothesis 库进行属性测试：

1. **下注金额属性测试**: 生成随机用户和余额，验证下注后余额变化正确
2. **累加下注属性测试**: 生成随机点击次数，验证总下注金额正确
3. **回调数据往返测试**: 生成随机动作和参数，验证解析-格式化往返一致

### Integration Tests

- 测试完整的游戏流程：启动 → 下注 → 开骰子 → 结算
- 测试多用户并发下注场景
- 测试超时自动开骰子
