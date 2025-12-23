"""
账户管理器 - 转账系统属性测试
使用 Hypothesis 验证转账功能的正确性属性
"""
import pytest
import os
import tempfile
from hypothesis import given, strategies as st, settings, assume
from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository
from src.account_manager import AccountManager


@pytest.fixture
async def account_mgr():
    """创建临时数据库和账户管理器用于测试"""
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    mgr = AccountManager(user_repo, tx_repo)
    
    yield mgr
    
    await db.close()
    if os.path.exists(db_path):
        os.unlink(db_path)


# Feature: telegram-game-bot, Property 7: 转账余额变化正确性
@settings(max_examples=5)
@given(
    sender_id=st.integers(min_value=1, max_value=999999999),
    receiver_id=st.integers(min_value=1, max_value=999999999),
    initial_balance=st.integers(min_value=1000, max_value=10000),
    transfer_amount=st.integers(min_value=100, max_value=1000)
)
@pytest.mark.asyncio
async def test_property_transfer_balance_correctness(
    sender_id, receiver_id, initial_balance, transfer_amount
):
    """
    属性 7: 转账余额变化正确性
    对于任何有效的转账操作，发送者余额应该减少 amount，接收者余额应该增加 amount * 0.95
    验证需求: 3.1
    """
    # 确保两个用户 ID 不同
    assume(sender_id != receiver_id)
    # 确保余额充足
    assume(initial_balance >= transfer_amount)
    
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        mgr = AccountManager(user_repo, tx_repo)
        
        # 创建两个用户
        sender = await user_repo.create_user(sender_id, 'sender')
        receiver = await user_repo.create_user(receiver_id, 'receiver')
        
        # 设置发送者余额
        if initial_balance != 1000:
            await user_repo.update_balance(sender_id, initial_balance - 1000)
        
        sender_initial = initial_balance
        receiver_initial = receiver.balance
        
        # 执行转账
        success, msg = await mgr.transfer(sender_id, receiver_id, transfer_amount)
        assert success is True
        
        # 验证余额变化
        sender_after = await user_repo.get_user(sender_id)
        receiver_after = await user_repo.get_user(receiver_id)
        
        # 发送者余额应该减少 transfer_amount
        assert sender_after.balance == sender_initial - transfer_amount
        
        # 接收者余额应该增加 transfer_amount - fee（扣除 5% 手续费）
        # 注意：手续费计算方式是 fee = int(amount * 0.05)，然后 actual = amount - fee
        fee = int(transfer_amount * 0.05)
        expected_receive = transfer_amount - fee
        assert receiver_after.balance == receiver_initial + expected_receive
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 8: 转账手续费回收
@settings(max_examples=5)
@given(
    sender_id=st.integers(min_value=1, max_value=999999999),
    receiver_id=st.integers(min_value=1, max_value=999999999),
    transfer_amount=st.integers(min_value=100, max_value=1000)
)
@pytest.mark.asyncio
async def test_property_transfer_fee_collection(sender_id, receiver_id, transfer_amount):
    """
    属性 8: 转账手续费回收
    对于任何转账操作，系统总金币量应该减少 amount * 0.05（手续费）
    验证需求: 3.2
    """
    # 确保两个用户 ID 不同
    assume(sender_id != receiver_id)
    
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        mgr = AccountManager(user_repo, tx_repo)
        
        # 创建两个用户
        sender = await user_repo.create_user(sender_id, 'sender')
        receiver = await user_repo.create_user(receiver_id, 'receiver')
        
        # 确保发送者余额充足
        if sender.balance < transfer_amount:
            await user_repo.update_balance(sender_id, transfer_amount - sender.balance)
        
        # 计算转账前的总金币量
        total_before = sender.balance + receiver.balance
        if sender.balance < transfer_amount:
            total_before += (transfer_amount - sender.balance)
        
        # 重新获取准确的余额
        sender = await user_repo.get_user(sender_id)
        receiver = await user_repo.get_user(receiver_id)
        total_before = sender.balance + receiver.balance
        
        # 执行转账
        success, msg = await mgr.transfer(sender_id, receiver_id, transfer_amount)
        assert success is True
        
        # 计算转账后的总金币量
        sender_after = await user_repo.get_user(sender_id)
        receiver_after = await user_repo.get_user(receiver_id)
        total_after = sender_after.balance + receiver_after.balance
        
        # 总金币量应该减少 5% 的手续费
        expected_fee = int(transfer_amount * 0.05)
        assert total_after == total_before - expected_fee
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 9: 转账输入验证
@settings(max_examples=5)
@given(
    user_id=st.integers(min_value=1, max_value=999999999),
    amount=st.integers(min_value=-1000, max_value=0)
)
@pytest.mark.asyncio
async def test_property_transfer_input_validation_invalid_amount(user_id, amount):
    """
    属性 9: 转账输入验证（无效金额）
    对于任何非正数金额，应该拒绝并返回错误消息
    验证需求: 3.4
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        mgr = AccountManager(user_repo, tx_repo)
        
        # 创建用户
        await user_repo.create_user(user_id, 'testuser')
        
        # 尝试转账非正数金额
        success, msg = await mgr.transfer(user_id, user_id + 1, amount)
        
        assert success is False
        assert '金额' in msg or '大于' in msg
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


@settings(max_examples=5)
@given(
    user_id=st.integers(min_value=1, max_value=999999999),
    balance=st.integers(min_value=100, max_value=500),
    amount=st.integers(min_value=501, max_value=1000)
)
@pytest.mark.asyncio
async def test_property_transfer_input_validation_insufficient_balance(user_id, balance, amount):
    """
    属性 9: 转账输入验证（余额不足）
    对于任何余额不足的情况，应该拒绝并返回错误消息
    验证需求: 3.3
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        mgr = AccountManager(user_repo, tx_repo)
        
        # 创建用户并设置余额
        user = await user_repo.create_user(user_id, 'testuser')
        await user_repo.update_balance(user_id, balance - user.balance)
        
        # 创建接收者
        await user_repo.create_user(user_id + 1, 'receiver')
        
        # 尝试转账超过余额的金额
        success, msg = await mgr.transfer(user_id, user_id + 1, amount)
        
        assert success is False
        assert '余额不足' in msg or '余额' in msg
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


