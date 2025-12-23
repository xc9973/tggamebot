"""
骰宝游戏命令处理器单元测试
测试命令格式解析、错误处理和用户反馈

需求: 7.5, 8.1, 8.2, 8.3, 8.4
"""
import pytest
import os
import tempfile
from unittest.mock import AsyncMock, MagicMock, patch
from telegram import Update, User as TelegramUser, Message, Chat

from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository
from src.account_manager import AccountManager
from src.sicbo_manager import SicBoManager
from src.bot import BotHandlers
from src.models import GamePhase, BetType


@pytest.fixture
async def db_manager():
    """创建临时数据库用于测试"""
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    db = DatabaseManager(db_path)
    await db.initialize()
    
    yield db
    
    await db.close()
    if os.path.exists(db_path):
        os.unlink(db_path)


@pytest.fixture
async def handlers(db_manager):
    """创建 BotHandlers 实例（包含骰宝管理器）"""
    user_repo = UserRepository(db_manager)
    tx_repo = TransactionRepository(db_manager)
    account_manager = AccountManager(user_repo, tx_repo)
    sicbo_manager = SicBoManager(account_manager, tx_repo)
    
    return BotHandlers(
        account_manager, 
        user_repo, 
        tx_repo,
        sicbo_manager=sicbo_manager
    )


def create_mock_update(
    user_id: int = 12345,
    username: str = "testuser",
    chat_id: int = -100123456,  # 群组 ID（负数）
    text: str = "/sicbo",
    args: list = None,
) -> tuple[Update, MagicMock]:
    """创建模拟的 Update 对象"""
    # 创建模拟用户
    mock_user = MagicMock(spec=TelegramUser)
    mock_user.id = user_id
    mock_user.username = username
    mock_user.first_name = username
    
    # 创建模拟聊天
    mock_chat = MagicMock(spec=Chat)
    mock_chat.id = chat_id
    
    # 创建模拟消息
    mock_message = MagicMock(spec=Message)
    mock_message.text = text
    mock_message.reply_text = AsyncMock()
    mock_message.from_user = mock_user
    mock_message.message_id = 1
    
    # 创建模拟 Update
    mock_update = MagicMock(spec=Update)
    mock_update.effective_user = mock_user
    mock_update.effective_chat = mock_chat
    mock_update.message = mock_message
    
    # 创建模拟 context
    mock_context = MagicMock()
    mock_context.args = args or []
    mock_context.bot = MagicMock()
    mock_context.bot.send_message = AsyncMock()
    mock_context.bot.send_dice = AsyncMock()
    
    return mock_update, mock_context


class TestSicboHandler:
    """测试 /sicbo 命令处理器"""
    
    async def test_sicbo_starts_new_game(self, handlers):
        """测试 /sicbo 开始新游戏"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        
        # 设置 reply_text 返回一个带 message_id 的消息
        mock_sent_message = MagicMock()
        mock_sent_message.message_id = 123
        mock_update.message.reply_text.return_value = mock_sent_message
        
        await handlers.sicbo_handler(mock_update, mock_context)
        
        # 验证回复消息被调用
        mock_update.message.reply_text.assert_called_once()
        
        # 获取调用参数（使用 keyword arguments）
        call_kwargs = mock_update.message.reply_text.call_args[1]
        reply_text = call_kwargs.get('text', '')
        reply_markup = call_kwargs.get('reply_markup', None)
        
        # 验证消息包含面板信息
        assert "骰宝" in reply_text
        assert "下注" in reply_text
        
        # 验证键盘被发送
        assert reply_markup is not None
        
        # 验证游戏已创建
        game = handlers.sicbo_manager.get_game(chat_id)
        assert game is not None
        assert game.phase == GamePhase.BETTING
        
        # 验证面板消息 ID 被存储
        assert game.panel_message_id == 123
    
    async def test_sicbo_rejects_duplicate_game(self, handlers):
        """测试 /sicbo 在游戏进行中时显示现有面板"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        
        # 设置 reply_text 返回一个带 message_id 的消息
        mock_sent_message = MagicMock()
        mock_sent_message.message_id = 123
        mock_update.message.reply_text.return_value = mock_sent_message
        
        # 第一次开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 第二次尝试开始游戏（应该显示现有面板）
        await handlers.sicbo_handler(mock_update, mock_context)
        
        # 验证回复消息被调用
        mock_update.message.reply_text.assert_called_once()
        
        # 获取调用参数
        call_kwargs = mock_update.message.reply_text.call_args[1]
        reply_text = call_kwargs.get('text', '')
        reply_markup = call_kwargs.get('reply_markup', None)
        
        # 验证显示的是面板（而不是错误消息）
        assert "骰宝" in reply_text
        assert reply_markup is not None
    
    async def test_sicbo_no_manager_shows_error(self, db_manager):
        """测试骰宝管理器不可用时显示错误"""
        user_repo = UserRepository(db_manager)
        tx_repo = TransactionRepository(db_manager)
        account_manager = AccountManager(user_repo, tx_repo)
        
        # 创建没有骰宝管理器的 handlers
        handlers_no_sicbo = BotHandlers(
            account_manager, 
            user_repo, 
            tx_repo,
            sicbo_manager=None
        )
        
        mock_update, mock_context = create_mock_update()
        
        await handlers_no_sicbo.sicbo_handler(mock_update, mock_context)
        
        # 验证错误消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "暂不可用" in reply_text


