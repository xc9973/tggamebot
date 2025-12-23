"""
错误处理属性测试
使用 Hypothesis 进行属性测试，验证命令格式错误反馈、参数验证反馈和 API 调用重试机制
"""
import pytest
import asyncio
from unittest.mock import AsyncMock, MagicMock, patch
from hypothesis import given, strategies as st, settings, HealthCheck
from telegram.error import NetworkError, TimedOut, RetryAfter, BadRequest, Forbidden

from src.error_handler import (
    ErrorMessages,
    CommandValidator,
    RetryConfig,
    retry_telegram_api,
    format_command_help,
)


# Feature: telegram-game-bot, Property 36: 命令格式错误反馈
@settings(max_examples=5)
@given(
    command=st.sampled_from(['/pay', '/dice', '/slot', '/bj', '/admin_add', '/admin_remove']),
    usage=st.text(min_size=5, max_size=50, alphabet=st.characters(blacklist_categories=('Cs',))),
    example=st.text(min_size=5, max_size=50, alphabet=st.characters(blacklist_categories=('Cs',)))
)
def test_property_command_format_error_feedback(command, usage, example):
    """
    属性 36: 命令格式错误反馈
    对于任何格式错误的命令，应该返回使用说明和正确的命令格式示例
    验证需求: 11.1
    """
    # 生成错误消息
    error_message = ErrorMessages.command_usage(command, usage, example)
    
    # 验证错误消息包含必要的信息
    assert "❌" in error_message  # 包含错误标识
    assert "命令格式错误" in error_message  # 包含错误类型
    assert "用法:" in error_message  # 包含使用说明
    assert usage in error_message  # 包含具体用法
    assert "示例:" in error_message  # 包含示例标签
    assert example in error_message  # 包含具体示例


# Feature: telegram-game-bot, Property 37: 参数验证反馈
@settings(max_examples=5)
@given(
    param_name=st.text(min_size=1, max_size=20, alphabet=st.characters(blacklist_categories=('Cs',))),
    reason=st.text(min_size=1, max_size=50, alphabet=st.characters(blacklist_categories=('Cs',)))
)
def test_property_parameter_validation_feedback(param_name, reason):
    """
    属性 37: 参数验证反馈
    对于任何无效参数，应该明确指出哪个参数无效及原因
    验证需求: 11.3
    """
    # 生成参数验证错误消息
    error_message = ErrorMessages.invalid_parameter(param_name, reason)
    
    # 验证错误消息包含必要的信息
    assert "❌" in error_message  # 包含错误标识
    assert "无效的参数" in error_message  # 包含错误类型
    assert param_name in error_message  # 包含参数名称
    assert reason in error_message  # 包含错误原因


# Feature: telegram-game-bot, Property 37: 金额参数验证
@settings(max_examples=5)
@given(
    amount_str=st.one_of(
        st.text(min_size=0, max_size=10, alphabet=st.characters(blacklist_categories=('Cs',))),
        st.integers().map(str),
        st.floats(allow_nan=False, allow_infinity=False).map(str)
    )
)
def test_property_amount_validation(amount_str):
    """
    属性 37: 金额参数验证
    对于任何金额输入，验证器应该正确判断有效性并返回适当的错误消息
    验证需求: 11.3
    """
    is_valid, amount, error_message = CommandValidator.validate_amount(amount_str)
    
    # 尝试解析为整数
    try:
        parsed_amount = int(amount_str)
        if parsed_amount > 0:
            # 应该是有效的
            assert is_valid is True
            assert amount == parsed_amount
            assert error_message == ""
        else:
            # 非正数应该无效
            assert is_valid is False
            assert "无效的参数" in error_message or "金额" in error_message
    except (ValueError, OverflowError):
        # 无法解析为整数，应该无效
        assert is_valid is False
        assert "无效的参数" in error_message


