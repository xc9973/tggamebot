"""
账户管理器
处理账户相关业务逻辑，包括用户创建、余额查询、每日签到和转账
"""
import time
from typing import Tuple, Optional
from src.repositories import UserRepository, TransactionRepository
from src.models import User


class AccountManager:
    """账户管理器，处理账户相关业务逻辑"""
    
    def __init__(self, user_repo: UserRepository, tx_repo: TransactionRepository):
        """
        初始化账户管理器
        
        Args:
            user_repo: 用户仓储实例
            tx_repo: 交易仓储实例
        """
        self.user_repo = user_repo
        self.tx_repo = tx_repo
    
    async def ensure_user_exists(self, telegram_id: int, username: str) -> User:
        """
        确保用户存在，不存在则创建
        
        Args:
            telegram_id: Telegram 用户 ID
            username: Telegram 用户名
            
        Returns:
            用户对象
        """
        user = await self.user_repo.get_user(telegram_id)
        
        if user is None:
            # 用户不存在，创建新用户
            user = await self.user_repo.create_user(telegram_id, username)
            # 记录初始化交易
            await self.tx_repo.log_transaction(
                user_id=telegram_id,
                amount=1000,
                transaction_type='init',
                description='账户初始化'
            )
        
        return user
    
    async def get_balance(self, telegram_id: int) -> int:
        """
        获取余额
        
        Args:
            telegram_id: Telegram 用户 ID
            
        Returns:
            用户余额
        """
        user = await self.user_repo.get_user(telegram_id)
        if user is None:
            return 0
        return user.balance
    
    async def claim_daily_reward(self, telegram_id: int) -> Tuple[bool, str]:
        """
        领取每日奖励（包含 24 小时检查）
        
        Args:
            telegram_id: Telegram 用户 ID
            
        Returns:
            (成功, 消息) 元组
        """
        user = await self.user_repo.get_user(telegram_id)
        
        if user is None:
            return False, "用户不存在"
        
        # 检查是否可以签到
        can_claim = await self.user_repo.can_claim_daily(telegram_id)
        
        if not can_claim:
            # 计算剩余等待时间
            now = int(time.time())
            time_passed = now - user.last_daily_claim
            time_remaining = 86400 - time_passed  # 24小时 = 86400秒
            
            hours = time_remaining // 3600
            minutes = (time_remaining % 3600) // 60
            
            return False, f"签到冷却中，还需等待 {hours} 小时 {minutes} 分钟"
        
        # 可以签到，增加 500 金币
        await self.user_repo.update_balance(telegram_id, 500)
        await self.user_repo.update_daily_claim(telegram_id)
        
        # 记录交易
        await self.tx_repo.log_transaction(
            user_id=telegram_id,
            amount=500,
            transaction_type='daily',
            description='每日签到奖励'
        )
        
        new_balance = user.balance + 500
        return True, f"签到成功！获得 500 金币，当前余额：{new_balance}"
    
    async def transfer(
        self,
        from_id: int,
        to_id: int,
        amount: int
    ) -> Tuple[bool, str]:
        """
        转账（包含 5% 手续费和验证）
        
        Args:
            from_id: 发送者 ID
            to_id: 接收者 ID
            amount: 转账金额
            
        Returns:
            (成功, 消息) 元组
        """
        # 验证：不能向自己转账
        if from_id == to_id:
            return False, "不能向自己转账"
        
        # 验证：金额必须为正数
        if amount <= 0:
            return False, "转账金额必须大于 0"
        
        # 获取发送者
        sender = await self.user_repo.get_user(from_id)
        if sender is None:
            return False, "发送者不存在"
        
        # 验证：余额是否充足
        if sender.balance < amount:
            return False, f"余额不足，当前余额：{sender.balance}"
        
        # 获取接收者
        receiver = await self.user_repo.get_user(to_id)
        if receiver is None:
            return False, "接收者不存在"
        
        # 计算手续费（5%）
        fee = int(amount * 0.05)
        actual_amount = amount - fee
        
        # 执行转账（使用事务确保原子性）
        async def deduct_from_sender(conn):
            await conn.execute(
                "UPDATE users SET balance = balance - ?, updated_at = ? WHERE telegram_id = ?",
                (amount, int(time.time()), from_id)
            )
        
        async def add_to_receiver(conn):
            await conn.execute(
                "UPDATE users SET balance = balance + ?, updated_at = ? WHERE telegram_id = ?",
                (actual_amount, int(time.time()), to_id)
            )
        
        async def log_sender_tx(conn):
            await conn.execute(
                """INSERT INTO transactions 
                   (user_id, amount, type, description, created_at) 
                   VALUES (?, ?, ?, ?, ?)""",
                (from_id, -amount, 'transfer_send', f'转账给 {receiver.username}', int(time.time()))
            )
        
        async def log_receiver_tx(conn):
            await conn.execute(
                """INSERT INTO transactions 
                   (user_id, amount, type, description, created_at) 
                   VALUES (?, ?, ?, ?, ?)""",
                (to_id, actual_amount, 'transfer_receive', f'收到来自 {sender.username} 的转账', int(time.time()))
            )
        
        async def log_fee_tx(conn):
            await conn.execute(
                """INSERT INTO transactions 
                   (user_id, amount, type, description, created_at) 
                   VALUES (?, ?, ?, ?, ?)""",
                (from_id, -fee, 'transfer_fee', '转账手续费', int(time.time()))
            )
        
        try:
            # 执行事务
            await self.user_repo.db.transaction([
                deduct_from_sender,
                add_to_receiver,
                log_sender_tx,
                log_receiver_tx,
                log_fee_tx
            ])
            
            new_sender_balance = sender.balance - amount
            return True, f"转账成功！转出 {amount} 金币（含手续费 {fee}），{receiver.username} 收到 {actual_amount} 金币。当前余额：{new_sender_balance}"
        
        except Exception as e:
            return False, f"转账失败：{str(e)}"
