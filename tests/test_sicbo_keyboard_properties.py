"""
骰宝键盘构建器属性测试
使用 Hypothesis 进行属性测试，验证回调数据编码/解码的正确性

Feature: sicbo-button-ui
"""
import pytest
import os
import tempfile
from hypothesis import given, strategies as st, settings, HealthCheck
from src.sicbo_keyboard import SicBoKeyboardBuilder, SicBoAction
from src.sicbo_manager import SicBoManager
from src.account_manager import AccountManager
from src.repositories import UserRepository, TransactionRepository
from src.database import DatabaseManager
from src.models import GamePhase, BetType


# ============================================================================
# 生成策略
# ============================================================================

# 有效的动作类型
valid_actions = st.sampled_from(["single", "big", "small", "sum", "roll", "mybets"])

# 有效的参数（数字字符串或空字符串）
valid_params = st.one_of(
    st.just(""),  # 空参数
    st.integers(min_value=1, max_value=17).map(str),  # 数字参数
)

# 单一数字参数 (1-6)
single_number_params = st.integers(min_value=1, max_value=6).map(str)

# 总和参数 (4-17)
sum_params = st.integers(min_value=4, max_value=17).map(str)


# ============================================================================
# 属性 5: 回调数据解析往返
# ============================================================================

# Feature: sicbo-button-ui, Property 5: 回调数据解析往返
@given(action=valid_actions, param=valid_params)
@settings(max_examples=100)
def test_property_5_callback_encode_decode_roundtrip(action, param):
    """
    属性 5: 回调数据解析往返
    
    *对于任何* 有效的 SicBoAction 和参数组合，编码为回调数据字符串后再解码，
    应该产生原始的 action 和 param。
    
    **Validates: Requirements 2.1, 3.1, 4.1, 5.1**
    """
    # 编码
    encoded = SicBoKeyboardBuilder.encode_callback(action, param)
    
    # 解码
    decoded_action, decoded_param = SicBoKeyboardBuilder.decode_callback(encoded)
    
    # 验证往返一致性
    assert decoded_action == action, \
        f"动作不匹配: 原始 '{action}', 解码后 '{decoded_action}', 编码数据 '{encoded}'"
    assert decoded_param == param, \
        f"参数不匹配: 原始 '{param}', 解码后 '{decoded_param}', 编码数据 '{encoded}'"


# Feature: sicbo-button-ui, Property 5: 回调数据解析往返 - 单一数字
@given(num=single_number_params)
@settings(max_examples=100)
def test_property_5_single_number_callback_roundtrip(num):
    """
    属性 5: 回调数据解析往返 - 单一数字下注
    
    *对于任何* 单一数字 (1-6)，编码为 single 动作的回调数据后再解码，
    应该产生原始的数字参数。
    
    **Validates: Requirements 3.1**
    """
    encoded = SicBoKeyboardBuilder.encode_callback("single", num)
    decoded_action, decoded_param = SicBoKeyboardBuilder.decode_callback(encoded)
    
    assert decoded_action == "single", \
        f"动作应为 'single', 实际 '{decoded_action}'"
    assert decoded_param == num, \
        f"参数应为 '{num}', 实际 '{decoded_param}'"


# Feature: sicbo-button-ui, Property 5: 回调数据解析往返 - 总和
@given(sum_val=sum_params)
@settings(max_examples=100)
def test_property_5_sum_callback_roundtrip(sum_val):
    """
    属性 5: 回调数据解析往返 - 总和下注
    
    *对于任何* 总和值 (4-17)，编码为 sum 动作的回调数据后再解码，
    应该产生原始的总和参数。
    
    **Validates: Requirements 5.1**
    """
    encoded = SicBoKeyboardBuilder.encode_callback("sum", sum_val)
    decoded_action, decoded_param = SicBoKeyboardBuilder.decode_callback(encoded)
    
    assert decoded_action == "sum", \
        f"动作应为 'sum', 实际 '{decoded_action}'"
    assert decoded_param == sum_val, \
        f"参数应为 '{sum_val}', 实际 '{decoded_param}'"


