"""
并发控制属性测试
使用 Hypothesis 进行属性测试，验证用户操作串行化、游戏会话互斥、用户操作隔离性和余额操作原子性
"""
import pytest
import asyncio
import os
import tempfile
from hypothesis import given, strategies as st, settings
from src.concurrency import UserLockManager, GameSessionManager, ConcurrencyManager
from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository
from src.account_manager import AccountManager


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
async def account_manager(db_manager):
    """创建账户管理器用于测试"""
    user_repo = UserRepository(db_manager)
    tx_repo = TransactionRepository(db_manager)
    return AccountManager(user_repo, tx_repo)


# Feature: telegram-game-bot, Property 32: 用户操作串行化
@settings(max_examples=5)
@given(
    user_id=st.integers(min_value=1, max_value=999999999),
    num_operations=st.integers(min_value=2, max_value=5)
)
@pytest.mark.asyncio
async def test_property_user_operations_serialized(user_id, num_operations):
    """
    属性 32: 用户操作串行化
    对于任何单个用户的多个并发请求，应该按顺序处理，确保状态一致性
    验证需求: 10.1
    """
    lock_manager = UserLockManager()
    execution_order = []
    
    async def operation(op_id: int):
        """模拟一个需要锁保护的操作"""
        await lock_manager.acquire(user_id)
        try:
            execution_order.append(f"start_{op_id}")
            await asyncio.sleep(0.01)  # 模拟操作耗时
            execution_order.append(f"end_{op_id}")
        finally:
            await lock_manager.release(user_id)
    
    # 并发执行多个操作
    tasks = [asyncio.create_task(operation(i)) for i in range(num_operations)]
    await asyncio.gather(*tasks)
    
    # 验证操作是串行执行的（每个操作的 start 和 end 应该连续出现）
    for i in range(num_operations):
        start_idx = execution_order.index(f"start_{i}")
        end_idx = execution_order.index(f"end_{i}")
        # 在 start 和 end 之间不应该有其他操作的 start
        for j in range(num_operations):
            if i != j:
                other_start_idx = execution_order.index(f"start_{j}")
                # 其他操作的 start 不应该在当前操作的 start 和 end 之间
                assert not (start_idx < other_start_idx < end_idx), \
                    f"Operation {j} started while operation {i} was in progress"


# Feature: telegram-game-bot, Property 33: 游戏会话互斥
@settings(max_examples=5)
@given(
    user_id=st.integers(min_value=1, max_value=999999999),
    game_type1=st.sampled_from(['dice', 'slot', 'blackjack']),
    game_type2=st.sampled_from(['dice', 'slot', 'blackjack'])
)
@pytest.mark.asyncio
async def test_property_game_session_mutual_exclusion(user_id, game_type1, game_type2):
    """
    属性 33: 游戏会话互斥
    对于任何用户，如果已有进行中的游戏会话，应该拒绝新的游戏请求
    验证需求: 10.2
    """
    session_manager = GameSessionManager()
    
    # 开始第一个游戏会话
    success1, message1 = await session_manager.start_session(user_id, game_type1)
    assert success1 is True
    assert session_manager.has_active_session(user_id)
    assert session_manager.get_active_game(user_id) == game_type1
    
    # 尝试开始第二个游戏会话（应该被拒绝）
    success2, message2 = await session_manager.start_session(user_id, game_type2)
    assert success2 is False
    assert game_type1 in message2  # 错误消息应该包含当前游戏类型
    
    # 结束第一个会话
    await session_manager.end_session(user_id)
    assert not session_manager.has_active_session(user_id)
    
    # 现在应该可以开始新游戏
    success3, message3 = await session_manager.start_session(user_id, game_type2)
    assert success3 is True
    assert session_manager.get_active_game(user_id) == game_type2
    
    # 清理
    await session_manager.end_session(user_id)


