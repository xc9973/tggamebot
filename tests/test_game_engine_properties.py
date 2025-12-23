"""
游戏引擎属性测试
使用 Hypothesis 进行属性测试，验证游戏逻辑的通用正确性

Feature: telegram-game-bot
"""
import pytest
import asyncio
from hypothesis import given, strategies as st, settings, HealthCheck
from src.game_engine import GameEngine
from src.account_manager import AccountManager
from src.repositories import UserRepository, TransactionRepository
from src.database import DatabaseManager


def create_game_engine_sync():
    """创建同步游戏引擎用于非异步测试"""
    async def setup():
        db = DatabaseManager(":memory:")
        await db.initialize()
        user_repo = UserRepository(db)
        tx_repo = TransactionRepository(db)
        account_mgr = AccountManager(user_repo, tx_repo)
        return GameEngine(account_mgr, tx_repo)
    
    return asyncio.run(setup())


async def create_game_engine_async():
    """创建异步游戏引擎"""
    db = DatabaseManager(":memory:")
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    return GameEngine(account_mgr, tx_repo), account_mgr


# ============================================================================
# 骰子游戏属性测试
# ============================================================================

# Feature: telegram-game-bot, Property 13: 骰子游戏赔率正确性
@given(
    dice_value=st.integers(min_value=1, max_value=6),
    bet=st.integers(min_value=1, max_value=100000)
)
@settings(max_examples=5)
def test_property_13_dice_payout_correctness(dice_value, bet):
    """
    属性 13: 骰子游戏赔率正确性
    
    *对于任何* 骰子游戏结果，余额变化应该符合赔率表：
    - 1-3 点输掉本金
    - 4-5 点赢得 1 倍本金
    - 6 点赢得 2 倍本金
    
    **验证需求: 5.2, 5.3, 5.4**
    """
    game_engine = create_game_engine_sync()
    payout = game_engine.calculate_dice_payout(dice_value, bet)
    
    if dice_value in [1, 2, 3]:
        # 1-3 点输掉本金
        assert payout == -bet, f"点数 {dice_value} 应该输掉本金 (-{bet})，实际为 {payout}"
    elif dice_value in [4, 5]:
        # 4-5 点赢得 1 倍本金
        assert payout == bet, f"点数 {dice_value} 应该赢得 1 倍本金 ({bet})，实际为 {payout}"
    elif dice_value == 6:
        # 6 点赢得 2 倍本金
        assert payout == bet * 2, f"点数 {dice_value} 应该赢得 2 倍本金 ({bet * 2})，实际为 {payout}"


# Feature: telegram-game-bot, Property 14: 骰子游戏前置条件验证
@given(
    bet=st.integers(min_value=-1000, max_value=100000),
    dice_value=st.integers(min_value=1, max_value=6)
)
@settings(max_examples=5)
def test_property_14_dice_precondition_validation(bet, dice_value):
    """
    属性 14: 骰子游戏前置条件验证
    
    *对于任何* 骰子游戏请求，如果余额不足或金额非正，应该拒绝并返回错误消息
    
    **验证需求: 5.5, 5.6**
    """
    async def run_test():
        game_engine, account_mgr = await create_game_engine_async()
        user_id = 12345
        await account_mgr.ensure_user_exists(user_id, "test_user")
        initial_balance = await account_mgr.get_balance(user_id)
        
        success, message, payout = await game_engine.play_dice(user_id, bet, dice_value)
        
        if bet <= 0:
            # 无效金额应该被拒绝
            assert success is False, f"无效金额 {bet} 应该被拒绝"
            assert "必须大于 0" in message, f"应该返回金额错误消息，实际为: {message}"
            assert payout == 0, "无效操作应该返回 0 奖金"
            # 余额不应该变化
            new_balance = await account_mgr.get_balance(user_id)
            assert new_balance == initial_balance, "无效操作不应该改变余额"
        elif bet > initial_balance:
            # 余额不足应该被拒绝
            assert success is False, f"余额不足时应该被拒绝 (bet={bet}, balance={initial_balance})"
            assert "余额不足" in message, f"应该返回余额不足消息，实际为: {message}"
            assert payout == 0, "无效操作应该返回 0 奖金"
            # 余额不应该变化
            new_balance = await account_mgr.get_balance(user_id)
            assert new_balance == initial_balance, "无效操作不应该改变余额"
        else:
            # 有效请求应该成功
            assert success is True, f"有效请求应该成功 (bet={bet}, balance={initial_balance})"
    
    asyncio.run(run_test())


