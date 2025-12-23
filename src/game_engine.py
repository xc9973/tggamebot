"""
游戏引擎
处理游戏逻辑和结算，包括骰子游戏和老虎机游戏
"""
import time
from typing import Tuple
from src.account_manager import AccountManager
from src.repositories import TransactionRepository


class GameEngine:
    """游戏引擎，处理游戏逻辑和结算"""
    
    def __init__(self, account_mgr: AccountManager, tx_repo: TransactionRepository):
        """
        初始化游戏引擎
        
        Args:
            account_mgr: 账户管理器实例
            tx_repo: 交易仓储实例
        """
        self.account_mgr = account_mgr
        self.tx_repo = tx_repo
    
    def calculate_dice_payout(self, dice_values: list, bet: int, bet_type: str, bet_number: int = None) -> int:
        """
        计算骰宝游戏奖金（三骰子）
        
        下注类型:
        - big: 大（总和11-17，三同号除外），赔率 1:1
        - small: 小（总和4-10，三同号除外），赔率 1:1
        - number: 单一数字，出现1个赔1:1，2个赔2:1，3个赔3:1
        
        Args:
            dice_values: 三个骰子的点数列表 [d1, d2, d3]
            bet: 下注金额
            bet_type: 下注类型 ('big', 'small', 'number')
            bet_number: 押注的数字（仅 number 类型需要）
            
        Returns:
            奖金金额（正数为赢，负数为输）
        """
        total = sum(dice_values)
        is_triple = dice_values[0] == dice_values[1] == dice_values[2]
        
        # 阶梯赔率系数（大额下注降低赔率）
        if bet <= 1000:
            rate_factor = 1.0
        elif bet <= 10000:
            rate_factor = 0.9
        elif bet <= 100000:
            rate_factor = 0.8
        else:
            rate_factor = 0.6
        
        if bet_type == 'big':
            # 大：11-17，三同号除外
            if is_triple:
                return -bet  # 三同号庄家通吃
            if 11 <= total <= 17:
                return int(bet * 1 * rate_factor)
            return -bet
            
        elif bet_type == 'small':
            # 小：4-10，三同号除外
            if is_triple:
                return -bet  # 三同号庄家通吃
            if 4 <= total <= 10:
                return int(bet * 1 * rate_factor)
            return -bet
            
        elif bet_type == 'number' and bet_number:
            # 单一数字：统计出现次数
            count = dice_values.count(bet_number)
            if count == 0:
                return -bet
            elif count == 1:
                return int(bet * 1 * rate_factor)
            elif count == 2:
                return int(bet * 2 * rate_factor)
            else:  # count == 3
                return int(bet * 3 * rate_factor)
        
        return -bet
    
    def decode_slot_value(self, slot_value: int) -> tuple[int, int, int]:
        """
        解码老虎机值为三个轮子的图案
        
        图案: 1=BAR, 2=葡萄, 3=柠檬, 4=七
        公式: value = 左 + (中-1)*4 + (右-1)*16
        
        Args:
            slot_value: 老虎机值 (1-64)
            
        Returns:
            (左轮, 中轮, 右轮) 图案编号
        """
        value = slot_value - 1  # 转为 0-63
        left = (value % 4) + 1
        middle = ((value // 4) % 4) + 1
        right = (value // 16) + 1
        return left, middle, right
    
    def calculate_slot_payout(self, slot_value: int, bet: int) -> int:
        """
        计算老虎机游戏奖金（按真实图案匹配 + 阶梯赔率）
        
        图案: 1=BAR, 2=葡萄, 3=柠檬, 4=七
        
        规则:
        - 三个图案完全一致: 根据下注金额给不同赔率
          - 下注 <= 1000: 赢 3 倍
          - 下注 1001-10000: 赢 2 倍
          - 下注 10001-100000: 赢 1.5 倍
          - 下注 > 100000: 赢 1 倍
        - 两个图案一致: 返还本金（不赔不赚）
        - 三个图案都不一致: 输掉本金
        
        Args:
            slot_value: 老虎机值 (1-64)
            bet: 下注金额
            
        Returns:
            奖金金额（正数为赢，负数为输，0为平）
        """
        left, middle, right = self.decode_slot_value(slot_value)
        
        # 三个一致
        if left == middle == right:
            if bet <= 1000:
                multiplier = 3.0
            elif bet <= 10000:
                multiplier = 2.0
            elif bet <= 100000:
                multiplier = 1.5
            else:
                multiplier = 1.0
            return int(bet * multiplier)
        
        # 两个一致
        if left == middle or middle == right or left == right:
            return 0  # 返还本金，不赔不赚
        
        # 都不一致
        return -bet
    
    async def play_dice(self, user_id: int, bet: int, dice_value: int, dice_value2: int = None) -> Tuple[bool, str, int]:
        """
        玩骰子游戏（双骰子版本）
        
        Args:
            user_id: 用户 ID
            bet: 下注金额
            dice_value: 第一个骰子点数 (1-6)
            dice_value2: 第二个骰子点数 (1-6)
            
        Returns:
            (成功, 消息, 奖金) 元组
        """
        # 验证：金额必须为正数
        if bet <= 0:
            return False, "下注金额必须大于 0", 0
        
        # 验证：余额是否充足
        balance = await self.account_mgr.get_balance(user_id)
        if balance < bet:
            return False, f"余额不足，当前余额：{balance}", 0
        
        # 计算奖金（双骰子）
        total = dice_value + (dice_value2 or 0)
        
        # 双骰子规则：
        # 2-6: 输掉本金
        # 7: 平局
        # 8-11: 赢得本金
        # 12: 大奖，赢 2 倍
        if total <= 6:
            payout = -bet
        elif total == 7:
            payout = 0
        elif total <= 11:
            payout = bet
        else:  # total == 12
            payout = bet * 2
        
        # 更新余额
        await self.account_mgr.user_repo.update_balance(user_id, payout)
        
        # 记录交易
        if dice_value2:
            dice_display = f"{dice_value}+{dice_value2}={total}"
        else:
            dice_display = str(dice_value)
            
        if payout > 0:
            description = f"骰子游戏获胜，点数 {dice_display}，赢得 {payout} 金币"
        elif payout == 0:
            description = f"骰子游戏平局，点数 {dice_display}，返还本金"
        else:
            description = f"骰子游戏失败，点数 {dice_display}，输掉 {abs(payout)} 金币"
        
        await self.tx_repo.log_transaction(
            user_id=user_id,
            amount=payout,
            transaction_type='dice',
            description=description
        )
        
        # 计算新余额
        new_balance = balance + payout
        
        # 构建结果消息
        if dice_value2:
            dice_msg = f"🎲🎲 点数: {dice_value} + {dice_value2} = {total}"
        else:
            dice_msg = f"🎲 骰子点数: {dice_value}"
            
        if payout > 0:
            message = f"{dice_msg}\n🎉 恭喜获胜！赢得 {payout} 金币\n💰 当前余额: {new_balance}"
        elif payout == 0:
            message = f"{dice_msg}\n😐 平局，返还本金\n💰 当前余额: {new_balance}"
        else:
            message = f"{dice_msg}\n😢 很遗憾，输掉 {abs(payout)} 金币\n💰 当前余额: {new_balance}"
        
        return True, message, payout
    
    async def play_slot(self, user_id: int, bet: int, slot_value: int) -> Tuple[bool, str, int]:
        """
        玩老虎机游戏（验证余额、扣款、结算）
        
        Args:
            user_id: 用户 ID
            bet: 下注金额
            slot_value: 老虎机值 (1-64)
            
        Returns:
            (成功, 消息, 奖金) 元组
        """
        # 验证：金额必须为正数
        if bet <= 0:
            return False, "下注金额必须大于 0", 0
        
        # 验证：余额是否充足
        balance = await self.account_mgr.get_balance(user_id)
        if balance < bet:
            return False, f"余额不足，当前余额：{balance}", 0
        
        # 解码图案
        left, middle, right = self.decode_slot_value(slot_value)
        symbols = {1: "BAR", 2: "🍇", 3: "🍋", 4: "7️⃣"}
        slot_display = f"{symbols[left]} {symbols[middle]} {symbols[right]}"
        
        # 计算奖金
        payout = self.calculate_slot_payout(slot_value, bet)
        
        # 更新余额
        await self.account_mgr.user_repo.update_balance(user_id, payout)
        
        # 记录交易
        if payout > 0:
            description = f"老虎机游戏获胜，{slot_display}，赢得 {payout} 金币"
        elif payout == 0:
            description = f"老虎机游戏平局，{slot_display}，返还本金"
        else:
            description = f"老虎机游戏失败，{slot_display}，输掉 {abs(payout)} 金币"
        
        await self.tx_repo.log_transaction(
            user_id=user_id,
            amount=payout,
            transaction_type='slot',
            description=description
        )
        
        # 计算新余额
        new_balance = balance + payout
        
        # 构建结果消息
        if payout > 0:
            message = f"🎰 {slot_display}\n🎊 大奖！三个图案一致！赢得 {payout} 金币\n💰 当前余额: {new_balance}"
        elif payout == 0:
            message = f"🎰 {slot_display}\n😐 两个图案一致，返还本金\n💰 当前余额: {new_balance}"
        else:
            message = f"🎰 {slot_display}\n😢 很遗憾，输掉 {abs(payout)} 金币\n💰 当前余额: {new_balance}"
        
        return True, message, payout