# Feature: telegram-game-bot, Property 34: 用户操作隔离性
@settings(max_examples=5)
@given(
    user1_id=st.integers(min_value=1, max_value=999999999),
    user2_id=st.integers(min_value=1, max_value=999999999)
)
@pytest.mark.asyncio
async def test_property_user_operations_isolated(user1_id, user2_id):
    """
    属性 34: 用户操作隔离性
    对于任何多个不同用户的并发操作，应该互不影响，各自独立处理
    验证需求: 10.3
    """
    # 确保两个用户 ID 不同
    if user1_id == user2_id:
        return
    
    lock_manager = UserLockManager()
    user1_operations = []
    user2_operations = []
    
    async def user1_operation():
        """用户1的操作"""
        await lock_manager.acquire(user1_id)
        try:
            user1_operations.append("start")
            await asyncio.sleep(0.02)  # 模拟较长的操作
            user1_operations.append("end")
        finally:
            await lock_manager.release(user1_id)
    
    async def user2_operation():
        """用户2的操作"""
        await lock_manager.acquire(user2_id)
        try:
            user2_operations.append("start")
            await asyncio.sleep(0.01)  # 模拟较短的操作
            user2_operations.append("end")
        finally:
            await lock_manager.release(user2_id)
    
    # 并发执行两个用户的操作
    await asyncio.gather(user1_operation(), user2_operation())
    
    # 验证两个用户的操作都完成了
    assert user1_operations == ["start", "end"]
    assert user2_operations == ["start", "end"]
    
    # 验证两个用户的锁是独立的
    assert not lock_manager.is_locked(user1_id)
    assert not lock_manager.is_locked(user2_id)


# Feature: telegram-game-bot, Property 35: 余额操作原子性
@settings(max_examples=5)
@given(
    user_id=st.integers(min_value=1, max_value=999999999),
    username=st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',))),
    initial_balance=st.integers(min_value=1000, max_value=10000),
    bet_amount=st.integers(min_value=10, max_value=100)
)
@pytest.mark.asyncio
async def test_property_balance_operations_atomic(user_id, username, initial_balance, bet_amount):
    """
    属性 35: 余额操作原子性
    对于任何余额检查和扣除操作，应该在同一事务中完成，防止竞态条件
    验证需求: 10.4, 10.5
    """
    # 创建临时数据库
    fd, db_path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    
    try:
        db = DatabaseManager(db_path)
        await db.initialize()
        
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_mgr = AccountManager(user_repo, tx_repo)
        lock_manager = UserLockManager()
        
        # 创建用户并设置初始余额
        await user_repo.create_user(user_id, username)
        # 调整余额到指定值
        balance_diff = initial_balance - 1000
        if balance_diff != 0:
            await user_repo.update_balance(user_id, balance_diff)
        
        # 计算可以进行的最大操作次数
        max_operations = initial_balance // bet_amount
        
        successful_operations = 0
        failed_operations = 0
        
        async def bet_operation():
            """模拟一个下注操作（检查余额并扣除）"""
            nonlocal successful_operations, failed_operations
            
            await lock_manager.acquire(user_id)
            try:
                # 检查余额
                balance = await account_mgr.get_balance(user_id)
                if balance >= bet_amount:
                    # 扣除余额
                    await user_repo.update_balance(user_id, -bet_amount)
                    successful_operations += 1
                else:
                    failed_operations += 1
            finally:
                await lock_manager.release(user_id)
        
        # 并发执行多个下注操作
        num_concurrent_ops = min(max_operations + 2, 10)  # 多于可能成功的次数
        tasks = [asyncio.create_task(bet_operation()) for _ in range(num_concurrent_ops)]
        await asyncio.gather(*tasks)
        
        # 验证最终余额
        final_balance = await account_mgr.get_balance(user_id)
        expected_balance = initial_balance - (successful_operations * bet_amount)
        
        assert final_balance == expected_balance, \
            f"Expected balance {expected_balance}, got {final_balance}"
        
        # 验证成功操作次数不超过最大可能次数
        assert successful_operations <= max_operations, \
            f"Too many successful operations: {successful_operations} > {max_operations}"
        
        # 验证余额不为负
        assert final_balance >= 0, f"Balance went negative: {final_balance}"
        
        await db.close()
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


# 额外测试：ConcurrencyManager 集成测试
@settings(max_examples=5)
@given(
    user_id=st.integers(min_value=1, max_value=999999999),
    game_type=st.sampled_from(['dice', 'slot', 'blackjack'])
)
@pytest.mark.asyncio
async def test_concurrency_manager_integration(user_id, game_type):
    """
    测试 ConcurrencyManager 的集成功能
    验证用户锁和游戏会话管理的协同工作
    """
    manager = ConcurrencyManager()
    
    # 获取用户锁
    await manager.acquire_user_lock(user_id)
    
    # 开始游戏
    success, message = await manager.start_game(user_id, game_type)
    assert success is True
    assert manager.has_active_game(user_id)
    
    # 释放用户锁
    await manager.release_user_lock(user_id)
    
    # 游戏会话应该仍然存在
    assert manager.has_active_game(user_id)
    
    # 结束游戏
    await manager.end_game(user_id)
    assert not manager.has_active_game(user_id)
