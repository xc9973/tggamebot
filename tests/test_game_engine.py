"""
游戏引擎单元测试
测试骰子和老虎机游戏的基本功能
"""
import pytest
from src.game_engine import GameEngine
from src.account_manager import AccountManager
from src.repositories import UserRepository, TransactionRepository
from src.database import DatabaseManager


@pytest.fixture
async def db():
    """创建测试数据库"""
    db = DatabaseManager(":memory:")
    await db.initialize()
    return db


@pytest.fixture
async def user_repo(db):
    """创建用户仓储"""
    return UserRepository(db)


@pytest.fixture
async def tx_repo(db):
    """创建交易仓储"""
    return TransactionRepository(db)


@pytest.fixture
async def account_mgr(user_repo, tx_repo):
    """创建账户管理器"""
    return AccountManager(user_repo, tx_repo)


@pytest.fixture
async def game_engine(account_mgr, tx_repo):
    """创建游戏引擎"""
    return GameEngine(account_mgr, tx_repo)


@pytest.mark.asyncio
async def test_calculate_dice_payout(game_engine):
    """测试骰子赔率计算"""
    # 1-3 点输掉本金
    assert game_engine.calculate_dice_payout(1, 100) == -100
    assert game_engine.calculate_dice_payout(2, 100) == -100
    assert game_engine.calculate_dice_payout(3, 100) == -100
    
    # 4-5 点赢得 1 倍本金
    assert game_engine.calculate_dice_payout(4, 100) == 100
    assert game_engine.calculate_dice_payout(5, 100) == 100
    
    # 6 点赢得 2 倍本金
    assert game_engine.calculate_dice_payout(6, 100) == 200


@pytest.mark.asyncio
async def test_calculate_slot_payout(game_engine):
    """测试老虎机赔率计算"""
    # 三个图案完全一致，赢得 10 倍本金
    assert game_engine.calculate_slot_payout(1, 100) == 1000
    assert game_engine.calculate_slot_payout(22, 100) == 1000
    assert game_engine.calculate_slot_payout(43, 100) == 1000
    assert game_engine.calculate_slot_payout(64, 100) == 1000
    
    # 两个图案一致，赢得 2 倍本金（偶数但不是大奖）
    assert game_engine.calculate_slot_payout(2, 100) == 200
    assert game_engine.calculate_slot_payout(20, 100) == 200
    
    # 三个图案不一致，输掉本金（奇数但不是大奖）
    assert game_engine.calculate_slot_payout(3, 100) == -100
    assert game_engine.calculate_slot_payout(21, 100) == -100


@pytest.mark.asyncio
async def test_play_dice_success(game_engine, account_mgr):
    """测试骰子游戏成功流程"""
    # 创建用户
    user = await account_mgr.ensure_user_exists(12345, "test_user")
    
    # 玩骰子游戏（点数 6，赢得 2 倍本金）
    success, message, payout = await game_engine.play_dice(12345, 100, 6)
    
    assert success is True
    assert payout == 200
    assert "获胜" in message
    assert "200" in message
    
    # 验证余额变化
    new_balance = await account_mgr.get_balance(12345)
    assert new_balance == 1000 + 200


@pytest.mark.asyncio
async def test_play_dice_lose(game_engine, account_mgr):
    """测试骰子游戏失败流程"""
    # 创建用户
    user = await account_mgr.ensure_user_exists(12346, "test_user2")
    
    # 玩骰子游戏（点数 1，输掉本金）
    success, message, payout = await game_engine.play_dice(12346, 100, 1)
    
    assert success is True
    assert payout == -100
    assert "遗憾" in message
    
    # 验证余额变化
    new_balance = await account_mgr.get_balance(12346)
    assert new_balance == 1000 - 100


@pytest.mark.asyncio
async def test_play_dice_insufficient_balance(game_engine, account_mgr):
    """测试骰子游戏余额不足"""
    # 创建用户
    user = await account_mgr.ensure_user_exists(12347, "test_user3")
    
    # 尝试下注超过余额的金额
    success, message, payout = await game_engine.play_dice(12347, 2000, 6)
    
    assert success is False
    assert "余额不足" in message
    assert payout == 0


@pytest.mark.asyncio
async def test_play_dice_invalid_amount(game_engine, account_mgr):
    """测试骰子游戏无效金额"""
    # 创建用户
    user = await account_mgr.ensure_user_exists(12348, "test_user4")
    
    # 尝试下注 0 或负数
    success, message, payout = await game_engine.play_dice(12348, 0, 6)
    assert success is False
    assert "必须大于 0" in message
    
    success, message, payout = await game_engine.play_dice(12348, -100, 6)
    assert success is False
    assert "必须大于 0" in message


@pytest.mark.asyncio
async def test_play_slot_success(game_engine, account_mgr):
    """测试老虎机游戏成功流程"""
    # 创建用户
    user = await account_mgr.ensure_user_exists(12349, "test_user5")
    
    # 玩老虎机游戏（值 1，大奖，赢得 10 倍本金）
    success, message, payout = await game_engine.play_slot(12349, 100, 1)
    
    assert success is True
    assert payout == 1000
    assert "大奖" in message
    
    # 验证余额变化
    new_balance = await account_mgr.get_balance(12349)
    assert new_balance == 1000 + 1000


@pytest.mark.asyncio
async def test_play_slot_lose(game_engine, account_mgr):
    """测试老虎机游戏失败流程"""
    # 创建用户
    user = await account_mgr.ensure_user_exists(12350, "test_user6")
    
    # 玩老虎机游戏（值 3，输掉本金）
    success, message, payout = await game_engine.play_slot(12350, 100, 3)
    
    assert success is True
    assert payout == -100
    assert "遗憾" in message
    
    # 验证余额变化
    new_balance = await account_mgr.get_balance(12350)
    assert new_balance == 1000 - 100


@pytest.mark.asyncio
async def test_play_slot_insufficient_balance(game_engine, account_mgr):
    """测试老虎机游戏余额不足"""
    # 创建用户
    user = await account_mgr.ensure_user_exists(12351, "test_user7")
    
    # 尝试下注超过余额的金额
    success, message, payout = await game_engine.play_slot(12351, 2000, 1)
    
    assert success is False
    assert "余额不足" in message
    assert payout == 0


@pytest.mark.asyncio
async def test_play_slot_invalid_amount(game_engine, account_mgr):
    """测试老虎机游戏无效金额"""
    # 创建用户
    user = await account_mgr.ensure_user_exists(12352, "test_user8")
    
    # 尝试下注 0 或负数
    success, message, payout = await game_engine.play_slot(12352, 0, 1)
    assert success is False
    assert "必须大于 0" in message
    
    success, message, payout = await game_engine.play_slot(12352, -100, 1)
    assert success is False
    assert "必须大于 0" in message