class TestBetHandler:
    """测试 /bet 命令处理器"""
    
    async def test_bet_missing_args_shows_usage(self, handlers):
        """测试 /bet 缺少参数时显示用法"""
        mock_update, mock_context = create_mock_update(args=[])
        
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证显示用法
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "命令格式错误" in reply_text
        assert "/bet single" in reply_text
    
    async def test_bet_single_valid_format(self, handlers):
        """测试 /bet single 有效格式"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        
        # 确保用户有余额
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 下注
        mock_context.args = ["single", "3", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证下注成功
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "下注成功" in reply_text or "✅" in reply_text
    
    async def test_bet_single_invalid_number_shows_error(self, handlers):
        """测试 /bet single 无效数字显示错误"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 下注无效数字
        mock_context.args = ["single", "7", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证错误消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "1-6" in reply_text or "❌" in reply_text
    
    async def test_bet_pair_same_numbers_shows_error(self, handlers):
        """测试 /bet pair 相同数字显示错误"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 下注相同数字
        mock_context.args = ["pair", "3", "3", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证错误消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "不同" in reply_text or "❌" in reply_text
    
    async def test_bet_sum_invalid_range_shows_error(self, handlers):
        """测试 /bet sum 无效范围显示错误"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 下注无效总和
        mock_context.args = ["sum", "3", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证错误消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "4-17" in reply_text or "❌" in reply_text
    
    async def test_bet_big_valid_format(self, handlers):
        """测试 /bet big 有效格式"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 下注大
        mock_context.args = ["big", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证下注成功
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "下注成功" in reply_text or "✅" in reply_text
    
    async def test_bet_small_valid_format(self, handlers):
        """测试 /bet small 有效格式"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 下注小
        mock_context.args = ["small", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证下注成功
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "下注成功" in reply_text or "✅" in reply_text
    
    async def test_bet_unknown_type_shows_error(self, handlers):
        """测试 /bet 未知类型显示错误"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 下注未知类型
        mock_context.args = ["unknown", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证错误消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "未知" in reply_text or "❌" in reply_text
    
    async def test_bet_no_game_shows_error(self, handlers):
        """测试没有游戏时下注显示错误"""
        mock_update, mock_context = create_mock_update()
        user_id = mock_update.effective_user.id
        
        # 确保用户存在
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 直接下注（没有开始游戏）
        mock_context.args = ["big", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证错误消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "没有进行中" in reply_text or "❌" in reply_text
    
    async def test_bet_insufficient_balance_shows_error(self, handlers):
        """测试余额不足时显示错误"""
        mock_update, mock_context = create_mock_update()
        chat_id = mock_update.effective_chat.id
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 下注超过余额
        mock_context.args = ["big", "999999"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 验证错误消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "余额不足" in reply_text or "❌" in reply_text


class TestSicboStatusHandler:
    """测试 /sicbo_status 命令处理器"""
    
    async def test_sicbo_status_no_game(self, handlers):
        """测试没有游戏时查询状态"""
        mock_update, mock_context = create_mock_update()
        
        await handlers.sicbo_status_handler(mock_update, mock_context)
        
        # 验证消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "没有进行中" in reply_text
    
    async def test_sicbo_status_with_game(self, handlers):
        """测试有游戏时查询状态"""
        mock_update, mock_context = create_mock_update()
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 查询状态
        await handlers.sicbo_status_handler(mock_update, mock_context)
        
        # 验证消息包含状态信息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "状态" in reply_text
        assert "参与人数" in reply_text


class TestMybetsHandler:
    """测试 /mybets 命令处理器"""
    
    async def test_mybets_no_game(self, handlers):
        """测试没有游戏时查询押注"""
        mock_update, mock_context = create_mock_update()
        
        await handlers.mybets_handler(mock_update, mock_context)
        
        # 验证消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "没有进行中" in reply_text
    
    async def test_mybets_no_bets(self, handlers):
        """测试有游戏但没有押注时查询"""
        mock_update, mock_context = create_mock_update()
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 查询押注
        await handlers.mybets_handler(mock_update, mock_context)
        
        # 验证消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "没有押注" in reply_text
    
    async def test_mybets_with_bets(self, handlers):
        """测试有押注时查询"""
        mock_update, mock_context = create_mock_update()
        user_id = mock_update.effective_user.id
        
        # 先开始游戏
        await handlers.sicbo_handler(mock_update, mock_context)
        await handlers.account_manager.ensure_user_exists(user_id, "testuser")
        
        # 下注
        mock_context.args = ["big", "100"]
        await handlers.bet_handler(mock_update, mock_context)
        
        # 重置 mock
        mock_update.message.reply_text.reset_mock()
        
        # 查询押注
        mock_context.args = []
        await handlers.mybets_handler(mock_update, mock_context)
        
        # 验证消息包含押注信息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "您的押注" in reply_text
        assert "大" in reply_text
        assert "100" in reply_text


class TestRollHandler:
    """测试 /roll 命令处理器"""
    
    async def test_roll_no_game(self, handlers):
        """测试没有游戏时开骰子"""
        mock_update, mock_context = create_mock_update()
        
        await handlers.roll_handler(mock_update, mock_context)
        
        # 验证错误消息
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "没有进行中" in reply_text or "❌" in reply_text