# Feature: sicbo-button-ui, Property 5: 回调数据解析往返 - 大小
@given(action=st.sampled_from(["big", "small"]))
@settings(max_examples=100)
def test_property_5_big_small_callback_roundtrip(action):
    """
    属性 5: 回调数据解析往返 - 大小下注
    
    *对于任何* 大小动作 (big/small)，编码为回调数据后再解码，
    应该产生原始的动作，参数为空。
    
    **Validates: Requirements 4.1**
    """
    encoded = SicBoKeyboardBuilder.encode_callback(action)
    decoded_action, decoded_param = SicBoKeyboardBuilder.decode_callback(encoded)
    
    assert decoded_action == action, \
        f"动作应为 '{action}', 实际 '{decoded_action}'"
    assert decoded_param == "", \
        f"参数应为空, 实际 '{decoded_param}'"


# Feature: sicbo-button-ui, Property 5: 回调数据解析往返 - 无效前缀
def test_property_5_invalid_prefix_returns_empty():
    """
    属性 5: 回调数据解析往返 - 无效前缀处理
    
    当回调数据不以 'sicbo_' 前缀开头时，解码应返回空字符串。
    
    **Validates: Requirements 2.1**
    """
    invalid_data = "invalid_callback_data"
    decoded_action, decoded_param = SicBoKeyboardBuilder.decode_callback(invalid_data)
    
    assert decoded_action == "", \
        f"无效前缀应返回空动作, 实际 '{decoded_action}'"
    assert decoded_param == "", \
        f"无效前缀应返回空参数, 实际 '{decoded_param}'"


# Feature: sicbo-button-ui, Property 5: 回调数据格式验证
def test_property_5_callback_format_has_prefix():
    """
    属性 5: 回调数据格式验证 - 编码数据包含正确前缀
    
    所有编码的回调数据应该以 'sicbo_' 前缀开头。
    
    **Validates: Requirements 2.1**
    """
    test_cases = [
        ("single", "3"),
        ("big", ""),
        ("small", ""),
        ("sum", "10"),
        ("roll", ""),
        ("mybets", ""),
    ]
    
    for action, param in test_cases:
        encoded = SicBoKeyboardBuilder.encode_callback(action, param)
        assert encoded.startswith(SicBoKeyboardBuilder.CALLBACK_PREFIX), \
            f"编码数据 '{encoded}' 应以 '{SicBoKeyboardBuilder.CALLBACK_PREFIX}' 开头"


# ============================================================================
# 属性 1: 固定下注金额一致性
# ============================================================================

# Feature: sicbo-button-ui, Property 1: 固定下注金额一致性
@pytest.mark.asyncio
@given(
    bet_type=st.sampled_from(["single", "big", "small", "sum"]),
    single_num=st.integers(min_value=1, max_value=6),
    sum_val=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_1_fixed_bet_amount_consistency(bet_type, single_num, sum_val, tmp_path):
    """
    属性 1: 固定下注金额一致性
    
    *对于任何* 有效的下注按钮点击（单一数字、大、小或总和），
    当用户余额充足时，记录的下注金额应该恰好等于 100 金币。
    
    **Validates: Requirements 2.1, 3.2, 4.2, 4.3, 5.3**
    """
    # 使用文件数据库而非内存数据库，避免连接池问题
    import uuid
    db_path = str(tmp_path / f"test_prop1_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户（初始余额 1000，足够下注）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 根据下注类型执行下注
        if bet_type == "single":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [single_num]
            )
        elif bet_type == "big":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.BIG,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "small":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SMALL,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "sum":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SUM,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [sum_val]
            )
        
        assert success is True, f"下注应该成功，实际消息: {msg}"
        
        # 验证下注金额恰好等于固定金额
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 1, f"应该有 1 条下注记录，实际: {len(user_bets)}"
        
        bet_record = user_bets[0]
        assert bet_record.amount == SicBoKeyboardBuilder.FIXED_BET_AMOUNT, \
            f"下注金额应该是 {SicBoKeyboardBuilder.FIXED_BET_AMOUNT}，实际: {bet_record.amount}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 1: 固定下注金额一致性 - 余额扣除验证
