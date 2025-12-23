"""
Telegram 游戏机器人命令处理器
实现基础命令、转账命令和游戏命令的处理
"""
import asyncio
import json
import logging
from functools import wraps
from typing import Optional
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import (
    Application,
    CommandHandler,
    CallbackQueryHandler,
    ContextTypes,
)
from telegram.constants import DiceEmoji

from src.database import DatabaseManager
from src.repositories import UserRepository, TransactionRepository
from src.account_manager import AccountManager
from src.game_engine import GameEngine
from src.blackjack import BlackjackManager
from src.sicbo_manager import SicBoManager
from src.sicbo_keyboard import SicBoKeyboardBuilder
from src.models import BetType, GamePhase
from src.concurrency import ConcurrencyManager, with_user_lock, with_game_session
from src.error_handler import (
    global_error_handler,
    ErrorMessages,
    CommandValidator,
    retry_telegram_api,
    RetryConfig,
)

# 配置日志
logging.basicConfig(
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    level=logging.INFO
)
logger = logging.getLogger(__name__)


def check_chat_allowed(func):
    """装饰器：检查群组是否在白名单中，或用户是否在白名单群组中使用过"""
    @wraps(func)
    async def wrapper(self, update: Update, context: ContextTypes.DEFAULT_TYPE, *args, **kwargs):
        chat = update.effective_chat
        user = update.effective_user
        
        if not chat:
            return await func(self, update, context, *args, **kwargs)
        
        # 私聊情况：检查用户是否在数据库中（说明在白名单群组用过）
        if chat.id > 0:
            if user:
                # 检查用户是否存在于数据库
                existing_user = await self.user_repo.get_user(user.id)
                if existing_user:
                    return await func(self, update, context, *args, **kwargs)
            # 用户不在数据库中，忽略
            logger.warning(f"User {user.id if user else 'unknown'} not registered, ignoring private chat")
            return
        
        # 群组情况：检查白名单
        if not self.is_chat_allowed(chat.id):
            logger.warning(f"Chat {chat.id} not in allowed list, ignoring command")
            return
        
        return await func(self, update, context, *args, **kwargs)
    return wrapper


class BotConfig:
    """Bot 配置类"""
    
    def __init__(self, config_path: str = "config/config.json"):
        """
        加载配置文件
        
        Args:
            config_path: 配置文件路径
        """
        with open(config_path, 'r') as f:
            config = json.load(f)
        
        self.bot_token: str = config['bot_token']
        self.database_path: str = config.get('database_path', 'data/bot.db')
        self.admin_ids: list[int] = config.get('admin_ids', [])
        self.allowed_chats: list[int] = config.get('allowed_chats', [])
    
    @classmethod
    def from_dict(cls, config: dict) -> 'BotConfig':
        """从字典创建配置对象（用于测试）"""
        instance = object.__new__(cls)
        instance.bot_token = config.get('bot_token', '')
        instance.database_path = config.get('database_path', 'data/bot.db')
        instance.admin_ids = config.get('admin_ids', [])
        instance.allowed_chats = config.get('allowed_chats', [])
        return instance