# ============================================================================
# 老虎机游戏属性测试
# ============================================================================

# Feature: telegram-game-bot, Property 16: 老虎机赔率正确性
@given(
    slot_value=st.integers(min_value=1, max_value=64),
    bet=st.integers(min_value=1, max_value=100000)
)
@settings(max_examples=5)
def test_property_16_slot_payout_correctness(slot_value, bet):
    """
    属性 16: 老虎机赔率正确性
    
    *对于任何* 老虎机结果，余额变化应该符合赔率表：
    - 三个图案完全一致赢得 10 倍本金
    - 两个图案一致赢得 2 倍本金
    - 三个图案不一致输掉本金
    
    **验证需求: 6.2, 6.3, 6.4**
    """
    game_engine = create_game_engine_sync()
    payout = game_engine.calculate_slot_payout(slot_value, bet)
    
    # 三个图案完全一致（大奖）
    if slot_value in [1, 22, 43, 64]:
        assert payout == bet * 10, f"值 {slot_value} 应该赢得 10 倍本金 ({bet * 10})，实际为 {payout}"
    # 两个图案一致（小奖）- 偶数但不是大奖
    elif slot_value % 2 == 0:
        assert payout == bet * 2, f"值 {slot_value} 应该赢得 2 倍本金 ({bet * 2})，实际为 {payout}"
    # 三个图案不一致（输）- 奇数但不是大奖
    else:
        assert payout == -bet, f"值 {slot_value} 应该输掉本金 (-{bet})，实际为 {payout}"


# Feature: telegram-game-bot, Property 17: 老虎机前置条件验证
@given(
    bet=st.integers(min_value=-1000, max_value=100000),
    slot_value=st.integers(min_value=1, max_value=64)
)
@settings(max_examples=5)
def test_property_17_slot_precondition_validation(bet, slot_value):
    """
    属性 17: 老虎机前置条件验证
    
    *对于任何* 老虎机游戏请求，如果余额不足或金额非正，应该拒绝并返回错误消息
    
    **验证需求: 6.5, 6.6**
    """
    async def run_test():
        game_engine, account_mgr = await create_game_engine_async()
        user_id = 12345
        await account_mgr.ensure_user_exists(user_id, "test_user")
        initial_balance = await account_mgr.get_balance(user_id)
        
        success, message, payout = await game_engine.play_slot(user_id, bet, slot_value)
        
        if bet <= 0:
            # 无效金额应该被拒绝
            assert success is False, f"无效金额 {bet} 应该被拒绝"
            assert "必须大于 0" in message, f"应该返回金额错误消息，实际为: {message}"
            assert payout == 0, "无效操作应该返回 0 奖金"
            # 余额不应该变化
            new_balance = await account_mgr.get_balance(user_id)
            assert new_balance == initial_balance, "无效操作不应该改变余额"
        elif bet > initial_balance:
            # 余额不足应该被拒绝
            assert success is False, f"余额不足时应该被拒绝 (bet={bet}, balance={initial_balance})"
            assert "余额不足" in message, f"应该返回余额不足消息，实际为: {message}"
            assert payout == 0, "无效操作应该返回 0 奖金"
            # 余额不应该变化
            new_balance = await account_mgr.get_balance(user_id)
            assert new_balance == initial_balance, "无效操作不应该改变余额"
        else:
            # 有效请求应该成功
            assert success is True, f"有效请求应该成功 (bet={bet}, balance={initial_balance})"
    
    asyncio.run(run_test())