@pytest.mark.asyncio
@given(
    bet_type=st.sampled_from(["single", "big", "small", "sum"]),
    single_num=st.integers(min_value=1, max_value=6),
    sum_val=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_1_fixed_bet_amount_balance_deduction(bet_type, single_num, sum_val, tmp_path):
    """
    属性 1: 固定下注金额一致性 - 余额扣除验证
    
    *对于任何* 有效的下注按钮点击，用户余额应该恰好减少 100 金币。
    
    **Validates: Requirements 2.1, 3.2, 4.2, 4.3, 5.3**
    """
    # 使用文件数据库而非内存数据库，避免连接池问题
    import uuid
    db_path = str(tmp_path / f"test_prop1_bal_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户（初始余额 1000）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        balance_before = await account_mgr.get_balance(user_id)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 根据下注类型执行下注
        if bet_type == "single":
            success, _ = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [single_num]
            )
        elif bet_type == "big":
            success, _ = await manager.place_bet(
                chat_id, user_id, BetType.BIG,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "small":
            success, _ = await manager.place_bet(
                chat_id, user_id, BetType.SMALL,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "sum":
            success, _ = await manager.place_bet(
                chat_id, user_id, BetType.SUM,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [sum_val]
            )
        
        assert success is True, "下注应该成功"
        
        # 验证余额恰好减少固定金额
        balance_after = await account_mgr.get_balance(user_id)
        expected_balance = balance_before - SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        assert balance_after == expected_balance, \
            f"余额应该从 {balance_before} 减少到 {expected_balance}，实际: {balance_after}"
    finally:
        await db.close()


# ============================================================================
# 属性 2: 余额不足时不扣款
# ============================================================================

# Feature: sicbo-button-ui, Property 2: 余额不足时不扣款
@pytest.mark.asyncio
@given(
    initial_balance=st.integers(min_value=0, max_value=99),
    bet_type=st.sampled_from(["single", "big", "small", "sum"]),
    single_num=st.integers(min_value=1, max_value=6),
    sum_val=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_2_insufficient_balance_no_deduction(initial_balance, bet_type, single_num, sum_val, tmp_path):
    """
    属性 2: 余额不足时不扣款
    
    *对于任何* 余额少于 100 金币的用户，点击任何下注按钮都不应该改变其余额，
    也不应该记录下注。
    
    **Validates: Requirements 2.3, 8.4**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop2_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户并设置余额为不足的金额
        await account_mgr.ensure_user_exists(user_id, "test_user")
        # 扣除余额使其低于 100
        current_balance = await account_mgr.get_balance(user_id)
        deduction = current_balance - initial_balance
        if deduction > 0:
            await account_mgr.user_repo.update_balance(user_id, -deduction)
        
        balance_before = await account_mgr.get_balance(user_id)
        assert balance_before == initial_balance, f"余额应该是 {initial_balance}，实际: {balance_before}"
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 根据下注类型尝试下注
        if bet_type == "single":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [single_num]
            )
        elif bet_type == "big":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.BIG,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "small":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SMALL,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "sum":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SUM,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [sum_val]
            )
        
        # 验证下注失败
        assert success is False, f"余额不足时下注应该失败，实际消息: {msg}"
        assert "余额不足" in msg, f"应该返回余额不足消息，实际: {msg}"
        
        # 验证余额没有变化
        balance_after = await account_mgr.get_balance(user_id)
        assert balance_after == balance_before, \
            f"余额不足时余额应该保持 {balance_before}，实际: {balance_after}"
        
        # 验证没有下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 0, f"余额不足时不应该有下注记录，实际: {len(user_bets)}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 2: 余额不足时不扣款 - 边界情况
@pytest.mark.asyncio
async def test_property_2_insufficient_balance_boundary(tmp_path):
    """
    属性 2: 余额不足时不扣款 - 边界情况
    
    当用户余额恰好等于 99（比固定金额少 1）时，下注应该失败且余额不变。
    
    **Validates: Requirements 2.3, 8.4**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop2_boundary_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        boundary_balance = SicBoKeyboardBuilder.FIXED_BET_AMOUNT - 1  # 99
        
        # 创建用户并设置余额为边界值
        await account_mgr.ensure_user_exists(user_id, "test_user")
        current_balance = await account_mgr.get_balance(user_id)
        deduction = current_balance - boundary_balance
        await account_mgr.user_repo.update_balance(user_id, -deduction)
        
        balance_before = await account_mgr.get_balance(user_id)
        assert balance_before == boundary_balance, f"余额应该是 {boundary_balance}"
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 尝试下注
        success, msg = await manager.place_bet(
            chat_id, user_id, BetType.BIG,
            SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
            []
        )
        
        # 验证下注失败
        assert success is False, "余额恰好不足时下注应该失败"
        assert "余额不足" in msg, f"应该返回余额不足消息，实际: {msg}"
        
        # 验证余额没有变化
        balance_after = await account_mgr.get_balance(user_id)
        assert balance_after == boundary_balance, \
            f"余额应该保持 {boundary_balance}，实际: {balance_after}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 2: 余额不足时不扣款 - 恰好足够
@pytest.mark.asyncio
async def test_property_2_exact_balance_succeeds(tmp_path):
    """
    属性 2: 余额不足时不扣款 - 恰好足够
    
    当用户余额恰好等于 100（固定金额）时，下注应该成功。
    
    **Validates: Requirements 2.1, 2.3**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop2_exact_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        exact_balance = SicBoKeyboardBuilder.FIXED_BET_AMOUNT  # 100
        
        # 创建用户并设置余额为恰好足够
        await account_mgr.ensure_user_exists(user_id, "test_user")
        current_balance = await account_mgr.get_balance(user_id)
        deduction = current_balance - exact_balance
        await account_mgr.user_repo.update_balance(user_id, -deduction)
        
        balance_before = await account_mgr.get_balance(user_id)
        assert balance_before == exact_balance, f"余额应该是 {exact_balance}"
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 尝试下注
        success, msg = await manager.place_bet(
            chat_id, user_id, BetType.BIG,
            SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
            []
        )
        
        # 验证下注成功
        assert success is True, f"余额恰好足够时下注应该成功，实际消息: {msg}"
        
        # 验证余额变为 0
        balance_after = await account_mgr.get_balance(user_id)
        assert balance_after == 0, f"余额应该变为 0，实际: {balance_after}"
        
        # 验证有下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 1, f"应该有 1 条下注记录，实际: {len(user_bets)}"
    finally:
        await db.close()


# ============================================================================
# 属性 3: 累加下注正确性
# ============================================================================

# Feature: sicbo-button-ui, Property 3: 累加下注正确性
@pytest.mark.asyncio
@given(
    click_count=st.integers(min_value=1, max_value=10),
    bet_type=st.sampled_from(["single", "big", "small", "sum"]),
    single_num=st.integers(min_value=1, max_value=6),
    sum_val=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_3_cumulative_bet_correctness(click_count, bet_type, single_num, sum_val, tmp_path):
    """
    属性 3: 累加下注正确性
    
    *对于任何* 用户点击同一下注按钮 N 次（N ≥ 1），
    其在该选项上的总下注金额应该等于 N × 100 金币。
    
    **Validates: Requirements 2.4**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop3_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户并设置足够的余额
        await account_mgr.ensure_user_exists(user_id, "test_user")
        # 确保余额足够多次下注
        needed_balance = click_count * SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        current_balance = await account_mgr.get_balance(user_id)
        if current_balance < needed_balance:
            await account_mgr.user_repo.update_balance(user_id, needed_balance - current_balance)
        
        balance_before = await account_mgr.get_balance(user_id)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 确定下注参数
        if bet_type == "single":
            bet_type_enum = BetType.SINGLE
            numbers = [single_num]
        elif bet_type == "big":
            bet_type_enum = BetType.BIG
            numbers = []
        elif bet_type == "small":
            bet_type_enum = BetType.SMALL
            numbers = []
        elif bet_type == "sum":
            bet_type_enum = BetType.SUM
            numbers = [sum_val]
        
        # 多次点击同一下注按钮
        for i in range(click_count):
            success, msg = await manager.place_bet(
                chat_id, user_id, bet_type_enum,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                numbers
            )
            assert success is True, f"第 {i+1} 次下注应该成功，实际消息: {msg}"
        
        # 验证总下注金额
        user_bets = manager.get_user_bets(chat_id, user_id)
        
        # 由于累加下注，应该只有一条记录
        assert len(user_bets) == 1, \
            f"累加下注后应该只有 1 条记录，实际: {len(user_bets)}"
        
        # 验证累计金额
        expected_total = click_count * SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        actual_total = user_bets[0].amount
        assert actual_total == expected_total, \
            f"累计下注金额应该是 {expected_total}，实际: {actual_total}"
        
        # 验证余额扣除正确
        balance_after = await account_mgr.get_balance(user_id)
        expected_balance = balance_before - expected_total
        assert balance_after == expected_balance, \
            f"余额应该从 {balance_before} 减少到 {expected_balance}，实际: {balance_after}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 3: 累加下注正确性 - 不同选项独立
@pytest.mark.asyncio
@given(
    click_count_big=st.integers(min_value=1, max_value=5),
    click_count_small=st.integers(min_value=1, max_value=5)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_3_different_options_independent(click_count_big, click_count_small, tmp_path):
    """
    属性 3: 累加下注正确性 - 不同选项独立
    
    *对于任何* 用户在不同选项上的下注，每个选项的累计金额应该独立计算。
    
    **Validates: Requirements 2.4**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop3_indep_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户并设置足够的余额
        await account_mgr.ensure_user_exists(user_id, "test_user")
        total_clicks = click_count_big + click_count_small
        needed_balance = total_clicks * SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        current_balance = await account_mgr.get_balance(user_id)
        if current_balance < needed_balance:
            await account_mgr.user_repo.update_balance(user_id, needed_balance - current_balance)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 多次点击"大"按钮
        for i in range(click_count_big):
            success, _ = await manager.place_bet(
                chat_id, user_id, BetType.BIG,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
            assert success is True, f"第 {i+1} 次押大应该成功"
        
        # 多次点击"小"按钮
        for i in range(click_count_small):
            success, _ = await manager.place_bet(
                chat_id, user_id, BetType.SMALL,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
            assert success is True, f"第 {i+1} 次押小应该成功"
        
        # 验证下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        
        # 应该有两条记录（大和小各一条）
        assert len(user_bets) == 2, \
            f"不同选项应该有 2 条记录，实际: {len(user_bets)}"
        
        # 验证每个选项的累计金额
        big_bet = next((b for b in user_bets if b.bet_type == BetType.BIG), None)
        small_bet = next((b for b in user_bets if b.bet_type == BetType.SMALL), None)
        
        assert big_bet is not None, "应该有押大的记录"
        assert small_bet is not None, "应该有押小的记录"
        
        expected_big = click_count_big * SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        expected_small = click_count_small * SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        
        assert big_bet.amount == expected_big, \
            f"押大累计金额应该是 {expected_big}，实际: {big_bet.amount}"
        assert small_bet.amount == expected_small, \
            f"押小累计金额应该是 {expected_small}，实际: {small_bet.amount}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 3: 累加下注正确性 - 单一数字不同数字独立
@pytest.mark.asyncio
@given(
    num1=st.integers(min_value=1, max_value=3),
    num2=st.integers(min_value=4, max_value=6),
    click_count1=st.integers(min_value=1, max_value=3),
    click_count2=st.integers(min_value=1, max_value=3)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_3_single_numbers_independent(num1, num2, click_count1, click_count2, tmp_path):
    """
    属性 3: 累加下注正确性 - 单一数字不同数字独立
    
    *对于任何* 用户在不同单一数字上的下注，每个数字的累计金额应该独立计算。
    
    **Validates: Requirements 2.4**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop3_single_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户并设置足够的余额
        await account_mgr.ensure_user_exists(user_id, "test_user")
        total_clicks = click_count1 + click_count2
        needed_balance = total_clicks * SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        current_balance = await account_mgr.get_balance(user_id)
        if current_balance < needed_balance:
            await account_mgr.user_repo.update_balance(user_id, needed_balance - current_balance)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 多次点击数字1按钮
        for i in range(click_count1):
            success, _ = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [num1]
            )
            assert success is True, f"第 {i+1} 次押数字 {num1} 应该成功"
        
        # 多次点击数字2按钮
        for i in range(click_count2):
            success, _ = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [num2]
            )
            assert success is True, f"第 {i+1} 次押数字 {num2} 应该成功"
        
        # 验证下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        
        # 应该有两条记录（两个不同数字各一条）
        assert len(user_bets) == 2, \
            f"不同数字应该有 2 条记录，实际: {len(user_bets)}"
        
        # 验证每个数字的累计金额
        bet1 = next((b for b in user_bets if b.numbers == [num1]), None)
        bet2 = next((b for b in user_bets if b.numbers == [num2]), None)
        
        assert bet1 is not None, f"应该有押数字 {num1} 的记录"
        assert bet2 is not None, f"应该有押数字 {num2} 的记录"
        
        expected1 = click_count1 * SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        expected2 = click_count2 * SicBoKeyboardBuilder.FIXED_BET_AMOUNT
        
        assert bet1.amount == expected1, \
            f"押数字 {num1} 累计金额应该是 {expected1}，实际: {bet1.amount}"
        assert bet2.amount == expected2, \
            f"押数字 {num2} 累计金额应该是 {expected2}，实际: {bet2.amount}"
    finally:
        await db.close()


# ============================================================================
# 属性 4: 游戏阶段验证
# ============================================================================

# Feature: sicbo-button-ui, Property 4: 游戏阶段验证
@pytest.mark.asyncio
@given(
    bet_type=st.sampled_from(["single", "big", "small", "sum"]),
    single_num=st.integers(min_value=1, max_value=6),
    sum_val=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_4_game_phase_validation_rolling(bet_type, single_num, sum_val, tmp_path):
    """
    属性 4: 游戏阶段验证 - ROLLING 阶段
    
    *对于任何* 按钮点击，当游戏不在 BETTING 阶段（如 ROLLING 阶段）时，
    点击应该被拒绝，不应该下注，用户余额应该保持不变。
    
    **Validates: Requirements 7.2, 8.1**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop4_rolling_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户（初始余额 1000，足够下注）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        balance_before = await account_mgr.get_balance(user_id)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 手动将游戏阶段改为 ROLLING
        game = manager.get_game(chat_id)
        game.phase = GamePhase.ROLLING
        
        # 尝试下注
        if bet_type == "single":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [single_num]
            )
        elif bet_type == "big":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.BIG,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "small":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SMALL,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "sum":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SUM,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [sum_val]
            )
        
        # 验证下注被拒绝
        assert success is False, f"ROLLING 阶段下注应该被拒绝，实际消息: {msg}"
        assert "不在下注阶段" in msg, f"应该返回不在下注阶段消息，实际: {msg}"
        
        # 验证余额没有变化
        balance_after = await account_mgr.get_balance(user_id)
        assert balance_after == balance_before, \
            f"ROLLING 阶段余额应该保持 {balance_before}，实际: {balance_after}"
        
        # 验证没有下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 0, f"ROLLING 阶段不应该有下注记录，实际: {len(user_bets)}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 4: 游戏阶段验证 - SETTLING 阶段
@pytest.mark.asyncio
@given(
    bet_type=st.sampled_from(["single", "big", "small", "sum"]),
    single_num=st.integers(min_value=1, max_value=6),
    sum_val=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_4_game_phase_validation_settling(bet_type, single_num, sum_val, tmp_path):
    """
    属性 4: 游戏阶段验证 - SETTLING 阶段
    
    *对于任何* 按钮点击，当游戏在 SETTLING 阶段时，
    点击应该被拒绝，不应该下注，用户余额应该保持不变。
    
    **Validates: Requirements 7.2, 8.1**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop4_settling_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户（初始余额 1000，足够下注）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        balance_before = await account_mgr.get_balance(user_id)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 手动将游戏阶段改为 SETTLING
        game = manager.get_game(chat_id)
        game.phase = GamePhase.SETTLING
        
        # 尝试下注
        if bet_type == "single":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [single_num]
            )
        elif bet_type == "big":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.BIG,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "small":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SMALL,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "sum":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SUM,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [sum_val]
            )
        
        # 验证下注被拒绝
        assert success is False, f"SETTLING 阶段下注应该被拒绝，实际消息: {msg}"
        assert "不在下注阶段" in msg, f"应该返回不在下注阶段消息，实际: {msg}"
        
        # 验证余额没有变化
        balance_after = await account_mgr.get_balance(user_id)
        assert balance_after == balance_before, \
            f"SETTLING 阶段余额应该保持 {balance_before}，实际: {balance_after}"
        
        # 验证没有下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 0, f"SETTLING 阶段不应该有下注记录，实际: {len(user_bets)}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 4: 游戏阶段验证 - IDLE 阶段
@pytest.mark.asyncio
@given(
    bet_type=st.sampled_from(["single", "big", "small", "sum"]),
    single_num=st.integers(min_value=1, max_value=6),
    sum_val=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_4_game_phase_validation_idle(bet_type, single_num, sum_val, tmp_path):
    """
    属性 4: 游戏阶段验证 - IDLE 阶段
    
    *对于任何* 按钮点击，当游戏在 IDLE 阶段时，
    点击应该被拒绝，不应该下注，用户余额应该保持不变。
    
    **Validates: Requirements 7.2, 8.1**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop4_idle_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户（初始余额 1000，足够下注）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        balance_before = await account_mgr.get_balance(user_id)
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 手动将游戏阶段改为 IDLE
        game = manager.get_game(chat_id)
        game.phase = GamePhase.IDLE
        
        # 尝试下注
        if bet_type == "single":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [single_num]
            )
        elif bet_type == "big":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.BIG,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "small":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SMALL,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "sum":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SUM,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [sum_val]
            )
        
        # 验证下注被拒绝
        assert success is False, f"IDLE 阶段下注应该被拒绝，实际消息: {msg}"
        assert "不在下注阶段" in msg, f"应该返回不在下注阶段消息，实际: {msg}"
        
        # 验证余额没有变化
        balance_after = await account_mgr.get_balance(user_id)
        assert balance_after == balance_before, \
            f"IDLE 阶段余额应该保持 {balance_before}，实际: {balance_after}"
        
        # 验证没有下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 0, f"IDLE 阶段不应该有下注记录，实际: {len(user_bets)}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 4: 游戏阶段验证 - 开骰子按钮阶段验证
@pytest.mark.asyncio
@given(
    phase=st.sampled_from([GamePhase.ROLLING, GamePhase.SETTLING, GamePhase.IDLE])
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_4_roll_button_phase_validation(phase, tmp_path):
    """
    属性 4: 游戏阶段验证 - 开骰子按钮阶段验证
    
    *对于任何* 非 BETTING 阶段，点击开骰子按钮应该被拒绝。
    
    **Validates: Requirements 7.2, 8.1**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop4_roll_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 手动将游戏阶段改为非 BETTING 阶段
        game = manager.get_game(chat_id)
        game.phase = phase
        
        # 尝试开骰子
        success, dice_results, msg = await manager.roll_dice(chat_id)
        
        # 验证开骰子被拒绝
        assert success is False, f"{phase.value} 阶段开骰子应该被拒绝，实际消息: {msg}"
        
        # 验证没有骰子结果
        assert dice_results == [], f"{phase.value} 阶段不应该有骰子结果，实际: {dice_results}"
    finally:
        await db.close()


# Feature: sicbo-button-ui, Property 4: 游戏阶段验证 - BETTING 阶段允许下注
@pytest.mark.asyncio
@given(
    bet_type=st.sampled_from(["single", "big", "small", "sum"]),
    single_num=st.integers(min_value=1, max_value=6),
    sum_val=st.integers(min_value=4, max_value=17)
)
@settings(max_examples=100, suppress_health_check=[HealthCheck.function_scoped_fixture], deadline=None)
async def test_property_4_betting_phase_allows_bet(bet_type, single_num, sum_val, tmp_path):
    """
    属性 4: 游戏阶段验证 - BETTING 阶段允许下注
    
    *对于任何* 按钮点击，当游戏在 BETTING 阶段时，
    下注应该被允许（假设余额充足）。
    
    **Validates: Requirements 7.2, 8.1**
    """
    import uuid
    db_path = str(tmp_path / f"test_prop4_betting_{uuid.uuid4().hex[:8]}.db")
    db = DatabaseManager(db_path)
    await db.initialize()
    user_repo = UserRepository(db)
    tx_repo = TransactionRepository(db)
    account_mgr = AccountManager(user_repo, tx_repo)
    manager = SicBoManager(account_mgr, tx_repo)
    
    try:
        chat_id = 12345
        user_id = 67890
        
        # 创建用户（初始余额 1000，足够下注）
        await account_mgr.ensure_user_exists(user_id, "test_user")
        
        # 开始游戏
        success, _ = await manager.start_game(chat_id)
        assert success is True, "游戏应该成功开始"
        
        # 验证游戏在 BETTING 阶段
        game = manager.get_game(chat_id)
        assert game.phase == GamePhase.BETTING, "游戏应该在 BETTING 阶段"
        
        # 尝试下注
        if bet_type == "single":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SINGLE,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [single_num]
            )
        elif bet_type == "big":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.BIG,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "small":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SMALL,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                []
            )
        elif bet_type == "sum":
            success, msg = await manager.place_bet(
                chat_id, user_id, BetType.SUM,
                SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
                [sum_val]
            )
        
        # 验证下注成功
        assert success is True, f"BETTING 阶段下注应该成功，实际消息: {msg}"
        
        # 验证有下注记录
        user_bets = manager.get_user_bets(chat_id, user_id)
        assert len(user_bets) == 1, f"BETTING 阶段应该有下注记录，实际: {len(user_bets)}"
    finally:
        await db.close()
