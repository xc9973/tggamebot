"""
集成测试
端到端测试完整的用户流程

测试场景:
1. 新用户注册 → 签到 → 游戏 → 转账 → 排行榜
2. 完整的 21 点游戏流程
3. 管理员操作流程
"""
import pytest
import os
import tempfile
import time
from unittest.mock import AsyncMock, MagicMock, patch

from telegram import Update, User as TelegramUser, Message, Chat, Chat

from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository
from src.account_manager import AccountManager
from src.game_engine import GameEngine
from src.blackjack import BlackjackManager, calculate_hand_value, is_blackjack
from src.sicbo_manager import SicBoManager
from src.models import BetType, GamePhase
from src.bot import BotHandlers
from src.concurrency import ConcurrencyManager


@pytest.fixture
async def integration_setup():
    """创建完整的集成测试环境"""
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    db = DatabaseManager(db_path)
    await db.initialize()
    
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_manager = AccountManager(user_repo, tx_repo)
    game_engine = GameEngine(account_manager, tx_repo)
    blackjack_manager = BlackjackManager(account_manager, tx_repo)
    sicbo_manager = SicBoManager(account_manager, tx_repo)
    concurrency_manager = ConcurrencyManager()
    
    # 管理员 ID 列表
    admin_ids = [99999]
    
    handlers = BotHandlers(
        account_manager=account_manager,
        user_repo=user_repo,
        tx_repo=tx_repo,
        game_engine=game_engine,
        blackjack_manager=blackjack_manager,
        sicbo_manager=sicbo_manager,
        admin_ids=admin_ids,
        concurrency_manager=concurrency_manager
    )
    
    yield {
        'db': db,
        'user_repo': user_repo,
        'tx_repo': tx_repo,
        'account_manager': account_manager,
        'game_engine': game_engine,
        'blackjack_manager': blackjack_manager,
        'sicbo_manager': sicbo_manager,
        'handlers': handlers,
        'admin_ids': admin_ids
    }
    
    await db.close()
    if os.path.exists(db_path):
        os.unlink(db_path)


def create_mock_update(
    user_id: int,
    username: str,
    text: str = "/start",
    args: list = None,
    reply_to_message: Message = None,
    entities: list = None,
    chat_id: int = None
) -> tuple[Update, MagicMock]:
    """创建模拟的 Update 对象"""
    mock_user = MagicMock(spec=TelegramUser)
    mock_user.id = user_id
    mock_user.username = username
    mock_user.first_name = username
    
    mock_message = MagicMock(spec=Message)
    mock_message.text = text
    mock_message.reply_text = AsyncMock()
    mock_message.from_user = mock_user
    mock_message.reply_to_message = reply_to_message
    mock_message.entities = entities or []
    mock_message.message_id = 1
    
    mock_chat = MagicMock(spec=Chat)
    # 默认使用负数 chat_id（群组），避免私聊检查
    mock_chat.id = chat_id if chat_id is not None else -user_id
    
    mock_update = MagicMock(spec=Update)
    mock_update.effective_user = mock_user
    mock_update.effective_chat = mock_chat
    mock_update.message = mock_message
    
    mock_context = MagicMock()
    mock_context.args = args or []
    mock_context.user_data = {}
    mock_context.bot = MagicMock()
    mock_context.bot.send_dice = AsyncMock()
    mock_context.bot.send_message = AsyncMock()
    
    return mock_update, mock_context


def create_mock_callback_query(
    user_id: int,
    username: str,
    callback_data: str,
    chat_id: int = None
) -> tuple[Update, MagicMock]:
    """创建模拟的 CallbackQuery Update 对象"""
    mock_user = MagicMock(spec=TelegramUser)
    mock_user.id = user_id
    mock_user.username = username
    mock_user.first_name = username
    
    mock_chat = MagicMock(spec=Chat)
    # 默认使用负数 chat_id（群组），避免私聊检查
    mock_chat.id = chat_id if chat_id is not None else -user_id
    
    mock_query = MagicMock()
    mock_query.data = callback_data
    mock_query.answer = AsyncMock()
    mock_query.edit_message_text = AsyncMock()
    
    mock_update = MagicMock(spec=Update)
    mock_update.effective_user = mock_user
    mock_update.effective_chat = mock_chat
    mock_update.callback_query = mock_query
    mock_update.message = None
    
    mock_context = MagicMock()
    mock_context.args = []
    
    return mock_update, mock_context


