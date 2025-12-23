"""
并发控制模块
提供用户级操作锁和游戏会话互斥检查功能
"""
import asyncio
import time
from typing import Dict, Optional, Set
from functools import wraps
import logging

logger = logging.getLogger(__name__)

# 游戏会话超时时间（秒）
GAME_SESSION_TIMEOUT = 60


class UserLockManager:
    """
    用户锁管理器
    为每个用户提供独立的操作锁，确保同一用户的操作串行执行
    """
    
    def __init__(self):
        """初始化用户锁管理器"""
        self._locks: Dict[int, asyncio.Lock] = {}
        self._global_lock = asyncio.Lock()
    
    async def get_lock(self, user_id: int) -> asyncio.Lock:
        """
        获取指定用户的锁
        如果锁不存在则创建
        
        Args:
            user_id: 用户 ID
            
        Returns:
            用户的 asyncio.Lock 实例
        """
        async with self._global_lock:
            if user_id not in self._locks:
                self._locks[user_id] = asyncio.Lock()
            return self._locks[user_id]
    
    async def acquire(self, user_id: int) -> bool:
        """
        获取用户锁
        
        Args:
            user_id: 用户 ID
            
        Returns:
            是否成功获取锁
        """
        lock = await self.get_lock(user_id)
        await lock.acquire()
        return True
    
    async def release(self, user_id: int) -> None:
        """
        释放用户锁
        
        Args:
            user_id: 用户 ID
        """
        async with self._global_lock:
            if user_id in self._locks:
                lock = self._locks[user_id]
                if lock.locked():
                    lock.release()
    
    def is_locked(self, user_id: int) -> bool:
        """
        检查用户是否被锁定
        
        Args:
            user_id: 用户 ID
            
        Returns:
            是否被锁定
        """
        if user_id not in self._locks:
            return False
        return self._locks[user_id].locked()
    
    async def cleanup_user(self, user_id: int) -> None:
        """
        清理用户的锁（当用户长时间不活跃时调用）
        
        Args:
            user_id: 用户 ID
        """
        async with self._global_lock:
            if user_id in self._locks and not self._locks[user_id].locked():
                del self._locks[user_id]


class GameSessionManager:
    """
    游戏会话管理器
    确保同一用户同时只能有一个进行中的游戏会话
    支持超时自动清理
    """
    
    def __init__(self, timeout: int = GAME_SESSION_TIMEOUT):
        """初始化游戏会话管理器"""
        # 存储用户当前进行中的游戏类型和开始时间
        self._active_sessions: Dict[int, tuple[str, float]] = {}
        self._lock = asyncio.Lock()
        self._timeout = timeout
    
    async def start_session(self, user_id: int, game_type: str) -> tuple[bool, str]:
        """
        开始游戏会话
        
        Args:
            user_id: 用户 ID
            game_type: 游戏类型 ('dice', 'slot', 'blackjack')
            
        Returns:
            (成功, 消息) 元组
        """
        async with self._lock:
            if user_id in self._active_sessions:
                current_game, start_time = self._active_sessions[user_id]
                elapsed = time.time() - start_time
                
                # 检查是否超时
                if elapsed > self._timeout:
                    # 超时自动清理
                    logger.info(f"Game session timeout for user {user_id}, game: {current_game}, elapsed: {elapsed:.1f}s")
                    del self._active_sessions[user_id]
                else:
                    return False, f"您已有进行中的 {current_game} 游戏，请先完成当前游戏"
            
            self._active_sessions[user_id] = (game_type, time.time())
            return True, "游戏会话已创建"
    
    async def end_session(self, user_id: int) -> None:
        """
        结束游戏会话
        
        Args:
            user_id: 用户 ID
        """
        async with self._lock:
            if user_id in self._active_sessions:
                del self._active_sessions[user_id]
    
    def has_active_session(self, user_id: int) -> bool:
        """
        检查用户是否有进行中的游戏会话（考虑超时）
        
        Args:
            user_id: 用户 ID
            
        Returns:
            是否有进行中的会话
        """
        if user_id not in self._active_sessions:
            return False
        
        _, start_time = self._active_sessions[user_id]
        if time.time() - start_time > self._timeout:
            return False
        return True
    
    def get_active_game(self, user_id: int) -> Optional[str]:
        """
        获取用户当前进行中的游戏类型
        
        Args:
            user_id: 用户 ID
            
        Returns:
            游戏类型，如果没有进行中的游戏返回 None
        """
        if user_id not in self._active_sessions:
            return None
        game_type, start_time = self._active_sessions[user_id]
        # 超时的不算
        if time.time() - start_time > self._timeout:
            return None
        return game_type
    
    async def cleanup_session(self, user_id: int) -> None:
        """
        清理用户的游戏会话（超时或异常时调用）
        
        Args:
            user_id: 用户 ID
        """
        await self.end_session(user_id)
    
    async def cleanup_expired_sessions(self) -> int:
        """
        清理所有过期的会话
        
        Returns:
            清理的会话数量
        """
        async with self._lock:
            now = time.time()
            expired = [
                user_id for user_id, (_, start_time) in self._active_sessions.items()
                if now - start_time > self._timeout
            ]
            for user_id in expired:
                logger.info(f"Cleaning up expired game session for user {user_id}")
                del self._active_sessions[user_id]
            return len(expired)


