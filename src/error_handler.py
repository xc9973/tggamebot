"""
错误处理模块
提供全局错误处理器、Telegram API 重试机制和友好的错误消息
"""
import asyncio
import logging
from typing import Optional, Callable, Any, TypeVar
from functools import wraps
from telegram import Update
from telegram.ext import ContextTypes
from telegram.error import (
    TelegramError,
    NetworkError,
    TimedOut,
    RetryAfter,
    BadRequest,
    Forbidden,
)

logger = logging.getLogger(__name__)

T = TypeVar('T')


class ErrorMessages:
    """错误消息常量"""
    
    # 系统错误
    SYSTEM_ERROR = "❌ 系统暂时不可用，请稍后再试"
    DATABASE_ERROR = "❌ 数据库连接失败，系统维护中"
    NETWORK_ERROR = "❌ 网络连接失败，请稍后再试"
    
    # 命令格式错误
    COMMAND_FORMAT_ERROR = "❌ 命令格式错误"
    
    # 参数验证错误
    INVALID_AMOUNT = "❌ 无效的金额，金额必须是正整数"
    AMOUNT_TOO_SMALL = "❌ 金额必须大于 0"
    INSUFFICIENT_BALANCE = "❌ 余额不足"
    USER_NOT_FOUND = "❌ 用户不存在"
    SELF_TRANSFER = "❌ 不能向自己转账"
    
    # 游戏错误
    GAME_IN_PROGRESS = "❌ 您已有进行中的游戏，请先完成当前游戏"
    GAME_NOT_FOUND = "❌ 没有进行中的游戏"
    GAME_UNAVAILABLE = "❌ 游戏功能暂不可用"
    
    # 权限错误
    PERMISSION_DENIED = "❌ 权限不足，只有管理员可以执行此操作"
    
    @staticmethod
    def command_usage(command: str, usage: str, example: str) -> str:
        """
        生成命令使用说明
        
        Args:
            command: 命令名称
            usage: 使用方法
            example: 示例
            
        Returns:
            格式化的使用说明
        """
        return f"❌ 命令格式错误\n\n用法: {usage}\n示例: {example}"
    
    @staticmethod
    def invalid_parameter(param_name: str, reason: str) -> str:
        """
        生成参数验证错误消息
        
        Args:
            param_name: 参数名称
            reason: 错误原因
            
        Returns:
            格式化的错误消息
        """
        return f"❌ 无效的参数 '{param_name}': {reason}"


class RetryConfig:
    """重试配置"""
    
    def __init__(
        self,
        max_retries: int = 3,
        base_delay: float = 1.0,
        max_delay: float = 30.0,
        exponential_base: float = 2.0
    ):
        """
        初始化重试配置
        
        Args:
            max_retries: 最大重试次数
            base_delay: 基础延迟（秒）
            max_delay: 最大延迟（秒）
            exponential_base: 指数退避基数
        """
        self.max_retries = max_retries
        self.base_delay = base_delay
        self.max_delay = max_delay
        self.exponential_base = exponential_base
    
    def get_delay(self, attempt: int) -> float:
        """
        计算第 n 次重试的延迟时间
        
        Args:
            attempt: 当前重试次数（从 0 开始）
            
        Returns:
            延迟时间（秒）
        """
        delay = self.base_delay * (self.exponential_base ** attempt)
        return min(delay, self.max_delay)


# 默认重试配置
DEFAULT_RETRY_CONFIG = RetryConfig(max_retries=3)