class TestEndToEndUserFlow:
    """
    端到端测试：新用户注册 → 签到 → 游戏 → 转账 → 排行榜
    """
    
    async def test_complete_user_journey(self, integration_setup):
        """测试完整的用户旅程"""
        handlers = integration_setup['handlers']
        user_repo = integration_setup['user_repo']
        
        user_id = 10001
        username = "testuser1"
        
        # 步骤 1: 新用户注册 (/start)
        mock_update, mock_context = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update, mock_context)
        
        # 验证用户已创建
        user = await user_repo.get_user(user_id)
        assert user is not None
        assert user.balance == 1000
        assert user.username == username
        
        # 验证欢迎消息
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "欢迎" in reply_text
        assert "1000" in reply_text
        
        # 步骤 2: 查询余额 (/balance)
        mock_update2, mock_context2 = create_mock_update(user_id, username)
        await handlers.balance_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "余额" in reply_text
        assert "1000" in reply_text
        
        # 步骤 3: 每日签到 (/daily)
        mock_update3, mock_context3 = create_mock_update(user_id, username)
        await handlers.daily_handler(mock_update3, mock_context3)
        
        reply_text = mock_update3.message.reply_text.call_args[0][0]
        assert "签到成功" in reply_text
        assert "500" in reply_text
        
        # 验证余额增加
        user = await user_repo.get_user(user_id)
        assert user.balance == 1500
        
        # 步骤 4: 查看排行榜 (/top)
        mock_update4, mock_context4 = create_mock_update(user_id, username)
        await handlers.top_handler(mock_update4, mock_context4)
        
        reply_text = mock_update4.message.reply_text.call_args[0][0]
        assert "排行榜" in reply_text
        assert username in reply_text
    
    async def test_user_registration_and_transfer(self, integration_setup):
        """测试用户注册和转账流程"""
        handlers = integration_setup['handlers']
        user_repo = integration_setup['user_repo']
        
        sender_id = 20001
        sender_name = "sender"
        receiver_id = 20002
        receiver_name = "receiver"
        
        # 创建发送者
        mock_update1, mock_context1 = create_mock_update(sender_id, sender_name)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 创建接收者
        mock_update2, mock_context2 = create_mock_update(receiver_id, receiver_name)
        await handlers.start_handler(mock_update2, mock_context2)
        
        # 验证两个用户都有 1000 金币
        sender = await user_repo.get_user(sender_id)
        receiver = await user_repo.get_user(receiver_id)
        assert sender.balance == 1000
        assert receiver.balance == 1000
        
        # 执行转账（使用 text_mention）
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = receiver_id
        
        mock_update3, mock_context3 = create_mock_update(
            sender_id, sender_name,
            args=["@receiver", "200"],
            entities=[mock_entity]
        )
        await handlers.pay_handler(mock_update3, mock_context3)
        
        reply_text = mock_update3.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
        
        # 验证余额变化（5% 手续费）
        sender = await user_repo.get_user(sender_id)
        receiver = await user_repo.get_user(receiver_id)
        
        assert sender.balance == 800  # 1000 - 200
        assert receiver.balance == 1190  # 1000 + 200 * 0.95
    
    async def test_dice_game_flow(self, integration_setup):
        """测试骰子游戏流程"""
        handlers = integration_setup['handlers']
        game_engine = integration_setup['game_engine']
        user_repo = integration_setup['user_repo']
        account_manager = integration_setup['account_manager']
        
        user_id = 30001
        username = "diceuser"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 直接测试游戏引擎（绕过 Telegram API）
        initial_balance = await account_manager.get_balance(user_id)
        assert initial_balance == 1000
        
        # 测试骰子游戏 - 点数 6（赢 2 倍）
        success, message, payout = await game_engine.play_dice(user_id, 100, 6)
        assert success is True
        assert payout == 200  # 赢 2 倍
        
        new_balance = await account_manager.get_balance(user_id)
        assert new_balance == 1200  # 1000 + 200
        
        # 测试骰子游戏 - 点数 1（输）
        success, message, payout = await game_engine.play_dice(user_id, 100, 1)
        assert success is True
        assert payout == -100  # 输掉本金
        
        final_balance = await account_manager.get_balance(user_id)
        assert final_balance == 1100  # 1200 - 100
    
    async def test_slot_game_flow(self, integration_setup):
        """测试老虎机游戏流程"""
        game_engine = integration_setup['game_engine']
        handlers = integration_setup['handlers']
        account_manager = integration_setup['account_manager']
        
        user_id = 30002
        username = "slotuser"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 测试老虎机 - 大奖（值 64，三个一致）
        success, message, payout = await game_engine.play_slot(user_id, 100, 64)
        assert success is True
        assert payout == 1000  # 赢 10 倍
        
        new_balance = await account_manager.get_balance(user_id)
        assert new_balance == 2000  # 1000 + 1000
    
    async def test_leaderboard_ranking(self, integration_setup):
        """测试排行榜排名"""
        handlers = integration_setup['handlers']
        user_repo = integration_setup['user_repo']
        account_manager = integration_setup['account_manager']
        
        # 创建多个用户并设置不同余额
        users = [
            (40001, "rich", 5000),
            (40002, "medium", 2000),
            (40003, "poor", 500),
        ]
        
        for user_id, username, balance in users:
            mock_update, mock_context = create_mock_update(user_id, username)
            await handlers.start_handler(mock_update, mock_context)
            
            # 调整余额
            balance_diff = balance - 1000
            await user_repo.update_balance(user_id, balance_diff)
        
        # 查看排行榜
        mock_update, mock_context = create_mock_update(40001, "rich")
        await handlers.top_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        
        # 验证排名顺序
        rich_pos = reply_text.find("rich")
        medium_pos = reply_text.find("medium")
        poor_pos = reply_text.find("poor")
        
        assert rich_pos < medium_pos < poor_pos



