"""
用户仓储层属性测试
使用 Hypothesis 验证用户账户管理的正确性属性
"""
import pytest
import os
import tempfile
import time
from hypothesis import given, strategies as st, settings, assume, HealthCheck
from src.database import DatabaseManager
from src.repositories import UserRepository


@pytest.fixture
async def user_repo():
    """创建临时数据库和用户仓储用于测试"""
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    db = DatabaseManager(db_path)
    await db.initialize()
    repo = UserRepository(db)
    
    yield repo
    
    await db.close()
    if os.path.exists(db_path):
        os.unlink(db_path)


# Feature: telegram-game-bot, Property 1: 新用户自动创建
@settings(max_examples=5)
@given(
    telegram_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',)))
)
@pytest.mark.asyncio
async def test_property_new_user_auto_creation(telegram_id, username):
    """
    属性 1: 新用户自动创建
    对于任何新的 telegram_id，当首次与系统交互时，应该自动创建账户且初始余额为 1000 金币
    验证需求: 1.1, 1.4
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        repo = UserRepository(db)
        
        # 创建新用户
        user = await repo.create_user(telegram_id, username)
        
        # 验证用户被创建且初始余额为 1000
        assert user.telegram_id == telegram_id
        assert user.username == username
        assert user.balance == 1000
        assert user.last_daily_claim == 0
        
        # 从数据库查询验证
        db_user = await repo.get_user(telegram_id)
        assert db_user is not None
        assert db_user.telegram_id == telegram_id
        assert db_user.balance == 1000
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 2: 账户查询幂等性
@settings(max_examples=5)
@given(
    telegram_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',)))
)
@pytest.mark.asyncio
async def test_property_account_query_idempotence(telegram_id, username):
    """
    属性 2: 账户查询幂等性
    对于任何用户，多次调用账户初始化或查询操作应该返回相同的账户状态
    验证需求: 1.2, 1.5
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        repo = UserRepository(db)
        
        # 创建用户
        user1 = await repo.create_user(telegram_id, username)
        
        # 多次查询应该返回相同的数据
        user2 = await repo.get_user(telegram_id)
        user3 = await repo.get_user(telegram_id)
        
        assert user2 is not None
        assert user3 is not None
        assert user2.telegram_id == user1.telegram_id
        assert user2.balance == user1.balance
        assert user3.telegram_id == user1.telegram_id
        assert user3.balance == user1.balance
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 3: 余额查询准确性
@settings(max_examples=5)
@given(
    telegram_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',))),
    balance_changes=st.lists(st.integers(min_value=-500, max_value=500), min_size=1, max_size=10)
)
@pytest.mark.asyncio
async def test_property_balance_query_accuracy(telegram_id, username, balance_changes):
    """
    属性 3: 余额查询准确性
    对于任何用户，查询余额应该返回数据库中存储的当前准确值
    验证需求: 1.3
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        repo = UserRepository(db)
        
        # 创建用户（初始余额 1000）
        await repo.create_user(telegram_id, username)
        expected_balance = 1000
        
        # 应用一系列余额变化
        for change in balance_changes:
            await repo.update_balance(telegram_id, change)
            expected_balance += change
        
        # 查询余额应该等于预期值
        user = await repo.get_user(telegram_id)
        assert user is not None
        assert user.balance == expected_balance
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 11: 排行榜排序正确性
@settings(max_examples=5, suppress_health_check=[HealthCheck.data_too_large, HealthCheck.too_slow])
@given(
    users_data=st.lists(
        st.tuples(
            st.integers(min_value=1, max_value=999999999),  # telegram_id
            st.text(min_size=1, max_size=10, alphabet=st.sampled_from('abcdefghijklmnopqrstuvwxyz0123456789')),  # username
            st.integers(min_value=0, max_value=100000)  # balance
        ),
        min_size=1,
        max_size=10,
        unique_by=lambda x: x[0]  # 确保 telegram_id 唯一
    )
)
@pytest.mark.asyncio
async def test_property_leaderboard_sorting(users_data):
    """
    属性 11: 排行榜排序正确性
    对于任何排行榜查询，返回的用户列表应该按余额降序排列，余额相同时按创建时间升序排列
    验证需求: 4.1, 4.4
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        repo = UserRepository(db)
        
        # 创建所有用户（添加小延迟确保创建时间不同）
        for telegram_id, username, balance in users_data:
            user = await repo.create_user(telegram_id, username)
            # 设置指定的余额
            if balance != 1000:
                await repo.update_balance(telegram_id, balance - 1000)
            time.sleep(0.001)  # 确保创建时间不同
        
        # 获取排行榜
        top_users = await repo.get_top_users(limit=len(users_data))
        
        # 验证排序正确性
        for i in range(len(top_users) - 1):
            current = top_users[i]
            next_user = top_users[i + 1]
            
            # 余额应该降序
            if current.balance == next_user.balance:
                # 余额相同时，创建时间应该升序
                assert current.created_at <= next_user.created_at
            else:
                assert current.balance >= next_user.balance
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 12: 排行榜数据完整性
@settings(max_examples=5)
@given(
    telegram_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',))),
    balance=st.integers(min_value=0, max_value=100000)
)
@pytest.mark.asyncio
async def test_property_leaderboard_data_completeness(telegram_id, username, balance):
    """
    属性 12: 排行榜数据完整性
    对于任何排行榜中的用户条目，应该包含排名、用户名和余额信息
    验证需求: 4.2
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        repo = UserRepository(db)
        
        # 创建用户
        await repo.create_user(telegram_id, username)
        if balance != 1000:
            await repo.update_balance(telegram_id, balance - 1000)
        
        # 获取排行榜
        top_users = await repo.get_top_users(limit=10)
        
        # 验证每个用户都有完整的数据
        assert len(top_users) > 0
        for user in top_users:
            assert user.telegram_id is not None
            assert user.username is not None
            assert user.balance is not None
            assert isinstance(user.telegram_id, int)
            assert isinstance(user.username, str)
            assert isinstance(user.balance, int)
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)
