"""
管理员功能属性测试
使用 Hypothesis 验证管理员功能的正确性属性

属性 25: 管理员金币操作
属性 26: 管理员权限验证
属性 27: 管理员重置操作
属性 28: 管理员操作审计

验证需求: 8.1, 8.2, 8.3, 8.4, 8.5
"""
import pytest
import os
import tempfile
from hypothesis import given, strategies as st, settings, assume
from unittest.mock import AsyncMock, MagicMock
from telegram import Update, User as TelegramUser, Message

from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository
from src.account_manager import AccountManager
from src.bot import BotHandlers


def create_mock_update(
    user_id: int,
    username: str = "testuser",
    text: str = "/admin_add",
    args: list = None,
    reply_to_message: Message = None,
    entities: list = None
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
    
    mock_update = MagicMock(spec=Update)
    mock_update.effective_user = mock_user
    mock_update.message = mock_message
    
    mock_context = MagicMock()
    mock_context.args = args or []
    
    return mock_update, mock_context


# Feature: telegram-game-bot, Property 25: 管理员金币操作
@settings(max_examples=10)
@given(
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999),
    add_amount=st.integers(min_value=1, max_value=10000)
)
@pytest.mark.asyncio
async def test_property_admin_add_coins(admin_id, target_id, add_amount):
    """
    属性 25: 管理员金币操作（添加金币）
    对于任何管理员的添加金币命令，目标用户余额应该相应增加指定金额
    **验证需求: 8.1**
    """
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        # 创建带管理员权限的 handlers
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        # 创建目标用户
        target_user = await user_repo.create_user(target_id, 'target')
        initial_balance = target_user.balance
        
        # 创建带有 text_mention 的请求
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update, mock_context = create_mock_update(
            user_id=admin_id,
            username="admin",
            args=["@target", str(add_amount)],
            entities=[mock_entity]
        )
        
        # 执行管理员添加金币
        await handlers.admin_add_handler(mock_update, mock_context)
        
        # 验证余额增加
        updated_user = await user_repo.get_user(target_id)
        assert updated_user.balance == initial_balance + add_amount
        
        # 验证成功消息
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


@settings(max_examples=10)
@given(
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999),
    remove_amount=st.integers(min_value=1, max_value=500)
)
@pytest.mark.asyncio
async def test_property_admin_remove_coins(admin_id, target_id, remove_amount):
    """
    属性 25: 管理员金币操作（扣除金币）
    对于任何管理员的扣除金币命令，目标用户余额应该相应减少指定金额
    **验证需求: 8.2**
    """
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        # 创建目标用户
        target_user = await user_repo.create_user(target_id, 'target')
        initial_balance = target_user.balance
        
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update, mock_context = create_mock_update(
            user_id=admin_id,
            username="admin",
            args=["@target", str(remove_amount)],
            entities=[mock_entity]
        )
        
        await handlers.admin_remove_handler(mock_update, mock_context)
        
        # 验证余额减少
        updated_user = await user_repo.get_user(target_id)
        assert updated_user.balance == initial_balance - remove_amount
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 26: 管理员权限验证
@settings(max_examples=10)
@given(
    non_admin_id=st.integers(min_value=1, max_value=999999999),
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999)
)
@pytest.mark.asyncio
async def test_property_admin_permission_validation(non_admin_id, admin_id, target_id):
    """
    属性 26: 管理员权限验证
    对于任何管理员命令，如果调用者不是管理员，应该拒绝并提示权限不足
    **验证需求: 8.3**
    """
    assume(non_admin_id != admin_id)
    assume(non_admin_id != target_id)
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        # 只有 admin_id 是管理员，non_admin_id 不是
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        # 创建目标用户
        target_user = await user_repo.create_user(target_id, 'target')
        initial_balance = target_user.balance
        
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        # 非管理员尝试添加金币
        mock_update, mock_context = create_mock_update(
            user_id=non_admin_id,
            username="non_admin",
            args=["@target", "1000"],
            entities=[mock_entity]
        )
        
        await handlers.admin_add_handler(mock_update, mock_context)
        
        # 验证被拒绝
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "权限不足" in reply_text
        
        # 验证余额未变化
        unchanged_user = await user_repo.get_user(target_id)
        assert unchanged_user.balance == initial_balance
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


@settings(max_examples=10)
@given(
    non_admin_id=st.integers(min_value=1, max_value=999999999),
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999)
)
@pytest.mark.asyncio
async def test_property_admin_permission_validation_remove(non_admin_id, admin_id, target_id):
    """
    属性 26: 管理员权限验证（扣除金币）
    非管理员尝试扣除金币应该被拒绝
    **验证需求: 8.3**
    """
    assume(non_admin_id != admin_id)
    assume(non_admin_id != target_id)
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        target_user = await user_repo.create_user(target_id, 'target')
        initial_balance = target_user.balance
        
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update, mock_context = create_mock_update(
            user_id=non_admin_id,
            username="non_admin",
            args=["@target", "500"],
            entities=[mock_entity]
        )
        
        await handlers.admin_remove_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "权限不足" in reply_text
        
        unchanged_user = await user_repo.get_user(target_id)
        assert unchanged_user.balance == initial_balance
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