async def retry_telegram_api(
    func: Callable[..., Any],
    *args,
    config: Optional[RetryConfig] = None,
    **kwargs
) -> Any:
    """
    带重试机制的 Telegram API 调用
    
    Args:
        func: 要调用的异步函数
        *args: 函数参数
        config: 重试配置
        **kwargs: 函数关键字参数
        
    Returns:
        函数返回值
        
    Raises:
        TelegramError: 重试次数用尽后仍然失败
    """
    if config is None:
        config = DEFAULT_RETRY_CONFIG
    
    last_exception = None
    
    for attempt in range(config.max_retries + 1):
        try:
            return await func(*args, **kwargs)
        except RetryAfter as e:
            # Telegram 要求等待特定时间
            wait_time = e.retry_after
            logger.warning(f"Rate limited, waiting {wait_time} seconds")
            await asyncio.sleep(wait_time)
            last_exception = e
        except (BadRequest, Forbidden) as e:
            # 请求错误或权限错误，不重试，直接抛出
            logger.error(f"Telegram API error (not retrying): {e}")
            raise
        except (NetworkError, TimedOut) as e:
            # 网络错误，可以重试
            if attempt < config.max_retries:
                delay = config.get_delay(attempt)
                logger.warning(f"Network error on attempt {attempt + 1}, retrying in {delay}s: {e}")
                await asyncio.sleep(delay)
                last_exception = e
            else:
                raise
        except TelegramError as e:
            # 其他 Telegram 错误
            if attempt < config.max_retries:
                delay = config.get_delay(attempt)
                logger.warning(f"Telegram error on attempt {attempt + 1}, retrying in {delay}s: {e}")
                await asyncio.sleep(delay)
                last_exception = e
            else:
                raise
    
    # 重试次数用尽
    if last_exception:
        raise last_exception


def with_retry(config: Optional[RetryConfig] = None):
    """
    装饰器：为异步函数添加重试机制
    
    Args:
        config: 重试配置
        
    Returns:
        装饰器函数
    """
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            return await retry_telegram_api(func, *args, config=config, **kwargs)
        return wrapper
    return decorator


async def global_error_handler(update: object, context: ContextTypes.DEFAULT_TYPE) -> None:
    """
    全局错误处理器
    捕获所有未处理的异常并向用户发送友好的错误消息
    
    Args:
        update: Telegram Update 对象
        context: 上下文对象，包含错误信息
    """
    error = context.error
    
    # 记录错误日志
    logger.error(f"Exception while handling an update: {error}", exc_info=error)
    
    # 确定错误消息
    error_message = ErrorMessages.SYSTEM_ERROR
    
    if isinstance(error, NetworkError):
        error_message = ErrorMessages.NETWORK_ERROR
    elif isinstance(error, TimedOut):
        error_message = ErrorMessages.NETWORK_ERROR
    elif isinstance(error, BadRequest):
        # 不向用户显示具体的 BadRequest 错误
        error_message = ErrorMessages.SYSTEM_ERROR
    elif isinstance(error, Forbidden):
        # Bot 被用户阻止或没有权限
        logger.warning(f"Bot forbidden: {error}")
        return  # 不发送消息
    
    # 尝试向用户发送错误消息
    if isinstance(update, Update):
        try:
            if update.effective_message:
                await update.effective_message.reply_text(error_message)
            elif update.callback_query:
                await update.callback_query.answer(error_message, show_alert=True)
        except TelegramError as e:
            logger.error(f"Failed to send error message to user: {e}")


class CommandValidator:
    """命令参数验证器"""
    
    @staticmethod
    def validate_amount(amount_str: str) -> tuple[bool, int, str]:
        """
        验证金额参数
        
        Args:
            amount_str: 金额字符串
            
        Returns:
            (是否有效, 金额值, 错误消息)
        """
        try:
            amount = int(amount_str)
        except ValueError:
            return False, 0, ErrorMessages.invalid_parameter("金额", "必须是整数")
        
        if amount <= 0:
            return False, 0, ErrorMessages.invalid_parameter("金额", "必须大于 0")
        
        return True, amount, ""
    
    @staticmethod
    def validate_username(username: str) -> tuple[bool, str, str]:
        """
        验证用户名参数
        
        Args:
            username: 用户名字符串
            
        Returns:
            (是否有效, 处理后的用户名, 错误消息)
        """
        if not username:
            return False, "", ErrorMessages.invalid_parameter("用户名", "不能为空")
        
        # 移除 @ 前缀
        clean_username = username.lstrip('@')
        
        if not clean_username:
            return False, "", ErrorMessages.invalid_parameter("用户名", "格式无效")
        
        return True, clean_username, ""


def format_command_help(command: str, description: str, usage: str, examples: list[str]) -> str:
    """
    格式化命令帮助信息
    
    Args:
        command: 命令名称
        description: 命令描述
        usage: 使用方法
        examples: 示例列表
        
    Returns:
        格式化的帮助信息
    """
    help_text = f"📖 {command} 命令帮助\n\n"
    help_text += f"📝 描述: {description}\n\n"
    help_text += f"💡 用法: {usage}\n\n"
    help_text += "📌 示例:\n"
    for example in examples:
        help_text += f"  {example}\n"
    
    return help_text