class TestBlackjackGameFlow:
    """
    测试完整的 21 点游戏流程
    """
    
    async def test_blackjack_start_hit_stand_flow(self, integration_setup):
        """测试 21 点游戏：开始 → 要牌 → 停牌"""
        handlers = integration_setup['handlers']
        blackjack_manager = integration_setup['blackjack_manager']
        account_manager = integration_setup['account_manager']
        
        user_id = 50001
        username = "bjplayer1"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        initial_balance = await account_manager.get_balance(user_id)
        assert initial_balance == 1000
        
        # 开始游戏
        success, message, game = await blackjack_manager.start_game(user_id, 100)
        assert success is True
        assert game is not None
        assert game.bet == 100
        assert len(game.player_cards) == 2
        assert len(game.dealer_cards) == 2
        
        # 如果不是 Blackjack，继续游戏
        if not game.is_finished:
            # 要牌
            success, message, game = await blackjack_manager.hit(user_id)
            assert success is True
            assert len(game.player_cards) >= 3
            
            # 如果没爆牌，停牌
            if not game.is_finished:
                success, message, game, payout = await blackjack_manager.stand(user_id)
                assert success is True
                assert game.is_finished is True
    
    async def test_blackjack_double_down_flow(self, integration_setup):
        """测试 21 点游戏：加倍操作"""
        blackjack_manager = integration_setup['blackjack_manager']
        handlers = integration_setup['handlers']
        account_manager = integration_setup['account_manager']
        
        user_id = 50002
        username = "bjplayer2"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 开始游戏
        success, message, game = await blackjack_manager.start_game(user_id, 100)
        
        if success and not game.is_finished:
            # 加倍
            success, message, game, payout = await blackjack_manager.double_down(user_id)
            assert success is True
            assert game.is_finished is True
            assert game.bet == 200  # 下注翻倍
    
    async def test_blackjack_bust_scenario(self, integration_setup):
        """测试 21 点游戏：爆牌场景"""
        blackjack_manager = integration_setup['blackjack_manager']
        handlers = integration_setup['handlers']
        account_manager = integration_setup['account_manager']
        
        user_id = 50003
        username = "bjplayer3"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 开始游戏
        success, message, game = await blackjack_manager.start_game(user_id, 100)
        
        if success and not game.is_finished:
            # 持续要牌直到爆牌或游戏结束
            while not game.is_finished:
                player_value = calculate_hand_value(game.player_cards)
                if player_value >= 21:
                    break
                success, message, game = await blackjack_manager.hit(user_id)
                if not success or game.is_finished:
                    break
    
    async def test_blackjack_callback_handlers(self, integration_setup):
        """测试 21 点游戏回调处理器"""
        handlers = integration_setup['handlers']
        blackjack_manager = integration_setup['blackjack_manager']
        concurrency_manager = integration_setup['handlers'].concurrency_manager
        
        user_id = 50004
        username = "bjplayer4"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 通过命令处理器开始游戏
        mock_update2, mock_context2 = create_mock_update(
            user_id, username, args=["100"]
        )
        await handlers.blackjack_handler(mock_update2, mock_context2)
        
        # 检查游戏是否已开始
        game = blackjack_manager.get_game(user_id)
        
        if game is not None and not game.is_finished:
            # 测试停牌回调
            mock_update3, mock_context3 = create_mock_callback_query(
                user_id, username, "bj_stand"
            )
            await handlers.blackjack_callback_handler(mock_update3, mock_context3)
            
            # 验证回调被处理
            mock_update3.callback_query.answer.assert_called_once()
            mock_update3.callback_query.edit_message_text.assert_called_once()
    
    async def test_blackjack_no_game_error(self, integration_setup):
        """测试没有游戏时的错误处理"""
        handlers = integration_setup['handlers']
        
        user_id = 50005
        username = "bjplayer5"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 尝试在没有游戏的情况下要牌
        mock_update2, mock_context2 = create_mock_callback_query(
            user_id, username, "bj_hit"
        )
        await handlers.blackjack_callback_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.callback_query.edit_message_text.call_args[0][0]
        assert "没有" in reply_text or "游戏" in reply_text
    
    async def test_blackjack_insufficient_balance(self, integration_setup):
        """测试余额不足时的错误处理"""
        handlers = integration_setup['handlers']
        blackjack_manager = integration_setup['blackjack_manager']
        
        user_id = 50006
        username = "bjplayer6"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 尝试下注超过余额
        success, message, game = await blackjack_manager.start_game(user_id, 2000)
        assert success is False
        assert "余额不足" in message



