"""
骰宝游戏管理器
管理游戏会话和业务逻辑
"""
import time
from typing import Optional, Tuple, List, Dict
from src.models import SicBoGame, SicBoBet, GamePhase, BetType
from src.sicbo_calculator import SicBoCalculator
from src.account_manager import AccountManager
from src.repositories import TransactionRepository, UserRepository


class SicBoManager:
    """骰宝游戏管理器"""
    
    # 下注阶段时长（秒）
    BETTING_DURATION = 60
    
    def __init__(
        self,
        account_mgr: AccountManager,
        tx_repo: TransactionRepository
    ):
        """
        初始化骰宝游戏管理器
        
        Args:
            account_mgr: 账户管理器实例
            tx_repo: 交易仓储实例
        """
        self.account_mgr = account_mgr
        self.tx_repo = tx_repo
        self.calculator = SicBoCalculator()
        self.active_games: Dict[int, SicBoGame] = {}  # chat_id -> game
    
    async def start_game(self, chat_id: int) -> Tuple[bool, str]:
        """
        开始新游戏
        
        Args:
            chat_id: 群组 ID
            
        Returns:
            (成功, 消息) 元组
        """
        # 检查是否已有进行中的游戏（互斥性）
        if chat_id in self.active_games:
            existing_game = self.active_games[chat_id]
            if existing_game.phase != GamePhase.IDLE:
                return False, "当前已有进行中的游戏，请等待游戏结束"
        
        # 创建新游戏会话
        now = time.time()
        game = SicBoGame(
            chat_id=chat_id,
            phase=GamePhase.BETTING,
            bets=[],
            dice_results=[],
            created_at=now,
            betting_end_time=now + self.BETTING_DURATION
        )
        
        self.active_games[chat_id] = game
        
        return True, self._get_game_start_message()
    
    def _get_game_start_message(self) -> str:
        """获取游戏开始消息"""
        return """🎲 骰宝游戏开始！

📋 下注区域和赔率：
• 单一数字 (1-6): 1:1 / 2:1 / 3:1
• 两数组合: 5:1
• 总和 (4-17): 6:1 ~ 60:1
• 大 (11-17): 1:1
• 小 (4-10): 1:1

⚠️ 围骰（三个相同）时，总和和大小押注庄家通吃

📝 下注命令：
/bet single <数字> <金额>
/bet pair <数字1> <数字2> <金额>
/bet sum <总和> <金额>
/bet big <金额>
/bet small <金额>

⏰ 下注时间：60 秒"""
    
    def get_game(self, chat_id: int) -> Optional[SicBoGame]:
        """
        获取当前游戏
        
        Args:
            chat_id: 群组 ID
            
        Returns:
            游戏对象，如果不存在返回 None
        """
        return self.active_games.get(chat_id)
    
    def get_game_stats(self, chat_id: int) -> Dict:
        """
        获取游戏统计
        
        Args:
            chat_id: 群组 ID
            
        Returns:
            统计信息字典，包含参与人数和总下注金额
        """
        game = self.get_game(chat_id)
        if not game:
            return {
                "exists": False,
                "phase": None,
                "player_count": 0,
                "total_bet_amount": 0,
                "bet_count": 0,
                "remaining_time": 0
            }
        
        # 统计参与人数（去重）
        unique_players = set(bet.user_id for bet in game.bets)
        
        # 统计总下注金额
        total_amount = sum(bet.amount for bet in game.bets)
        
        # 计算剩余时间
        remaining_time = max(0, game.betting_end_time - time.time())
        
        return {
            "exists": True,
            "phase": game.phase.value,
            "player_count": len(unique_players),
            "total_bet_amount": total_amount,
            "bet_count": len(game.bets),
            "remaining_time": int(remaining_time)
        }

    
    def validate_bet_input(
        self,
        bet_type: BetType,
        numbers: List[int]
    ) -> Tuple[bool, str]:
        """
        验证下注输入
        
        Args:
            bet_type: 押注类型
            numbers: 押注的数字列表
            
        Returns:
            (有效, 错误消息) 元组
        """
        if bet_type == BetType.SINGLE:
            if len(numbers) != 1:
                return False, "单一数字押注需要指定一个数字"
            if not (1 <= numbers[0] <= 6):
                return False, "数字必须在 1-6 之间"
        
        elif bet_type == BetType.PAIR:
            if len(numbers) != 2:
                return False, "组合押注需要指定两个数字"
            if numbers[0] == numbers[1]:
                return False, "两个数字必须不同"
            if not (1 <= numbers[0] <= 6 and 1 <= numbers[1] <= 6):
                return False, "数字必须在 1-6 之间"
        
        elif bet_type == BetType.SUM:
            if len(numbers) != 1:
                return False, "总和押注需要指定一个总和值"
            if not (4 <= numbers[0] <= 17):
                return False, "总和必须在 4-17 之间（3和18不可押注）"
        
        elif bet_type in (BetType.BIG, BetType.SMALL):
            # 大小押注不需要数字参数
            pass
        
        return True, ""
    
    async def place_bet(
        self,
        chat_id: int,
        user_id: int,
        bet_type: BetType,
        amount: int,
        numbers: List[int] = None,
        username: str = ""
    ) -> Tuple[bool, str]:
        """
        下注
        
        支持同一选项多次下注累加：如果用户已经在同一选项上下注，
        则累加金额而不是创建新记录。
        
        Args:
            chat_id: 群组 ID
            user_id: 用户 ID
            bet_type: 押注类型
            amount: 押注金额
            numbers: 押注的数字列表（可选）
            username: 用户名（用于显示）
            
        Returns:
            (成功, 消息) 元组
        """
        if numbers is None:
            numbers = []
        
        # 检查游戏是否存在
        game = self.get_game(chat_id)
        if not game:
            return False, "当前没有进行中的骰宝游戏"
        
        # 检查是否在下注阶段
        if game.phase != GamePhase.BETTING:
            return False, "当前不在下注阶段，请等待新游戏开始"
        
        # 检查下注时间是否已过
        if time.time() > game.betting_end_time:
            return False, "下注时间已结束"
        
        # 验证金额
        if amount <= 0:
            return False, "下注金额必须大于 0"
        
        # 验证下注输入
        valid, error_msg = self.validate_bet_input(bet_type, numbers)
        if not valid:
            return False, error_msg
        
        # 检查余额
        balance = await self.account_mgr.get_balance(user_id)
        if balance < amount:
            return False, f"余额不足，当前余额：{balance}"
        
        # 扣除余额
        await self.account_mgr.user_repo.update_balance(user_id, -amount)
        
        # 记录交易
        bet_type_name = self._get_bet_type_name(bet_type, numbers)
        await self.tx_repo.log_transaction(
            user_id=user_id,
            amount=-amount,
            transaction_type='sicbo_bet',
            description=f'骰宝押注: {bet_type_name}'
        )
        
        # 查找是否已有相同选项的押注（累加下注）
        existing_bet = self._find_existing_bet(game, user_id, bet_type, numbers)
        
        if existing_bet:
            # 累加到现有押注
            existing_bet.amount += amount
            total_amount = existing_bet.amount
            return True, f"下注成功！{bet_type_name}，累计金额：{total_amount}"
        else:
            # 创建新押注记录
            bet = SicBoBet(
                user_id=user_id,
                bet_type=bet_type,
                amount=amount,
                numbers=numbers,
                created_at=time.time(),
                username=username
            )
            game.bets.append(bet)
            return True, f"下注成功！{bet_type_name}，金额：{amount}"
    
    def _find_existing_bet(
        self,
        game: SicBoGame,
        user_id: int,
        bet_type: BetType,
        numbers: List[int]
    ) -> Optional[SicBoBet]:
        """
        查找用户在同一选项上的现有押注
        
        Args:
            game: 游戏对象
            user_id: 用户 ID
            bet_type: 押注类型
            numbers: 押注的数字列表
            
        Returns:
            现有押注对象，如果不存在返回 None
        """
        for bet in game.bets:
            if (bet.user_id == user_id and 
                bet.bet_type == bet_type and 
                bet.numbers == numbers):
                return bet
        return None
    
    def _get_bet_type_name(self, bet_type: BetType, numbers: List[int]) -> str:
        """获取押注类型的显示名称"""
        if bet_type == BetType.SINGLE:
            return f"单一数字 {numbers[0]}"
        elif bet_type == BetType.PAIR:
            return f"组合 {numbers[0]}-{numbers[1]}"
        elif bet_type == BetType.SUM:
            return f"总和 {numbers[0]}"
        elif bet_type == BetType.BIG:
            return "大"
        elif bet_type == BetType.SMALL:
            return "小"
        return "未知"
    
    def get_user_bets(self, chat_id: int, user_id: int) -> List[SicBoBet]:
        """
        获取用户在当前游戏的所有押注
        
        Args:
            chat_id: 群组 ID
            user_id: 用户 ID
            
        Returns:
            押注列表
        """
        game = self.get_game(chat_id)
        if not game:
            return []
        
        return [bet for bet in game.bets if bet.user_id == user_id]
    
    async def roll_dice(self, chat_id: int, dice_results: List[int] = None) -> Tuple[bool, List[int], str]:
        """
        开骰子（结束下注阶段，生成骰子结果）
        
        Args:
            chat_id: 群组 ID
            dice_results: 可选的骰子结果（用于测试），如果不提供则随机生成
            
        Returns:
            (成功, 骰子结果, 消息) 元组
        """
        import random
        
        game = self.get_game(chat_id)
        if not game:
            return False, [], "当前没有进行中的骰宝游戏"
        
        # 检查游戏状态，只有在 BETTING 阶段才能开骰子
        if game.phase != GamePhase.BETTING:
            return False, [], "当前不在下注阶段，无法开骰子"
        
        # 状态转换: BETTING -> ROLLING
        game.phase = GamePhase.ROLLING
        
        # 生成骰子结果（如果没有提供）
        if dice_results is None:
            dice_results = [random.randint(1, 6) for _ in range(3)]
        
        game.dice_results = dice_results
        
        # 计算总和
        total = sum(dice_results)
        is_triple = self.calculator.is_triple(dice_results)
        
        # 构建结果消息
        dice_str = " ".join([f"🎲{d}" for d in dice_results])
        msg = f"🎲 骰子结果: {dice_str}\n"
        msg += f"📊 总和: {total}\n"
        
        if is_triple:
            msg += "⚠️ 围骰！庄家通吃大小和总和押注！"
        elif total >= 11:
            msg += "📈 大"
        else:
            msg += "📉 小"
        
        return True, dice_results, msg
    
    async def settle_game(self, chat_id: int) -> Tuple[bool, Dict[int, int], str]:
        """
        结算游戏（计算所有押注赔付，更新余额，结束游戏）
        
        Args:
            chat_id: 群组 ID
            
        Returns:
            (成功, {user_id: 净收益}, 消息) 元组
        """
        game = self.get_game(chat_id)
        if not game:
            return False, {}, "当前没有进行中的骰宝游戏"
        
        # 检查游戏状态，只有在 ROLLING 阶段才能结算
        if game.phase != GamePhase.ROLLING:
            return False, {}, "游戏尚未开骰子，无法结算"
        
        # 检查是否有骰子结果
        if not game.dice_results or len(game.dice_results) != 3:
            return False, {}, "骰子结果无效，无法结算"
        
        # 状态转换: ROLLING -> SETTLING
        game.phase = GamePhase.SETTLING
        
        # 计算每个玩家的结果
        player_results = self._calculate_player_results(game)
        
        # 更新玩家余额并记录交易
        for user_id, result in player_results.items():
            payout = result['total_payout']
            if payout > 0:
                # 增加余额（赔付金额）
                await self.account_mgr.user_repo.update_balance(user_id, payout)
                
                # 记录交易
                await self.tx_repo.log_transaction(
                    user_id=user_id,
                    amount=payout,
                    transaction_type='sicbo_win',
                    description=f'骰宝赢钱: {payout}'
                )
        
        # 构建结算消息
        msg = self._build_settlement_message(game, player_results)
        
        # 计算净收益（赔付 - 下注金额）
        net_results = {}
        for user_id, result in player_results.items():
            net_results[user_id] = result['total_payout'] - result['total_bet']
        
        # 状态转换: SETTLING -> IDLE，结束游戏
        game.phase = GamePhase.IDLE
        
        # 从活跃游戏中移除
        del self.active_games[chat_id]
        
        return True, net_results, msg
    
    def _calculate_player_results(self, game: SicBoGame) -> Dict[int, Dict]:
        """
        计算单个游戏中所有玩家的押注结果
        
        Args:
            game: 游戏对象
            
        Returns:
            {user_id: {'bets': [...], 'total_bet': int, 'total_payout': int, 'username': str}} 字典
        """
        results = {}
        
        for bet in game.bets:
            user_id = bet.user_id
            
            # 初始化玩家结果
            if user_id not in results:
                results[user_id] = {
                    'bets': [],
                    'total_bet': 0,
                    'total_payout': 0,
                    'username': bet.username or str(user_id)
                }
            
            # 计算该押注的赔付
            payout = self.calculator.calculate_bet_payout(bet, game.dice_results)
            
            # 记录押注详情
            bet_detail = {
                'bet_type': bet.bet_type,
                'numbers': bet.numbers,
                'amount': bet.amount,
                'payout': payout
            }
            
            results[user_id]['bets'].append(bet_detail)
            results[user_id]['total_bet'] += bet.amount
            results[user_id]['total_payout'] += payout
        
        return results
    
    def _build_settlement_message(self, game: SicBoGame, player_results: Dict[int, Dict]) -> str:
        """
        构建结算消息
        
        显示 @username 和净胜负金额，使用 emoji 区分胜负
        
        Args:
            game: 游戏对象
            player_results: 玩家结果字典
            
        Returns:
            结算消息字符串
            
        Requirements: 7.6, 7.7
        """
        dice_str = " ".join([f"🎲{d}" for d in game.dice_results])
        total = sum(game.dice_results)
        is_triple = self.calculator.is_triple(game.dice_results)
        
        msg = f"🎰 骰宝结算\n"
        msg += f"━━━━━━━━━━━━━━━\n"
        msg += f"骰子: {dice_str} = {total}"
        
        if is_triple:
            msg += " (围骰)\n"
        else:
            msg += f" ({'大' if total >= 11 else '小'})\n"
        
        msg += f"━━━━━━━━━━━━━━━\n"
        
        if not player_results:
            msg += "本局无人下注\n"
        else:
            for user_id, result in player_results.items():
                username = result.get('username', str(user_id))
                net = result['total_payout'] - result['total_bet']
                # 格式: emoji @username +/-金额
                if net > 0:
                    msg += f"🎉 @{username} +{net}\n"
                elif net < 0:
                    msg += f"😢 @{username} {net}\n"
                else:
                    msg += f"😐 @{username} ±0\n"
        
        msg += f"━━━━━━━━━━━━━━━\n"
        msg += "游戏结束"
        
        return msg
