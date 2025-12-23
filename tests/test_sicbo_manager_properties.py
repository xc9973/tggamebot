"""
骰宝游戏管理器属性测试
使用 Hypothesis 进行属性测试，验证游戏管理逻辑的通用正确性

Feature: sic-bo-game
"""
import pytest
from hypothesis import given, strategies as st, settings, HealthCheck
from src.sicbo_manager import SicBoManager
from src.account_manager import AccountManager
from src.repositories import UserRepository, TransactionRepository
from src.database import DatabaseManager
from src.models import GamePhase, BetType


# ============================================================================
# 属性 1: 游戏会话互斥性
# ============================================================================

# Feature: sic-bo-game, Property 1: 游戏会话互斥性
@pytest.mark.asyncio
@given(chat_id=st.integers(min_value=1, max_value=1000000))
@settings(max_examples=5, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_1_game_session_mutual_exclusivity(chat_id):
    """
    属性 1: 游戏会话互斥性
    
    *对于任何* 群组，在任意时刻最多只能有一个进行中的骰宝游戏会话；
    当已有游戏进行时，创建新游戏的请求应该被拒绝
    
    **验证需求: 1.1, 1.2**
    """
    db = DatabaseManager(":memory:")
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        # 第一次创建游戏应该成功
        success1, msg1 = await manager.start_game(chat_id)
        assert success1 is True, f"第一次创建游戏应该成功，实际消息: {msg1}"
        
        # 验证游戏存在且处于下注阶段
        game = manager.get_game(chat_id)
        assert game is not None, "游戏应该存在"
        assert game.phase == GamePhase.BETTING, "游戏应该处于下注阶段"
        
        # 第二次创建游戏应该失败（互斥性）
        success2, msg2 = await manager.start_game(chat_id)
        assert success2 is False, f"第二次创建游戏应该失败（互斥性），实际消息: {msg2}"
        assert "已有进行中的游戏" in msg2, f"应该返回互斥性错误消息，实际: {msg2}"
        
        # 验证仍然只有一个游戏
        game_after = manager.get_game(chat_id)
        assert game_after is not None, "原游戏应该仍然存在"
        assert game_after.phase == GamePhase.BETTING, "原游戏状态不应该改变"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 1: 游戏会话互斥性 - 不同群组独立
@pytest.mark.asyncio
@given(
    chat_id1=st.integers(min_value=1, max_value=500000),
    chat_id2=st.integers(min_value=500001, max_value=1000000)
)
@settings(max_examples=5, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_1_different_chats_independent(chat_id1, chat_id2):
    """
    属性 1: 游戏会话互斥性 - 不同群组独立
    
    *对于任何* 两个不同的群组，它们的游戏会话应该相互独立，
    一个群组有游戏不影响另一个群组创建游戏
    
    **验证需求: 1.1, 1.2**
    """
    db = DatabaseManager(":memory:")
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        # 在群组1创建游戏
        success1, msg1 = await manager.start_game(chat_id1)
        assert success1 is True, f"群组1创建游戏应该成功"
        
        # 在群组2创建游戏应该也成功（独立性）
        success2, msg2 = await manager.start_game(chat_id2)
        assert success2 is True, f"群组2创建游戏应该成功（独立于群组1）"
        
        # 验证两个游戏都存在
        game1 = manager.get_game(chat_id1)
        game2 = manager.get_game(chat_id2)
        assert game1 is not None, "群组1的游戏应该存在"
        assert game2 is not None, "群组2的游戏应该存在"
        assert game1.chat_id == chat_id1, "群组1的游戏应该属于群组1"
        assert game2.chat_id == chat_id2, "群组2的游戏应该属于群组2"
    finally:
        await db.close()



# ============================================================================
# 属性 3: 下注输入验证
# ============================================================================

# Feature: sic-bo-game, Property 3: 下注输入验证 - 单一数字范围
@given(
    invalid_number=st.integers().filter(lambda x: x < 1 or x > 6)
)
@settings(max_examples=5, deadline=None)
def test_property_3_single_number_out_of_range(invalid_number):
    """
    属性 3: 下注输入验证 - 单一数字超出范围
    
    *对于任何* 下注请求，如果单一数字参数超出 1-6 范围，应该被拒绝
    
    **验证需求: 2.2**
    """
    from src.sicbo_manager import SicBoManager
    from src.account_manager import AccountManager
    from src.repositories import UserRepository, TransactionRepository
    from src.database import DatabaseManager
    
    manager = SicBoManager.__new__(SicBoManager)
    manager.active_games = {}
    manager.calculator = None
    
    valid, error_msg = manager.validate_bet_input(BetType.SINGLE, [invalid_number])
    assert valid is False, f"数字 {invalid_number} 超出范围应该被拒绝"
    assert "1-6" in error_msg, f"应该提示有效范围，实际: {error_msg}"


# Feature: sic-bo-game, Property 3: 下注输入验证 - 组合数字相同
@given(
    same_number=st.integers(min_value=1, max_value=6)
)
@settings(max_examples=5, deadline=None)
def test_property_3_pair_same_numbers(same_number):
    """
    属性 3: 下注输入验证 - 组合数字相同
    
    *对于任何* 组合押注，如果两个数字相同，应该被拒绝
    
    **验证需求: 3.2**
    """
    from src.sicbo_manager import SicBoManager
    
    manager = SicBoManager.__new__(SicBoManager)
    manager.active_games = {}
    manager.calculator = None
    
    valid, error_msg = manager.validate_bet_input(BetType.PAIR, [same_number, same_number])
    assert valid is False, f"相同数字 [{same_number}, {same_number}] 应该被拒绝"
    assert "不同" in error_msg, f"应该提示数字必须不同，实际: {error_msg}"


# Feature: sic-bo-game, Property 3: 下注输入验证 - 组合数字超出范围
@given(
    num1=st.integers().filter(lambda x: x < 1 or x > 6),
    num2=st.integers(min_value=1, max_value=6)
)
@settings(max_examples=5, deadline=None)
def test_property_3_pair_number_out_of_range(num1, num2):
    """
    属性 3: 下注输入验证 - 组合数字超出范围
    
    *对于任何* 组合押注，如果任一数字超出 1-6 范围，应该被拒绝
    
    **验证需求: 3.3**
    """
    from src.sicbo_manager import SicBoManager
    
    manager = SicBoManager.__new__(SicBoManager)
    manager.active_games = {}
    manager.calculator = None
    
    valid, error_msg = manager.validate_bet_input(BetType.PAIR, [num1, num2])
    assert valid is False, f"数字 [{num1}, {num2}] 超出范围应该被拒绝"
    assert "1-6" in error_msg, f"应该提示有效范围，实际: {error_msg}"


# Feature: sic-bo-game, Property 3: 下注输入验证 - 总和超出范围
@given(
    invalid_sum=st.integers().filter(lambda x: x < 4 or x > 17)
)
@settings(max_examples=5, deadline=None)
def test_property_3_sum_out_of_range(invalid_sum):
    """
    属性 3: 下注输入验证 - 总和超出范围
    
    *对于任何* 总和押注，如果总和参数超出 4-17 范围，应该被拒绝
    
    **验证需求: 4.2**
    """
    from src.sicbo_manager import SicBoManager
    
    manager = SicBoManager.__new__(SicBoManager)
    manager.active_games = {}
    manager.calculator = None
    
    valid, error_msg = manager.validate_bet_input(BetType.SUM, [invalid_sum])
    assert valid is False, f"总和 {invalid_sum} 超出范围应该被拒绝"
    assert "4-17" in error_msg, f"应该提示有效范围，实际: {error_msg}"


# Feature: sic-bo-game, Property 3: 下注输入验证 - 有效输入应该通过
@given(
    single_num=st.integers(min_value=1, max_value=6),
    pair_num1=st.integers(min_value=1, max_value=5),
    valid_sum=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=5, deadline=None)
def test_property_3_valid_inputs_accepted(single_num, pair_num1, valid_sum):
    """
    属性 3: 下注输入验证 - 有效输入应该通过
    
    *对于任何* 有效的下注输入，应该通过验证
    
    **验证需求: 2.2, 3.2, 3.3, 4.2**
    """
    from src.sicbo_manager import SicBoManager
    
    manager = SicBoManager.__new__(SicBoManager)
    manager.active_games = {}
    manager.calculator = None
    
    # 单一数字有效
    valid, _ = manager.validate_bet_input(BetType.SINGLE, [single_num])
    assert valid is True, f"有效单一数字 {single_num} 应该通过"
    
    # 组合有效（确保两个数字不同）
    pair_num2 = pair_num1 + 1  # 保证不同
    valid, _ = manager.validate_bet_input(BetType.PAIR, [pair_num1, pair_num2])
    assert valid is True, f"有效组合 [{pair_num1}, {pair_num2}] 应该通过"
    
    # 总和有效
    valid, _ = manager.validate_bet_input(BetType.SUM, [valid_sum])
    assert valid is True, f"有效总和 {valid_sum} 应该通过"
    
    # 大小有效（不需要数字参数）
    valid, _ = manager.validate_bet_input(BetType.BIG, [])
    assert valid is True, "押大应该通过"
    
    valid, _ = manager.validate_bet_input(BetType.SMALL, [])
    assert valid is True, "押小应该通过"



# ============================================================================
# 属性 4: 下注前置条件验证
# ============================================================================

# Feature: sic-bo-game, Property 4: 下注前置条件验证 - 余额不足
@pytest.mark.asyncio
async def test_property_4_insufficient_balance(tmp_path):
    """
    属性 4: 下注前置条件验证 - 余额不足
    
    *对于任何* 下注请求，如果余额不足，应该被拒绝
    
    **验证需求: 7.1**
    """
    db_path = str(tmp_path / "test_balance.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 123
        user_id = 456
        
        # 创建用户（初始余额 1000）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        
        # 开始游戏
        await manager.start_game(chat_id)
        
        # 测试多个超过余额的金额
        test_amounts = [1001, 2000, 5000, 10000]
        for bet_amount in test_amounts:
            success, msg = await manager.place_bet(chat_id, user_id, BetType.BIG, bet_amount)
            assert success is False, f"余额不足时应该拒绝下注 (bet={bet_amount}, balance=1000)"
            assert "余额不足" in msg, f"应该返回余额不足消息，实际: {msg}"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 4: 下注前置条件验证 - 金额非正
@pytest.mark.asyncio
async def test_property_4_non_positive_amount(tmp_path):
    """
    属性 4: 下注前置条件验证 - 金额非正
    
    *对于任何* 下注请求，如果金额小于或等于 0，应该被拒绝
    
    **验证需求: 7.2**
    """
    db_path = str(tmp_path / "test_amount.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 123
        user_id = 456
        
        # 创建用户
        await account_mgr.ensure_user_exists(user_id, "test_user")
        
        # 开始游戏
        await manager.start_game(chat_id)
        
        # 测试多个非正金额
        test_amounts = [0, -1, -100, -1000]
        for invalid_amount in test_amounts:
            success, msg = await manager.place_bet(chat_id, user_id, BetType.BIG, invalid_amount)
            assert success is False, f"非正金额 {invalid_amount} 应该被拒绝"
            assert "大于 0" in msg, f"应该返回金额错误消息，实际: {msg}"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 4: 下注前置条件验证 - 非下注阶段
@pytest.mark.asyncio
async def test_property_4_not_in_betting_phase(tmp_path):
    """
    属性 4: 下注前置条件验证 - 非下注阶段
    
    *对于任何* 下注请求，如果不在下注阶段，应该被拒绝
    
    **验证需求: 7.3**
    """
    db_path = str(tmp_path / "test_phase.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        user_id = 456
        
        # 创建用户
        await account_mgr.ensure_user_exists(user_id, "test_user")
        
        # 测试多个不同的 chat_id，不开始游戏直接尝试下注
        test_chat_ids = [1, 100, 999999]
        for chat_id in test_chat_ids:
            success, msg = await manager.place_bet(chat_id, user_id, BetType.BIG, 100)
            assert success is False, f"没有游戏时应该拒绝下注 (chat_id={chat_id})"
            assert "没有进行中的" in msg or "不在下注阶段" in msg, f"应该返回状态错误消息，实际: {msg}"
    finally:
        await db.close()


# ============================================================================
# 属性 5: 下注余额扣除原子性
# ============================================================================

# Feature: sic-bo-game, Property 5: 下注余额扣除原子性
@pytest.mark.asyncio
async def test_property_5_bet_balance_deduction_atomicity(tmp_path):
    """
    属性 5: 下注余额扣除原子性
    
    *对于任何* 成功的下注，玩家账户余额应该立即减少下注金额，
    且下注记录应该被正确保存
    
    **验证需求: 2.1, 3.1, 4.1, 5.1, 5.2, 7.4**
    """
    db_path = str(tmp_path / "test_balance_deduction.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 测试多种下注类型和金额
        test_cases = [
            (100, BetType.SINGLE, [3]),
            (200, BetType.PAIR, [2, 5]),
            (150, BetType.SUM, [10]),
            (50, BetType.BIG, []),
            (75, BetType.SMALL, [])
        ]
        
        for bet_amount, bet_type, numbers in test_cases:
            # 创建新用户（初始余额 1000）
            test_user_id = user_id + bet_amount
            await account_mgr.ensure_user_exists(test_user_id, f"test_user_{bet_amount}")
            
            # 获取下注前余额
            balance_before = await account_mgr.get_balance(test_user_id)
            assert balance_before == 1000, f"初始余额应该是 1000，实际: {balance_before}"
            
            # 开始游戏（每次用不同的 chat_id）
            test_chat_id = chat_id + bet_amount
            success, _ = await manager.start_game(test_chat_id)
            assert success is True, "游戏应该成功开始"
            
            # 下注
            success, msg = await manager.place_bet(test_chat_id, test_user_id, bet_type, bet_amount, numbers)
            assert success is True, f"下注应该成功，实际消息: {msg}"
            
            # 验证余额立即减少
            balance_after = await account_mgr.get_balance(test_user_id)
            expected_balance = balance_before - bet_amount
            assert balance_after == expected_balance, \
                f"余额应该从 {balance_before} 减少到 {expected_balance}，实际: {balance_after}"
            
            # 验证下注记录被正确保存
            user_bets = manager.get_user_bets(test_chat_id, test_user_id)
            assert len(user_bets) == 1, f"应该有 1 条下注记录，实际: {len(user_bets)}"
            
            bet_record = user_bets[0]
            assert bet_record.user_id == test_user_id, "下注记录的用户 ID 应该正确"
            assert bet_record.bet_type == bet_type, "下注记录的类型应该正确"
            assert bet_record.amount == bet_amount, f"下注记录的金额应该是 {bet_amount}，实际: {bet_record.amount}"
            assert bet_record.numbers == numbers, f"下注记录的数字应该是 {numbers}，实际: {bet_record.numbers}"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 5: 下注余额扣除原子性 - 多次下注累计扣除
@pytest.mark.asyncio
async def test_property_5_multiple_bets_cumulative_deduction(tmp_path):
    """
    属性 5: 下注余额扣除原子性 - 多次下注累计扣除
    
    *对于任何* 多次成功的下注，玩家账户余额应该累计减少所有下注金额，
    且下注记录应该被正确保存（同一选项的下注会累加到一条记录）
    
    **验证需求: 2.1, 3.1, 4.1, 5.1, 5.2, 7.4**
    """
    db_path = str(tmp_path / "test_multiple_bets.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 测试多次下注
        bet_amounts = [50, 100, 75, 25, 150]
        
        # 创建用户（初始余额 1000）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        initial_balance = await account_mgr.get_balance(user_id)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 过滤掉会导致余额不足的下注
        total_bet = 0
        valid_bets = []
        for amount in bet_amounts:
            if total_bet + amount <= initial_balance:
                valid_bets.append(amount)
                total_bet += amount
        
        # 执行多次下注（同一选项 BIG）
        for i, amount in enumerate(valid_bets):
            success, msg = await manager.place_bet(chat_id, user_id, BetType.BIG, amount)
            assert success is True, f"第 {i+1} 次下注应该成功，实际消息: {msg}"
        
        # 验证余额累计减少
        final_balance = await account_mgr.get_balance(user_id)
        expected_balance = initial_balance - sum(valid_bets)
        assert final_balance == expected_balance, \
            f"余额应该从 {initial_balance} 减少到 {expected_balance}，实际: {final_balance}"
        
        # 验证下注记录（同一选项的下注会累加到一条记录）
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 1, \
            f"同一选项的多次下注应该累加为 1 条记录，实际: {len(user_bets)}"
        
        # 验证累计下注金额
        total_amount = user_bets[0].amount
        expected_total = sum(valid_bets)
        assert total_amount == expected_total, \
            f"累计下注金额应该是 {expected_total}，实际: {total_amount}"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 5: 下注余额扣除原子性 - 失败下注不扣款
@pytest.mark.asyncio
async def test_property_5_failed_bet_no_deduction(tmp_path):
    """
    属性 5: 下注余额扣除原子性 - 失败下注不扣款
    
    *对于任何* 失败的下注（如余额不足），玩家账户余额应该保持不变，
    且不应该有下注记录被保存
    
    **验证需求: 7.1, 7.4**
    """
    db_path = str(tmp_path / "test_failed_bet.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 测试多个超过余额的金额
        test_amounts = [1001, 2000, 5000, 10000]
        
        # 创建用户（初始余额 1000）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        balance_before = await account_mgr.get_balance(user_id)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        for bet_amount in test_amounts:
            # 尝试下注超过余额的金额（应该失败）
            success, msg = await manager.place_bet(chat_id, user_id, BetType.BIG, bet_amount)
            assert success is False, f"余额不足时下注应该失败，实际消息: {msg}"
            
            # 验证余额没有变化
            balance_after = await account_mgr.get_balance(user_id)
            assert balance_after == balance_before, \
                f"失败下注后余额应该保持 {balance_before}，实际: {balance_after}"
        
        # 验证没有下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 0, f"失败下注不应该有记录，实际: {len(user_bets)}"
    finally:
        await db.close()


# ============================================================================
# 属性 2: 游戏状态转换正确性
# ============================================================================

# Feature: sic-bo-game, Property 2: 游戏状态转换正确性 - 正常流程
@pytest.mark.asyncio
@given(chat_id=st.integers(min_value=1, max_value=1000000))
@settings(max_examples=3, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_2_game_state_transition_normal_flow(chat_id):
    """
    属性 2: 游戏状态转换正确性 - 正常流程
    
    *对于任何* 骰宝游戏会话，状态转换必须遵循 IDLE → BETTING → ROLLING → SETTLING → IDLE 的顺序
    
    **验证需求: 1.5, 6.5**
    """
    db = DatabaseManager(":memory:")
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        # 初始状态：没有游戏（相当于 IDLE）
        game = manager.get_game(chat_id)
        assert game is None, "初始状态应该没有游戏"
        
        # 开始游戏：IDLE -> BETTING
        success, _ = await manager.start_game(chat_id)
        assert success is True, "开始游戏应该成功"
        
        game = manager.get_game(chat_id)
        assert game is not None, "游戏应该存在"
        assert game.phase == GamePhase.BETTING, "开始游戏后应该处于 BETTING 阶段"
        
        # 开骰子：BETTING -> ROLLING
        success, dice_results, _ = await manager.roll_dice(chat_id)
        assert success is True, "开骰子应该成功"
        assert len(dice_results) == 3, "应该有 3 个骰子结果"
        
        game = manager.get_game(chat_id)
        assert game.phase == GamePhase.ROLLING, "开骰子后应该处于 ROLLING 阶段"
        
        # 结算：ROLLING -> SETTLING -> IDLE（游戏结束）
        success, _, _ = await manager.settle_game(chat_id)
        assert success is True, "结算应该成功"
        
        # 游戏结束后应该被移除
        game = manager.get_game(chat_id)
        assert game is None, "结算后游戏应该被移除"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 2: 游戏状态转换正确性 - 不允许跳过状态
@pytest.mark.asyncio
@given(chat_id=st.integers(min_value=1, max_value=1000000))
@settings(max_examples=3, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_2_cannot_skip_states(chat_id):
    """
    属性 2: 游戏状态转换正确性 - 不允许跳过状态
    
    *对于任何* 骰宝游戏会话，不允许跳过状态（如直接从 BETTING 到 SETTLING）
    
    **验证需求: 1.5, 6.5**
    """
    db = DatabaseManager(":memory:")
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "开始游戏应该成功"
        
        game = manager.get_game(chat_id)
        assert game.phase == GamePhase.BETTING, "应该处于 BETTING 阶段"
        
        # 尝试直接结算（跳过 ROLLING）应该失败
        success, _, msg = await manager.settle_game(chat_id)
        assert success is False, "在 BETTING 阶段直接结算应该失败"
        assert "尚未开骰子" in msg, f"应该返回状态错误消息，实际: {msg}"
        
        # 验证状态没有改变
        game = manager.get_game(chat_id)
        assert game.phase == GamePhase.BETTING, "失败的结算不应该改变状态"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 2: 游戏状态转换正确性 - 不允许逆向转换
@pytest.mark.asyncio
@given(chat_id=st.integers(min_value=1, max_value=1000000))
@settings(max_examples=3, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_2_cannot_reverse_states(chat_id):
    """
    属性 2: 游戏状态转换正确性 - 不允许逆向转换
    
    *对于任何* 骰宝游戏会话，不允许逆向状态转换（如从 ROLLING 回到 BETTING）
    
    **验证需求: 1.5, 6.5**
    """
    db = DatabaseManager(":memory:")
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        # 开始游戏并进入 ROLLING 阶段
        success, _ = await manager.start_game(chat_id)
        assert success is True, "开始游戏应该成功"
        
        success, _, _ = await manager.roll_dice(chat_id)
        assert success is True, "开骰子应该成功"
        
        game = manager.get_game(chat_id)
        assert game.phase == GamePhase.ROLLING, "应该处于 ROLLING 阶段"
        
        # 尝试再次开骰子（逆向回到 BETTING 再开骰子）应该失败
        success, _, msg = await manager.roll_dice(chat_id)
        assert success is False, "在 ROLLING 阶段再次开骰子应该失败"
        assert "不在下注阶段" in msg, f"应该返回状态错误消息，实际: {msg}"
        
        # 验证状态没有改变
        game = manager.get_game(chat_id)
        assert game.phase == GamePhase.ROLLING, "失败的操作不应该改变状态"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 2: 游戏状态转换正确性 - 结算后可以开始新游戏
@pytest.mark.asyncio
@given(chat_id=st.integers(min_value=1, max_value=1000000))
@settings(max_examples=3, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_2_can_start_new_game_after_settlement(chat_id):
    """
    属性 2: 游戏状态转换正确性 - 结算后可以开始新游戏
    
    *对于任何* 骰宝游戏会话，结算完成后应该允许开始新游戏
    
    **验证需求: 6.5**
    """
    db = DatabaseManager(":memory:")
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        # 完成第一局游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "第一局开始游戏应该成功"
        
        success, _, _ = await manager.roll_dice(chat_id)
        assert success is True, "第一局开骰子应该成功"
        
        success, _, _ = await manager.settle_game(chat_id)
        assert success is True, "第一局结算应该成功"
        
        # 验证游戏已结束
        game = manager.get_game(chat_id)
        assert game is None, "第一局结算后游戏应该被移除"
        
        # 开始第二局游戏应该成功
        success, _ = await manager.start_game(chat_id)
        assert success is True, "第二局开始游戏应该成功"
        
        game = manager.get_game(chat_id)
        assert game is not None, "第二局游戏应该存在"
        assert game.phase == GamePhase.BETTING, "第二局应该处于 BETTING 阶段"
    finally:
        await db.close()


# ============================================================================
# 属性 10: 多押注结算正确性
# ============================================================================

# Feature: sic-bo-game, Property 10: 多押注结算正确性 - 单玩家多押注
@pytest.mark.asyncio
async def test_property_10_single_player_multiple_bets_settlement(tmp_path):
    """
    属性 10: 多押注结算正确性 - 单玩家多押注
    
    *对于任何* 游戏结算，每个玩家的每个押注应该独立计算赔付，
    玩家的总收益等于所有押注赔付之和减去所有押注金额
    
    **验证需求: 6.3, 6.6**
    """
    from src.sicbo_calculator import SicBoCalculator
    
    db_path = str(tmp_path / "test_multi_bet.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    calculator = SicBoCalculator()
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 测试多组骰子结果
        test_dice_results = [
            [1, 2, 3],  # 总和6，小
            [4, 5, 6],  # 总和15，大
            [3, 3, 3],  # 围骰
        ]
        
        for dice in test_dice_results:
            # 创建新用户
            test_user_id = user_id + sum(dice)
            await account_mgr.ensure_user_exists(test_user_id, f"test_user_{sum(dice)}")
            await account_mgr.user_repo.update_balance(test_user_id, 9000)
            
            initial_balance = await account_mgr.get_balance(test_user_id)
            
            # 开始游戏
            test_chat_id = chat_id + sum(dice)
            success, _ = await manager.start_game(test_chat_id)
            assert success is True, "开始游戏应该成功"
            
            # 下多个不同类型的押注
            bets_to_place = [
                (BetType.SINGLE, 100, [3]),
                (BetType.BIG, 200, []),
                (BetType.SMALL, 150, []),
            ]
            
            total_bet_amount = 0
            for bet_type, amount, numbers in bets_to_place:
                success, _ = await manager.place_bet(test_chat_id, test_user_id, bet_type, amount, numbers)
                assert success is True, f"下注 {bet_type} 应该成功"
                total_bet_amount += amount
            
            # 开骰子
            success, _, _ = await manager.roll_dice(test_chat_id, dice)
            assert success is True, "开骰子应该成功"
            
            # 手动计算预期赔付
            expected_payout = 0
            for bet_type, amount, numbers in bets_to_place:
                if bet_type == BetType.SINGLE:
                    expected_payout += calculator.calculate_single_payout(numbers[0], dice, amount)
                elif bet_type == BetType.BIG:
                    expected_payout += calculator.calculate_big_small_payout(True, dice, amount)
                elif bet_type == BetType.SMALL:
                    expected_payout += calculator.calculate_big_small_payout(False, dice, amount)
            
            # 结算
            success, net_results, _ = await manager.settle_game(test_chat_id)
            assert success is True, "结算应该成功"
            
            # 验证净收益计算正确
            expected_net = expected_payout - total_bet_amount
            actual_net = net_results.get(test_user_id, 0)
            assert actual_net == expected_net, f"骰子 {dice}: 净收益应该是 {expected_net}，实际: {actual_net}"
            
            # 验证最终余额
            final_balance = await account_mgr.get_balance(test_user_id)
            expected_final = initial_balance - total_bet_amount + expected_payout
            assert final_balance == expected_final, f"骰子 {dice}: 最终余额应该是 {expected_final}，实际: {final_balance}"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 10: 多押注结算正确性 - 多玩家结算
@pytest.mark.asyncio
async def test_property_10_multiple_players_settlement(tmp_path):
    """
    属性 10: 多押注结算正确性 - 多玩家结算
    
    *对于任何* 游戏结算，每个玩家的押注应该独立计算，
    不同玩家之间的结算互不影响
    
    **验证需求: 6.3, 6.6**
    """
    from src.sicbo_calculator import SicBoCalculator
    
    db_path = str(tmp_path / "test_multi_player.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    calculator = SicBoCalculator()
    
    try:
        chat_id = 12345
        
        # 测试多组骰子结果
        test_dice_results = [
            [1, 2, 3],  # 总和6，小
            [4, 5, 6],  # 总和15，大
        ]
        
        for idx, dice in enumerate(test_dice_results):
            test_chat_id = chat_id + idx
            
            # 创建多个用户
            players = [
                (1001 + idx * 100, "player1", BetType.BIG, 100, []),
                (1002 + idx * 100, "player2", BetType.SMALL, 200, []),
                (1003 + idx * 100, "player3", BetType.SINGLE, 150, [3]),
            ]
            
            initial_balances = {}
            for user_id, username, _, _, _ in players:
                await account_mgr.ensure_user_exists(user_id, username)
                initial_balances[user_id] = await account_mgr.get_balance(user_id)
            
            # 开始游戏
            success, _ = await manager.start_game(test_chat_id)
            assert success is True, "开始游戏应该成功"
            
            # 每个玩家下注
            for user_id, _, bet_type, amount, numbers in players:
                success, _ = await manager.place_bet(test_chat_id, user_id, bet_type, amount, numbers)
                assert success is True, f"玩家 {user_id} 下注应该成功"
            
            # 开骰子
            success, _, _ = await manager.roll_dice(test_chat_id, dice)
            assert success is True, "开骰子应该成功"
            
            # 计算每个玩家的预期赔付
            expected_payouts = {}
            for user_id, _, bet_type, amount, numbers in players:
                if bet_type == BetType.BIG:
                    expected_payouts[user_id] = calculator.calculate_big_small_payout(True, dice, amount)
                elif bet_type == BetType.SMALL:
                    expected_payouts[user_id] = calculator.calculate_big_small_payout(False, dice, amount)
                elif bet_type == BetType.SINGLE:
                    expected_payouts[user_id] = calculator.calculate_single_payout(numbers[0], dice, amount)
            
            # 结算
            success, net_results, _ = await manager.settle_game(test_chat_id)
            assert success is True, "结算应该成功"
            
            # 验证每个玩家的结算结果
            for user_id, _, bet_type, amount, numbers in players:
                expected_net = expected_payouts[user_id] - amount
                actual_net = net_results.get(user_id, 0)
                assert actual_net == expected_net, \
                    f"骰子 {dice}, 玩家 {user_id} 净收益应该是 {expected_net}，实际: {actual_net}"
                
                # 验证最终余额
                final_balance = await account_mgr.get_balance(user_id)
                expected_final = initial_balances[user_id] - amount + expected_payouts[user_id]
                assert final_balance == expected_final, \
                    f"骰子 {dice}, 玩家 {user_id} 最终余额应该是 {expected_final}，实际: {final_balance}"
    finally:
        await db.close()


# Feature: sic-bo-game, Property 10: 多押注结算正确性 - 无押注结算
@pytest.mark.asyncio
async def test_property_10_no_bets_settlement(tmp_path):
    """
    属性 10: 多押注结算正确性 - 无押注结算
    
    *对于任何* 没有押注的游戏，结算应该成功且净收益为空
    
    **验证需求: 6.3**
    """
    db_path = str(tmp_path / "test_no_bets.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        # 测试多组骰子结果
        test_dice_results = [
            [1, 2, 3],
            [4, 5, 6],
            [3, 3, 3],
        ]
        
        for idx, dice in enumerate(test_dice_results):
            chat_id = 12345 + idx
            
            # 开始游戏
            success, _ = await manager.start_game(chat_id)
            assert success is True, "开始游戏应该成功"
            
            # 不下注，直接开骰子
            success, _, _ = await manager.roll_dice(chat_id, dice)
            assert success is True, "开骰子应该成功"
            
            # 结算
            success, net_results, msg = await manager.settle_game(chat_id)
            assert success is True, "结算应该成功"
            assert len(net_results) == 0, f"骰子 {dice}: 没有押注时净收益应该为空"
            assert "无人下注" in msg, f"骰子 {dice}: 消息应该提示无人下注，实际: {msg}"
    finally:
        await db.close()
