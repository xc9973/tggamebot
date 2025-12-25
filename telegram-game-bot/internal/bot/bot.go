// Package bot provides the Telegram bot initialization and handler registration.
// Requirements: 7.3 - Load whitelist from configuration file
package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/config"
	"telegram-game-bot/internal/game"
	"telegram-game-bot/internal/game/rob"
	"telegram-game-bot/internal/game/sicbo"
	"telegram-game-bot/internal/handler"
	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/service"
)

// Bot wraps the telebot instance with application dependencies.
type Bot struct {
	bot             *tele.Bot
	cfg             *config.Config
	accountService  *service.AccountService
	transferService *service.TransferService
	rankingService  *service.RankingService
	shopService     *service.ShopService
	gameRegistry    *game.Registry
	sicboGame       *sicbo.SicBoGame
	robGame         *rob.RobGame
	userLock        *lock.UserLock

	// Handlers
	accountHandler  *handler.AccountHandler
	transferHandler *handler.TransferHandler
	adminHandler    *handler.AdminHandler
	rankingHandler  *handler.RankingHandler
	gameHandler     *handler.GameHandler
	shopHandler     *handler.ShopHandler
}

// Dependencies holds all the dependencies needed by the bot handlers.
type Dependencies struct {
	Config          *config.Config
	AccountService  *service.AccountService
	TransferService *service.TransferService
	RankingService  *service.RankingService
	ShopService     *service.ShopService
	GameRegistry    *game.Registry
	SicBoGame       *sicbo.SicBoGame
	RobGame         *rob.RobGame
	UserLock        *lock.UserLock
}

// New creates a new Bot instance with the given dependencies.
// Requirements: 7.3
func New(deps *Dependencies) (*Bot, error) {
	if deps.Config.Bot.Token == "" {
		return nil, fmt.Errorf("bot token is required")
	}

	pref := tele.Settings{
		Token:  deps.Config.Bot.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	teleBot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	b := &Bot{
		bot:             teleBot,
		cfg:             deps.Config,
		accountService:  deps.AccountService,
		transferService: deps.TransferService,
		rankingService:  deps.RankingService,
		shopService:     deps.ShopService,
		gameRegistry:    deps.GameRegistry,
		sicboGame:       deps.SicBoGame,
		robGame:         deps.RobGame,
		userLock:        deps.UserLock,
	}

	// Initialize handlers
	b.accountHandler = handler.NewAccountHandler(deps.AccountService, deps.RankingService, deps.UserLock)
	b.transferHandler = handler.NewTransferHandler(deps.AccountService, deps.TransferService, deps.UserLock)
	b.adminHandler = handler.NewAdminHandler(deps.AccountService, deps.UserLock)
	b.rankingHandler = handler.NewRankingHandler(deps.RankingService)
	b.gameHandler = handler.NewGameHandler(deps.Config, deps.AccountService, deps.GameRegistry, deps.SicBoGame, deps.RobGame, deps.UserLock)
	b.shopHandler = handler.NewShopHandler(deps.ShopService, deps.AccountService)

	// Register middleware
	b.registerMiddleware()

	// Register handlers
	b.registerHandlers()

	return b, nil
}

// registerMiddleware registers all middleware.
func (b *Bot) registerMiddleware() {
	// Whitelist middleware - check if chat is allowed
	b.bot.Use(WhitelistMiddleware(b.cfg))

	// Logging middleware
	b.bot.Use(LoggingMiddleware())
}

// registerHandlers registers all command and callback handlers.
func (b *Bot) registerHandlers() {
	// Account handlers
	b.bot.Handle("/start", b.handleStart) // Custom handler to route private/group
	b.bot.Handle("/balance", b.accountHandler.HandleBalance)
	b.bot.Handle("/my", b.accountHandler.HandleMy)
	b.bot.Handle("/daily", b.accountHandler.HandleDaily)
	b.bot.Handle("/top", b.accountHandler.HandleTop)

	// Transfer handler
	b.bot.Handle("/pay", b.transferHandler.HandlePay)

	// Admin handlers (with admin middleware)
	adminGroup := b.bot.Group()
	adminGroup.Use(AdminMiddleware(b.cfg))
	adminGroup.Handle("/admin_add", b.adminHandler.HandleAdminAdd)
	adminGroup.Handle("/admin_sub", b.adminHandler.HandleAdminSub)
	adminGroup.Handle("/admin_set", b.adminHandler.HandleAdminSet)
	adminGroup.Handle("/admin_gift_all", b.adminHandler.HandleAdminGiftAll)

	// Ranking handler
	b.bot.Handle("/daily_top", b.rankingHandler.HandleDailyTop)

	// Game handlers
	b.bot.Handle("/dice", b.gameHandler.HandleDice)
	b.bot.Handle("/slot", b.gameHandler.HandleSlot)

	// SicBo handlers
	b.bot.Handle("/sicbo", b.gameHandler.HandleSicBoStart)
	b.bot.Handle("/sicbo_settle", b.gameHandler.HandleSicBoSettle)
	b.bot.Handle("/mybets", b.gameHandler.HandleMyBets)

	// Rob game handler
	b.bot.Handle("/dajie", b.gameHandler.HandleDajie)

	// Shop handlers
	b.bot.Handle("/bag", b.shopHandler.HandleBag)
	b.bot.Handle("/handcuff", b.shopHandler.HandleHandcuff)

	// Generic callback handler for sicbo and shop buttons
	b.bot.Handle(tele.OnCallback, b.handleCallback)
}

// handleStart routes /start to shop (private) or account (group)
func (b *Bot) handleStart(c tele.Context) error {
	chat := c.Chat()
	if chat != nil && chat.Type == tele.ChatPrivate {
		return b.shopHandler.HandleShopStart(c)
	}
	return b.accountHandler.HandleStart(c)
}

// handleCallback routes callbacks to appropriate handlers
func (b *Bot) handleCallback(c tele.Context) error {
	callback := c.Callback()
	if callback == nil {
		return nil
	}

	data := callback.Data

	// Route shop callbacks
	if strings.HasPrefix(data, "shop_") {
		return b.shopHandler.HandleShopCallback(c)
	}

	// Route sicbo callbacks
	return b.gameHandler.HandleSicBoCallback(c)
}

// Start starts the bot polling.
func (b *Bot) Start() {
	log.Info().Msg("Starting bot...")
	
	// Start message cleaner for auto-deleting old bot messages
	b.gameHandler.StartMessageCleaner(b.bot)
	log.Info().Msg("Message cleaner started (30 min interval)")
	
	b.bot.Start()
}

// Stop stops the bot gracefully.
func (b *Bot) Stop() {
	log.Info().Msg("Stopping bot...")
	b.bot.Stop()
}

// GetBot returns the underlying telebot instance.
func (b *Bot) GetBot() *tele.Bot {
	return b.bot
}

// Context returns a background context for handlers.
func (b *Bot) Context() context.Context {
	return context.Background()
}
