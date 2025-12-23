"""
数据库层属性测试
使用 Hypothesis 进行属性测试，验证数据持久化和事务原子性
"""
import pytest
import os
import tempfile
from hypothesis import given, strategies as st, settings
from src.database import DatabaseManager


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


# Feature: telegram-game-bot, Property 29: 数据变更即时持久化
@settings(max_examples=5)
@given(
    telegram_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',))),
    balance=st.integers(min_value=0, max_value=1000000)
)
@pytest.mark.asyncio
async def test_property_data_persistence(telegram_id, username, balance):
    """
    属性 29: 数据变更即时持久化
    对于任何用户数据变更，应该立即写入数据库
    验证需求: 9.1
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        
        # 插入用户数据
        await db.execute(
            "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
            (telegram_id, username, balance, 1000000, 1000000)
        )
        
        # 立即查询，应该能获取到数据
        result = await db.fetch_one(
            "SELECT * FROM users WHERE telegram_id = ?",
            (telegram_id,)
        )
        
        assert result is not None
        assert result['telegram_id'] == telegram_id
        assert result['username'] == username
        assert result['balance'] == balance
        
        # 更新余额
        new_balance = balance + 500
        await db.execute(
            "UPDATE users SET balance = ? WHERE telegram_id = ?",
            (new_balance, telegram_id)
        )
        
        # 立即查询，应该获取到更新后的数据
        result = await db.fetch_one(
            "SELECT balance FROM users WHERE telegram_id = ?",
            (telegram_id,)
        )
        
        assert result['balance'] == new_balance
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 30: 事务原子性
@settings(max_examples=5)
@given(
    user1_id=st.integers(min_value=1, max_value=999999999),
    user2_id=st.integers(min_value=1, max_value=999999999),
    initial_balance1=st.integers(min_value=100, max_value=10000),
    initial_balance2=st.integers(min_value=100, max_value=10000),
    transfer_amount=st.integers(min_value=1, max_value=100)
)
@pytest.mark.asyncio
async def test_property_transaction_atomicity_success(
    user1_id, user2_id, initial_balance1, initial_balance2, transfer_amount
):
    """
    属性 30: 事务原子性（成功情况）
    对于任何涉及多个数据库操作的业务逻辑，要么全部成功要么全部回滚
    验证需求: 9.3, 9.5
    """
    # 确保两个用户 ID 不同
    if user1_id == user2_id:
        return
    
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        
        # 创建两个用户
        await db.execute(
            "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
            (user1_id, 'user1', initial_balance1, 1000000, 1000000)
        )
        await db.execute(
            "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
            (user2_id, 'user2', initial_balance2, 1000000, 1000000)
        )
        
        # 转账事务
        async def deduct(conn):
            await conn.execute(
                "UPDATE users SET balance = balance - ? WHERE telegram_id = ?",
                (transfer_amount, user1_id)
            )
        
        async def add(conn):
            await conn.execute(
                "UPDATE users SET balance = balance + ? WHERE telegram_id = ?",
                (transfer_amount, user2_id)
            )
        
        # 执行事务
        success = await db.transaction([deduct, add])
        assert success is True
        
        # 验证两个用户的余额都正确更新
        user1 = await db.fetch_one("SELECT balance FROM users WHERE telegram_id = ?", (user1_id,))
        user2 = await db.fetch_one("SELECT balance FROM users WHERE telegram_id = ?", (user2_id,))
        
        assert user1['balance'] == initial_balance1 - transfer_amount
        assert user2['balance'] == initial_balance2 + transfer_amount
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# Feature: telegram-game-bot, Property 30: 事务原子性（失败回滚）
@settings(max_examples=5)
@given(
    user_id=st.integers(min_value=1, max_value=999999999),
    initial_balance=st.integers(min_value=100, max_value=10000)
)
@pytest.mark.asyncio
async def test_property_transaction_atomicity_rollback(user_id, initial_balance):
    """
    属性 30: 事务原子性（失败回滚）
    对于任何失败的事务，所有操作都应该回滚
    验证需求: 9.3, 9.4, 9.5
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        
        # 创建用户
        await db.execute(
            "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
            (user_id, 'testuser', initial_balance, 1000000, 1000000)
        )
        
        # 尝试执行会失败的事务
        async def operation1(conn):
            await conn.execute(
                "UPDATE users SET balance = balance + 100 WHERE telegram_id = ?",
                (user_id,)
            )
        
        async def operation2(conn):
            # 这个操作会失败（插入重复的主键）
            await conn.execute(
                "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
                (user_id, 'duplicate', 1000, 1000000, 1000000)
            )
        
        # 事务应该失败并抛出异常
        with pytest.raises(Exception):
            await db.transaction([operation1, operation2])
        
        # 验证第一个操作也被回滚了（余额没有变化）
        user = await db.fetch_one("SELECT balance FROM users WHERE telegram_id = ?", (user_id,))
        assert user['balance'] == initial_balance
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)
