"""
21点游戏管理器
处理21点游戏的逻辑，包括发牌、计算点数、游戏流程和结算
"""
import random
import time
from typing import Tuple, Optional, List
from src.models import BlackjackGame
from src.account_manager import AccountManager
from src.repositories import TransactionRepository


# 牌组定义：1-10 代表 A-10，11-13 代表 J, Q, K
# 在21点中，J, Q, K 都算作 10 点，A 可以是 1 或 11
CARD_VALUES = {
    1: [1, 11],  # A 可以是 1 或 11
    2: [2],
    3: [3],
    4: [4],
    5: [5],
    6: [6],
    7: [7],
    8: [8],
    9: [9],
    10: [10],
    11: [10],  # J
    12: [10],  # Q
    13: [10],  # K
}


def get_card_display(card: int) -> str:
    """
    获取牌的显示名称
    
    Args:
        card: 牌值 (1-13)
        
    Returns:
        牌的显示名称
    """
    if card == 1:
        return 'A'
    elif card == 11:
        return 'J'
    elif card == 12:
        return 'Q'
    elif card == 13:
        return 'K'
    else:
        return str(card)


def calculate_hand_value(cards: List[int]) -> int:
    """
    计算手牌点数（A 可为 1 或 11，自动选择最优值）
    
    规则：
    - 2-10 按面值计算
    - J, Q, K 都算作 10 点
    - A 可以是 1 或 11，自动选择不爆牌的最大值
    
    Args:
        cards: 手牌列表，每张牌用 1-13 表示
        
    Returns:
        手牌点数
    """
    if not cards:
        return 0
    
    # 计算非 A 牌的总点数
    total = 0
    ace_count = 0
    
    for card in cards:
        if card == 1:  # A
            ace_count += 1
        elif card >= 11:  # J, Q, K
            total += 10
        else:
            total += card
    
    # 处理 A 的点数
    # 先把所有 A 当作 11 点
    total += ace_count * 11
    
    # 如果爆牌，把 A 从 11 改为 1（每次减 10）
    while total > 21 and ace_count > 0:
        total -= 10
        ace_count -= 1
    
    return total


def is_blackjack(cards: List[int]) -> bool:
    """
    判断是否为 Blackjack（首两张牌点数为 21）
    
    Args:
        cards: 手牌列表
        
    Returns:
        是否为 Blackjack
    """
    return len(cards) == 2 and calculate_hand_value(cards) == 21


def is_bust(cards: List[int]) -> bool:
    """
    判断是否爆牌（点数超过 21）
    
    Args:
        cards: 手牌列表
        
    Returns:
        是否爆牌
    """
    return calculate_hand_value(cards) > 21


def deal_card() -> int:
    """
    发一张牌（随机 1-13）
    
    Returns:
        牌值 (1-13)
    """
    return random.randint(1, 13)


def format_hand(cards: List[int], hide_second: bool = False) -> str:
    """
    格式化手牌显示
    
    Args:
        cards: 手牌列表
        hide_second: 是否隐藏第二张牌（庄家暗牌）
        
    Returns:
        格式化的手牌字符串
    """
    if not cards:
        return "无"
    
    if hide_second and len(cards) >= 2:
        return f"{get_card_display(cards[0])} [?]"
    
    return ' '.join(get_card_display(card) for card in cards)