class TestAdminOperationFlow:
    """
    测试管理员操作流程
    """
    
    async def test_admin_add_coins(self, integration_setup):
        """测试管理员添加金币"""
        handlers = integration_setup['handlers']
        user_repo = integration_setup['user_repo']
        admin_id = integration_setup['admin_ids'][0]
        
        target_id = 60001
        target_name = "targetuser1"
        
        # 创建目标用户
        mock_update1, mock_context1 = create_mock_update(target_id, target_name)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 验证初始余额
        user = await user_repo.get_user(target_id)
        assert user.balance == 1000
        
        # 管理员添加金币
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update2, mock_context2 = create_mock_update(
            admin_id, "admin",
            args=["@targetuser1", "500"],
            entities=[mock_entity]
        )
        await handlers.admin_add_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
        
        # 验证余额增加
        user = await user_repo.get_user(target_id)
        assert user.balance == 1500
    
    async def test_admin_remove_coins(self, integration_setup):
        """测试管理员扣除金币"""
        handlers = integration_setup['handlers']
        user_repo = integration_setup['user_repo']
        admin_id = integration_setup['admin_ids'][0]
        
        target_id = 60002
        target_name = "targetuser2"
        
        # 创建目标用户
        mock_update1, mock_context1 = create_mock_update(target_id, target_name)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 管理员扣除金币
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update2, mock_context2 = create_mock_update(
            admin_id, "admin",
            args=["@targetuser2", "300"],
            entities=[mock_entity]
        )
        await handlers.admin_remove_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
        
        # 验证余额减少
        user = await user_repo.get_user(target_id)
        assert user.balance == 700
    
    async def test_admin_reset_account(self, integration_setup):
        """测试管理员重置账户"""
        handlers = integration_setup['handlers']
        user_repo = integration_setup['user_repo']
        admin_id = integration_setup['admin_ids'][0]
        
        target_id = 60003
        target_name = "targetuser3"
        
        # 创建目标用户
        mock_update1, mock_context1 = create_mock_update(target_id, target_name)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 修改用户余额
        await user_repo.update_balance(target_id, 5000)
        user = await user_repo.get_user(target_id)
        assert user.balance == 6000
        
        # 管理员重置账户
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update2, mock_context2 = create_mock_update(
            admin_id, "admin",
            args=["@targetuser3"],
            entities=[mock_entity]
        )
        await handlers.admin_reset_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
        assert "重置" in reply_text
        
        # 验证余额重置为 1000
        user = await user_repo.get_user(target_id)
        assert user.balance == 1000
    
    async def test_non_admin_permission_denied(self, integration_setup):
        """测试非管理员权限拒绝"""
        handlers = integration_setup['handlers']
        
        non_admin_id = 60004
        target_id = 60005
        
        # 创建目标用户
        mock_update1, mock_context1 = create_mock_update(target_id, "target")
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 非管理员尝试添加金币
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update2, mock_context2 = create_mock_update(
            non_admin_id, "nonadmin",
            args=["@target", "500"],
            entities=[mock_entity]
        )
        await handlers.admin_add_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "权限不足" in reply_text
    
    async def test_admin_operation_audit_log(self, integration_setup):
        """测试管理员操作审计日志"""
        handlers = integration_setup['handlers']
        tx_repo = integration_setup['tx_repo']
        admin_id = integration_setup['admin_ids'][0]
        
        target_id = 60006
        target_name = "targetuser6"
        
        # 创建目标用户
        mock_update1, mock_context1 = create_mock_update(target_id, target_name)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 管理员添加金币
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update2, mock_context2 = create_mock_update(
            admin_id, "admin",
            args=["@targetuser6", "1000"],
            entities=[mock_entity]
        )
        await handlers.admin_add_handler(mock_update2, mock_context2)
        
        # 验证交易日志
        history = await tx_repo.get_user_history(target_id)
        admin_tx = [tx for tx in history if tx.type == 'admin_add']
        assert len(admin_tx) >= 1
        assert admin_tx[0].amount == 1000
        assert "管理员" in admin_tx[0].description