class BotHandlers:
    """Bot 命令处理器集合"""
    
    def __init__(
        self,
        account_manager: AccountManager,
        user_repo: UserRepository,
        tx_repo: TransactionRepository,
        game_engine: Optional[GameEngine] = None,
        blackjack_manager: Optional[BlackjackManager] = None,
        sicbo_manager: Optional[SicBoManager] = None,
        admin_ids: Optional[list[int]] = None,
        concurrency_manager: Optional[ConcurrencyManager] = None,
        allowed_chats: Optional[list[int]] = None
    ):
        """
        初始化处理器
        
        Args:
            account_manager: 账户管理器
            user_repo: 用户仓储
            tx_repo: 交易仓储
            game_engine: 游戏引擎（可选）
            blackjack_manager: 21点游戏管理器（可选）
            sicbo_manager: 骰宝游戏管理器（可选）
            admin_ids: 管理员 ID 列表（可选）
            concurrency_manager: 并发控制管理器（可选）
            allowed_chats: 允许使用的群组 ID 列表（可选，为空则不限制）
        """
        self.account_manager = account_manager
        self.user_repo = user_repo
        self.tx_repo = tx_repo
        self.game_engine = game_engine
        self.blackjack_manager = blackjack_manager
        self.sicbo_manager = sicbo_manager
        self.admin_ids = admin_ids or []
        self.concurrency_manager = concurrency_manager or ConcurrencyManager()
        self.allowed_chats = allowed_chats or []
        # 骰宝游戏定时器存储
        self._sicbo_timers: dict[int, asyncio.Task] = {}
    
    def is_admin(self, user_id: int) -> bool:
        """
        检查用户是否为管理员
        
        Args:
            user_id: 用户 ID
            
        Returns:
            是否为管理员
        """
        return user_id in self.admin_ids
    
    def is_chat_allowed(self, chat_id: int) -> bool:
        """
        检查群组是否在白名单中
        
        Args:
            chat_id: 群组 ID
            
        Returns:
            是否允许使用
        """
        # 如果白名单为空，允许所有群组
        if not self.allowed_chats:
            return True
        return chat_id in self.allowed_chats

    @check_chat_allowed
    @with_user_lock()
    async def start_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /start 命令
        初始化用户账户
        
        需求: 1.1, 1.2
        """
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        
        try:
            # 确保用户存在（不存在则创建）
            account = await self.account_manager.ensure_user_exists(telegram_id, username)
            
            await update.message.reply_text(
                f"🎮 欢迎来到游戏机器人！\n\n"
                f"👤 用户: {username}\n"
                f"💰 余额: {account.balance} 金币\n\n"
                f"📋 可用命令:\n"
                f"/balance - 查询余额\n"
                f"/daily - 每日签到\n"
                f"/top - 财富排行榜\n"
                f"/pay @用户 金额 - 转账\n"
                f"/dice 金额 - 骰子游戏\n"
                f"/slot 金额 - 老虎机游戏\n"
                f"/bj 金额 - 21点游戏"
            )
        except Exception as e:
            logger.error(f"start_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    @check_chat_allowed
    @with_user_lock()
    async def balance_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /balance 或 /my 命令
        查询用户余额
        
        需求: 1.3
        """
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        
        try:
            # 确保用户存在
            account = await self.account_manager.ensure_user_exists(telegram_id, username)
            
            await update.message.reply_text(
                f"💰 账户余额\n\n"
                f"👤 用户: {username}\n"
                f"💵 余额: {account.balance} 金币"
            )
        except Exception as e:
            logger.error(f"balance_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    @check_chat_allowed
    @with_user_lock()
    async def daily_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /daily 命令
        每日签到领取奖励
        
        需求: 2.1, 2.2, 2.3, 2.4, 2.5
        """
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        
        try:
            # 确保用户存在
            await self.account_manager.ensure_user_exists(telegram_id, username)
            
            # 尝试签到
            success, message = await self.account_manager.claim_daily_reward(telegram_id)
            
            if success:
                await update.message.reply_text(f"✅ {message}")
            else:
                await update.message.reply_text(f"⏰ {message}")
        except Exception as e:
            logger.error(f"daily_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    @check_chat_allowed
    @with_user_lock()
    async def top_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /top 命令
        显示财富排行榜
        
        需求: 4.1, 4.2, 4.3, 4.4
        """
        try:
            # 获取前 10 名用户
            top_users = await self.user_repo.get_top_users(limit=10)
            
            if not top_users:
                await update.message.reply_text("📊 排行榜暂无数据")
                return
            
            # 构建排行榜消息
            lines = ["🏆 财富排行榜 TOP 10\n"]
            
            medals = ["🥇", "🥈", "🥉"]
            
            for i, user in enumerate(top_users):
                rank = i + 1
                medal = medals[i] if i < 3 else f"{rank}."
                lines.append(f"{medal} {user.username}: {user.balance} 金币")
            
            await update.message.reply_text("\n".join(lines))
        except Exception as e:
            logger.error(f"top_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")


    @check_chat_allowed
    @with_user_lock()
    async def pay_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /pay 命令
        用户间转账
        
        用法: /pay @用户名 金额
        
        需求: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7
        """
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        
        # 解析参数
        args = context.args
        
        if not args or len(args) < 2:
            await update.message.reply_text(
                "❌ 命令格式错误\n\n"
                "用法: /pay @用户名 金额\n"
                "示例: /pay @zhangsan 100"
            )
            return
        
        # 解析目标用户
        target_input = args[0]
        
        # 解析金额
        try:
            amount = int(args[1])
        except ValueError:
            await update.message.reply_text(
                "❌ 无效的金额\n\n"
                "金额必须是正整数\n"
                "示例: /pay @zhangsan 100"
            )
            return
        
        # 验证金额
        if amount <= 0:
            await update.message.reply_text("❌ 转账金额必须大于 0")
            return
        
        try:
            # 确保发送者存在
            await self.account_manager.ensure_user_exists(telegram_id, username)
            
            # 尝试从回复消息获取目标用户
            target_user = None
            target_id = None
            
            # 检查是否回复了某条消息
            if update.message.reply_to_message and update.message.reply_to_message.from_user:
                target_user = update.message.reply_to_message.from_user
                target_id = target_user.id
            # 检查是否提及了用户（通过 entities）
            elif update.message.entities:
                for entity in update.message.entities:
                    if entity.type == "text_mention" and entity.user:
                        target_user = entity.user
                        target_id = entity.user.id
                        break
                    elif entity.type == "mention":
                        # @username 格式，需要从数据库查找
                        mention_text = update.message.text[entity.offset:entity.offset + entity.length]
                        target_username = mention_text.lstrip('@')
                        # 尝试从数据库查找用户
                        result = await self.user_repo.db.fetch_one(
                            "SELECT telegram_id FROM users WHERE username = ?",
                            (target_username,)
                        )
                        if result:
                            target_id = result['telegram_id']
                        break
            
            if target_id is None:
                await update.message.reply_text(
                    "❌ 找不到目标用户\n\n"
                    "请通过以下方式指定用户:\n"
                    "1. 回复目标用户的消息并使用 /pay 金额\n"
                    "2. 使用 /pay @用户名 金额"
                )
                return
            
            # 执行转账
            success, message = await self.account_manager.transfer(telegram_id, target_id, amount)
            
            if success:
                await update.message.reply_text(f"✅ {message}")
            else:
                await update.message.reply_text(f"❌ {message}")
                
        except Exception as e:
            logger.error(f"pay_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    @check_chat_allowed
    @with_game_session('dice')
    async def dice_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /dice 命令
        双骰子游戏：发送两个骰子，根据点数之和判断输赢
        
        用法: /dice 金额
        """
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        
        # 检查游戏引擎是否可用
        if self.game_engine is None:
            await update.message.reply_text("❌ 游戏功能暂不可用")
            return
        
        # 检查冷却时间
        cooldown_key = f"dice_cooldown_{telegram_id}"
        last_play = context.user_data.get(cooldown_key, 0)
        now = asyncio.get_event_loop().time()
        if now - last_play < 3:  # 3秒冷却
            remaining = int(3 - (now - last_play))
            await update.message.reply_text(f"⏳ 请等待 {remaining} 秒后再玩")
            return
        
        # 解析参数
        args = context.args
        
        if not args or len(args) < 1:
            await update.message.reply_text(
                "❌ 命令格式错误\n\n"
                "用法: /dice 金额\n"
                "示例: /dice 100\n\n"
                "规则（双骰子点数之和）:\n"
                "🎲 2-6: 输掉本金\n"
                "🎲 7: 平局，返还本金\n"
                "🎲 8-11: 赢得奖金\n"
                "🎲 12: 大奖！"
            )
            return
        
        # 解析金额
        try:
            bet = int(args[0])
        except ValueError:
            await update.message.reply_text(
                "❌ 无效的金额\n\n"
                "金额必须是正整数\n"
                "示例: /dice 100"
            )
            return
        
        # 验证金额
        if bet <= 0:
            await update.message.reply_text("❌ 下注金额必须大于 0")
            return
        
        # 验证最大下注金额
        if bet > 1000:
            await update.message.reply_text("❌ 骰子游戏最大下注金额为 1000")
            return
        
        try:
            # 确保用户存在并检查余额（加锁保护）
            await self.concurrency_manager.acquire_user_lock(telegram_id)
            try:
                await self.account_manager.ensure_user_exists(telegram_id, username)
                balance = await self.account_manager.get_balance(telegram_id)
                if balance < bet:
                    await update.message.reply_text(f"❌ 余额不足，当前余额：{balance}")
                    return
            finally:
                await self.concurrency_manager.release_user_lock(telegram_id)
            
            chat_id = update.effective_chat.id
            message_id = update.message.message_id
            
            # 发送第一个骰子
            dice_message1 = await context.bot.send_dice(
                chat_id=chat_id,
                emoji=DiceEmoji.DICE
            )
            dice_value1 = dice_message1.dice.value
            
            # 稍等一下再发第二个
            await asyncio.sleep(0.5)
            
            # 发送第二个骰子
            dice_message2 = await context.bot.send_dice(
                chat_id=chat_id,
                emoji=DiceEmoji.DICE
            )
            dice_value2 = dice_message2.dice.value
            
            # 等待动画完成
            await asyncio.sleep(2)
            
            # 记录冷却时间
            context.user_data[cooldown_key] = asyncio.get_event_loop().time()
            
            # 执行游戏逻辑（重新加锁）
            await self.concurrency_manager.acquire_user_lock(telegram_id)
            try:
                success, result_message, payout = await self.game_engine.play_dice(
                    telegram_id, bet, dice_value1, dice_value2
                )
            finally:
                await self.concurrency_manager.release_user_lock(telegram_id)
            
            # 发送结果
            text = f"@{username} {result_message}" if success else f"@{username} ❌ {result_message}"
            try:
                await context.bot.send_message(
                    chat_id=chat_id, 
                    text=text,
                    reply_to_message_id=message_id
                )
            except Exception:
                await context.bot.send_message(chat_id=chat_id, text=text)
                
        except Exception as e:
            logger.error(f"dice_handler error: {e}")
            try:
                await context.bot.send_message(
                    chat_id=update.effective_chat.id,
                    text="❌ 系统错误，请稍后再试"
                )
            except Exception:
                pass

    @check_chat_allowed
    @with_game_session('slot')
    async def slot_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /slot 命令
        老虎机游戏：使用 Telegram sendDice API 发送老虎机动画
        
        用法: /slot 金额
        """
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        
        # 检查游戏引擎是否可用
        if self.game_engine is None:
            await update.message.reply_text("❌ 游戏功能暂不可用")
            return
        
        # 检查冷却时间
        cooldown_key = f"slot_cooldown_{telegram_id}"
        last_play = context.user_data.get(cooldown_key, 0)
        now = asyncio.get_event_loop().time()
        if now - last_play < 5:  # 5秒冷却
            remaining = int(5 - (now - last_play))
            await update.message.reply_text(f"⏳ 请等待 {remaining} 秒后再玩")
            return
        
        # 解析参数
        args = context.args
        
        if not args or len(args) < 1:
            await update.message.reply_text(
                "❌ 命令格式错误\n\n"
                "用法: /slot 金额\n"
                "示例: /slot 100\n\n"
                "规则:\n"
                "🎰 三个图案一致: 大奖\n"
                "🎰 两个图案一致: 返还本金\n"
                "🎰 三个图案不一致: 输掉本金"
            )
            return
        
        # 解析金额
        try:
            bet = int(args[0])
        except ValueError:
            await update.message.reply_text(
                "❌ 无效的金额\n\n"
                "金额必须是正整数\n"
                "示例: /slot 100"
            )
            return
        
        # 验证金额
        if bet <= 0:
            await update.message.reply_text("❌ 下注金额必须大于 0")
            return
        
        try:
            # 确保用户存在并检查余额（加锁保护）
            await self.concurrency_manager.acquire_user_lock(telegram_id)
            try:
                await self.account_manager.ensure_user_exists(telegram_id, username)
                balance = await self.account_manager.get_balance(telegram_id)
                if balance < bet:
                    await update.message.reply_text(f"❌ 余额不足，当前余额：{balance}")
                    return
            finally:
                await self.concurrency_manager.release_user_lock(telegram_id)
            
            chat_id = update.effective_chat.id
            message_id = update.message.message_id
            
            # 发送老虎机动画（直接发送到聊天，不回复）
            slot_message = await context.bot.send_dice(
                chat_id=chat_id,
                emoji=DiceEmoji.SLOT_MACHINE
            )
            slot_value = slot_message.dice.value
            
            # 等待动画完成（不持有锁，允许其他用户操作）
            await asyncio.sleep(2)
            
            # 记录冷却时间
            context.user_data[cooldown_key] = asyncio.get_event_loop().time()
            
            # 执行游戏逻辑（重新加锁）
            await self.concurrency_manager.acquire_user_lock(telegram_id)
            try:
                success, result_message, payout = await self.game_engine.play_slot(
                    telegram_id, bet, slot_value
                )
            finally:
                await self.concurrency_manager.release_user_lock(telegram_id)
            
            # 尝试回复原消息，失败则直接发送
            # 加上用户名方便识别
            text = f"@{username} {result_message}" if success else f"@{username} ❌ {result_message}"
            try:
                await context.bot.send_message(
                    chat_id=chat_id,
                    text=text,
                    reply_to_message_id=message_id
                )
            except Exception:
                await context.bot.send_message(chat_id=chat_id, text=text)
                
        except Exception as e:
            logger.error(f"slot_handler error: {e}")
            try:
                await context.bot.send_message(
                    chat_id=update.effective_chat.id, 
                    text="❌ 系统错误，请稍后再试"
                )
            except Exception:
                pass

    def _create_blackjack_keyboard(self, can_double: bool = True) -> InlineKeyboardMarkup:
        """
        创建21点游戏的内联键盘
        
        Args:
            can_double: 是否可以加倍（只有首两张牌时可以）
            
        Returns:
            InlineKeyboardMarkup 对象
        """
        buttons = [
            [
                InlineKeyboardButton("🃏 要牌", callback_data="bj_hit"),
                InlineKeyboardButton("✋ 停牌", callback_data="bj_stand"),
            ]
        ]
        
        if can_double:
            buttons.append([
                InlineKeyboardButton("💰 加倍", callback_data="bj_double")
            ])
        
        return InlineKeyboardMarkup(buttons)

    @check_chat_allowed
    @with_user_lock()
    @with_game_session('blackjack')
    async def blackjack_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /bj 命令
        开始21点游戏，发送 Inline Keyboard
        
        用法: /bj 金额
        
        需求: 7.1, 7.2
        """
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        
        # 检查21点管理器是否可用
        if self.blackjack_manager is None:
            await update.message.reply_text("❌ 21点游戏功能暂不可用")
            return
        
        # 解析参数
        args = context.args
        
        if not args or len(args) < 1:
            await update.message.reply_text(
                "❌ 命令格式错误\n\n"
                "用法: /bj 金额\n"
                "示例: /bj 100\n\n"
                "规则:\n"
                "🃏 目标是让手牌点数尽量接近21点但不超过\n"
                "🃏 A可以算1点或11点\n"
                "🃏 J、Q、K都算10点\n"
                "🃏 Blackjack（首两张21点）赢1.5倍"
            )
            return
        
        # 解析金额
        try:
            bet = int(args[0])
        except ValueError:
            await update.message.reply_text(
                "❌ 无效的金额\n\n"
                "金额必须是正整数\n"
                "示例: /bj 100"
            )
            return
        
        # 验证金额
        if bet <= 0:
            await update.message.reply_text("❌ 下注金额必须大于 0")
            return
        
        try:
            # 确保用户存在
            await self.account_manager.ensure_user_exists(telegram_id, username)
            
            # 开始游戏
            success, message, game = await self.blackjack_manager.start_game(telegram_id, bet)
            
            if not success:
                # 游戏未能开始，结束会话
                await self.concurrency_manager.end_game(telegram_id)
                await update.message.reply_text(f"❌ {message}")
                return
            
            # 检查是否已经结束（Blackjack）
            if game and game.is_finished:
                # 游戏已结束（玩家或庄家 Blackjack），结束会话
                await self.concurrency_manager.end_game(telegram_id)
                await update.message.reply_text(message)
            else:
                # 游戏进行中，发送带按钮的消息
                keyboard = self._create_blackjack_keyboard(can_double=True)
                await update.message.reply_text(message, reply_markup=keyboard)
                
        except Exception as e:
            logger.error(f"blackjack_handler error: {e}")
            # 发生异常，结束会话
            await self.concurrency_manager.end_game(telegram_id)
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    @check_chat_allowed
    @with_user_lock()
    async def blackjack_callback_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理21点游戏的按钮回调
        
        回调数据:
        - bj_hit: 要牌
        - bj_stand: 停牌
        - bj_double: 加倍
        
        需求: 7.3, 7.4, 7.5
        """
        query = update.callback_query
        if not query:
            return
        
        await query.answer()
        
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        
        # 检查21点管理器是否可用
        if self.blackjack_manager is None:
            await query.edit_message_text("❌ 21点游戏功能暂不可用")
            return
        
        callback_data = query.data
        
        try:
            if callback_data == "bj_hit":
                # 要牌
                success, message, game = await self.blackjack_manager.hit(telegram_id)
                
                if not success:
                    await query.edit_message_text(f"❌ {message}")
                    return
                
                if game and game.is_finished:
                    # 游戏结束（爆牌），结束会话
                    await self.concurrency_manager.end_game(telegram_id)
                    await query.edit_message_text(message)
                else:
                    # 游戏继续，更新消息（要牌后不能加倍）
                    can_double = game and len(game.player_cards) == 2
                    keyboard = self._create_blackjack_keyboard(can_double=can_double)
                    await query.edit_message_text(message, reply_markup=keyboard)
                    
            elif callback_data == "bj_stand":
                # 停牌
                success, message, game, payout = await self.blackjack_manager.stand(telegram_id)
                
                if not success:
                    await query.edit_message_text(f"❌ {message}")
                    return
                
                # 游戏结束，结束会话并显示结果
                await self.concurrency_manager.end_game(telegram_id)
                await query.edit_message_text(message)
                
            elif callback_data == "bj_double":
                # 加倍
                success, message, game, payout = await self.blackjack_manager.double_down(telegram_id)
                
                if not success:
                    await query.edit_message_text(f"❌ {message}")
                    return
                
                # 游戏结束，结束会话并显示结果
                await self.concurrency_manager.end_game(telegram_id)
                await query.edit_message_text(message)
                
            else:
                await query.edit_message_text("❌ 未知的操作")
                
        except Exception as e:
            logger.error(f"blackjack_callback_handler error: {e}")
            # 发生异常，结束会话
            await self.concurrency_manager.end_game(telegram_id)
            await query.edit_message_text("❌ 系统错误，请稍后再试")

    async def _parse_admin_target_user(
        self, 
        update: Update, 
        args: list
    ) -> Optional[int]:
        """
        解析管理员命令的目标用户
        
        Args:
            update: Telegram Update 对象
            args: 命令参数列表
            
        Returns:
            目标用户 ID，如果无法解析返回 None
        """
        target_id = None
        
        # 检查是否回复了某条消息
        if update.message.reply_to_message and update.message.reply_to_message.from_user:
            target_id = update.message.reply_to_message.from_user.id
        # 检查是否提及了用户（通过 entities）
        elif update.message.entities:
            for entity in update.message.entities:
                if entity.type == "text_mention" and entity.user:
                    target_id = entity.user.id
                    break
                elif entity.type == "mention" and args:
                    # @username 格式，需要从数据库查找
                    mention_text = update.message.text[entity.offset:entity.offset + entity.length]
                    target_username = mention_text.lstrip('@')
                    # 尝试从数据库查找用户
                    result = await self.user_repo.db.fetch_one(
                        "SELECT telegram_id FROM users WHERE username = ?",
                        (target_username,)
                    )
                    if result:
                        target_id = result['telegram_id']
                    break
        
        return target_id

    @with_user_lock()
    async def admin_add_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /admin_add 命令
        管理员向指定用户添加金币
        
        用法: /admin_add @用户名 金额
        
        需求: 8.1, 8.3, 8.5
        """
        user = update.effective_user
        if not user:
            return
        
        admin_id = user.id
        admin_username = user.username or user.first_name or str(admin_id)
        
        # 权限检查
        if not self.is_admin(admin_id):
            await update.message.reply_text("❌ 权限不足，只有管理员可以执行此操作")
            return
        
        # 解析参数
        args = context.args
        
        if not args or len(args) < 2:
            await update.message.reply_text(
                "❌ 命令格式错误\n\n"
                "用法: /admin_add @用户名 金额\n"
                "示例: /admin_add @zhangsan 1000\n"
                "或回复用户消息: /admin_add 1000"
            )
            return
        
        # 解析金额（最后一个参数）
        try:
            amount = int(args[-1])
        except ValueError:
            await update.message.reply_text(
                "❌ 无效的金额\n\n"
                "金额必须是正整数\n"
                "示例: /admin_add @zhangsan 1000"
            )
            return
        
        # 验证金额
        if amount <= 0:
            await update.message.reply_text("❌ 添加金额必须大于 0")
            return
        
        try:
            # 解析目标用户
            target_id = await self._parse_admin_target_user(update, args)
            
            if target_id is None:
                await update.message.reply_text(
                    "❌ 找不到目标用户\n\n"
                    "请通过以下方式指定用户:\n"
                    "1. 回复目标用户的消息并使用 /admin_add 金额\n"
                    "2. 使用 /admin_add @用户名 金额"
                )
                return
            
            # 获取目标用户
            target_user = await self.user_repo.get_user(target_id)
            if target_user is None:
                await update.message.reply_text("❌ 目标用户不存在")
                return
            
            # 添加金币
            await self.user_repo.update_balance(target_id, amount)
            
            # 记录交易日志
            await self.tx_repo.log_transaction(
                user_id=target_id,
                amount=amount,
                transaction_type='admin_add',
                description=f'管理员 {admin_username} 添加金币'
            )
            
            # 获取新余额
            updated_user = await self.user_repo.get_user(target_id)
            new_balance = updated_user.balance if updated_user else target_user.balance + amount
            
            await update.message.reply_text(
                f"✅ 管理员操作成功\n\n"
                f"👤 目标用户: {target_user.username}\n"
                f"💰 添加金币: +{amount}\n"
                f"💵 当前余额: {new_balance}"
            )
            
            logger.info(f"Admin {admin_username}({admin_id}) added {amount} coins to {target_user.username}({target_id})")
            
        except Exception as e:
            logger.error(f"admin_add_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    @with_user_lock()
    async def admin_remove_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /admin_remove 命令
        管理员从指定用户扣除金币
        
        用法: /admin_remove @用户名 金额
        
        需求: 8.2, 8.3, 8.5
        """
        user = update.effective_user
        if not user:
            return
        
        admin_id = user.id
        admin_username = user.username or user.first_name or str(admin_id)
        
        # 权限检查
        if not self.is_admin(admin_id):
            await update.message.reply_text("❌ 权限不足，只有管理员可以执行此操作")
            return
        
        # 解析参数
        args = context.args
        
        if not args or len(args) < 2:
            await update.message.reply_text(
                "❌ 命令格式错误\n\n"
                "用法: /admin_remove @用户名 金额\n"
                "示例: /admin_remove @zhangsan 500\n"
                "或回复用户消息: /admin_remove 500"
            )
            return
        
        # 解析金额（最后一个参数）
        try:
            amount = int(args[-1])
        except ValueError:
            await update.message.reply_text(
                "❌ 无效的金额\n\n"
                "金额必须是正整数\n"
                "示例: /admin_remove @zhangsan 500"
            )
            return
        
        # 验证金额
        if amount <= 0:
            await update.message.reply_text("❌ 扣除金额必须大于 0")
            return
        
        try:
            # 解析目标用户
            target_id = await self._parse_admin_target_user(update, args)
            
            if target_id is None:
                await update.message.reply_text(
                    "❌ 找不到目标用户\n\n"
                    "请通过以下方式指定用户:\n"
                    "1. 回复目标用户的消息并使用 /admin_remove 金额\n"
                    "2. 使用 /admin_remove @用户名 金额"
                )
                return
            
            # 获取目标用户
            target_user = await self.user_repo.get_user(target_id)
            if target_user is None:
                await update.message.reply_text("❌ 目标用户不存在")
                return
            
            # 扣除金币（允许余额变为负数，由管理员决定）
            await self.user_repo.update_balance(target_id, -amount)
            
            # 记录交易日志
            await self.tx_repo.log_transaction(
                user_id=target_id,
                amount=-amount,
                transaction_type='admin_remove',
                description=f'管理员 {admin_username} 扣除金币'
            )
            
            # 获取新余额
            updated_user = await self.user_repo.get_user(target_id)
            new_balance = updated_user.balance if updated_user else target_user.balance - amount
            
            await update.message.reply_text(
                f"✅ 管理员操作成功\n\n"
                f"👤 目标用户: {target_user.username}\n"
                f"💰 扣除金币: -{amount}\n"
                f"💵 当前余额: {new_balance}"
            )
            
            logger.info(f"Admin {admin_username}({admin_id}) removed {amount} coins from {target_user.username}({target_id})")
            
        except Exception as e:
            logger.error(f"admin_remove_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    @with_user_lock()
    async def admin_reset_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /admin_reset 命令
        管理员重置指定用户账户为初始状态（1000 金币，签到时间重置）
        
        用法: /admin_reset @用户名
        
        需求: 8.4, 8.3, 8.5
        """
        user = update.effective_user
        if not user:
            return
        
        admin_id = user.id
        admin_username = user.username or user.first_name or str(admin_id)
        
        # 权限检查
        if not self.is_admin(admin_id):
            await update.message.reply_text("❌ 权限不足，只有管理员可以执行此操作")
            return
        
        # 解析参数
        args = context.args
        
        if not args and not (update.message.reply_to_message and update.message.reply_to_message.from_user):
            await update.message.reply_text(
                "❌ 命令格式错误\n\n"
                "用法: /admin_reset @用户名\n"
                "示例: /admin_reset @zhangsan\n"
                "或回复用户消息: /admin_reset"
            )
            return
        
        try:
            # 解析目标用户
            target_id = await self._parse_admin_target_user(update, args)
            
            if target_id is None:
                await update.message.reply_text(
                    "❌ 找不到目标用户\n\n"
                    "请通过以下方式指定用户:\n"
                    "1. 回复目标用户的消息并使用 /admin_reset\n"
                    "2. 使用 /admin_reset @用户名"
                )
                return
            
            # 获取目标用户
            target_user = await self.user_repo.get_user(target_id)
            if target_user is None:
                await update.message.reply_text("❌ 目标用户不存在")
                return
            
            old_balance = target_user.balance
            
            # 重置账户：设置余额为 1000，签到时间为 0
            import time
            now = int(time.time())
            
            await self.user_repo.db.execute(
                """UPDATE users 
                   SET balance = 1000, last_daily_claim = 0, updated_at = ? 
                   WHERE telegram_id = ?""",
                (now, target_id)
            )
            
            # 记录交易日志
            balance_change = 1000 - old_balance
            await self.tx_repo.log_transaction(
                user_id=target_id,
                amount=balance_change,
                transaction_type='admin_reset',
                description=f'管理员 {admin_username} 重置账户'
            )
            
            await update.message.reply_text(
                f"✅ 管理员操作成功\n\n"
                f"👤 目标用户: {target_user.username}\n"
                f"🔄 账户已重置\n"
                f"💰 原余额: {old_balance}\n"
                f"💵 新余额: 1000\n"
                f"⏰ 签到时间已重置"
            )
            
            logger.info(f"Admin {admin_username}({admin_id}) reset account of {target_user.username}({target_id})")
            
        except Exception as e:
            logger.error(f"admin_reset_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    # ============ 骰宝游戏命令处理器 ============
    
    @check_chat_allowed
    async def sicbo_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /sicbo 命令
        开始新的骰宝游戏，显示按钮下注面板
        
        需求: 1.1, 1.6
        """
        user = update.effective_user
        chat = update.effective_chat
        if not user or not chat:
            return
        
        # 检查骰宝管理器是否可用
        if self.sicbo_manager is None:
            await update.message.reply_text("❌ 骰宝游戏功能暂不可用")
            return
        
        chat_id = chat.id
        
        try:
            # 检查是否已有进行中的游戏
            existing_game = self.sicbo_manager.get_game(chat_id)
            if existing_game and existing_game.phase == GamePhase.BETTING:
                # 游戏已存在且在下注阶段，显示现有面板
                stats = self.sicbo_manager.get_game_stats(chat_id)
                panel_message = SicBoKeyboardBuilder.format_panel_message(
                    remaining_time=stats['remaining_time'],
                    player_count=stats['player_count'],
                    total_bet_amount=stats['total_bet_amount']
                )
                keyboard = SicBoKeyboardBuilder.build_main_panel()
                await update.message.reply_text(
                    text=panel_message,
                    reply_markup=keyboard
                )
                return
            
            # 尝试开始新游戏
            success, message = await self.sicbo_manager.start_game(chat_id)
            
            if success:
                # 获取游戏统计信息
                stats = self.sicbo_manager.get_game_stats(chat_id)
                
                # 构建面板消息和键盘
                panel_message = SicBoKeyboardBuilder.format_panel_message(
                    remaining_time=stats['remaining_time'],
                    player_count=stats['player_count'],
                    total_bet_amount=stats['total_bet_amount']
                )
                keyboard = SicBoKeyboardBuilder.build_main_panel()
                
                # 发送带按钮的面板消息
                sent_message = await update.message.reply_text(
                    text=panel_message,
                    reply_markup=keyboard
                )
                
                # 存储面板消息 ID 用于后续更新
                game = self.sicbo_manager.get_game(chat_id)
                if game:
                    game.panel_message_id = sent_message.message_id
                
                # 启动下注计时器（60秒后自动结束下注阶段）
                await self._start_sicbo_betting_timer(chat_id, context)
            else:
                await update.message.reply_text(f"❌ {message}")
                
        except Exception as e:
            logger.error(f"sicbo_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")
    
    async def _start_sicbo_betting_timer(self, chat_id: int, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        启动骰宝下注计时器
        
        Args:
            chat_id: 群组 ID
            context: Telegram 上下文
        """
        # 取消已有的计时器
        if chat_id in self._sicbo_timers:
            self._sicbo_timers[chat_id].cancel()
        
        # 创建新的计时器任务
        async def betting_timeout():
            await asyncio.sleep(60)  # 60秒下注时间
            await self._end_sicbo_betting_phase(chat_id, context)
        
        self._sicbo_timers[chat_id] = asyncio.create_task(betting_timeout())
    
    async def _cancel_sicbo_timer(self, chat_id: int) -> None:
        """
        取消骰宝下注计时器
        
        Args:
            chat_id: 群组 ID
        """
        if chat_id in self._sicbo_timers:
            self._sicbo_timers[chat_id].cancel()
            del self._sicbo_timers[chat_id]
    
    async def _end_sicbo_betting_phase(self, chat_id: int, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        结束骰宝下注阶段（超时自动触发）
        
        Args:
            chat_id: 群组 ID
            context: Telegram 上下文
        """
        if self.sicbo_manager is None:
            return
        
        game = self.sicbo_manager.get_game(chat_id)
        if not game or game.phase != GamePhase.BETTING:
            return
        
        try:
            # 发送下注结束提示
            await context.bot.send_message(
                chat_id=chat_id,
                text="⏰ 下注时间结束！正在开骰子..."
            )
            
            # 自动开骰子
            await self._do_roll_and_settle(chat_id, context)
            
        except Exception as e:
            logger.error(f"_end_sicbo_betting_phase error: {e}")
        finally:
            # 清理计时器
            if chat_id in self._sicbo_timers:
                del self._sicbo_timers[chat_id]
    
    async def _do_roll_and_settle(self, chat_id: int, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        执行开骰子和结算流程
        
        Args:
            chat_id: 群组 ID
            context: Telegram 上下文
        """
        if self.sicbo_manager is None:
            return
        
        # 发送三个骰子动画
        dice_values = []
        for i in range(3):
            dice_message = await context.bot.send_dice(
                chat_id=chat_id,
                emoji=DiceEmoji.DICE
            )
            dice_values.append(dice_message.dice.value)
            if i < 2:
                await asyncio.sleep(0.5)
        
        # 等待动画完成
        await asyncio.sleep(2)
        
        # 开骰子（使用 Telegram 返回的骰子值）
        success, dice_results, roll_message = await self.sicbo_manager.roll_dice(chat_id, dice_values)
        
        if not success:
            await context.bot.send_message(chat_id=chat_id, text=f"❌ {roll_message}")
            return
        
        # 发送骰子结果
        await context.bot.send_message(chat_id=chat_id, text=roll_message)
        
        # 结算游戏
        success, results, settle_message = await self.sicbo_manager.settle_game(chat_id)
        
        if success:
            await context.bot.send_message(chat_id=chat_id, text=settle_message)
        else:
            await context.bot.send_message(chat_id=chat_id, text=f"❌ {settle_message}")
    
    @check_chat_allowed
    @with_user_lock()
    async def bet_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /bet 命令
        骰宝游戏下注
        
        用法:
        - /bet single <数字> <金额>
        - /bet pair <数字1> <数字2> <金额>
        - /bet sum <总和> <金额>
        - /bet big <金额>
        - /bet small <金额>
        
        需求: 2.1, 3.1, 4.1, 5.1, 5.2, 7.1, 7.2, 7.3, 7.4, 7.5
        """
        user = update.effective_user
        chat = update.effective_chat
        if not user or not chat:
            return
        
        # 检查骰宝管理器是否可用
        if self.sicbo_manager is None:
            await update.message.reply_text("❌ 骰宝游戏功能暂不可用")
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        chat_id = chat.id
        
        # 解析参数
        args = context.args
        
        if not args or len(args) < 2:
            await update.message.reply_text(
                "❌ 命令格式错误\n\n"
                "用法:\n"
                "/bet single <数字> <金额> - 押单一数字\n"
                "/bet pair <数字1> <数字2> <金额> - 押两数组合\n"
                "/bet sum <总和> <金额> - 押总和\n"
                "/bet big <金额> - 押大\n"
                "/bet small <金额> - 押小\n\n"
                "示例: /bet single 3 100"
            )
            return
        
        bet_type_str = args[0].lower()
        
        try:
            # 确保用户存在
            await self.account_manager.ensure_user_exists(telegram_id, username)
            
            # 解析下注类型和参数
            bet_type = None
            numbers = []
            amount = 0
            
            if bet_type_str == "single":
                if len(args) < 3:
                    await update.message.reply_text(
                        "❌ 命令格式错误\n\n"
                        "用法: /bet single <数字> <金额>\n"
                        "示例: /bet single 3 100"
                    )
                    return
                bet_type = BetType.SINGLE
                try:
                    numbers = [int(args[1])]
                    amount = int(args[2])
                except ValueError:
                    await update.message.reply_text("❌ 数字和金额必须是整数")
                    return
                    
            elif bet_type_str == "pair":
                if len(args) < 4:
                    await update.message.reply_text(
                        "❌ 命令格式错误\n\n"
                        "用法: /bet pair <数字1> <数字2> <金额>\n"
                        "示例: /bet pair 3 5 100"
                    )
                    return
                bet_type = BetType.PAIR
                try:
                    numbers = [int(args[1]), int(args[2])]
                    amount = int(args[3])
                except ValueError:
                    await update.message.reply_text("❌ 数字和金额必须是整数")
                    return
                    
            elif bet_type_str == "sum":
                if len(args) < 3:
                    await update.message.reply_text(
                        "❌ 命令格式错误\n\n"
                        "用法: /bet sum <总和> <金额>\n"
                        "示例: /bet sum 10 100"
                    )
                    return
                bet_type = BetType.SUM
                try:
                    numbers = [int(args[1])]
                    amount = int(args[2])
                except ValueError:
                    await update.message.reply_text("❌ 总和和金额必须是整数")
                    return
                    
            elif bet_type_str == "big":
                if len(args) < 2:
                    await update.message.reply_text(
                        "❌ 命令格式错误\n\n"
                        "用法: /bet big <金额>\n"
                        "示例: /bet big 100"
                    )
                    return
                bet_type = BetType.BIG
                try:
                    amount = int(args[1])
                except ValueError:
                    await update.message.reply_text("❌ 金额必须是整数")
                    return
                    
            elif bet_type_str == "small":
                if len(args) < 2:
                    await update.message.reply_text(
                        "❌ 命令格式错误\n\n"
                        "用法: /bet small <金额>\n"
                        "示例: /bet small 100"
                    )
                    return
                bet_type = BetType.SMALL
                try:
                    amount = int(args[1])
                except ValueError:
                    await update.message.reply_text("❌ 金额必须是整数")
                    return
            else:
                await update.message.reply_text(
                    "❌ 未知的下注类型\n\n"
                    "可用类型: single, pair, sum, big, small"
                )
                return
            
            # 执行下注
            success, message = await self.sicbo_manager.place_bet(
                chat_id=chat_id,
                user_id=telegram_id,
                bet_type=bet_type,
                amount=amount,
                numbers=numbers,
                username=username
            )
            
            if success:
                await update.message.reply_text(f"✅ {message}")
            else:
                await update.message.reply_text(f"❌ {message}")
                
        except Exception as e:
            logger.error(f"bet_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")
    
    @check_chat_allowed
    async def roll_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /roll 命令
        开骰子（结束下注阶段）
        
        需求: 1.5, 6.1, 6.2, 6.3, 6.4, 6.5
        """
        user = update.effective_user
        chat = update.effective_chat
        if not user or not chat:
            return
        
        # 检查骰宝管理器是否可用
        if self.sicbo_manager is None:
            await update.message.reply_text("❌ 骰宝游戏功能暂不可用")
            return
        
        chat_id = chat.id
        
        # 检查是否有进行中的游戏
        game = self.sicbo_manager.get_game(chat_id)
        if not game:
            await update.message.reply_text("❌ 当前没有进行中的骰宝游戏")
            return
        
        # 检查游戏状态
        if game.phase != GamePhase.BETTING:
            await update.message.reply_text("❌ 当前不在下注阶段，无法开骰子")
            return
        
        try:
            # 取消下注计时器
            await self._cancel_sicbo_timer(chat_id)
            
            # 发送开骰子提示
            await update.message.reply_text("🎲 开骰子！")
            
            # 执行开骰子和结算
            await self._do_roll_and_settle(chat_id, context)
            
        except Exception as e:
            logger.error(f"roll_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")
    
    @check_chat_allowed
    async def sicbo_status_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /sicbo_status 命令
        查看当前骰宝游戏状态
        
        需求: 8.1, 8.2, 8.4
        """
        chat = update.effective_chat
        if not chat:
            return
        
        # 检查骰宝管理器是否可用
        if self.sicbo_manager is None:
            await update.message.reply_text("❌ 骰宝游戏功能暂不可用")
            return
        
        chat_id = chat.id
        
        try:
            stats = self.sicbo_manager.get_game_stats(chat_id)
            
            if not stats["exists"]:
                await update.message.reply_text("ℹ️ 当前没有进行中的骰宝游戏\n\n使用 /sicbo 开始新游戏")
                return
            
            # 构建状态消息
            phase_names = {
                "idle": "空闲",
                "betting": "下注中",
                "rolling": "开骰子中",
                "settling": "结算中"
            }
            phase_name = phase_names.get(stats["phase"], stats["phase"])
            
            msg = f"🎲 骰宝游戏状态\n"
            msg += f"━━━━━━━━━━━━━━━\n"
            msg += f"📊 状态: {phase_name}\n"
            msg += f"👥 参与人数: {stats['player_count']}\n"
            msg += f"💰 总下注: {stats['total_bet_amount']}\n"
            msg += f"📝 下注数: {stats['bet_count']}\n"
            
            if stats["phase"] == "betting" and stats["remaining_time"] > 0:
                msg += f"⏰ 剩余时间: {stats['remaining_time']} 秒"
            
            await update.message.reply_text(msg)
            
        except Exception as e:
            logger.error(f"sicbo_status_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")
    
    @check_chat_allowed
    async def mybets_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /mybets 命令
        查看我在当前游戏中的押注
        
        需求: 8.3
        """
        user = update.effective_user
        chat = update.effective_chat
        if not user or not chat:
            return
        
        # 检查骰宝管理器是否可用
        if self.sicbo_manager is None:
            await update.message.reply_text("❌ 骰宝游戏功能暂不可用")
            return
        
        telegram_id = user.id
        chat_id = chat.id
        
        try:
            # 检查是否有进行中的游戏
            game = self.sicbo_manager.get_game(chat_id)
            if not game:
                await update.message.reply_text("ℹ️ 当前没有进行中的骰宝游戏")
                return
            
            # 获取用户押注
            user_bets = self.sicbo_manager.get_user_bets(chat_id, telegram_id)
            
            if not user_bets:
                await update.message.reply_text("ℹ️ 您在当前游戏中没有押注")
                return
            
            # 构建押注列表消息
            msg = f"📋 您的押注\n"
            msg += f"━━━━━━━━━━━━━━━\n"
            
            total_amount = 0
            for i, bet in enumerate(user_bets, 1):
                bet_name = self.sicbo_manager._get_bet_type_name(bet.bet_type, bet.numbers)
                msg += f"{i}. {bet_name}: {bet.amount} 金币\n"
                total_amount += bet.amount
            
            msg += f"━━━━━━━━━━━━━━━\n"
            msg += f"💰 总计: {total_amount} 金币"
            
            await update.message.reply_text(msg)
            
        except Exception as e:
            logger.error(f"mybets_handler error: {e}")
            await update.message.reply_text("❌ 系统错误，请稍后再试")

    @check_chat_allowed
    async def sicbo_callback_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理骰宝游戏的按钮回调
        
        回调数据格式: sicbo_{action}_{param}
        
        需求: 8.1, 8.2
        """
        query = update.callback_query
        if not query:
            return
        
        user = update.effective_user
        chat = update.effective_chat
        if not user or not chat:
            await query.answer("❌ 无法获取用户信息")
            return
        
        # 检查骰宝管理器是否可用
        if self.sicbo_manager is None:
            await query.answer("❌ 骰宝游戏功能暂不可用")
            return
        
        telegram_id = user.id
        username = user.username or user.first_name or str(telegram_id)
        chat_id = chat.id
        callback_data = query.data
        
        try:
            # 解析回调数据
            action, param = SicBoKeyboardBuilder.decode_callback(callback_data)
            
            if not action:
                await query.answer("❌ 无效的回调数据")
                return
            
            # 检查游戏是否存在
            game = self.sicbo_manager.get_game(chat_id)
            if not game:
                await query.answer("❌ 当前没有进行中的游戏", show_alert=True)
                return
            
            # 分发到对应处理函数
            if action == "single":
                await self._handle_sicbo_single_bet(query, chat_id, telegram_id, username, param, context)
            elif action == "big":
                await self._handle_sicbo_big_bet(query, chat_id, telegram_id, username, context)
            elif action == "small":
                await self._handle_sicbo_small_bet(query, chat_id, telegram_id, username, context)
            elif action == "sum":
                await self._handle_sicbo_sum_bet(query, chat_id, telegram_id, username, param, context)
            elif action == "roll":
                await self._handle_sicbo_roll(query, chat_id, context)
            elif action == "mybets":
                await self._handle_sicbo_mybets(query, chat_id, telegram_id)
            else:
                await query.answer("❌ 未知的操作")
                
        except Exception as e:
            logger.error(f"sicbo_callback_handler error: {e}")
            await query.answer("❌ 系统错误，请稍后再试", show_alert=True)

    async def _handle_sicbo_single_bet(
        self,
        query,
        chat_id: int,
        user_id: int,
        username: str,
        param: str,
        context: ContextTypes.DEFAULT_TYPE
    ) -> None:
        """
        处理单一数字下注
        
        需求: 2.1, 3.2
        """
        # 验证游戏阶段
        game = self.sicbo_manager.get_game(chat_id)
        if not game or game.phase != GamePhase.BETTING:
            await query.answer("❌ 下注已结束", show_alert=True)
            return
        
        # 解析数字参数
        try:
            number = int(param)
        except ValueError:
            await query.answer("❌ 无效的数字")
            return
        
        # 确保用户存在
        await self.account_manager.ensure_user_exists(user_id, username)
        
        # 检查余额
        balance = await self.account_manager.get_balance(user_id)
        if balance < SicBoKeyboardBuilder.FIXED_BET_AMOUNT:
            await query.answer(f"❌ 余额不足，当前余额：{balance}", show_alert=True)
            return
        
        # 下注
        success, message = await self.sicbo_manager.place_bet(
            chat_id=chat_id,
            user_id=user_id,
            bet_type=BetType.SINGLE,
            amount=SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
            numbers=[number],
            username=username
        )
        
        if success:
            await query.answer(f"✅ 押注数字 {number}，金额 {SicBoKeyboardBuilder.FIXED_BET_AMOUNT}")
            # 更新面板
            await self._update_sicbo_panel(chat_id, context)
        else:
            await query.answer(f"❌ {message}", show_alert=True)

    async def _handle_sicbo_big_bet(
        self,
        query,
        chat_id: int,
        user_id: int,
        username: str,
        context: ContextTypes.DEFAULT_TYPE
    ) -> None:
        """
        处理大下注
        
        需求: 4.2
        """
        # 验证游戏阶段
        game = self.sicbo_manager.get_game(chat_id)
        if not game or game.phase != GamePhase.BETTING:
            await query.answer("❌ 下注已结束", show_alert=True)
            return
        
        # 确保用户存在
        await self.account_manager.ensure_user_exists(user_id, username)
        
        # 检查余额
        balance = await self.account_manager.get_balance(user_id)
        if balance < SicBoKeyboardBuilder.FIXED_BET_AMOUNT:
            await query.answer(f"❌ 余额不足，当前余额：{balance}", show_alert=True)
            return
        
        # 下注
        success, message = await self.sicbo_manager.place_bet(
            chat_id=chat_id,
            user_id=user_id,
            bet_type=BetType.BIG,
            amount=SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
            numbers=[],
            username=username
        )
        
        if success:
            await query.answer(f"✅ 押注大，金额 {SicBoKeyboardBuilder.FIXED_BET_AMOUNT}")
            # 更新面板
            await self._update_sicbo_panel(chat_id, context)
        else:
            await query.answer(f"❌ {message}", show_alert=True)

    async def _handle_sicbo_small_bet(
        self,
        query,
        chat_id: int,
        user_id: int,
        username: str,
        context: ContextTypes.DEFAULT_TYPE
    ) -> None:
        """
        处理小下注
        
        需求: 4.3
        """
        # 验证游戏阶段
        game = self.sicbo_manager.get_game(chat_id)
        if not game or game.phase != GamePhase.BETTING:
            await query.answer("❌ 下注已结束", show_alert=True)
            return
        
        # 确保用户存在
        await self.account_manager.ensure_user_exists(user_id, username)
        
        # 检查余额
        balance = await self.account_manager.get_balance(user_id)
        if balance < SicBoKeyboardBuilder.FIXED_BET_AMOUNT:
            await query.answer(f"❌ 余额不足，当前余额：{balance}", show_alert=True)
            return
        
        # 下注
        success, message = await self.sicbo_manager.place_bet(
            chat_id=chat_id,
            user_id=user_id,
            bet_type=BetType.SMALL,
            amount=SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
            numbers=[],
            username=username
        )
        
        if success:
            await query.answer(f"✅ 押注小，金额 {SicBoKeyboardBuilder.FIXED_BET_AMOUNT}")
            # 更新面板
            await self._update_sicbo_panel(chat_id, context)
        else:
            await query.answer(f"❌ {message}", show_alert=True)

    async def _handle_sicbo_sum_bet(
        self,
        query,
        chat_id: int,
        user_id: int,
        username: str,
        param: str,
        context: ContextTypes.DEFAULT_TYPE
    ) -> None:
        """
        处理总和下注
        
        需求: 5.3
        """
        # 验证游戏阶段
        game = self.sicbo_manager.get_game(chat_id)
        if not game or game.phase != GamePhase.BETTING:
            await query.answer("❌ 下注已结束", show_alert=True)
            return
        
        # 解析总和参数
        try:
            sum_value = int(param)
        except ValueError:
            await query.answer("❌ 无效的总和值")
            return
        
        # 确保用户存在
        await self.account_manager.ensure_user_exists(user_id, username)
        
        # 检查余额
        balance = await self.account_manager.get_balance(user_id)
        if balance < SicBoKeyboardBuilder.FIXED_BET_AMOUNT:
            await query.answer(f"❌ 余额不足，当前余额：{balance}", show_alert=True)
            return
        
        # 下注
        success, message = await self.sicbo_manager.place_bet(
            chat_id=chat_id,
            user_id=user_id,
            bet_type=BetType.SUM,
            amount=SicBoKeyboardBuilder.FIXED_BET_AMOUNT,
            numbers=[sum_value],
            username=username
        )
        
        if success:
            await query.answer(f"✅ 押注总和 {sum_value}，金额 {SicBoKeyboardBuilder.FIXED_BET_AMOUNT}")
            # 更新面板
            await self._update_sicbo_panel(chat_id, context)
        else:
            await query.answer(f"❌ {message}", show_alert=True)

    async def _handle_sicbo_roll(
        self,
        query,
        chat_id: int,
        context: ContextTypes.DEFAULT_TYPE
    ) -> None:
        """
        处理开骰子按钮
        
        需求: 7.2
        """
        # 验证游戏阶段
        game = self.sicbo_manager.get_game(chat_id)
        if not game or game.phase != GamePhase.BETTING:
            await query.answer("❌ 当前不在下注阶段", show_alert=True)
            return
        
        await query.answer("🎲 开骰子！")
        
        # 取消下注计时器
        await self._cancel_sicbo_timer(chat_id)
        
        # 发送开骰子提示
        await context.bot.send_message(chat_id=chat_id, text="🎲 开骰子！")
        
        # 执行开骰子和结算
        await self._do_roll_and_settle(chat_id, context)

    async def _handle_sicbo_mybets(
        self,
        query,
        chat_id: int,
        user_id: int
    ) -> None:
        """
        处理我的押注按钮
        
        需求: 6.4
        """
        # 获取用户押注
        user_bets = self.sicbo_manager.get_user_bets(chat_id, user_id)
        
        # 格式化押注信息
        bets_text = SicBoKeyboardBuilder.format_my_bets(user_bets)
        
        # 显示弹窗
        await query.answer(bets_text, show_alert=True)

    async def _update_sicbo_panel(self, chat_id: int, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        更新骰宝面板消息
        
        需求: 2.5, 6.3
        """
        game = self.sicbo_manager.get_game(chat_id)
        if not game or not game.panel_message_id:
            return
        
        try:
            # 获取游戏统计信息
            stats = self.sicbo_manager.get_game_stats(chat_id)
            
            # 构建新的面板消息
            panel_message = SicBoKeyboardBuilder.format_panel_message(
                remaining_time=stats['remaining_time'],
                player_count=stats['player_count'],
                total_bet_amount=stats['total_bet_amount']
            )
            keyboard = SicBoKeyboardBuilder.build_main_panel()
            
            # 更新消息
            await context.bot.edit_message_text(
                chat_id=chat_id,
                message_id=game.panel_message_id,
                text=panel_message,
                reply_markup=keyboard
            )
        except Exception as e:
            # 消息更新失败（可能消息已被删除），忽略错误
            logger.debug(f"Failed to update sicbo panel: {e}")

    async def cancel_handler(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """
        处理 /cancel 命令
        强制结束当前进行中的游戏会话
        """
        user = update.effective_user
        if not user:
            return
        
        telegram_id = user.id
        
        # 检查是否有进行中的游戏
        if not self.concurrency_manager.has_active_game(telegram_id):
            await update.message.reply_text("ℹ️ 您当前没有进行中的游戏")
            return
        
        # 获取当前游戏类型
        game_type = self.concurrency_manager.game_sessions.get_active_game(telegram_id)
        
        # 结束游戏会话
        await self.concurrency_manager.end_game(telegram_id)
        
        # 如果是21点游戏，清理游戏状态
        if game_type == 'blackjack' and self.blackjack_manager:
            if telegram_id in self.blackjack_manager.active_games:
                del self.blackjack_manager.active_games[telegram_id]
        
        await update.message.reply_text(f"✅ 已取消 {game_type} 游戏，下注金额不退还")


def create_bot_application(config: BotConfig, handlers: BotHandlers) -> Application:
    """
    创建 Bot 应用实例
    
    Args:
        config: Bot 配置
        handlers: 命令处理器
        
    Returns:
        Application 实例
    """
    # 配置并发处理和连接池
    from telegram.ext import Defaults
    from telegram.request import HTTPXRequest
    
    # 自定义请求配置，增加连接池
    request = HTTPXRequest(
        connection_pool_size=100,  # 连接池大小
        read_timeout=10.0,
        write_timeout=10.0,
        connect_timeout=10.0,
    )
    
    application = (
        Application.builder()
        .token(config.bot_token)
        .concurrent_updates(True)  # 启用并发处理
        .request(request)
        .build()
    )
    
    # 注册基础命令处理器
    application.add_handler(CommandHandler("start", handlers.start_handler))
    application.add_handler(CommandHandler("balance", handlers.balance_handler))
    application.add_handler(CommandHandler("my", handlers.balance_handler))
    application.add_handler(CommandHandler("daily", handlers.daily_handler))
    application.add_handler(CommandHandler("top", handlers.top_handler))
    application.add_handler(CommandHandler("pay", handlers.pay_handler))
    
    # 注册游戏命令处理器
    application.add_handler(CommandHandler("dice", handlers.dice_handler))
    application.add_handler(CommandHandler("slot", handlers.slot_handler))
    application.add_handler(CommandHandler("bj", handlers.blackjack_handler))
    application.add_handler(CommandHandler("cancel", handlers.cancel_handler))
    
    # 注册21点回调处理器
    application.add_handler(CallbackQueryHandler(
        handlers.blackjack_callback_handler,
        pattern="^bj_"
    ))
    
    # 注册骰宝回调处理器
    application.add_handler(CallbackQueryHandler(
        handlers.sicbo_callback_handler,
        pattern="^sicbo_"
    ))
    
    # 注册骰宝游戏命令处理器
    application.add_handler(CommandHandler("sicbo", handlers.sicbo_handler))
    application.add_handler(CommandHandler("bet", handlers.bet_handler))
    application.add_handler(CommandHandler("roll", handlers.roll_handler))
    application.add_handler(CommandHandler("sicbo_status", handlers.sicbo_status_handler))
    application.add_handler(CommandHandler("mybets", handlers.mybets_handler))
    
    # 注册管理员命令处理器
    application.add_handler(CommandHandler("admin_add", handlers.admin_add_handler))
    application.add_handler(CommandHandler("admin_remove", handlers.admin_remove_handler))
    application.add_handler(CommandHandler("admin_reset", handlers.admin_reset_handler))
    
    # 注册全局错误处理器
    application.add_error_handler(global_error_handler)
    
    return application
