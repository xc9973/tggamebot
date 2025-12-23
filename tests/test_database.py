"""
数据库层单元测试
测试数据库初始化、表创建、事务提交和回滚
"""
import pytest
import os
import tempfile
from src.database import DatabaseManager


@pytest.fixture
async def db_manager():
    """创建临时数据库用于测试"""
    # 创建临时文件
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    db = DatabaseManager(db_path)
    await db.initialize()
    
    yield db
    
    # 清理
    await db.close()
    if os.path.exists(db_path):
        os.unlink(db_path)


class TestDatabaseInitialization:
    """测试数据库初始化"""
    
    async def test_database_creates_users_table(self, db_manager):
        """测试创建 users 表"""
        # 查询表是否存在
        result = await db_manager.fetch_one(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='users'"
        )
        assert result is not None
        assert result['name'] == 'users'
    
    async def test_database_creates_transactions_table(self, db_manager):
        """测试创建 transactions 表"""
        result = await db_manager.fetch_one(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='transactions'"
        )
        assert result is not None
        assert result['name'] == 'transactions'
    
    async def test_database_creates_balance_index(self, db_manager):
        """测试创建余额索引"""
        result = await db_manager.fetch_one(
            "SELECT name FROM sqlite_master WHERE type='index' AND name='idx_balance'"
        )
        assert result is not None
    
    async def test_database_enables_wal_mode(self, db_manager):
        """测试启用 WAL 模式"""
        result = await db_manager.fetch_one("PRAGMA journal_mode")
        assert result['journal_mode'].upper() == 'WAL'


class TestDatabaseOperations:
    """测试数据库 CRUD 操作"""
    
    async def test_execute_insert(self, db_manager):
        """测试插入操作"""
        await db_manager.execute(
            "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
            (12345, 'testuser', 1000, 1000000, 1000000)
        )
        
        result = await db_manager.fetch_one(
            "SELECT * FROM users WHERE telegram_id = ?",
            (12345,)
        )
        assert result is not None
        assert result['telegram_id'] == 12345
        assert result['username'] == 'testuser'
        assert result['balance'] == 1000
    
    async def test_execute_update(self, db_manager):
        """测试更新操作"""
        # 先插入
        await db_manager.execute(
            "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
            (12345, 'testuser', 1000, 1000000, 1000000)
        )
        
        # 更新
        await db_manager.execute(
            "UPDATE users SET balance = ? WHERE telegram_id = ?",
            (2000, 12345)
        )
        
        result = await db_manager.fetch_one(
            "SELECT balance FROM users WHERE telegram_id = ?",
            (12345,)
        )
        assert result['balance'] == 2000
    
    async def test_fetch_one_returns_none_when_no_result(self, db_manager):
        """测试查询不存在的数据返回 None"""
        result = await db_manager.fetch_one(
            "SELECT * FROM users WHERE telegram_id = ?",
            (99999,)
        )
        assert result is None
    
    async def test_fetch_all_returns_multiple_rows(self, db_manager):
        """测试查询多行数据"""
        # 插入多个用户
        for i in range(3):
            await db_manager.execute(
                "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
                (i, f'user{i}', 1000 + i * 100, 1000000, 1000000)
            )
        
        results = await db_manager.fetch_all("SELECT * FROM users ORDER BY telegram_id")
        assert len(results) == 3
        assert results[0]['telegram_id'] == 0
        assert results[1]['telegram_id'] == 1
        assert results[2]['telegram_id'] == 2


class TestDatabaseTransactions:
    """测试事务提交和回滚"""
    
    async def test_transaction_commits_on_success(self, db_manager):
        """测试事务成功时提交"""
        async def operation1(conn):
            await conn.execute(
                "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
                (1, 'user1', 1000, 1000000, 1000000)
            )
        
        async def operation2(conn):
            await conn.execute(
                "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
                (2, 'user2', 2000, 1000000, 1000000)
            )
        
        success = await db_manager.transaction([operation1, operation2])
        assert success is True
        
        # 验证两个用户都被插入
        result1 = await db_manager.fetch_one("SELECT * FROM users WHERE telegram_id = ?", (1,))
        result2 = await db_manager.fetch_one("SELECT * FROM users WHERE telegram_id = ?", (2,))
        assert result1 is not None
        assert result2 is not None
    
    async def test_transaction_rolls_back_on_error(self, db_manager):
        """测试事务失败时回滚"""
        async def operation1(conn):
            await conn.execute(
                "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
                (1, 'user1', 1000, 1000000, 1000000)
            )
        
        async def operation2(conn):
            # 这个操作会失败（重复的主键）
            await conn.execute(
                "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
                (1, 'user2', 2000, 1000000, 1000000)
            )
        
        # 事务应该失败
        with pytest.raises(Exception):
            await db_manager.transaction([operation1, operation2])
        
        # 验证第一个用户也没有被插入（回滚）
        result = await db_manager.fetch_one("SELECT * FROM users WHERE telegram_id = ?", (1,))
        assert result is None
    
    async def test_transaction_with_balance_update(self, db_manager):
        """测试转账事务（扣除和增加余额）"""
        # 先创建两个用户
        await db_manager.execute(
            "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
            (1, 'user1', 1000, 1000000, 1000000)
        )
        await db_manager.execute(
            "INSERT INTO users (telegram_id, username, balance, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
            (2, 'user2', 500, 1000000, 1000000)
        )
        
        # 转账事务
        async def deduct(conn):
            await conn.execute(
                "UPDATE users SET balance = balance - ? WHERE telegram_id = ?",
                (200, 1)
            )
        
        async def add(conn):
            await conn.execute(
                "UPDATE users SET balance = balance + ? WHERE telegram_id = ?",
                (190, 2)  # 扣除 5% 手续费
            )
        
        success = await db_manager.transaction([deduct, add])
        assert success is True
        
        # 验证余额变化
        user1 = await db_manager.fetch_one("SELECT balance FROM users WHERE telegram_id = ?", (1,))
        user2 = await db_manager.fetch_one("SELECT balance FROM users WHERE telegram_id = ?", (2,))
        assert user1['balance'] == 800
        assert user2['balance'] == 690
