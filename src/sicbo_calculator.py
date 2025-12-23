"""
骰宝赔率计算器
负责计算各种押注类型的赔付
"""
from typing import List
from src.models import SicBoBet, BetType


class SicBoCalculator:
    """骰宝赔率计算器"""
    
    # 总和赔率表（赔率为 N:1，返还 = bet * (N + 1)）
    SUM_PAYOUTS = {
        4: 61, 17: 61,   # 60:1 赔率，返还 bet * 61
        5: 31, 16: 31,   # 30:1 赔率，返还 bet * 31
        6: 18, 15: 18,   # 17:1 赔率，返还 bet * 18
        7: 13, 14: 13,   # 12:1 赔率，返还 bet * 13
        8: 9, 13: 9,     # 8:1 赔率，返还 bet * 9
        9: 7, 12: 7,     # 6:1 赔率，返还 bet * 7
        10: 7, 11: 7,    # 6:1 赔率，返还 bet * 7
    }
    
    @staticmethod
    def is_triple(dice: List[int]) -> bool:
        """
        判断是否为围骰（三个相同）
        
        Args:
            dice: 三个骰子的结果列表
            
        Returns:
            True 如果三个骰子点数相同，否则 False
        """
        if len(dice) != 3:
            return False
        return dice[0] == dice[1] == dice[2]
    
    @staticmethod
    def calculate_single_payout(
        bet_number: int,
        dice: List[int],
        bet_amount: int
    ) -> int:
        """
        计算单一数字押注的赔付
        
        规则:
        - 0个匹配: 返回 0（输掉本金）
        - 1个匹配: 返回 bet * 2（1:1 赔率）
        - 2个匹配: 返回 bet * 3（2:1 赔率）
        - 3个匹配: 返回 bet * 4（3:1 赔率）
        
        Args:
            bet_number: 押注的数字 (1-6)
            dice: 三个骰子的结果列表
            bet_amount: 押注金额
            
        Returns:
            总返还金额（0 表示输）
        """
        match_count = dice.count(bet_number)
        if match_count == 0:
            return 0
        # 1匹配返回2倍，2匹配返回3倍，3匹配返回4倍
        return bet_amount * (match_count + 1)
    
    @staticmethod
    def calculate_pair_payout(
        numbers: List[int],
        dice: List[int],
        bet_amount: int
    ) -> int:
        """
        计算两个数字组合押注的赔付
        
        规则:
        - 骰子包含两个押注数字: 返回 bet * 6（5:1 赔率）
        - 否则: 返回 0
        - 重复数字不多次计算
        
        Args:
            numbers: 押注的两个数字 [num1, num2]
            dice: 三个骰子的结果列表
            bet_amount: 押注金额
            
        Returns:
            总返还金额
        """
        if len(numbers) != 2:
            return 0
        
        num1, num2 = numbers[0], numbers[1]
        
        # 检查骰子是否同时包含两个数字
        if num1 in dice and num2 in dice:
            return bet_amount * 6
        return 0
    
    @staticmethod
    def calculate_sum_payout(
        target_sum: int,
        dice: List[int],
        bet_amount: int
    ) -> int:
        """
        计算总和押注的赔付
        
        规则:
        - 围骰时返回 0（庄家通吃）
        - 总和匹配时按赔率表返回
        - 不匹配返回 0
        
        Args:
            target_sum: 押注的总和 (4-17)
            dice: 三个骰子的结果列表
            bet_amount: 押注金额
            
        Returns:
            总返还金额
        """
        # 围骰时庄家通吃
        if SicBoCalculator.is_triple(dice):
            return 0
        
        actual_sum = sum(dice)
        if actual_sum == target_sum:
            multiplier = SicBoCalculator.SUM_PAYOUTS.get(target_sum, 0)
            return bet_amount * multiplier
        return 0
    
    @staticmethod
    def calculate_big_small_payout(
        is_big: bool,
        dice: List[int],
        bet_amount: int
    ) -> int:
        """
        计算大小押注的赔付
        
        规则:
        - 围骰时返回 0（庄家通吃）
        - 大 (11-17) 匹配时返回 bet * 2（1:1 赔率）
        - 小 (4-10) 匹配时返回 bet * 2（1:1 赔率）
        - 不匹配返回 0
        
        Args:
            is_big: True 表示押大，False 表示押小
            dice: 三个骰子的结果列表
            bet_amount: 押注金额
            
        Returns:
            总返还金额
        """
        # 围骰时庄家通吃
        if SicBoCalculator.is_triple(dice):
            return 0
        
        total = sum(dice)
        
        if is_big:
            # 大: 11-17
            if 11 <= total <= 17:
                return bet_amount * 2
        else:
            # 小: 4-10
            if 4 <= total <= 10:
                return bet_amount * 2
        
        return 0
    
    @staticmethod
    def calculate_bet_payout(bet: SicBoBet, dice: List[int]) -> int:
        """
        计算任意押注的赔付（统一入口）
        
        Args:
            bet: 押注对象
            dice: 三个骰子的结果列表
            
        Returns:
            总返还金额
        """
        if bet.bet_type == BetType.SINGLE:
            return SicBoCalculator.calculate_single_payout(
                bet.numbers[0], dice, bet.amount
            )
        elif bet.bet_type == BetType.PAIR:
            return SicBoCalculator.calculate_pair_payout(
                bet.numbers, dice, bet.amount
            )
        elif bet.bet_type == BetType.SUM:
            return SicBoCalculator.calculate_sum_payout(
                bet.numbers[0], dice, bet.amount
            )
        elif bet.bet_type == BetType.BIG:
            return SicBoCalculator.calculate_big_small_payout(
                True, dice, bet.amount
            )
        elif bet.bet_type == BetType.SMALL:
            return SicBoCalculator.calculate_big_small_payout(
                False, dice, bet.amount
            )
        return 0
