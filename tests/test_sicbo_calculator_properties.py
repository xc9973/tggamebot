"""
骰宝赔率计算器属性测试
使用 Hypothesis 进行属性测试，验证赔率计算的通用正确性

Feature: sic-bo-game
"""
import pytest
from hypothesis import given, strategies as st, settings
from src.sicbo_calculator import SicBoCalculator
from src.models import SicBoBet, BetType


# ============================================================================
# 骰子生成策略
# ============================================================================

# 生成三个骰子结果 (1-6)
dice_strategy = st.lists(
    st.integers(min_value=1, max_value=6),
    min_size=3,
    max_size=3
)

# 生成围骰（三个相同的骰子）
triple_dice_strategy = st.integers(min_value=1, max_value=6).map(lambda x: [x, x, x])

# 生成非围骰的骰子
non_triple_dice_strategy = dice_strategy.filter(
    lambda d: not (d[0] == d[1] == d[2])
)

# 生成单一数字 (1-6)
single_number_strategy = st.integers(min_value=1, max_value=6)

# 生成两个不同的数字组合
pair_numbers_strategy = st.tuples(
    st.integers(min_value=1, max_value=6),
    st.integers(min_value=1, max_value=6)
).filter(lambda x: x[0] != x[1]).map(lambda x: list(x))

# 生成总和 (4-17)
sum_strategy = st.integers(min_value=4, max_value=17)

# 生成押注金额
bet_amount_strategy = st.integers(min_value=1, max_value=10000)


# ============================================================================
# 属性 12: 围骰判定正确性
# ============================================================================

# Feature: sic-bo-game, Property 12: 围骰判定正确性
@given(dice=triple_dice_strategy)
@settings(max_examples=100)
def test_property_12_is_triple_returns_true_for_triples(dice):
    """
    属性 12: 围骰判定正确性 - 三个相同的骰子应该返回 True
    
    *对于任何* 三个骰子结果，当且仅当三个骰子点数完全相同时，is_triple 应该返回 True
    
    **验证需求: 4.4, 5.5**
    """
    assert SicBoCalculator.is_triple(dice) is True, \
        f"三个相同的骰子 {dice} 应该被判定为围骰"


# Feature: sic-bo-game, Property 12: 围骰判定正确性
@given(dice=non_triple_dice_strategy)
@settings(max_examples=100)
def test_property_12_is_triple_returns_false_for_non_triples(dice):
    """
    属性 12: 围骰判定正确性 - 非围骰应该返回 False
    
    *对于任何* 三个骰子结果，当三个骰子点数不完全相同时，is_triple 应该返回 False
    
    **验证需求: 4.4, 5.5**
    """
    assert SicBoCalculator.is_triple(dice) is False, \
        f"非围骰 {dice} 不应该被判定为围骰"



# ============================================================================
# 属性 6: 单一数字赔率正确性
# ============================================================================