# Feature: telegram-game-bot, Property 37: 用户名参数验证
@settings(max_examples=5)
@given(
    username=st.one_of(
        st.text(min_size=0, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',))),
        st.just(""),
        st.just("@"),
        st.text(min_size=1, max_size=32, alphabet=st.characters(blacklist_categories=('Cs',))).map(lambda x: f"@{x}")
    )
)
def test_property_username_validation(username):
    """
    属性 37: 用户名参数验证
    对于任何用户名输入，验证器应该正确判断有效性
    验证需求: 11.3
    """
    is_valid, clean_username, error_message = CommandValidator.validate_username(username)
    
    # 移除 @ 前缀后的用户名
    expected_clean = username.lstrip('@') if username else ""
    
    if not username or not expected_clean:
        # 空用户名应该无效
        assert is_valid is False
        assert "无效的参数" in error_message
    else:
        # 非空用户名应该有效
        assert is_valid is True
        assert clean_username == expected_clean
        assert error_message == ""


# Feature: telegram-game-bot, Property 38: API 调用重试机制
@settings(max_examples=5, deadline=1000)  # 增加 deadline 到 1000ms
@given(
    max_retries=st.integers(min_value=1, max_value=5),
    fail_count=st.integers(min_value=0, max_value=6)
)
@pytest.mark.asyncio
async def test_property_api_retry_mechanism(max_retries, fail_count):
    """
    属性 38: API 调用重试机制
    对于任何 Telegram API 调用失败，应该自动重试最多指定次数
    验证需求: 11.4
    """
    config = RetryConfig(max_retries=max_retries, base_delay=0.01, max_delay=0.1)
    
    call_count = 0
    
    async def mock_api_call():
        nonlocal call_count
        call_count += 1
        if call_count <= fail_count:
            raise NetworkError("Network error")
        return "success"
    
    if fail_count <= max_retries:
        # 应该成功（在重试次数内恢复）
        result = await retry_telegram_api(mock_api_call, config=config)
        assert result == "success"
        assert call_count == fail_count + 1
    else:
        # 应该失败（超过重试次数）
        with pytest.raises(NetworkError):
            await retry_telegram_api(mock_api_call, config=config)
        assert call_count == max_retries + 1


# Feature: telegram-game-bot, Property 38: 重试延迟计算
@settings(max_examples=5, suppress_health_check=[HealthCheck.data_too_large])
@given(
    base_delay=st.floats(min_value=0.1, max_value=5.0),
    max_delay=st.floats(min_value=5.0, max_value=60.0),
    exponential_base=st.floats(min_value=1.5, max_value=3.0),
    attempt=st.integers(min_value=0, max_value=10)
)
def test_property_retry_delay_calculation(base_delay, max_delay, exponential_base, attempt):
    """
    属性 38: 重试延迟计算
    对于任何重试配置，延迟时间应该按指数增长但不超过最大值
    验证需求: 11.4
    """
    config = RetryConfig(
        max_retries=10,
        base_delay=base_delay,
        max_delay=max_delay,
        exponential_base=exponential_base
    )
    
    delay = config.get_delay(attempt)
    
    # 验证延迟不超过最大值
    assert delay <= max_delay
    
    # 验证延迟是正数
    assert delay > 0
    
    # 验证延迟计算正确
    expected_delay = min(base_delay * (exponential_base ** attempt), max_delay)
    assert abs(delay - expected_delay) < 0.0001


# Feature: telegram-game-bot, Property 38: RetryAfter 处理
@pytest.mark.asyncio
async def test_property_retry_after_handling():
    """
    属性 38: RetryAfter 处理
    当 Telegram 返回 RetryAfter 错误时，应该等待指定时间后重试
    验证需求: 11.4
    """
    config = RetryConfig(max_retries=3, base_delay=0.01, max_delay=0.1)
    
    call_count = 0
    
    async def mock_api_call():
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            raise RetryAfter(retry_after=0.01)  # 要求等待 0.01 秒
        return "success"
    
    result = await retry_telegram_api(mock_api_call, config=config)
    assert result == "success"
    assert call_count == 2


# Feature: telegram-game-bot, Property 38: 不可重试错误处理
@pytest.mark.asyncio
async def test_property_non_retryable_errors():
    """
    属性 38: 不可重试错误处理
    对于 BadRequest 和 Forbidden 错误，不应该重试
    验证需求: 11.4
    """
    config = RetryConfig(max_retries=3, base_delay=0.01, max_delay=0.1)
    
    # BadRequest 不应该重试
    call_count_bad_request = 0
    
    async def mock_bad_request():
        nonlocal call_count_bad_request
        call_count_bad_request += 1
        raise BadRequest("Bad request")
    
    with pytest.raises(BadRequest):
        await retry_telegram_api(mock_bad_request, config=config)
    assert call_count_bad_request == 1  # 只调用一次，不重试
    
    # Forbidden 不应该重试
    call_count_forbidden = 0
    
    async def mock_forbidden():
        nonlocal call_count_forbidden
        call_count_forbidden += 1
        raise Forbidden("Forbidden")
    
    with pytest.raises(Forbidden):
        await retry_telegram_api(mock_forbidden, config=config)
    assert call_count_forbidden == 1  # 只调用一次，不重试


# 额外测试：命令帮助格式化
@settings(max_examples=5)
@given(
    command=st.text(min_size=1, max_size=20, alphabet=st.characters(blacklist_categories=('Cs',))),
    description=st.text(min_size=1, max_size=100, alphabet=st.characters(blacklist_categories=('Cs',))),
    usage=st.text(min_size=1, max_size=50, alphabet=st.characters(blacklist_categories=('Cs',))),
    examples=st.lists(
        st.text(min_size=1, max_size=30, alphabet=st.characters(blacklist_categories=('Cs',))),
        min_size=1,
        max_size=5
    )
)
def test_command_help_formatting(command, description, usage, examples):
    """
    测试命令帮助信息格式化
    """
    help_text = format_command_help(command, description, usage, examples)
    
    # 验证帮助信息包含所有必要部分
    assert command in help_text
    assert description in help_text
    assert usage in help_text
    for example in examples:
        assert example in help_text
    
    # 验证格式正确
    assert "📖" in help_text  # 标题图标
    assert "📝 描述:" in help_text
    assert "💡 用法:" in help_text
    assert "📌 示例:" in help_text
