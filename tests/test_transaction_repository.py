"""
交易仓储层单元测试
测试交易日志记录和查询功能
"""
import pytest
import os
import tempfile
from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository


@pytest.fixture
async def repos():
    """创建临时数据库和仓储用于测试"""
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    
    yield user_repo, tx_repo
    
    await db.close()
    if os.path.exists(db_path):
        os.unlink(db_path)


class TestTransactionLogging:
    """测试交易日志记录"""
    
    async def test_log_transaction_creates_record(self, repos):
        """测试记录交易日志"""
        user_repo, tx_repo = repos
        
        # 创建用户
        user = await user_repo.create_user(12345, 'testuser')
        
        # 记录交易
        await tx_repo.log_transaction(
            user_id=user.telegram_id,
            amount=500,
            transaction_type='daily',
            description='每日签到奖励'
        )
        
        # 查询交易历史
        history = await tx_repo.get_user_history(user.telegram_id)
        
        assert len(history) == 1
        assert history[0].user_id == user.telegram_id
        assert history[0].amount == 500
        assert history[0].type == 'daily'
        assert history[0].description == '每日签到奖励'
    
    async def test_log_multiple_transactions(self, repos):
        """测试记录多个交易"""
        user_repo, tx_repo = repos
        
        # 创建用户
        user = await user_repo.create_user(12345, 'testuser')
        
        # 记录多个交易
        transactions = [
            (500, 'daily', '每日签到'),
            (-100, 'dice', '骰子游戏'),
            (200, 'dice', '骰子游戏赢'),
            (-50, 'transfer_send', '转账给其他用户')
        ]
        
        for amount, tx_type, desc in transactions:
            await tx_repo.log_transaction(user.telegram_id, amount, tx_type, desc)
        
        # 查询交易历史
        history = await tx_repo.get_user_history(user.telegram_id)
        
        assert len(history) == 4
        # 验证所有交易都被记录
        descriptions = {tx.description for tx in history}
        assert '每日签到' in descriptions
        assert '骰子游戏' in descriptions
        assert '骰子游戏赢' in descriptions
        assert '转账给其他用户' in descriptions
    
    async def test_log_transaction_without_description(self, repos):
        """测试记录没有描述的交易"""
        user_repo, tx_repo = repos
        
        user = await user_repo.create_user(12345, 'testuser')
        
        await tx_repo.log_transaction(
            user_id=user.telegram_id,
            amount=100,
            transaction_type='admin_add'
        )
        
        history = await tx_repo.get_user_history(user.telegram_id)
        
        assert len(history) == 1
        assert history[0].description is None


class TestTransactionHistory:
    """测试交易历史查询"""
    
    async def test_get_user_history_returns_empty_for_new_user(self, repos):
        """测试新用户的交易历史为空"""
        user_repo, tx_repo = repos
        
        user = await user_repo.create_user(12345, 'testuser')
        history = await tx_repo.get_user_history(user.telegram_id)
        
        assert len(history) == 0
    
    async def test_get_user_history_respects_limit(self, repos):
        """测试交易历史查询限制"""
        user_repo, tx_repo = repos
        
        user = await user_repo.create_user(12345, 'testuser')
        
        # 记录 10 个交易
        for i in range(10):
            await tx_repo.log_transaction(
                user.telegram_id,
                i * 10,
                'test',
                f'交易 {i}'
            )
        
        # 只查询最近 5 个
        history = await tx_repo.get_user_history(user.telegram_id, limit=5)
        
        assert len(history) == 5
        # 验证返回的是 5 个交易
        descriptions = {tx.description for tx in history}
        assert len(descriptions) == 5
    
    async def test_get_user_history_only_returns_user_transactions(self, repos):
        """测试只返回指定用户的交易"""
        user_repo, tx_repo = repos
        
        # 创建两个用户
        user1 = await user_repo.create_user(11111, 'user1')
        user2 = await user_repo.create_user(22222, 'user2')
        
        # 为两个用户记录交易
        await tx_repo.log_transaction(user1.telegram_id, 100, 'test', 'user1 交易')
        await tx_repo.log_transaction(user2.telegram_id, 200, 'test', 'user2 交易')
        await tx_repo.log_transaction(user1.telegram_id, 300, 'test', 'user1 交易2')
        
        # 查询 user1 的历史
        history1 = await tx_repo.get_user_history(user1.telegram_id)
        
        assert len(history1) == 2
        assert all(tx.user_id == user1.telegram_id for tx in history1)
        
        # 查询 user2 的历史
        history2 = await tx_repo.get_user_history(user2.telegram_id)
        
        assert len(history2) == 1
        assert history2[0].user_id == user2.telegram_id


class TestTransactionTypes:
    """测试不同类型的交易"""
    
    async def test_log_different_transaction_types(self, repos):
        """测试记录不同类型的交易"""
        user_repo, tx_repo = repos
        
        user = await user_repo.create_user(12345, 'testuser')
        
        # 记录各种类型的交易
        transaction_types = [
            'daily',
            'dice',
            'slot',
            'blackjack',
            'transfer_send',
            'transfer_receive',
            'admin_add',
            'admin_remove'
        ]
        
        for tx_type in transaction_types:
            await tx_repo.log_transaction(
                user.telegram_id,
                100,
                tx_type,
                f'{tx_type} 交易'
            )
        
        history = await tx_repo.get_user_history(user.telegram_id, limit=100)
        
        assert len(history) == len(transaction_types)
        
        # 验证所有类型都被记录
        recorded_types = {tx.type for tx in history}
        assert recorded_types == set(transaction_types)