# Feature: sic-bo-game, Property 6: 单一数字赔率正确性
@given(
    bet_number=single_number_strategy,
    dice=dice_strategy,
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_6_single_number_payout_correctness(bet_number, dice, bet_amount):
    """
    属性 6: 单一数字赔率正确性
    
    *对于任何* 单一数字押注和骰子结果组合，赔付应该符合规则：
    - 0个匹配返回0
    - 1个匹配返回bet*2
    - 2个匹配返回bet*3
    - 3个匹配返回bet*4
    
    **验证需求: 2.3, 2.4, 2.5, 2.6**
    """
    payout = SicBoCalculator.calculate_single_payout(bet_number, dice, bet_amount)
    match_count = dice.count(bet_number)
    
    if match_count == 0:
        expected = 0
    else:
        expected = bet_amount * (match_count + 1)
    
    assert payout == expected, \
        f"押注数字 {bet_number}，骰子 {dice}，匹配 {match_count} 个，" \
        f"押注金额 {bet_amount}，期望赔付 {expected}，实际 {payout}"



# ============================================================================
# 属性 7: 两个数字组合赔率正确性
# ============================================================================

# Feature: sic-bo-game, Property 7: 两个数字组合赔率正确性
@given(
    numbers=pair_numbers_strategy,
    dice=dice_strategy,
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_7_pair_payout_correctness(numbers, dice, bet_amount):
    """
    属性 7: 两个数字组合赔率正确性
    
    *对于任何* 两个数字组合押注和骰子结果，如果骰子包含两个押注数字则返回bet*6，
    否则返回0；重复数字不多次计算
    
    **验证需求: 3.4, 3.5, 3.6**
    """
    payout = SicBoCalculator.calculate_pair_payout(numbers, dice, bet_amount)
    
    num1, num2 = numbers[0], numbers[1]
    # 检查骰子是否同时包含两个数字
    contains_both = num1 in dice and num2 in dice
    
    if contains_both:
        expected = bet_amount * 6
    else:
        expected = 0
    
    assert payout == expected, \
        f"押注组合 {numbers}，骰子 {dice}，" \
        f"押注金额 {bet_amount}，期望赔付 {expected}，实际 {payout}"


# Feature: sic-bo-game, Property 7: 两个数字组合赔率正确性 - 重复数字不多次计算
@given(bet_amount=bet_amount_strategy)
@settings(max_examples=100)
def test_property_7_pair_payout_no_double_counting(bet_amount):
    """
    属性 7: 两个数字组合赔率正确性 - 重复数字不多次计算
    
    *对于任何* 包含重复数字的骰子结果（如 [3, 3, 5]），
    押注 3-5 组合只计算一次赢钱，不因重复数字多次计算
    
    **验证需求: 3.6**
    """
    # 测试 [3, 3, 5] 押注 3-5 组合
    dice = [3, 3, 5]
    numbers = [3, 5]
    payout = SicBoCalculator.calculate_pair_payout(numbers, dice, bet_amount)
    
    # 应该只返回一次 bet * 6，不因为有两个 3 而多次计算
    expected = bet_amount * 6
    assert payout == expected, \
        f"重复数字骰子 {dice}，押注组合 {numbers}，" \
        f"应该只计算一次，期望 {expected}，实际 {payout}"



# ============================================================================
# 属性 8: 总和赔率正确性
# ============================================================================

# 总和赔率表（赔率为 N:1，返还 = bet * (N + 1)）
SUM_PAYOUT_TABLE = {
    4: 61, 17: 61,   # 60:1 赔率
    5: 31, 16: 31,   # 30:1 赔率
    6: 18, 15: 18,   # 17:1 赔率
    7: 13, 14: 13,   # 12:1 赔率
    8: 9, 13: 9,     # 8:1 赔率
    9: 7, 12: 7,     # 6:1 赔率
    10: 7, 11: 7,    # 6:1 赔率
}


# Feature: sic-bo-game, Property 8: 总和赔率正确性
@given(
    target_sum=sum_strategy,
    dice=non_triple_dice_strategy,
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_8_sum_payout_correctness_non_triple(target_sum, dice, bet_amount):
    """
    属性 8: 总和赔率正确性 - 非围骰情况
    
    *对于任何* 总和押注和非围骰骰子结果，如果总和匹配则按赔率表返回奖金，不匹配返回0
    
    **验证需求: 4.3, 4.5**
    """
    payout = SicBoCalculator.calculate_sum_payout(target_sum, dice, bet_amount)
    actual_sum = sum(dice)
    
    if actual_sum == target_sum:
        expected = bet_amount * SUM_PAYOUT_TABLE.get(target_sum, 0)
    else:
        expected = 0
    
    assert payout == expected, \
        f"押注总和 {target_sum}，骰子 {dice}（总和 {actual_sum}），" \
        f"押注金额 {bet_amount}，期望赔付 {expected}，实际 {payout}"


# Feature: sic-bo-game, Property 8: 总和赔率正确性 - 围骰返回0
@given(
    target_sum=sum_strategy,
    dice=triple_dice_strategy,
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_8_sum_payout_triple_returns_zero(target_sum, dice, bet_amount):
    """
    属性 8: 总和赔率正确性 - 围骰时返回0
    
    *对于任何* 总和押注和围骰骰子结果，无论总和是否匹配，都应该返回0（庄家通吃）
    
    **验证需求: 4.4**
    """
    payout = SicBoCalculator.calculate_sum_payout(target_sum, dice, bet_amount)
    
    assert payout == 0, \
        f"围骰 {dice}，押注总和 {target_sum}，应该返回 0（庄家通吃），实际 {payout}"



# ============================================================================
# 属性 9: 大小赔率正确性
# ============================================================================

# Feature: sic-bo-game, Property 9: 大小赔率正确性
@given(
    is_big=st.booleans(),
    dice=non_triple_dice_strategy,
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_9_big_small_payout_correctness_non_triple(is_big, dice, bet_amount):
    """
    属性 9: 大小赔率正确性 - 非围骰情况
    
    *对于任何* 大小押注和非围骰骰子结果：
    - 大 (11-17) 匹配时返回 bet * 2
    - 小 (4-10) 匹配时返回 bet * 2
    - 不匹配返回 0
    
    **验证需求: 5.3, 5.4**
    """
    payout = SicBoCalculator.calculate_big_small_payout(is_big, dice, bet_amount)
    total = sum(dice)
    
    if is_big:
        # 押大：11-17 赢
        if 11 <= total <= 17:
            expected = bet_amount * 2
        else:
            expected = 0
    else:
        # 押小：4-10 赢
        if 4 <= total <= 10:
            expected = bet_amount * 2
        else:
            expected = 0
    
    bet_type = "大" if is_big else "小"
    assert payout == expected, \
        f"押注 {bet_type}，骰子 {dice}（总和 {total}），" \
        f"押注金额 {bet_amount}，期望赔付 {expected}，实际 {payout}"


# Feature: sic-bo-game, Property 9: 大小赔率正确性 - 围骰返回0
@given(
    is_big=st.booleans(),
    dice=triple_dice_strategy,
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_9_big_small_payout_triple_returns_zero(is_big, dice, bet_amount):
    """
    属性 9: 大小赔率正确性 - 围骰时返回0
    
    *对于任何* 大小押注和围骰骰子结果，无论总和是否在大小范围内，都应该返回0（庄家通吃）
    
    **验证需求: 5.5**
    """
    payout = SicBoCalculator.calculate_big_small_payout(is_big, dice, bet_amount)
    
    bet_type = "大" if is_big else "小"
    assert payout == 0, \
        f"围骰 {dice}，押注 {bet_type}，应该返回 0（庄家通吃），实际 {payout}"



# ============================================================================
# 属性 11: 围骰通吃规则
# ============================================================================

# Feature: sic-bo-game, Property 11: 围骰通吃规则
@given(
    dice=triple_dice_strategy,
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_11_triple_house_edge_sum_bets(dice, bet_amount):
    """
    属性 11: 围骰通吃规则 - 总和押注
    
    *对于任何* 围骰结果（三个骰子相同），所有总和押注应该返回0（庄家通吃）
    
    **验证需求: 4.4**
    """
    actual_sum = sum(dice)
    
    # 测试所有可能的总和押注
    for target_sum in range(4, 18):
        payout = SicBoCalculator.calculate_sum_payout(target_sum, dice, bet_amount)
        assert payout == 0, \
            f"围骰 {dice}（总和 {actual_sum}），押注总和 {target_sum}，" \
            f"应该返回 0（庄家通吃），实际 {payout}"


# Feature: sic-bo-game, Property 11: 围骰通吃规则
@given(
    dice=triple_dice_strategy,
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_11_triple_house_edge_big_small_bets(dice, bet_amount):
    """
    属性 11: 围骰通吃规则 - 大小押注
    
    *对于任何* 围骰结果（三个骰子相同），所有大小押注应该返回0（庄家通吃）
    
    **验证需求: 5.5**
    """
    # 测试押大
    payout_big = SicBoCalculator.calculate_big_small_payout(True, dice, bet_amount)
    assert payout_big == 0, \
        f"围骰 {dice}，押大应该返回 0（庄家通吃），实际 {payout_big}"
    
    # 测试押小
    payout_small = SicBoCalculator.calculate_big_small_payout(False, dice, bet_amount)
    assert payout_small == 0, \
        f"围骰 {dice}，押小应该返回 0（庄家通吃），实际 {payout_small}"


# Feature: sic-bo-game, Property 11: 围骰通吃规则 - 综合测试
@given(
    triple_value=st.integers(min_value=1, max_value=6),
    bet_amount=bet_amount_strategy
)
@settings(max_examples=100)
def test_property_11_triple_house_edge_comprehensive(triple_value, bet_amount):
    """
    属性 11: 围骰通吃规则 - 综合测试
    
    *对于任何* 围骰结果，使用 calculate_bet_payout 统一入口验证：
    - 所有总和押注返回 0
    - 所有大小押注返回 0
    
    **验证需求: 4.4, 5.5**
    """
    dice = [triple_value, triple_value, triple_value]
    actual_sum = sum(dice)
    
    # 测试总和押注
    for target_sum in range(4, 18):
        bet = SicBoBet(
            user_id=1,
            bet_type=BetType.SUM,
            amount=bet_amount,
            numbers=[target_sum],
            created_at=0.0
        )
        payout = SicBoCalculator.calculate_bet_payout(bet, dice)
        assert payout == 0, \
            f"围骰 {dice}，总和押注 {target_sum}，应该返回 0，实际 {payout}"
    
    # 测试押大
    bet_big = SicBoBet(
        user_id=1,
        bet_type=BetType.BIG,
        amount=bet_amount,
        numbers=[],
        created_at=0.0
    )
    payout_big = SicBoCalculator.calculate_bet_payout(bet_big, dice)
    assert payout_big == 0, \
        f"围骰 {dice}，押大应该返回 0，实际 {payout_big}"
    
    # 测试押小
    bet_small = SicBoBet(
        user_id=1,
        bet_type=BetType.SMALL,
        amount=bet_amount,
        numbers=[],
        created_at=0.0
    )
    payout_small = SicBoCalculator.calculate_bet_payout(bet_small, dice)
    assert payout_small == 0, \
        f"围骰 {dice}，押小应该返回 0，实际 {payout_small}"
