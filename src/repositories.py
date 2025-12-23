"""
仓储层
管理用户和交易数据的 CRUD 操作
"""
from typing import Optional, List
import time
from src.database import DatabaseManager
from src.models import User, Transaction


class UserRepository:
    """用户仓储，管理用户账户的 CRUD 操作"""
    
    def __init__(self, db: DatabaseManager):
        """
        初始化用户仓储
        
        Args:
            db: 数据库管理器实例
        """
        self.db = db
    
    async def create_user(self, telegram_id: int, username: str) -> User:
        """
        创建新用户，初始 1000 金币
        
        Args:
            telegram_id: Telegram 用户 ID
            username: Telegram 用户名
            
        Returns:
            创建的用户对象
        """
        now = int(time.time())
        
        await self.db.execute(
            """INSERT INTO users 
               (telegram_id, username, balance, last_daily_claim, created_at, updated_at) 
               VALUES (?, ?, ?, ?, ?, ?)""",
            (telegram_id, username, 1000, 0, now, now)
        )
        
        return User(
            telegram_id=telegram_id,
            username=username,
            balance=1000,
            last_daily_claim=0,
            created_at=now,
            updated_at=now
        )
    
    async def get_user(self, telegram_id: int) -> Optional[User]:
        """
        获取用户信息
        
        Args:
            telegram_id: Telegram 用户 ID
            
        Returns:
            用户对象，如果不存在返回 None
        """
        result = await self.db.fetch_one(
            "SELECT * FROM users WHERE telegram_id = ?",
            (telegram_id,)
        )
        
        if result:
            return User.from_dict(result)
        return None
    
    async def update_balance(self, telegram_id: int, amount: int) -> bool:
        """
        更新用户余额（可正可负）
        
        Args:
            telegram_id: Telegram 用户 ID
            amount: 金币变动量（正数为增加，负数为减少）
            
        Returns:
            是否更新成功
        """
        now = int(time.time())
        
        await self.db.execute(
            """UPDATE users 
               SET balance = balance + ?, updated_at = ? 
               WHERE telegram_id = ?""",
            (amount, now, telegram_id)
        )
        
        return True
    
    async def get_top_users(self, limit: int = 10) -> List[User]:
        """
        获取财富榜（按余额降序，余额相同按创建时间升序）
        
        Args:
            limit: 返回的用户数量
            
        Returns:
            用户列表
        """
        results = await self.db.fetch_all(
            """SELECT * FROM users 
               ORDER BY balance DESC, created_at ASC 
               LIMIT ?""",
            (limit,)
        )
        
        return [User.from_dict(row) for row in results]
    
    async def update_daily_claim(self, telegram_id: int) -> bool:
        """
        更新每日签到时间
        
        Args:
            telegram_id: Telegram 用户 ID
            
        Returns:
            是否更新成功
        """
        now = int(time.time())
        
        await self.db.execute(
            """UPDATE users 
               SET last_daily_claim = ?, updated_at = ? 
               WHERE telegram_id = ?""",
            (now, now, telegram_id)
        )
        
        return True
    
    async def can_claim_daily(self, telegram_id: int) -> bool:
        """
        检查是否可以签到（距离上次签到超过 24 小时）
        
        Args:
            telegram_id: Telegram 用户 ID
            
        Returns:
            是否可以签到
        """
        user = await self.get_user(telegram_id)
        if not user:
            return False
        
        # 如果从未签到过（last_daily_claim = 0），可以签到
        if user.last_daily_claim == 0:
            return True
        
        # 检查是否超过 24 小时（86400 秒）
        now = int(time.time())
        return (now - user.last_daily_claim) >= 86400


class TransactionRepository:
    """交易仓储，管理交易记录的 CRUD 操作"""
    
    def __init__(self, db: DatabaseManager):
        """
        初始化交易仓储
        
        Args:
            db: 数据库管理器实例
        """
        self.db = db
    
    async def log_transaction(
        self,
        user_id: int,
        amount: int,
        transaction_type: str,
        description: Optional[str] = None
    ) -> None:
        """
        记录交易日志
        
        Args:
            user_id: 用户 ID
            amount: 金币变动量（正数为增加，负数为减少）
            transaction_type: 交易类型
            description: 交易描述
        """
        now = int(time.time())
        
        await self.db.execute(
            """INSERT INTO transactions 
               (user_id, amount, type, description, created_at) 
               VALUES (?, ?, ?, ?, ?)""",
            (user_id, amount, transaction_type, description, now)
        )
    
    async def get_user_history(
        self,
        user_id: int,
        limit: int = 50
    ) -> List[Transaction]:
        """
        获取用户交易历史
        
        Args:
            user_id: 用户 ID
            limit: 返回的记录数量
            
        Returns:
            交易记录列表
        """
        results = await self.db.fetch_all(
            """SELECT * FROM transactions 
               WHERE user_id = ? 
               ORDER BY created_at DESC 
               LIMIT ?""",
            (user_id, limit)
        )
        
        return [Transaction.from_dict(row) for row in results]