class TestCompleteGameScenarios:
    """
    测试完整的游戏场景
    """
    
    async def test_user_plays_multiple_games(self, integration_setup):
        """测试用户玩多个游戏"""
        handlers = integration_setup['handlers']
        game_engine = integration_setup['game_engine']
        account_manager = integration_setup['account_manager']
        
        user_id = 70001
        username = "multigamer"
        
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(user_id, username)
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 签到获取更多金币
        mock_update2, mock_context2 = create_mock_update(user_id, username)
        await handlers.daily_handler(mock_update2, mock_context2)
        
        balance = await account_manager.get_balance(user_id)
        assert balance == 1500
        
        # 玩骰子游戏
        await game_engine.play_dice(user_id, 100, 5)  # 赢 1 倍
        balance = await account_manager.get_balance(user_id)
        assert balance == 1600
        
        # 玩老虎机游戏
        await game_engine.play_slot(user_id, 100, 3)  # 输
        balance = await account_manager.get_balance(user_id)
        assert balance == 1500
    
    async def test_transfer_chain(self, integration_setup):
        """测试转账链"""
        handlers = integration_setup['handlers']
        account_manager = integration_setup['account_manager']
        user_repo = integration_setup['user_repo']
        
        # 创建三个用户
        users = [(70002, "user_a"), (70003, "user_b"), (70004, "user_c")]
        for user_id, username in users:
            mock_update, mock_context = create_mock_update(user_id, username)
            await handlers.start_handler(mock_update, mock_context)
        
        # A 转给 B
        success, message = await account_manager.transfer(70002, 70003, 200)
        assert success is True
        
        # B 转给 C
        success, message = await account_manager.transfer(70003, 70004, 100)
        assert success is True
        
        # 验证最终余额
        user_a = await user_repo.get_user(70002)
        user_b = await user_repo.get_user(70003)
        user_c = await user_repo.get_user(70004)
        
        assert user_a.balance == 800  # 1000 - 200
        assert user_b.balance == 1090  # 1000 + 190 - 100
        assert user_c.balance == 1095  # 1000 + 95



