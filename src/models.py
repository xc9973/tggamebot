"""
数据模型
定义系统中使用的数据类
"""
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional, List
import time


# ============ 骰宝游戏枚举和数据类 ============

class GamePhase(Enum):
    """骰宝游戏阶段"""
    IDLE = "idle"           # 空闲状态
    BETTING = "betting"     # 下注阶段
    ROLLING = "rolling"     # 开骰子阶段
    SETTLING = "settling"   # 结算阶段


class BetType(Enum):
    """骰宝押注类型"""
    SINGLE = "single"   # 单一数字
    PAIR = "pair"       # 两个数字组合
    SUM = "sum"         # 数字总和
    BIG = "big"         # 大
    SMALL = "small"     # 小


@dataclass
class SicBoBet:
    """骰宝押注模型"""
    user_id: int                    # 用户 ID
    bet_type: BetType               # 押注类型
    amount: int                     # 押注金额
    numbers: List[int]              # 押注的数字（单一数字或组合）
    created_at: float               # 创建时间
    username: str = ""              # 用户名（用于显示）


@dataclass
class SicBoGame:
    """骰宝游戏会话模型"""
    chat_id: int                                            # 群组 ID
    phase: GamePhase                                        # 游戏阶段
    bets: List[SicBoBet] = field(default_factory=list)      # 所有押注
    dice_results: List[int] = field(default_factory=list)   # 三个骰子结果
    created_at: float = 0.0                                 # 创建时间
    betting_end_time: float = 0.0                           # 下注结束时间
    panel_message_id: Optional[int] = None                  # 面板消息 ID（用于后续更新）


@dataclass
class User:
    """用户模型"""
    telegram_id: int
    username: str
    balance: int
    last_daily_claim: int
    created_at: int
    updated_at: int
    
    @classmethod
    def from_dict(cls, data: dict) -> 'User':
        """从字典创建 User 对象"""
        return cls(
            telegram_id=data['telegram_id'],
            username=data['username'],
            balance=data['balance'],
            last_daily_claim=data['last_daily_claim'],
            created_at=data['created_at'],
            updated_at=data['updated_at']
        )


@dataclass
class Transaction:
    """交易记录模型"""
    id: int
    user_id: int
    amount: int
    type: str
    description: Optional[str]
    created_at: int
    
    @classmethod
    def from_dict(cls, data: dict) -> 'Transaction':
        """从字典创建 Transaction 对象"""
        return cls(
            id=data['id'],
            user_id=data['user_id'],
            amount=data['amount'],
            type=data['type'],
            description=data.get('description'),
            created_at=data['created_at']
        )


@dataclass
class BlackjackGame:
    """21点游戏会话模型"""
    user_id: int
    bet: int
    player_cards: List[int] = field(default_factory=list)
    dealer_cards: List[int] = field(default_factory=list)
    is_finished: bool = False
    created_at: float = field(default_factory=time.time)
    
    @classmethod
    def from_dict(cls, data: dict) -> 'BlackjackGame':
        """从字典创建 BlackjackGame 对象"""
        return cls(
            user_id=data['user_id'],
            bet=data['bet'],
            player_cards=data.get('player_cards', []),
            dealer_cards=data.get('dealer_cards', []),
            is_finished=data.get('is_finished', False),
            created_at=data.get('created_at', time.time())
        )