@settings(max_examples=10)
@given(
    non_admin_id=st.integers(min_value=1, max_value=999999999),
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999)
)
@pytest.mark.asyncio
async def test_property_admin_permission_validation_reset(non_admin_id, admin_id, target_id):
    """
    属性 26: 管理员权限验证（重置账户）
    非管理员尝试重置账户应该被拒绝
    **验证需求: 8.3**
    """
    assume(non_admin_id != admin_id)
    assume(non_admin_id != target_id)
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        # 创建目标用户并修改余额
        target_user = await user_repo.create_user(target_id, 'target')
        await user_repo.update_balance(target_id, 5000)
        modified_user = await user_repo.get_user(target_id)
        initial_balance = modified_user.balance
        
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update, mock_context = create_mock_update(
            user_id=non_admin_id,
            username="non_admin",
            args=["@target"],
            entities=[mock_entity]
        )
        
        await handlers.admin_reset_handler(mock_update, mock_context)
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "权限不足" in reply_text
        
        unchanged_user = await user_repo.get_user(target_id)
        assert unchanged_user.balance == initial_balance
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 27: 管理员重置操作
@settings(max_examples=10)
@given(
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999),
    modified_balance=st.integers(min_value=0, max_value=100000)
)
@pytest.mark.asyncio
async def test_property_admin_reset_operation(admin_id, target_id, modified_balance):
    """
    属性 27: 管理员重置操作
    对于任何管理员重置命令，目标用户余额应该重置为 1000，签到时间应该重置为 0
    **验证需求: 8.4**
    """
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        # 创建目标用户并修改状态
        target_user = await user_repo.create_user(target_id, 'target')
        
        # 修改余额和签到时间
        balance_diff = modified_balance - target_user.balance
        await user_repo.update_balance(target_id, balance_diff)
        await user_repo.update_daily_claim(target_id)  # 设置签到时间
        
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update, mock_context = create_mock_update(
            user_id=admin_id,
            username="admin",
            args=["@target"],
            entities=[mock_entity]
        )
        
        await handlers.admin_reset_handler(mock_update, mock_context)
        
        # 验证重置结果
        reset_user = await user_repo.get_user(target_id)
        assert reset_user.balance == 1000
        assert reset_user.last_daily_claim == 0
        
        reply_text = mock_update.message.reply_text.call_args[0][0]
        assert "成功" in reply_text
        assert "重置" in reply_text
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 28: 管理员操作审计
@settings(max_examples=10)
@given(
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999),
    amount=st.integers(min_value=1, max_value=10000)
)
@pytest.mark.asyncio
async def test_property_admin_operation_audit_add(admin_id, target_id, amount):
    """
    属性 28: 管理员操作审计（添加金币）
    对于任何管理员操作，应该在 transactions 表中记录日志
    **验证需求: 8.5**
    """
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        await user_repo.create_user(target_id, 'target')
        
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update, mock_context = create_mock_update(
            user_id=admin_id,
            username="admin",
            args=["@target", str(amount)],
            entities=[mock_entity]
        )
        
        await handlers.admin_add_handler(mock_update, mock_context)
        
        # 验证交易日志
        history = await tx_repo.get_user_history(target_id)
        admin_add_txs = [tx for tx in history if tx.type == 'admin_add']
        
        assert len(admin_add_txs) >= 1
        latest_tx = admin_add_txs[0]
        assert latest_tx.amount == amount
        assert '管理员' in latest_tx.description
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


@settings(max_examples=10)
@given(
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999),
    amount=st.integers(min_value=1, max_value=500)
)
@pytest.mark.asyncio
async def test_property_admin_operation_audit_remove(admin_id, target_id, amount):
    """
    属性 28: 管理员操作审计（扣除金币）
    对于任何管理员扣除操作，应该在 transactions 表中记录日志
    **验证需求: 8.5**
    """
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        await user_repo.create_user(target_id, 'target')
        
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update, mock_context = create_mock_update(
            user_id=admin_id,
            username="admin",
            args=["@target", str(amount)],
            entities=[mock_entity]
        )
        
        await handlers.admin_remove_handler(mock_update, mock_context)
        
        history = await tx_repo.get_user_history(target_id)
        admin_remove_txs = [tx for tx in history if tx.type == 'admin_remove']
        
        assert len(admin_remove_txs) >= 1
        latest_tx = admin_remove_txs[0]
        assert latest_tx.amount == -amount
        assert '管理员' in latest_tx.description
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


@settings(max_examples=10)
@given(
    admin_id=st.integers(min_value=1, max_value=999999999),
    target_id=st.integers(min_value=1, max_value=999999999)
)
@pytest.mark.asyncio
async def test_property_admin_operation_audit_reset(admin_id, target_id):
    """
    属性 28: 管理员操作审计（重置账户）
    对于任何管理员重置操作，应该在 transactions 表中记录日志
    **验证需求: 8.5**
    """
    assume(admin_id != target_id)
    
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_manager = AccountManager(user_repo, tx_repo)
        
        handlers = BotHandlers(
            account_manager, user_repo, tx_repo,
            admin_ids=[admin_id]
        )
        
        # 创建用户并修改余额
        await user_repo.create_user(target_id, 'target')
        await user_repo.update_balance(target_id, 5000)
        
        mock_entity = MagicMock()
        mock_entity.type = "text_mention"
        mock_entity.user = MagicMock()
        mock_entity.user.id = target_id
        
        mock_update, mock_context = create_mock_update(
            user_id=admin_id,
            username="admin",
            args=["@target"],
            entities=[mock_entity]
        )
        
        await handlers.admin_reset_handler(mock_update, mock_context)
        
        history = await tx_repo.get_user_history(target_id)
        admin_reset_txs = [tx for tx in history if tx.type == 'admin_reset']
        
        assert len(admin_reset_txs) >= 1
        latest_tx = admin_reset_txs[0]
        assert '管理员' in latest_tx.description
        assert '重置' in latest_tx.description
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)