class ConcurrencyManager:
    """
    并发控制管理器
    整合用户锁和游戏会话管理功能
    """
    
    def __init__(self):
        """初始化并发控制管理器"""
        self.user_locks = UserLockManager()
        self.game_sessions = GameSessionManager()
    
    async def acquire_user_lock(self, user_id: int) -> bool:
        """
        获取用户操作锁
        
        Args:
            user_id: 用户 ID
            
        Returns:
            是否成功获取锁
        """
        return await self.user_locks.acquire(user_id)
    
    async def release_user_lock(self, user_id: int) -> None:
        """
        释放用户操作锁
        
        Args:
            user_id: 用户 ID
        """
        await self.user_locks.release(user_id)
    
    async def start_game(self, user_id: int, game_type: str) -> tuple[bool, str]:
        """
        开始游戏（检查会话互斥）
        
        Args:
            user_id: 用户 ID
            game_type: 游戏类型
            
        Returns:
            (成功, 消息) 元组
        """
        return await self.game_sessions.start_session(user_id, game_type)
    
    async def end_game(self, user_id: int) -> None:
        """
        结束游戏
        
        Args:
            user_id: 用户 ID
        """
        await self.game_sessions.end_session(user_id)
    
    def has_active_game(self, user_id: int) -> bool:
        """
        检查用户是否有进行中的游戏
        
        Args:
            user_id: 用户 ID
            
        Returns:
            是否有进行中的游戏
        """
        return self.game_sessions.has_active_session(user_id)


def with_user_lock(concurrency_manager_attr: str = 'concurrency_manager'):
    """
    装饰器：为命令处理器添加用户锁保护
    
    Args:
        concurrency_manager_attr: BotHandlers 实例中 ConcurrencyManager 的属性名
        
    Returns:
        装饰器函数
    """
    def decorator(func):
        @wraps(func)
        async def wrapper(self, update, context, *args, **kwargs):
            user = update.effective_user
            if not user:
                return
            
            user_id = user.id
            concurrency_manager = getattr(self, concurrency_manager_attr, None)
            
            if concurrency_manager is None:
                # 如果没有并发管理器，直接执行
                return await func(self, update, context, *args, **kwargs)
            
            try:
                await concurrency_manager.acquire_user_lock(user_id)
                return await func(self, update, context, *args, **kwargs)
            finally:
                await concurrency_manager.release_user_lock(user_id)
        
        return wrapper
    return decorator


def with_game_session(game_type: str, concurrency_manager_attr: str = 'concurrency_manager'):
    """
    装饰器：为游戏命令处理器添加会话互斥检查
    
    Args:
        game_type: 游戏类型
        concurrency_manager_attr: BotHandlers 实例中 ConcurrencyManager 的属性名
        
    Returns:
        装饰器函数
    """
    def decorator(func):
        @wraps(func)
        async def wrapper(self, update, context, *args, **kwargs):
            user = update.effective_user
            if not user:
                return
            
            user_id = user.id
            concurrency_manager = getattr(self, concurrency_manager_attr, None)
            
            if concurrency_manager is None:
                # 如果没有并发管理器，直接执行
                return await func(self, update, context, *args, **kwargs)
            
            # 检查是否有进行中的游戏
            success, message = await concurrency_manager.start_game(user_id, game_type)
            if not success:
                await update.message.reply_text(f"❌ {message}")
                return
            
            try:
                return await func(self, update, context, *args, **kwargs)
            finally:
                # 对于非持续性游戏（dice, slot），立即结束会话
                # blackjack 会话由游戏逻辑管理
                if game_type in ['dice', 'slot']:
                    await concurrency_manager.end_game(user_id)
        
        return wrapper
    return decorator