# ============================================================================
# 游戏结果反馈属性测试
# ============================================================================

# Feature: telegram-game-bot, Property 15: 骰子游戏结果反馈
@given(
    dice_value=st.integers(min_value=1, max_value=6),
    bet=st.integers(min_value=1, max_value=500)  # 限制在初始余额范围内
)
@settings(max_examples=5, suppress_health_check=[HealthCheck.function_scoped_fixture])
def test_property_15_dice_result_feedback(dice_value, bet):
    """
    属性 15: 骰子游戏结果反馈
    
    *对于任何* 骰子游戏结束，应该显示骰子点数、输赢结果和当前余额
    
    **验证需求: 5.7**
    """
    async def run_test():
        game_engine, account_mgr = await create_game_engine_async()
        user_id = 12345
        await account_mgr.ensure_user_exists(user_id, "test_user")
        
        success, message, payout = await game_engine.play_dice(user_id, bet, dice_value)
        
        # 游戏应该成功（余额充足）
        assert success is True, f"游戏应该成功，实际为: {message}"
        
        # 消息应该包含骰子点数
        assert str(dice_value) in message, f"消息应该包含骰子点数 {dice_value}，实际消息: {message}"
        
        # 消息应该包含输赢结果
        if payout > 0:
            assert "赢" in message or "获胜" in message, f"赢的消息应该包含'赢'或'获胜'，实际消息: {message}"
        else:
            assert "输" in message or "遗憾" in message, f"输的消息应该包含'输'或'遗憾'，实际消息: {message}"
        
        # 消息应该包含当前余额
        assert "余额" in message, f"消息应该包含'余额'，实际消息: {message}"
        
        # 验证余额数值在消息中
        new_balance = await account_mgr.get_balance(user_id)
        assert str(new_balance) in message, f"消息应该包含当前余额 {new_balance}，实际消息: {message}"
    
    asyncio.run(run_test())


# Feature: telegram-game-bot, Property 18: 老虎机结果反馈
@given(
    slot_value=st.integers(min_value=1, max_value=64),
    bet=st.integers(min_value=1, max_value=500)  # 限制在初始余额范围内
)
@settings(max_examples=5, suppress_health_check=[HealthCheck.function_scoped_fixture])
def test_property_18_slot_result_feedback(slot_value, bet):
    """
    属性 18: 老虎机结果反馈
    
    *对于任何* 老虎机游戏结束，应该显示老虎机结果、输赢情况和当前余额
    
    **验证需求: 6.7**
    """
    async def run_test():
        game_engine, account_mgr = await create_game_engine_async()
        user_id = 12345
        await account_mgr.ensure_user_exists(user_id, "test_user")
        
        success, message, payout = await game_engine.play_slot(user_id, bet, slot_value)
        
        # 游戏应该成功（余额充足）
        assert success is True, f"游戏应该成功，实际为: {message}"
        
        # 消息应该包含老虎机值
        assert str(slot_value) in message, f"消息应该包含老虎机值 {slot_value}，实际消息: {message}"
        
        # 消息应该包含输赢结果
        if payout > 0:
            assert "赢" in message or "获胜" in message or "大奖" in message, \
                f"赢的消息应该包含'赢'、'获胜'或'大奖'，实际消息: {message}"
        else:
            assert "输" in message or "遗憾" in message, \
                f"输的消息应该包含'输'或'遗憾'，实际消息: {message}"
        
        # 消息应该包含当前余额
        assert "余额" in message, f"消息应该包含'余额'，实际消息: {message}"
        
        # 验证余额数值在消息中
        new_balance = await account_mgr.get_balance(user_id)
        assert str(new_balance) in message, f"消息应该包含当前余额 {new_balance}，实际消息: {message}"
    
    asyncio.run(run_test())