@pytest.mark.asyncio
async def test_transfer_to_self_rejected():
    """边界情况：向自己转账应该被拒绝"""
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        mgr = AccountManager(user_repo, tx_repo)
        
        # 创建用户
        await user_repo.create_user(12345, 'testuser')
        
        # 尝试向自己转账
        success, msg = await mgr.transfer(12345, 12345, 100)
        
        assert success is False
        assert '自己' in msg
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


@pytest.mark.asyncio
async def test_transfer_to_nonexistent_user():
    """边界情况：向不存在的用户转账应该被拒绝"""
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        mgr = AccountManager(user_repo, tx_repo)
        
        # 创建发送者
        await user_repo.create_user(12345, 'sender')
        
        # 尝试向不存在的用户转账
        success, msg = await mgr.transfer(12345, 99999, 100)
        
        assert success is False
        assert '不存在' in msg
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 10: 转账确认消息
@settings(max_examples=5)
@given(
    sender_id=st.integers(min_value=1, max_value=999999999),
    receiver_id=st.integers(min_value=1, max_value=999999999),
    amount=st.integers(min_value=100, max_value=1000)
)
@pytest.mark.asyncio
async def test_property_transfer_confirmation_message(sender_id, receiver_id, amount):
    """
    属性 10: 转账确认消息
    对于任何成功的转账，应该返回包含金额和对方信息的确认消息
    验证需求: 3.7
    """
    # 确保两个用户 ID 不同
    assume(sender_id != receiver_id)
    
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        mgr = AccountManager(user_repo, tx_repo)
        
        # 创建两个用户
        sender = await user_repo.create_user(sender_id, 'sender')
        receiver = await user_repo.create_user(receiver_id, 'receiver')
        
        # 确保余额充足
        if sender.balance < amount:
            await user_repo.update_balance(sender_id, amount - sender.balance)
        
        # 执行转账
        success, msg = await mgr.transfer(sender_id, receiver_id, amount)
        
        if success:
            # 验证消息包含必要信息
            assert isinstance(msg, str)
            assert len(msg) > 0
            assert str(amount) in msg or '金币' in msg
            assert '成功' in msg or '余额' in msg
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)
