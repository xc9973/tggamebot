"""
骰宝键盘构建器
构建 Telegram Inline Keyboard 用于骰宝游戏按钮交互
"""
from typing import Tuple, List, Optional
from enum import Enum
from telegram import InlineKeyboardButton, InlineKeyboardMarkup
from src.models import SicBoBet, BetType


class SicBoAction(Enum):
    """骰宝按钮动作类型"""
    SINGLE = "single"   # 单一数字
    BIG = "big"         # 大
    SMALL = "small"     # 小
    SUM = "sum"         # 总和
    ROLL = "roll"       # 开骰子
    MYBETS = "mybets"   # 我的押注


class SicBoKeyboardBuilder:
    """骰宝键盘构建器"""
    
    # 固定下注金额
    FIXED_BET_AMOUNT = 100
    
    # 回调数据前缀
    CALLBACK_PREFIX = "sicbo_"
    
    # 单一数字
    SINGLE_NUMBERS = [1, 2, 3, 4, 5, 6]
    
    # 大小选项
    BIG_SMALL = [("大", "big"), ("小", "small")]
    
    # 总和按钮 (按赔率分组)
    SUM_HIGH_ODDS = [(4, "60:1"), (5, "30:1"), (6, "17:1"), (15, "17:1"), (16, "30:1"), (17, "60:1")]
    SUM_MED_ODDS = [(7, "12:1"), (8, "8:1"), (13, "8:1"), (14, "12:1")]
    SUM_LOW_ODDS = [(9, "6:1"), (10, "6:1"), (11, "6:1"), (12, "6:1")]
    
    @staticmethod
    def encode_callback(action: str, param: str = "") -> str:
        """
        编码回调数据
        
        Args:
            action: 动作类型 (single, big, small, sum, roll, mybets)
            param: 参数 (如数字)
            
        Returns:
            编码后的回调数据字符串
        """
        if param:
            return f"{SicBoKeyboardBuilder.CALLBACK_PREFIX}{action}_{param}"
        return f"{SicBoKeyboardBuilder.CALLBACK_PREFIX}{action}"
    
    @staticmethod
    def decode_callback(data: str) -> Tuple[str, str]:
        """
        解码回调数据
        
        Args:
            data: 回调数据字符串
            
        Returns:
            (action, param) 元组
        """
        if not data.startswith(SicBoKeyboardBuilder.CALLBACK_PREFIX):
            return "", ""
        
        # 移除前缀
        content = data[len(SicBoKeyboardBuilder.CALLBACK_PREFIX):]
        
        # 分割 action 和 param
        parts = content.split("_", 1)
        action = parts[0]
        param = parts[1] if len(parts) > 1 else ""
        
        return action, param
    
    @classmethod
    def build_main_panel(cls) -> InlineKeyboardMarkup:
        """
        构建主下注面板
        
        Returns:
            InlineKeyboardMarkup 对象
        """
        keyboard = []
        
        # 单一数字行：[1] [2] [3] [4] [5] [6]
        single_row = [
            InlineKeyboardButton(
                str(num),
                callback_data=cls.encode_callback("single", str(num))
            )
            for num in cls.SINGLE_NUMBERS
        ]
        keyboard.append(single_row)
        
        # 大小行：[大] [小]
        big_small_row = [
            InlineKeyboardButton(
                label,
                callback_data=cls.encode_callback(action)
            )
            for label, action in cls.BIG_SMALL
        ]
        keyboard.append(big_small_row)
        
        # 高赔率总和行：[4(60:1)] [5(30:1)] [6(17:1)] [15(17:1)] [16(30:1)] [17(60:1)]
        high_odds_row = [
            InlineKeyboardButton(
                f"{num}({odds})",
                callback_data=cls.encode_callback("sum", str(num))
            )
            for num, odds in cls.SUM_HIGH_ODDS
        ]
        keyboard.append(high_odds_row)
        
        # 中赔率总和行：[7(12:1)] [8(8:1)] [13(8:1)] [14(12:1)]
        med_odds_row = [
            InlineKeyboardButton(
                f"{num}({odds})",
                callback_data=cls.encode_callback("sum", str(num))
            )
            for num, odds in cls.SUM_MED_ODDS
        ]
        keyboard.append(med_odds_row)
        
        # 低赔率总和行：[9(6:1)] [10(6:1)] [11(6:1)] [12(6:1)]
        low_odds_row = [
            InlineKeyboardButton(
                f"{num}({odds})",
                callback_data=cls.encode_callback("sum", str(num))
            )
            for num, odds in cls.SUM_LOW_ODDS
        ]
        keyboard.append(low_odds_row)
        
        # 操作行：[🎲 开骰子] [我的押注]
        action_row = [
            InlineKeyboardButton(
                "🎲 开骰子",
                callback_data=cls.encode_callback("roll")
            ),
            InlineKeyboardButton(
                "我的押注",
                callback_data=cls.encode_callback("mybets")
            )
        ]
        keyboard.append(action_row)
        
        return InlineKeyboardMarkup(keyboard)
    
    @staticmethod
    def format_panel_message(
        remaining_time: int,
        player_count: int,
        total_bet_amount: int
    ) -> str:
        """
        格式化下注面板消息
        
        显示游戏状态、剩余时间、参与人数、总下注金额
        
        Args:
            remaining_time: 剩余下注时间（秒）
            player_count: 参与人数
            total_bet_amount: 总下注金额
            
        Returns:
            格式化的面板消息文本
            
        Requirements: 6.1, 6.2
        """
        msg = "🎲 骰宝 - 下注中\n"
        msg += f"⏰ 剩余 {remaining_time} 秒 | 👥 {player_count} 人 | 💰 {total_bet_amount}\n"
        msg += "\n"
        msg += f"点击按钮下注 (每次 {SicBoKeyboardBuilder.FIXED_BET_AMOUNT} 金币)"
        return msg

    @staticmethod
    def _get_bet_type_display(bet: SicBoBet) -> str:
        """
        获取押注类型的显示名称
        
        Args:
            bet: 押注对象
            
        Returns:
            押注类型的中文显示名称
        """
        if bet.bet_type == BetType.SINGLE:
            return f"单一数字 {bet.numbers[0]}"
        elif bet.bet_type == BetType.PAIR:
            return f"组合 {bet.numbers[0]}-{bet.numbers[1]}"
        elif bet.bet_type == BetType.SUM:
            return f"总和 {bet.numbers[0]}"
        elif bet.bet_type == BetType.BIG:
            return "大"
        elif bet.bet_type == BetType.SMALL:
            return "小"
        return "未知"
    
    @classmethod
    def format_my_bets(cls, bets: List[SicBoBet]) -> str:
        """
        格式化用户的押注详情列表
        
        Args:
            bets: 用户的押注列表
            
        Returns:
            格式化的押注详情文本
            
        Requirements: 6.4
        """
        if not bets:
            return "您还没有下注"
        
        msg = "📋 您的押注:\n"
        msg += "━━━━━━━━━━━━━━━\n"
        
        total_amount = 0
        for bet in bets:
            bet_type_name = cls._get_bet_type_display(bet)
            msg += f"• {bet_type_name}: {bet.amount} 金币\n"
            total_amount += bet.amount
        
        msg += "━━━━━━━━━━━━━━━\n"
        msg += f"💰 总计: {total_amount} 金币"
        
        return msg

    @staticmethod
    def format_settlement_message(
        dice_results: List[int],
        player_results: dict,
        is_triple: bool = False
    ) -> str:
        """
        格式化结算消息
        
        显示骰子结果、每个玩家的用户名和胜负金额
        使用 🎉 和 😢 emoji 区分胜负
        
        Args:
            dice_results: 三个骰子的结果列表
            player_results: 玩家结果字典 {user_id: {'username': str, 'total_bet': int, 'total_payout': int}}
            is_triple: 是否为围骰
            
        Returns:
            格式化的结算消息文本
            
        Requirements: 7.6, 7.7
        """
        dice_str = " ".join([f"🎲{d}" for d in dice_results])
        total = sum(dice_results)
        
        msg = "🎰 骰宝结算\n"
        msg += "━━━━━━━━━━━━━━━\n"
        msg += f"骰子: {dice_str} = {total}"
        
        if is_triple:
            msg += " (围骰)\n"
        else:
            msg += f" ({'大' if total >= 11 else '小'})\n"
        
        msg += "━━━━━━━━━━━━━━━\n"
        
        if not player_results:
            msg += "本局无人下注\n"
        else:
            for user_id, result in player_results.items():
                username = result.get('username', str(user_id))
                # 确保 username 以 @ 开头显示
                display_name = f"@{username}" if username and not username.startswith('@') else username
                
                net = result['total_payout'] - result['total_bet']
                if net > 0:
                    msg += f"🎉 {display_name} +{net}\n"
                elif net < 0:
                    msg += f"😢 {display_name} {net}\n"
                else:
                    msg += f"😐 {display_name} ±0\n"
        
        msg += "━━━━━━━━━━━━━━━\n"
        msg += "游戏结束"
        
        return msg
