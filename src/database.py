"""
数据库管理模块
提供 SQLite 数据库连接池、初始化和事务管理功能
"""
import aiosqlite
import asyncio
from typing import Optional, Any, Callable, List
from datetime import datetime
from contextlib import asynccontextmanager


class DatabaseManager:
    """数据库管理器，使用连接池支持高并发"""
    
    def __init__(self, db_path: str, pool_size: int = 50):
        """
        初始化数据库管理器
        
        Args:
            db_path: 数据库文件路径
            pool_size: 连接池大小
        """
        self.db_path = db_path
        self.pool_size = pool_size
        self._pool: asyncio.Queue = asyncio.Queue(maxsize=pool_size)
        self._initialized = False
        self._init_lock = asyncio.Lock()
        # 保留单连接用于向后兼容
        self._connection: Optional[aiosqlite.Connection] = None
        self._lock = asyncio.Lock()
    
    async def _create_connection(self) -> aiosqlite.Connection:
        """创建单个数据库连接"""
        conn = await aiosqlite.connect(self.db_path)
        conn.row_factory = aiosqlite.Row
        # 优化 SQLite 性能
        await conn.execute("PRAGMA journal_mode=WAL")
        await conn.execute("PRAGMA synchronous=NORMAL")
        await conn.execute("PRAGMA cache_size=10000")
        await conn.execute("PRAGMA temp_store=MEMORY")
        return conn
    
    async def connect(self) -> None:
        """初始化连接池"""
        async with self._init_lock:
            if self._initialized:
                return
            # 创建连接池
            for _ in range(self.pool_size):
                conn = await self._create_connection()
                await self._pool.put(conn)
            # 保留一个主连接用于初始化
            self._connection = await self._create_connection()
            self._initialized = True
    
    async def close(self) -> None:
        """关闭所有连接"""
        if self._connection:
            await self._connection.close()
            self._connection = None
        # 关闭连接池中的所有连接
        while not self._pool.empty():
            conn = await self._pool.get()
            await conn.close()
        self._initialized = False
    
    @asynccontextmanager
    async def get_connection(self):
        """从连接池获取连接"""
        conn = await self._pool.get()
        try:
            yield conn
        finally:
            await self._pool.put(conn)
    
    async def initialize(self) -> None:
        """
        创建表结构，启用 WAL 模式
        WAL (Write-Ahead Logging) 模式允许并发读写
        """
        await self.connect()
        
        # 创建 users 表
        await self._connection.execute("""
            CREATE TABLE IF NOT EXISTS users (
                telegram_id INTEGER PRIMARY KEY,
                username TEXT NOT NULL,
                balance INTEGER NOT NULL DEFAULT 1000,
                last_daily_claim INTEGER DEFAULT 0,
                created_at INTEGER NOT NULL,
                updated_at INTEGER NOT NULL
            )
        """)
        
        # 创建余额索引（用于排行榜）
        await self._connection.execute("""
            CREATE INDEX IF NOT EXISTS idx_balance 
            ON users(balance DESC, created_at ASC)
        """)
        
        # 创建 transactions 表
        await self._connection.execute("""
            CREATE TABLE IF NOT EXISTS transactions (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                user_id INTEGER NOT NULL,
                amount INTEGER NOT NULL,
                type TEXT NOT NULL,
                description TEXT,
                created_at INTEGER NOT NULL,
                FOREIGN KEY (user_id) REFERENCES users(telegram_id)
            )
        """)
        
        # 创建交易索引
        await self._connection.execute("""
            CREATE INDEX IF NOT EXISTS idx_user_transactions 
            ON transactions(user_id, created_at DESC)
        """)
        
        await self._connection.commit()
    
    async def execute(self, query: str, params: tuple = ()) -> None:
        """
        执行单个查询（INSERT, UPDATE, DELETE）
        
        Args:
            query: SQL 查询语句
            params: 查询参数
        """
        async with self.get_connection() as conn:
            await conn.execute(query, params)
            await conn.commit()
    
    async def fetch_one(self, query: str, params: tuple = ()) -> Optional[dict]:
        """
        查询单行数据
        
        Args:
            query: SQL 查询语句
            params: 查询参数
            
        Returns:
            查询结果字典，如果没有结果返回 None
        """
        async with self.get_connection() as conn:
            cursor = await conn.execute(query, params)
            row = await cursor.fetchone()
            if row:
                return dict(row)
            return None
    
    async def fetch_all(self, query: str, params: tuple = ()) -> List[dict]:
        """
        查询多行数据
        
        Args:
            query: SQL 查询语句
            params: 查询参数
            
        Returns:
            查询结果列表
        """
        async with self.get_connection() as conn:
            cursor = await conn.execute(query, params)
            rows = await cursor.fetchall()
            return [dict(row) for row in rows]
    
    async def transaction(self, operations: List[Callable]) -> bool:
        """
        执行事务，全部成功或全部回滚
        
        Args:
            operations: 操作函数列表，每个函数应该是 async 函数
            
        Returns:
            事务是否成功
        """
        async with self.get_connection() as conn:
            try:
                # 开始事务
                await conn.execute("BEGIN IMMEDIATE")
                
                # 执行所有操作
                for operation in operations:
                    await operation(conn)
                
                # 提交事务
                await conn.commit()
                return True
            except Exception as e:
                # 回滚事务
                await conn.rollback()
                raise e
    
    async def execute_in_transaction(self, query: str, params: tuple = ()) -> None:
        """
        在当前事务中执行查询（不自动提交）
        用于 transaction() 方法中的操作
        
        Args:
            query: SQL 查询语句
            params: 查询参数
        """
        await self._connection.execute(query, params)