class TestSicBoGameFlow:
    """
    骰宝游戏端到端测试
    测试完整游戏流程（开始 → 下注 → 开骰子 → 结算）
    
    需求: 6.3, 6.6
    """
    
    async def test_complete_sicbo_game_flow(self, integration_setup):
        """测试完整的骰宝游戏流程：开始 → 下注 → 开骰子 → 结算"""
        sicbo_manager = integration_setup['sicbo_manager']
        account_manager = integration_setup['account_manager']
        
        chat_id = -100001
        user_id = 80001
        username = "sicboplayer1"
        
        # 直接创建用户（绕过 handler 的白名单检查）
        await account_manager.ensure_user_exists(user_id, username)
        
        initial_balance = await account_manager.get_balance(user_id)
        assert initial_balance == 1000
        
        # 步骤 1: 开始游戏
        success, message = await sicbo_manager.start_game(chat_id)
        assert success is True
        assert "骰宝游戏开始" in message
        
        game = sicbo_manager.get_game(chat_id)
        assert game is not None
        assert game.phase == GamePhase.BETTING
        
        # 步骤 2: 下注（押大）
        success, message = await sicbo_manager.place_bet(
            chat_id=chat_id,
            user_id=user_id,
            bet_type=BetType.BIG,
            amount=100,
            numbers=[]
        )
        assert success is True
        assert "下注成功" in message
        
        # 验证余额已扣除
        balance_after_bet = await account_manager.get_balance(user_id)
        assert balance_after_bet == 900
        
        # 步骤 3: 开骰子（使用固定结果：4, 5, 6 = 15，大）
        success, dice_results, message = await sicbo_manager.roll_dice(chat_id, [4, 5, 6])
        assert success is True
        assert dice_results == [4, 5, 6]
        assert "15" in message  # 总和
        assert "大" in message
        
        # 步骤 4: 结算
        success, results, message = await sicbo_manager.settle_game(chat_id)
        assert success is True
        assert user_id in results
        
        # 验证赢钱（押大赢，1:1 赔率，返还 200）
        final_balance = await account_manager.get_balance(user_id)
        assert final_balance == 1100  # 900 + 200
        
        # 验证游戏已结束
        game = sicbo_manager.get_game(chat_id)
        assert game is None
    
    async def test_multiple_players_betting(self, integration_setup):
        """测试多人同时下注场景"""
        sicbo_manager = integration_setup['sicbo_manager']
        account_manager = integration_setup['account_manager']
        
        chat_id = -100002
        
        # 创建三个玩家
        players = [
            (80002, "player_a"),
            (80003, "player_b"),
            (80004, "player_c"),
        ]
        
        for user_id, username in players:
            await account_manager.ensure_user_exists(user_id, username)
        
        # 开始游戏
        success, _ = await sicbo_manager.start_game(chat_id)
        assert success is True
        
        # 玩家 A 押大
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=80002,
            bet_type=BetType.BIG, amount=100, numbers=[]
        )
        assert success is True
        
        # 玩家 B 押小
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=80003,
            bet_type=BetType.SMALL, amount=150, numbers=[]
        )
        assert success is True
        
        # 玩家 C 押单一数字 3
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=80004,
            bet_type=BetType.SINGLE, amount=50, numbers=[3]
        )
        assert success is True
        
        # 验证游戏统计
        stats = sicbo_manager.get_game_stats(chat_id)
        assert stats["player_count"] == 3
        assert stats["total_bet_amount"] == 300
        assert stats["bet_count"] == 3
        
        # 开骰子（结果：2, 3, 6 = 11，大，包含 3）
        success, _, _ = await sicbo_manager.roll_dice(chat_id, [2, 3, 6])
        assert success is True
        
        # 结算
        success, results, _ = await sicbo_manager.settle_game(chat_id)
        assert success is True
        
        # 验证结果
        # 玩家 A 押大赢：净收益 +100
        assert results[80002] == 100
        # 玩家 B 押小输：净收益 -150
        assert results[80003] == -150
        # 玩家 C 押单一数字 3 赢（1个匹配）：净收益 +50
        assert results[80004] == 50
    
    async def test_triple_house_wins(self, integration_setup):
        """测试围骰情况下的结算（庄家通吃大小和总和）"""
        sicbo_manager = integration_setup['sicbo_manager']
        account_manager = integration_setup['account_manager']
        
        chat_id = -100003
        user_id = 80005
        username = "tripleplayer"
        
        # 创建用户
        await account_manager.ensure_user_exists(user_id, username)
        
        # 开始游戏
        success, _ = await sicbo_manager.start_game(chat_id)
        assert success is True
        
        # 押大
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.BIG, amount=100, numbers=[]
        )
        assert success is True
        
        # 押总和 12
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.SUM, amount=100, numbers=[12]
        )
        assert success is True
        
        # 押单一数字 4
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.SINGLE, amount=100, numbers=[4]
        )
        assert success is True
        
        # 开骰子（围骰：4, 4, 4 = 12）
        success, dice_results, message = await sicbo_manager.roll_dice(chat_id, [4, 4, 4])
        assert success is True
        assert "围骰" in message
        
        # 结算
        success, results, _ = await sicbo_manager.settle_game(chat_id)
        assert success is True
        
        # 验证结果
        # 押大：围骰庄家通吃，输 100
        # 押总和 12：围骰庄家通吃，输 100
        # 押单一数字 4：3 个匹配，赢 4 倍 = 400，净收益 +300
        # 总净收益：-100 - 100 + 300 = 100
        assert results[user_id] == 100
        
        # 验证最终余额
        final_balance = await account_manager.get_balance(user_id)
        assert final_balance == 1100  # 1000 - 300 + 400
    
    async def test_same_player_multiple_bets(self, integration_setup):
        """测试同一玩家多种押注的结算"""
        sicbo_manager = integration_setup['sicbo_manager']
        account_manager = integration_setup['account_manager']
        
        chat_id = -100004
        user_id = 80006
        username = "multibetplayer"
        
        # 创建用户
        await account_manager.ensure_user_exists(user_id, username)
        
        # 开始游戏
        success, _ = await sicbo_manager.start_game(chat_id)
        assert success is True
        
        # 押大
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.BIG, amount=100, numbers=[]
        )
        assert success is True
        
        # 押组合 3-5
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.PAIR, amount=50, numbers=[3, 5]
        )
        assert success is True
        
        # 押总和 11
        success, _ = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.SUM, amount=50, numbers=[11]
        )
        assert success is True
        
        # 验证用户押注
        user_bets = sicbo_manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 3
        
        # 开骰子（结果：3, 3, 5 = 11，大，包含 3-5 组合）
        success, _, _ = await sicbo_manager.roll_dice(chat_id, [3, 3, 5])
        assert success is True
        
        # 结算
        success, results, _ = await sicbo_manager.settle_game(chat_id)
        assert success is True
        
        # 验证结果
        # 押大赢：+100
        # 押组合 3-5 赢：+250 (5:1 赔率，返还 300，净收益 250)
        # 押总和 11 赢：+300 (6:1 赔率，返还 350，净收益 300)
        # 总净收益：100 + 250 + 300 = 650
        assert results[user_id] == 650
    
    async def test_game_session_exclusivity(self, integration_setup):
        """测试游戏会话互斥性"""
        sicbo_manager = integration_setup['sicbo_manager']
        
        chat_id = -100005
        
        # 开始第一个游戏
        success, _ = await sicbo_manager.start_game(chat_id)
        assert success is True
        
        # 尝试开始第二个游戏（应该失败）
        success, message = await sicbo_manager.start_game(chat_id)
        assert success is False
        assert "已有进行中的游戏" in message
    
    async def test_betting_validation(self, integration_setup):
        """测试下注验证"""
        sicbo_manager = integration_setup['sicbo_manager']
        account_manager = integration_setup['account_manager']
        
        chat_id = -100006
        user_id = 80007
        username = "validationplayer"
        
        # 创建用户
        await account_manager.ensure_user_exists(user_id, username)
        
        # 开始游戏
        success, _ = await sicbo_manager.start_game(chat_id)
        assert success is True
        
        # 测试无效的单一数字（超出范围）
        success, message = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.SINGLE, amount=100, numbers=[7]
        )
        assert success is False
        assert "1-6" in message
        
        # 测试无效的组合（相同数字）
        success, message = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.PAIR, amount=100, numbers=[3, 3]
        )
        assert success is False
        assert "不同" in message
        
        # 测试无效的总和（超出范围）
        success, message = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.SUM, amount=100, numbers=[3]
        )
        assert success is False
        assert "4-17" in message
        
        # 测试无效金额
        success, message = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.BIG, amount=0, numbers=[]
        )
        assert success is False
        assert "大于 0" in message
    
    async def test_insufficient_balance(self, integration_setup):
        """测试余额不足"""
        sicbo_manager = integration_setup['sicbo_manager']
        account_manager = integration_setup['account_manager']
        
        chat_id = -100007
        user_id = 80008
        username = "poorplayer"
        
        # 创建用户
        await account_manager.ensure_user_exists(user_id, username)
        
        # 开始游戏
        success, _ = await sicbo_manager.start_game(chat_id)
        assert success is True
        
        # 尝试下注超过余额
        success, message = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.BIG, amount=2000, numbers=[]
        )
        assert success is False
        assert "余额不足" in message
    
    async def test_betting_not_in_betting_phase(self, integration_setup):
        """测试非下注阶段下注"""
        sicbo_manager = integration_setup['sicbo_manager']
        account_manager = integration_setup['account_manager']
        
        chat_id = -100008
        user_id = 80009
        username = "lateplayer"
        
        # 创建用户
        await account_manager.ensure_user_exists(user_id, username)
        
        # 开始游戏
        success, _ = await sicbo_manager.start_game(chat_id)
        assert success is True
        
        # 开骰子（结束下注阶段）
        success, _, _ = await sicbo_manager.roll_dice(chat_id, [1, 2, 3])
        assert success is True
        
        # 尝试在非下注阶段下注
        success, message = await sicbo_manager.place_bet(
            chat_id=chat_id, user_id=user_id,
            bet_type=BetType.BIG, amount=100, numbers=[]
        )
        assert success is False
        assert "不在下注阶段" in message
    
    async def test_no_bets_settlement(self, integration_setup):
        """测试无人下注时的结算"""
        sicbo_manager = integration_setup['sicbo_manager']
        
        chat_id = -100009
        
        # 开始游戏
        success, _ = await sicbo_manager.start_game(chat_id)
        assert success is True
        
        # 开骰子
        success, _, _ = await sicbo_manager.roll_dice(chat_id, [1, 2, 3])
        assert success is True
        
        # 结算
        success, results, message = await sicbo_manager.settle_game(chat_id)
        assert success is True
        assert len(results) == 0
        assert "无人下注" in message
