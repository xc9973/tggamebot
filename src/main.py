"""
Telegram 游戏机器人主程序入口
实现 Bot 初始化、启动逻辑和优雅关闭
"""
import asyncio
import logging
import signal
import sys
import os
from pathlib import Path

from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository
from src.account_manager import AccountManager
from src.game_engine import GameEngine
from src.blackjack import BlackjackManager
from src.sicbo_manager import SicBoManager
from src.concurrency import ConcurrencyManager
from src.bot import BotConfig, BotHandlers, create_bot_application

# 配置日志
logging.basicConfig(
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    level=logging.INFO
)
logger = logging.getLogger(__name__)


class BotApplication:
    """Bot 应用程序类，管理生命周期"""
    
    def __init__(self, config_path: str = "config/config.json"):
        """
        初始化 Bot 应用
        
        Args:
            config_path: 配置文件路径
        """
        self.config_path = config_path
        self.config = None
        self.db = None
        self.application = None
        self._shutdown_event = asyncio.Event()
    
    async def initialize(self) -> None:
        """初始化所有组件"""
        logger.info("正在初始化 Bot...")
        
        # 加载配置
        self.config = BotConfig(self.config_path)
        logger.info(f"配置加载完成，数据库路径: {self.config.database_path}")
        
        # 确保数据目录存在
        db_dir = Path(self.config.database_path).parent
        db_dir.mkdir(parents=True, exist_ok=True)
        
        # 初始化数据库
        self.db = DatabaseManager(self.config.database_path)
        await self.db.initialize()
        logger.info("数据库初始化完成")
        
        # 初始化仓储层
        user_repo = UserRepository(self.db)
        tx_repo = TransactionRepository(self.db)
        
        # 初始化业务层
        account_manager = AccountManager(user_repo, tx_repo)
        game_engine = GameEngine(account_manager, tx_repo)
        blackjack_manager = BlackjackManager(account_manager, tx_repo)
        sicbo_manager = SicBoManager(account_manager, tx_repo)
        concurrency_manager = ConcurrencyManager()
        
        # 初始化处理器
        handlers = BotHandlers(
            account_manager=account_manager,
            user_repo=user_repo,
            tx_repo=tx_repo,
            game_engine=game_engine,
            blackjack_manager=blackjack_manager,
            sicbo_manager=sicbo_manager,
            admin_ids=self.config.admin_ids,
            concurrency_manager=concurrency_manager,
            allowed_chats=self.config.allowed_chats
        )
        
        # 创建 Bot 应用
        self.application = create_bot_application(self.config, handlers)
        logger.info("Bot 应用创建完成")
    
    async def start(self) -> None:
        """启动 Bot"""
        if self.application is None:
            await self.initialize()
        
        logger.info("正在启动 Bot...")
        
        # 初始化应用
        await self.application.initialize()
        
        # 启动轮询（优化并发处理）
        await self.application.start()
        await self.application.updater.start_polling(
            drop_pending_updates=True,
            allowed_updates=["message", "callback_query"],  # 只接收需要的更新类型
            pool_timeout=1.0,  # 减少轮询超时
            read_timeout=5.0,  # 读取超时
            write_timeout=5.0,  # 写入超时
            connect_timeout=5.0,  # 连接超时
        )
        
        logger.info("Bot 已启动，正在监听消息...")
        
        # 等待关闭信号
        await self._shutdown_event.wait()
    
    async def stop(self) -> None:
        """停止 Bot 并清理资源"""
        logger.info("正在关闭 Bot...")
        
        if self.application:
            # 停止轮询
            if self.application.updater and self.application.updater.running:
                await self.application.updater.stop()
            
            # 停止应用
            if self.application.running:
                await self.application.stop()
            
            # 关闭应用
            await self.application.shutdown()
        
        # 关闭数据库连接
        if self.db:
            await self.db.close()
            logger.info("数据库连接已关闭")
        
        logger.info("Bot 已关闭")
    
    def request_shutdown(self) -> None:
        """请求关闭"""
        self._shutdown_event.set()


async def main() -> None:
    """主函数"""
    # 确定配置文件路径
    config_path = os.environ.get("BOT_CONFIG_PATH", "config/config.json")
    
    # 检查配置文件是否存在
    if not Path(config_path).exists():
        logger.error(f"配置文件不存在: {config_path}")
        logger.error("请复制 config/config.example.json 到 config/config.json 并填写配置")
        sys.exit(1)
    
    # 创建应用实例
    app = BotApplication(config_path)
    
    # 设置信号处理
    loop = asyncio.get_running_loop()
    
    def signal_handler():
        logger.info("收到关闭信号")
        app.request_shutdown()
    
    # 注册信号处理器（仅在 Unix 系统上）
    if sys.platform != "win32":
        for sig in (signal.SIGINT, signal.SIGTERM):
            loop.add_signal_handler(sig, signal_handler)
    
    try:
        await app.initialize()
        await app.start()
    except KeyboardInterrupt:
        logger.info("收到键盘中断")
    except Exception as e:
        logger.error(f"Bot 运行出错: {e}", exc_info=True)
    finally:
        await app.stop()


if __name__ == "__main__":
    asyncio.run(main())
