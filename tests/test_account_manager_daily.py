"""
账户管理器 - 每日签到属性测试
使用 Hypothesis 验证每日签到功能的正确性属性
"""
import pytest
import os
import tempfile
import time
from hypothesis import given, strategies as st, settings
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


# Feature: telegram-game-bot, Property 4: 签到时间间隔验证
@settings(max_examples=5)
@given(
    telegram_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',)))
)
@pytest.mark.asyncio
async def test_property_daily_claim_interval_validation(telegram_id, username):
    """
    属性 4: 签到时间间隔验证
    对于任何用户，如果距离上次签到超过 24 小时，应该允许签到并增加 500 金币；
    如果不足 24 小时，应该拒绝并显示剩余时间
    验证需求: 2.1, 2.2
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
        user = await user_repo.create_user(telegram_id, username)
        initial_balance = user.balance
        
        # 首次签到应该成功
        success1, msg1 = await mgr.claim_daily_reward(telegram_id)
        assert success1 is True
        assert '签到成功' in msg1
        
        # 验证余额增加了 500
        user = await user_repo.get_user(telegram_id)
        assert user.balance == initial_balance + 500
        
        # 立即再次签到应该失败（不足 24 小时）
        success2, msg2 = await mgr.claim_daily_reward(telegram_id)
        assert success2 is False
        assert '签到冷却中' in msg2 or '等待' in msg2
        
        # 余额不应该变化
        user = await user_repo.get_user(telegram_id)
        assert user.balance == initial_balance + 500
        
        # 模拟 24 小时后（修改数据库中的时间戳）
        await db.execute(
            "UPDATE users SET last_daily_claim = ? WHERE telegram_id = ?",
            (int(time.time()) - 86401, telegram_id)  # 24小时1秒前
        )
        
        # 现在应该可以再次签到
        success3, msg3 = await mgr.claim_daily_reward(telegram_id)
        assert success3 is True
        assert '签到成功' in msg3
        
        # 验证余额再次增加了 500
        user = await user_repo.get_user(telegram_id)
        assert user.balance == initial_balance + 1000
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 5: 签到时间戳更新
@settings(max_examples=5)
@given(
    telegram_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',)))
)
@pytest.mark.asyncio
async def test_property_daily_claim_timestamp_update(telegram_id, username):
    """
    属性 5: 签到时间戳更新
    对于任何成功的签到操作，用户的 last_daily_claim 时间戳应该更新为当前时间
    验证需求: 2.4
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
        user = await user_repo.create_user(telegram_id, username)
        assert user.last_daily_claim == 0  # 初始为 0
        
        # 签到前记录时间
        before_claim = int(time.time())
        
        # 签到
        success, msg = await mgr.claim_daily_reward(telegram_id)
        assert success is True
        
        # 签到后记录时间
        after_claim = int(time.time())
        
        # 验证时间戳被更新
        user = await user_repo.get_user(telegram_id)
        assert user.last_daily_claim >= before_claim
        assert user.last_daily_claim <= after_claim
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 6: 签到反馈完整性
@settings(max_examples=5)
@given(
    telegram_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',)))
)
@pytest.mark.asyncio
async def test_property_daily_claim_feedback_completeness(telegram_id, username):
    """
    属性 6: 签到反馈完整性
    对于任何签到操作，返回的消息应该包含操作结果和获得的金币数量
    验证需求: 2.5
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
        await user_repo.create_user(telegram_id, username)
        
        # 签到
        success, msg = await mgr.claim_daily_reward(telegram_id)
        
        # 验证消息包含必要信息
        assert isinstance(msg, str)
        assert len(msg) > 0
        
        if success:
            # 成功消息应该包含金币数量
            assert '500' in msg or '金币' in msg
            assert '成功' in msg or '余额' in msg
        else:
            # 失败消息应该包含原因
            assert len(msg) > 0
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# 边界情况测试：首次签到
@pytest.mark.asyncio
async def test_first_time_daily_claim():
    """测试首次签到（last_daily_claim = 0）"""
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        mgr = AccountManager(user_repo, tx_repo)
        
        # 创建用户
        user = await user_repo.create_user(12345, 'testuser')
        assert user.last_daily_claim == 0
        
        # 首次签到应该立即成功
        success, msg = await mgr.claim_daily_reward(12345)
        assert success is True
        assert '签到成功' in msg
        
        # 验证余额增加
        user = await user_repo.get_user(12345)
        assert user.balance == 1500  # 1000 初始 + 500 签到
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)