class BlackjackManager:
    """21点游戏管理器，管理游戏会话和游戏逻辑"""
    
    # 游戏会话超时时间（10分钟）
    SESSION_TIMEOUT = 600
    
    def __init__(self, account_mgr: AccountManager, tx_repo: TransactionRepository):
        """
        初始化21点游戏管理器
        
        Args:
            account_mgr: 账户管理器实例
            tx_repo: 交易仓储实例
        """
        self.account_mgr = account_mgr
        self.tx_repo = tx_repo
        self.active_games: dict[int, BlackjackGame] = {}
    
    def get_game(self, user_id: int) -> Optional[BlackjackGame]:
        """
        获取用户当前的游戏会话
        
        Args:
            user_id: 用户 ID
            
        Returns:
            游戏会话，如果不存在或已超时则返回 None
        """
        game = self.active_games.get(user_id)
        
        if game is None:
            return None
        
        # 检查是否超时
        if time.time() - game.created_at > self.SESSION_TIMEOUT:
            # 游戏超时，清理会话
            del self.active_games[user_id]
            return None
        
        return game
    
    async def start_game(self, user_id: int, bet: int) -> Tuple[bool, str, Optional[BlackjackGame]]:
        """
        开始新的21点游戏
        
        Args:
            user_id: 用户 ID
            bet: 下注金额
            
        Returns:
            (成功, 消息, 游戏会话) 元组
        """
        # 检查是否已有进行中的游戏
        existing_game = self.get_game(user_id)
        if existing_game is not None and not existing_game.is_finished:
            return False, "您已有进行中的游戏，请先完成当前游戏", None
        
        # 验证：金额必须为正数
        if bet <= 0:
            return False, "下注金额必须大于 0", None
        
        # 验证：余额是否充足
        balance = await self.account_mgr.get_balance(user_id)
        if balance < bet:
            return False, f"余额不足，当前余额：{balance}", None
        
        # 扣除下注金额
        await self.account_mgr.user_repo.update_balance(user_id, -bet)
        
        # 创建新游戏
        game = BlackjackGame(
            user_id=user_id,
            bet=bet,
            player_cards=[],
            dealer_cards=[],
            is_finished=False,
            created_at=time.time()
        )
        
        # 发初始牌：玩家 2 张，庄家 2 张
        game.player_cards.append(deal_card())
        game.dealer_cards.append(deal_card())
        game.player_cards.append(deal_card())
        game.dealer_cards.append(deal_card())
        
        # 保存游戏会话
        self.active_games[user_id] = game
        
        # 检查玩家是否 Blackjack
        if is_blackjack(game.player_cards):
            # 玩家 Blackjack，直接结算
            return await self._settle_blackjack(game)
        
        # 构建消息
        player_value = calculate_hand_value(game.player_cards)
        message = self._format_game_status(game, hide_dealer=True)
        message += f"\n\n您的点数: {player_value}"
        message += "\n\n请选择操作：要牌 / 停牌 / 加倍"
        
        return True, message, game
    
    async def hit(self, user_id: int) -> Tuple[bool, str, Optional[BlackjackGame]]:
        """
        要牌操作
        
        Args:
            user_id: 用户 ID
            
        Returns:
            (成功, 消息, 游戏会话) 元组
        """
        game = self.get_game(user_id)
        
        if game is None:
            return False, "没有进行中的游戏，请使用 /bj 开始新游戏", None
        
        if game.is_finished:
            return False, "游戏已结束，请使用 /bj 开始新游戏", None
        
        # 发一张牌给玩家
        game.player_cards.append(deal_card())
        
        player_value = calculate_hand_value(game.player_cards)
        
        # 检查是否爆牌
        if is_bust(game.player_cards):
            # 玩家爆牌，游戏结束
            game.is_finished = True
            
            # 记录交易
            await self.tx_repo.log_transaction(
                user_id=user_id,
                amount=-game.bet,
                transaction_type='blackjack',
                description=f'21点游戏爆牌，输掉 {game.bet} 金币'
            )
            
            # 获取新余额
            new_balance = await self.account_mgr.get_balance(user_id)
            
            message = self._format_game_status(game, hide_dealer=False)
            message += f"\n\n您的点数: {player_value}"
            message += f"\n\n💥 爆牌！您输掉了 {game.bet} 金币"
            message += f"\n💰 当前余额: {new_balance}"
            
            # 清理游戏会话
            del self.active_games[user_id]
            
            return True, message, game
        
        # 未爆牌，继续游戏
        message = self._format_game_status(game, hide_dealer=True)
        message += f"\n\n您的点数: {player_value}"
        message += "\n\n请选择操作：要牌 / 停牌"
        
        return True, message, game
    
    async def stand(self, user_id: int) -> Tuple[bool, str, Optional[BlackjackGame], int]:
        """
        停牌操作，执行庄家逻辑并结算
        
        Args:
            user_id: 用户 ID
            
        Returns:
            (成功, 消息, 游戏会话, 奖金) 元组
        """
        game = self.get_game(user_id)
        
        if game is None:
            return False, "没有进行中的游戏，请使用 /bj 开始新游戏", None, 0
        
        if game.is_finished:
            return False, "游戏已结束，请使用 /bj 开始新游戏", None, 0
        
        # 执行庄家逻辑：点数小于 17 时继续要牌
        while calculate_hand_value(game.dealer_cards) < 17:
            game.dealer_cards.append(deal_card())
        
        # 结算游戏
        return await self._settle_game(game)
    
    async def double_down(self, user_id: int) -> Tuple[bool, str, Optional[BlackjackGame], int]:
        """
        加倍操作：下注金额翻倍，发一张牌后自动停牌
        
        Args:
            user_id: 用户 ID
            
        Returns:
            (成功, 消息, 游戏会话, 奖金) 元组
        """
        game = self.get_game(user_id)
        
        if game is None:
            return False, "没有进行中的游戏，请使用 /bj 开始新游戏", None, 0
        
        if game.is_finished:
            return False, "游戏已结束，请使用 /bj 开始新游戏", None, 0
        
        # 只能在首两张牌时加倍
        if len(game.player_cards) != 2:
            return False, "只能在首两张牌时选择加倍", None, 0
        
        # 验证：余额是否充足
        balance = await self.account_mgr.get_balance(user_id)
        if balance < game.bet:
            return False, f"余额不足，无法加倍。当前余额：{balance}，需要：{game.bet}", None, 0
        
        # 扣除额外的下注金额
        await self.account_mgr.user_repo.update_balance(user_id, -game.bet)
        
        # 下注金额翻倍
        game.bet *= 2
        
        # 发一张牌
        game.player_cards.append(deal_card())
        
        player_value = calculate_hand_value(game.player_cards)
        
        # 检查是否爆牌
        if is_bust(game.player_cards):
            # 玩家爆牌，游戏结束
            game.is_finished = True
            
            # 记录交易
            await self.tx_repo.log_transaction(
                user_id=user_id,
                amount=-game.bet,
                transaction_type='blackjack',
                description=f'21点游戏加倍后爆牌，输掉 {game.bet} 金币'
            )
            
            # 获取新余额
            new_balance = await self.account_mgr.get_balance(user_id)
            
            message = self._format_game_status(game, hide_dealer=False)
            message += f"\n\n您的点数: {player_value}"
            message += f"\n\n💥 加倍后爆牌！您输掉了 {game.bet} 金币"
            message += f"\n💰 当前余额: {new_balance}"
            
            # 清理游戏会话
            del self.active_games[user_id]
            
            return True, message, game, -game.bet
        
        # 未爆牌，执行庄家逻辑并结算
        while calculate_hand_value(game.dealer_cards) < 17:
            game.dealer_cards.append(deal_card())
        
        return await self._settle_game(game)
    
    async def _settle_blackjack(self, game: BlackjackGame) -> Tuple[bool, str, Optional[BlackjackGame]]:
        """
        处理玩家 Blackjack 的结算
        
        Args:
            game: 游戏会话
            
        Returns:
            (成功, 消息, 游戏会话) 元组
        """
        game.is_finished = True
        
        # 检查庄家是否也是 Blackjack
        if is_blackjack(game.dealer_cards):
            # 双方都是 Blackjack，平局，返还本金
            await self.account_mgr.user_repo.update_balance(game.user_id, game.bet)
            
            await self.tx_repo.log_transaction(
                user_id=game.user_id,
                amount=0,
                transaction_type='blackjack',
                description='21点游戏双方 Blackjack 平局'
            )
            
            new_balance = await self.account_mgr.get_balance(game.user_id)
            
            message = self._format_game_status(game, hide_dealer=False)
            message += f"\n\n🃏 双方都是 Blackjack！平局，返还本金 {game.bet} 金币"
            message += f"\n💰 当前余额: {new_balance}"
            
            del self.active_games[game.user_id]
            return True, message, game
        
        # 玩家 Blackjack，赢得 1.5 倍本金
        payout = int(game.bet * 2.5)  # 返还本金 + 1.5 倍奖励
        await self.account_mgr.user_repo.update_balance(game.user_id, payout)
        
        winnings = int(game.bet * 1.5)
        await self.tx_repo.log_transaction(
            user_id=game.user_id,
            amount=winnings,
            transaction_type='blackjack',
            description=f'21点游戏 Blackjack，赢得 {winnings} 金币'
        )
        
        new_balance = await self.account_mgr.get_balance(game.user_id)
        
        message = self._format_game_status(game, hide_dealer=False)
        message += f"\n\n🎊 Blackjack！您赢得了 {winnings} 金币"
        message += f"\n💰 当前余额: {new_balance}"
        
        del self.active_games[game.user_id]
        return True, message, game
    
    async def _settle_game(self, game: BlackjackGame) -> Tuple[bool, str, Optional[BlackjackGame], int]:
        """
        结算游戏
        
        Args:
            game: 游戏会话
            
        Returns:
            (成功, 消息, 游戏会话, 奖金) 元组
        """
        game.is_finished = True
        
        player_value = calculate_hand_value(game.player_cards)
        dealer_value = calculate_hand_value(game.dealer_cards)
        
        payout = 0
        result_message = ""
        
        if is_bust(game.dealer_cards):
            # 庄家爆牌，玩家赢
            payout = game.bet * 2  # 返还本金 + 1 倍奖励
            result_message = f"🎉 庄家爆牌！您赢得了 {game.bet} 金币"
        elif player_value > dealer_value:
            # 玩家点数大于庄家，玩家赢
            payout = game.bet * 2  # 返还本金 + 1 倍奖励
            result_message = f"🎉 您赢了！赢得 {game.bet} 金币"
        elif player_value == dealer_value:
            # 平局，返还本金
            payout = game.bet
            result_message = f"🤝 平局！返还本金 {game.bet} 金币"
        else:
            # 玩家点数小于庄家，玩家输
            payout = 0
            result_message = f"😢 您输了，失去 {game.bet} 金币"
        
        # 更新余额
        if payout > 0:
            await self.account_mgr.user_repo.update_balance(game.user_id, payout)
        
        # 计算实际盈亏
        actual_payout = payout - game.bet  # 减去已扣除的本金
        
        # 记录交易
        if actual_payout > 0:
            await self.tx_repo.log_transaction(
                user_id=game.user_id,
                amount=actual_payout,
                transaction_type='blackjack',
                description=f'21点游戏获胜，赢得 {actual_payout} 金币'
            )
        elif actual_payout == 0:
            await self.tx_repo.log_transaction(
                user_id=game.user_id,
                amount=0,
                transaction_type='blackjack',
                description='21点游戏平局'
            )
        else:
            await self.tx_repo.log_transaction(
                user_id=game.user_id,
                amount=-game.bet,
                transaction_type='blackjack',
                description=f'21点游戏失败，输掉 {game.bet} 金币'
            )
        
        # 获取新余额
        new_balance = await self.account_mgr.get_balance(game.user_id)
        
        message = self._format_game_status(game, hide_dealer=False)
        message += f"\n\n您的点数: {player_value} | 庄家点数: {dealer_value}"
        message += f"\n\n{result_message}"
        message += f"\n💰 当前余额: {new_balance}"
        
        # 清理游戏会话
        del self.active_games[game.user_id]
        
        return True, message, game, actual_payout
    
    def _format_game_status(self, game: BlackjackGame, hide_dealer: bool = True) -> str:
        """
        格式化游戏状态显示
        
        Args:
            game: 游戏会话
            hide_dealer: 是否隐藏庄家第二张牌
            
        Returns:
            格式化的游戏状态字符串
        """
        player_hand = format_hand(game.player_cards)
        dealer_hand = format_hand(game.dealer_cards, hide_second=hide_dealer)
        
        if hide_dealer:
            dealer_value = "?"
        else:
            dealer_value = str(calculate_hand_value(game.dealer_cards))
        
        return f"🃏 21点游戏\n\n" \
               f"💰 下注: {game.bet} 金币\n\n" \
               f"👤 您的手牌: {player_hand}\n" \
               f"🏠 庄家手牌: {dealer_hand}"
