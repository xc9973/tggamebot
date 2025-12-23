"""
Bot 命令处理器单元测试
测试命令格式解析、错误处理和用户反馈

需求: 11.1, 11.3
"""
import pytest
import os
import tempfile
from unittest.mock import AsyncMock, MagicMock, patch
from telegram import Update, User as TelegramUser, Message, Chat

from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository
from src.account_manager import AccountManager
from src.bot import BotHandlers


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
    """创建 BotHandlers 实例"""
    user_repo = UserRepository(db_manager)
    tx_repo = TransactionRepository(db_manager)
    account_manager = AccountManager(user_repo, tx_repo)
    
    return BotHandlers(account_manager, user_repo, tx_repo)


def create_mock_update(
    user_id: int = 12345,
    username: str = "testuser",
    text: str = "/start",
    args: list = None,
    reply_to_message: Message = None,
    entities: list = None
) -> tuple[Update, MagicMock]:
    """创建模拟的 Update 对象"""
    # 创建模拟用户
    mock_user = MagicMock(spec=TelegramUser)
    mock_user.id = user_id
    mock_user.username = username
    mock_user.first_name = username
    
    # 创建模拟消息
    mock_message = MagicMock(spec=Message)
    mock_message.text = text
    mock_message.reply_text = AsyncMock()
    mock_message.from_user = mock_user
    mock_message.reply_to_message = reply_to_message
    mock_message.entities = entities or []
    
    # 创建模拟 Update
    mock_update = MagicMock(spec=Update)
    mock_update.effective_user = mock_user
    mock_update.message = mock_message
    
    # 创建模拟 context
    mock_context = MagicMock()
    mock_context.args = args or []
    
    return mock_update, mock_context


class TestStartHandler:
    """测试 /start 命令处理器"""
    
    async def test_start_creates_new_user(self, handlers):
        """测试 /start 为新用户创建账户"""
        mock_update, mock_context = create_mock_update(
            user_id=11111,
            username="newuser"
        )
        
        await handlers.start_handler(mock_update, mock_context)
        
        # 验证回复消息被调用
        mock_update.message.reply_text.assert_called_once()
        reply_text = mock_update.message.reply_text.call_args[0][0]
        
        # 验证消息包含欢迎信息和余额
        assert "欢迎" in reply_text
        assert "1000" in reply_text
        assert "newuser" in reply_text
    
    async def test_start_shows_existing_user_balance(self, handlers):
        """测试 /start 显示已存在用户的余额"""
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=22222,
            username="existinguser"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 再次调用 /start
        mock_update2, mock_context2 = create_mock_update(
            user_id=22222,
            username="existinguser"
        )
        await handlers.start_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "1000" in reply_text


class TestBalanceHandler:
    """测试 /balance 命令处理器"""
    
    async def test_balance_shows_user_balance(self, handlers):
        """测试 /balance 显示用户余额"""
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=33333,
            username="balanceuser"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 查询余额
        mock_update2, mock_context2 = create_mock_update(
            user_id=33333,
            username="balanceuser"
        )
        await handlers.balance_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "余额" in reply_text
        assert "1000" in reply_text
    
    async def test_balance_creates_user_if_not_exists(self, handlers):
        """测试 /balance 为不存在的用户创建账户"""
        mock_update, mock_context = create_mock_update(
            user_id=44444,
            username="newbalanceuser"
        )
        
        await handlers.balance_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "1000" in reply_text


class TestDailyHandler:
    """测试 /daily 命令处理器"""
    
    async def test_daily_first_claim_success(self, handlers):
        """测试首次签到成功"""
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=55555,
            username="dailyuser"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 签到
        mock_update2, mock_context2 = create_mock_update(
            user_id=55555,
            username="dailyuser"
        )
        await handlers.daily_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "签到成功" in reply_text
        assert "500" in reply_text
    
    async def test_daily_cooldown_message(self, handlers):
        """测试签到冷却提示"""
        # 先创建用户并签到
        mock_update1, mock_context1 = create_mock_update(
            user_id=66666,
            username="cooldownuser"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        mock_update2, mock_context2 = create_mock_update(
            user_id=66666,
            username="cooldownuser"
        )
        await handlers.daily_handler(mock_update2, mock_context2)
        
        # 再次签到（应该显示冷却）
        mock_update3, mock_context3 = create_mock_update(
            user_id=66666,
            username="cooldownuser"
        )
        await handlers.daily_handler(mock_update3, mock_context3)
        
        reply_text = mock_update3.message.reply_text.call_args[0][0]
        assert "冷却" in reply_text or "等待" in reply_text


class TestTopHandler:
    """测试 /top 命令处理器"""
    
    async def test_top_shows_empty_message(self, handlers):
        """测试排行榜为空时的提示"""
        mock_update, mock_context = create_mock_update()
        
        await handlers.top_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "排行榜" in reply_text
    
    async def test_top_shows_users_ranked(self, handlers):
        """测试排行榜显示用户排名"""
        # 创建多个用户
        for i, (user_id, username) in enumerate([
            (77771, "user1"),
            (77772, "user2"),
            (77773, "user3")
        ]):
            mock_update, mock_context = create_mock_update(
                user_id=user_id,
                username=username
            )
            await handlers.start_handler(mock_update, mock_context)
        
        # 查看排行榜
        mock_update, mock_context = create_mock_update()
        await handlers.top_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "排行榜" in reply_text
        assert "user1" in reply_text or "user2" in reply_text or "user3" in reply_text


class TestPayHandler:
    """测试 /pay 命令处理器"""
    
    async def test_pay_missing_args_shows_usage(self, handlers):
        """测试缺少参数时显示用法说明"""
        mock_update, mock_context = create_mock_update(
            user_id=88881,
            username="payuser",
            args=[]  # 没有参数
        )
        
        await handlers.pay_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "格式错误" in reply_text or "用法" in reply_text
    
    async def test_pay_invalid_amount_shows_error(self, handlers):
        """测试无效金额显示错误"""
        mock_update, mock_context = create_mock_update(
            user_id=88882,
            username="payuser2",
            args=["@target", "abc"]  # 无效金额
        )
        
        await handlers.pay_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "无效" in reply_text or "金额" in reply_text
    
    async def test_pay_zero_amount_shows_error(self, handlers):
        """测试零金额显示错误"""
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=88883,
            username="payuser3"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        mock_update, mock_context = create_mock_update(
            user_id=88883,
            username="payuser3",
            args=["@target", "0"]
        )
        
        await handlers.pay_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "大于 0" in reply_text or "必须" in reply_text
    
    async def test_pay_negative_amount_shows_error(self, handlers):
        """测试负数金额显示错误"""
        mock_update1, mock_context1 = create_mock_update(
            user_id=88884,
            username="payuser4"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        mock_update, mock_context = create_mock_update(
            user_id=88884,
            username="payuser4",
            args=["@target", "-100"]
        )
        
        await handlers.pay_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "大于 0" in reply_text or "必须" in reply_text
    
    async def test_pay_target_not_found_shows_error(self, handlers):
        """测试找不到目标用户显示错误"""
        mock_update1, mock_context1 = create_mock_update(
            user_id=88885,
            username="payuser5"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        mock_update, mock_context = create_mock_update(
            user_id=88885,
            username="payuser5",
            args=["@nonexistent", "100"]
        )
        
        await handlers.pay_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "找不到" in reply_text or "不存在" in reply_text


class TestCommandFormatErrors:
    """测试命令格式错误处理 - 需求 11.1"""
    
    async def test_pay_shows_correct_format_example(self, handlers):
        """测试 /pay 命令格式错误时显示正确示例"""
        mock_update, mock_context = create_mock_update(
            user_id=99991,
            username="formatuser",
            args=[]
        )
        
        await handlers.pay_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        # 应该包含示例
        assert "/pay" in reply_text
        assert "示例" in reply_text or "@" in reply_text


class TestParameterValidationFeedback:
    """测试参数验证反馈 - 需求 11.3"""
    
    async def test_pay_invalid_amount_specifies_reason(self, handlers):
        """测试无效金额时指出具体原因"""
        mock_update, mock_context = create_mock_update(
            user_id=99992,
            username="validationuser",
            args=["@target", "not_a_number"]
        )
        
        await handlers.pay_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        # 应该指出金额无效
        assert "金额" in reply_text
        assert "无效" in reply_text or "整数" in reply_text


class TestPayHandlerTransfer:
    """测试 /pay 转账功能 - 需求 3.3, 3.4, 3.5"""
    
    async def test_pay_insufficient_balance_shows_error(self, handlers):
        """测试余额不足时显示错误 - 需求 3.3"""
        # 创建发送者（初始 1000 金币）
        mock_update1, mock_context1 = create_mock_update(
            user_id=100001,
            username="sender1"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 创建接收者
        mock_update2, mock_context2 = create_mock_update(
            user_id=100002,
            username="receiver1"
        )
        await handlers.start_handler(mock_update2, mock_context2)
        
        # 创建带有 text_mention 的转账请求（金额超过余额）
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = 100002
        
        mock_update3, mock_context3 = create_mock_update(
            user_id=100001,
            username="sender1",
            args=["@receiver1", "2000"],  # 超过 1000 余额
            entities=[mock_entity]
        )
        
        await handlers.pay_handler(mock_update3, mock_context3)
        
        reply_text = mock_update3.message.reply_text.call_args[0][0]
        assert "余额不足" in reply_text or "不足" in reply_text
    
    async def test_pay_to_self_shows_error(self, handlers):
        """测试向自己转账显示错误 - 需求 3.6"""
        # 创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=100003,
            username="selfpayer"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 创建带有 text_mention 指向自己的转账请求
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = 100003  # 同一个用户
        
        mock_update2, mock_context2 = create_mock_update(
            user_id=100003,
            username="selfpayer",
            args=["@selfpayer", "100"],
            entities=[mock_entity]
        )
        
        await handlers.pay_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "自己" in reply_text
    
    async def test_pay_success_shows_confirmation(self, handlers):
        """测试转账成功显示确认消息 - 需求 3.7"""
        # 创建发送者
        mock_update1, mock_context1 = create_mock_update(
            user_id=100004,
            username="sender2"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 创建接收者
        mock_update2, mock_context2 = create_mock_update(
            user_id=100005,
            username="receiver2"
        )
        await handlers.start_handler(mock_update2, mock_context2)
        
        # 创建带有 text_mention 的转账请求
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = 100005
        
        mock_update3, mock_context3 = create_mock_update(
            user_id=100004,
            username="sender2",
            args=["@receiver2", "100"],
            entities=[mock_entity]
        )
        
        await handlers.pay_handler(mock_update3, mock_context3)
        
        reply_text = mock_update3.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
        assert "100" in reply_text
    
    async def test_pay_with_reply_to_message(self, handlers):
        """测试通过回复消息进行转账"""
        # 创建发送者
        mock_update1, mock_context1 = create_mock_update(
            user_id=100006,
            username="sender3"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 创建接收者
        mock_update2, mock_context2 = create_mock_update(
            user_id=100007,
            username="receiver3"
        )
        await handlers.start_handler(mock_update2, mock_context2)
        
        # 创建模拟的回复消息
        mock_reply_user = MagicMock()
        mock_reply_user.id = 100007
        
        mock_reply_message = MagicMock()
        mock_reply_message.from_user = mock_reply_user
        
        mock_update3, mock_context3 = create_mock_update(
            user_id=100006,
            username="sender3",
            args=["@receiver3", "50"],
            reply_to_message=mock_reply_message
        )
        
        await handlers.pay_handler(mock_update3, mock_context3)
        
        reply_text = mock_update3.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
    
    async def test_pay_receiver_not_exists_shows_error(self, handlers):
        """测试接收者不存在时显示错误 - 需求 3.5"""
        # 创建发送者
        mock_update1, mock_context1 = create_mock_update(
            user_id=100008,
            username="sender4"
        )
        await handlers.start_handler(mock_update1, mock_context1)
        
        # 创建带有 text_mention 指向不存在用户的转账请求
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = 999999  # 不存在的用户
        
        mock_update2, mock_context2 = create_mock_update(
            user_id=100008,
            username="sender4",
            args=["@nonexistent", "100"],
            entities=[mock_entity]
        )
        
        await handlers.pay_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "不存在" in reply_text


# 导入21点相关模块
from src.game_engine import GameEngine
from src.blackjack import BlackjackManager


@pytest.fixture
async def handlers_with_blackjack(db_manager):
    """创建带有21点管理器的 BotHandlers 实例"""
    user_repo = UserRepository(db_manager)
    tx_repo = TransactionRepository(db_manager)
    account_manager = AccountManager(user_repo, tx_repo)
    game_engine = GameEngine(account_manager, tx_repo)
    blackjack_manager = BlackjackManager(account_manager, tx_repo)
    
    return BotHandlers(
        account_manager, 
        user_repo, 
        tx_repo,
        game_engine=game_engine,
        blackjack_manager=blackjack_manager
    )


def create_mock_callback_query(
    user_id: int = 12345,
    username: str = "testuser",
    callback_data: str = "bj_hit"
) -> tuple[Update, MagicMock]:
    """创建模拟的 CallbackQuery Update 对象"""
    # 创建模拟用户
    mock_user = MagicMock(spec=TelegramUser)
    mock_user.id = user_id
    mock_user.username = username
    mock_user.first_name = username
    
    # 创建模拟 CallbackQuery
    mock_query = MagicMock()
    mock_query.data = callback_data
    mock_query.answer = AsyncMock()
    mock_query.edit_message_text = AsyncMock()
    
    # 创建模拟 Update
    mock_update = MagicMock(spec=Update)
    mock_update.effective_user = mock_user
    mock_update.callback_query = mock_query
    mock_update.message = None
    
    # 创建模拟 context
    mock_context = MagicMock()
    mock_context.args = []
    
    return mock_update, mock_context


class TestBlackjackHandler:
    """测试 /bj 命令处理器 - 需求 7.1"""
    
    async def test_bj_missing_args_shows_usage(self, handlers_with_blackjack):
        """测试缺少参数时显示用法说明"""
        mock_update, mock_context = create_mock_update(
            user_id=200001,
            username="bjuser1",
            args=[]
        )
        
        await handlers_with_blackjack.blackjack_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "格式错误" in reply_text or "用法" in reply_text
        assert "/bj" in reply_text
    
    async def test_bj_invalid_amount_shows_error(self, handlers_with_blackjack):
        """测试无效金额显示错误"""
        mock_update, mock_context = create_mock_update(
            user_id=200002,
            username="bjuser2",
            args=["abc"]
        )
        
        await handlers_with_blackjack.blackjack_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "无效" in reply_text or "金额" in reply_text
    
    async def test_bj_zero_amount_shows_error(self, handlers_with_blackjack):
        """测试零金额显示错误"""
        mock_update, mock_context = create_mock_update(
            user_id=200003,
            username="bjuser3",
            args=["0"]
        )
        
        await handlers_with_blackjack.blackjack_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "大于 0" in reply_text or "必须" in reply_text
    
    async def test_bj_negative_amount_shows_error(self, handlers_with_blackjack):
        """测试负数金额显示错误"""
        mock_update, mock_context = create_mock_update(
            user_id=200004,
            username="bjuser4",
            args=["-100"]
        )
        
        await handlers_with_blackjack.blackjack_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "大于 0" in reply_text or "必须" in reply_text
    
    async def test_bj_insufficient_balance_shows_error(self, handlers_with_blackjack):
        """测试余额不足显示错误"""
        # 先创建用户（初始 1000 金币）
        mock_update1, mock_context1 = create_mock_update(
            user_id=200005,
            username="bjuser5"
        )
        await handlers_with_blackjack.start_handler(mock_update1, mock_context1)
        
        # 尝试下注超过余额
        mock_update2, mock_context2 = create_mock_update(
            user_id=200005,
            username="bjuser5",
            args=["2000"]
        )
        
        await handlers_with_blackjack.blackjack_handler(mock_update2, mock_context2)
        
        reply_text = mock_update2.message.reply_text.call_args[0][0]
        assert "余额不足" in reply_text or "不足" in reply_text
    
    async def test_bj_start_game_success(self, handlers_with_blackjack):
        """测试成功开始游戏 - 需求 7.1"""
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=200006,
            username="bjuser6"
        )
        await handlers_with_blackjack.start_handler(mock_update1, mock_context1)
        
        # 开始游戏
        mock_update2, mock_context2 = create_mock_update(
            user_id=200006,
            username="bjuser6",
            args=["100"]
        )
        
        await handlers_with_blackjack.blackjack_handler(mock_update2, mock_context2)
        
        # 验证回复被调用
        assert mock_update2.message.reply_text.called
        
        # 获取回复内容
        call_args = mock_update2.message.reply_text.call_args
        reply_text = call_args[0][0]
        
        # 验证消息包含游戏信息
        assert "21点" in reply_text or "手牌" in reply_text or "下注" in reply_text
    
    async def test_bj_duplicate_game_shows_error(self, handlers_with_blackjack):
        """测试重复开始游戏显示错误"""
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=200007,
            username="bjuser7"
        )
        await handlers_with_blackjack.start_handler(mock_update1, mock_context1)
        
        # 开始第一个游戏
        mock_update2, mock_context2 = create_mock_update(
            user_id=200007,
            username="bjuser7",
            args=["100"]
        )
        await handlers_with_blackjack.blackjack_handler(mock_update2, mock_context2)
        
        # 检查游戏是否已结束（可能是 Blackjack）
        game = handlers_with_blackjack.blackjack_manager.get_game(200007)
        
        if game is not None and not game.is_finished:
            # 尝试开始第二个游戏
            mock_update3, mock_context3 = create_mock_update(
                user_id=200007,
                username="bjuser7",
                args=["100"]
            )
            await handlers_with_blackjack.blackjack_handler(mock_update3, mock_context3)
            
            reply_text = mock_update3.message.reply_text.call_args[0][0]
            assert "进行中" in reply_text or "已有" in reply_text


class TestBlackjackCallbackHandler:
    """测试21点按钮回调处理器 - 需求 7.3, 7.4, 7.5"""
    
    async def test_bj_hit_no_game_shows_error(self, handlers_with_blackjack):
        """测试没有游戏时要牌显示错误"""
        mock_update, mock_context = create_mock_callback_query(
            user_id=300001,
            username="cbuser1",
            callback_data="bj_hit"
        )
        
        await handlers_with_blackjack.blackjack_callback_handler(mock_update, mock_context)
        
        # 验证 answer 被调用
        mock_update.callback_query.answer.assert_called_once()
        
        # 验证 edit_message_text 被调用
        mock_update.callback_query.edit_message_text.assert_called_once()
        reply_text = mock_update.callback_query.edit_message_text.call_args[0][0]
        assert "没有" in reply_text or "游戏" in reply_text
    
    async def test_bj_stand_no_game_shows_error(self, handlers_with_blackjack):
        """测试没有游戏时停牌显示错误"""
        mock_update, mock_context = create_mock_callback_query(
            user_id=300002,
            username="cbuser2",
            callback_data="bj_stand"
        )
        
        await handlers_with_blackjack.blackjack_callback_handler(mock_update, mock_context)
        
        mock_update.callback_query.answer.assert_called_once()
        reply_text = mock_update.callback_query.edit_message_text.call_args[0][0]
        assert "没有" in reply_text or "游戏" in reply_text
    
    async def test_bj_double_no_game_shows_error(self, handlers_with_blackjack):
        """测试没有游戏时加倍显示错误"""
        mock_update, mock_context = create_mock_callback_query(
            user_id=300003,
            username="cbuser3",
            callback_data="bj_double"
        )
        
        await handlers_with_blackjack.blackjack_callback_handler(mock_update, mock_context)
        
        mock_update.callback_query.answer.assert_called_once()
        reply_text = mock_update.callback_query.edit_message_text.call_args[0][0]
        assert "没有" in reply_text or "游戏" in reply_text
    
    async def test_bj_hit_with_active_game(self, handlers_with_blackjack):
        """测试有游戏时要牌 - 需求 7.3"""
        user_id = 300004
        
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=user_id,
            username="cbuser4"
        )
        await handlers_with_blackjack.start_handler(mock_update1, mock_context1)
        
        # 开始游戏
        mock_update2, mock_context2 = create_mock_update(
            user_id=user_id,
            username="cbuser4",
            args=["100"]
        )
        await handlers_with_blackjack.blackjack_handler(mock_update2, mock_context2)
        
        # 检查游戏是否已结束（可能是 Blackjack）
        game = handlers_with_blackjack.blackjack_manager.get_game(user_id)
        
        if game is not None and not game.is_finished:
            initial_cards = len(game.player_cards)
            
            # 要牌
            mock_update3, mock_context3 = create_mock_callback_query(
                user_id=user_id,
                username="cbuser4",
                callback_data="bj_hit"
            )
            await handlers_with_blackjack.blackjack_callback_handler(mock_update3, mock_context3)
            
            # 验证回调被处理
            mock_update3.callback_query.answer.assert_called_once()
            mock_update3.callback_query.edit_message_text.assert_called_once()
    
    async def test_bj_stand_with_active_game(self, handlers_with_blackjack):
        """测试有游戏时停牌 - 需求 7.4"""
        user_id = 300005
        
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=user_id,
            username="cbuser5"
        )
        await handlers_with_blackjack.start_handler(mock_update1, mock_context1)
        
        # 开始游戏
        mock_update2, mock_context2 = create_mock_update(
            user_id=user_id,
            username="cbuser5",
            args=["100"]
        )
        await handlers_with_blackjack.blackjack_handler(mock_update2, mock_context2)
        
        # 检查游戏是否已结束
        game = handlers_with_blackjack.blackjack_manager.get_game(user_id)
        
        if game is not None and not game.is_finished:
            # 停牌
            mock_update3, mock_context3 = create_mock_callback_query(
                user_id=user_id,
                username="cbuser5",
                callback_data="bj_stand"
            )
            await handlers_with_blackjack.blackjack_callback_handler(mock_update3, mock_context3)
            
            # 验证回调被处理
            mock_update3.callback_query.answer.assert_called_once()
            mock_update3.callback_query.edit_message_text.assert_called_once()
            
            # 验证游戏结束
            reply_text = mock_update3.callback_query.edit_message_text.call_args[0][0]
            # 游戏结束后应该显示结果
            assert "点数" in reply_text or "余额" in reply_text
    
    async def test_bj_double_with_active_game(self, handlers_with_blackjack):
        """测试有游戏时加倍 - 需求 7.5"""
        user_id = 300006
        
        # 先创建用户
        mock_update1, mock_context1 = create_mock_update(
            user_id=user_id,
            username="cbuser6"
        )
        await handlers_with_blackjack.start_handler(mock_update1, mock_context1)
        
        # 开始游戏
        mock_update2, mock_context2 = create_mock_update(
            user_id=user_id,
            username="cbuser6",
            args=["100"]
        )
        await handlers_with_blackjack.blackjack_handler(mock_update2, mock_context2)
        
        # 检查游戏是否已结束
        game = handlers_with_blackjack.blackjack_manager.get_game(user_id)
        
        if game is not None and not game.is_finished:
            # 加倍
            mock_update3, mock_context3 = create_mock_callback_query(
                user_id=user_id,
                username="cbuser6",
                callback_data="bj_double"
            )
            await handlers_with_blackjack.blackjack_callback_handler(mock_update3, mock_context3)
            
            # 验证回调被处理
            mock_update3.callback_query.answer.assert_called_once()
            mock_update3.callback_query.edit_message_text.assert_called_once()
    
    async def test_bj_unknown_callback_shows_error(self, handlers_with_blackjack):
        """测试未知回调显示错误"""
        mock_update, mock_context = create_mock_callback_query(
            user_id=300007,
            username="cbuser7",
            callback_data="bj_unknown"
        )
        
        await handlers_with_blackjack.blackjack_callback_handler(mock_update, mock_context)
        
        mock_update.callback_query.answer.assert_called_once()
        reply_text = mock_update.callback_query.edit_message_text.call_args[0][0]
        assert "未知" in reply_text or "操作" in reply_text
