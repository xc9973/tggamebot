// Package main is the entry point for the Telegram Game Bot.
// Requirements: 8.4 - Database migrations for schema management
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"telegram-game-bot/internal/bot"
	"telegram-game-bot/internal/config"
	"telegram-game-bot/internal/game"
	"telegram-game-bot/internal/game/dice"
	"telegram-game-bot/internal/game/rob"
	"telegram-game-bot/internal/game/sicbo"
	"telegram-game-bot/internal/game/slot"
	"telegram-game-bot/internal/pkg/db"
	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/repository"
	"telegram-game-bot/internal/service"
)

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Load configuration
	cfg, err := config.Load("config")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().Msg("Configuration loaded successfully")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection pool
	dbPool, err := db.NewPool(ctx, &cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer dbPool.Close()

	// Run database migrations
	if err := runMigrations(ctx, dbPool); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbPool.Pool)
	txRepo := repository.NewTransactionRepository(dbPool.Pool)
	inventoryRepo := repository.NewInventoryRepository(dbPool.Pool)

	// Initialize services
	accountService := service.NewAccountService(
		userRepo,
		txRepo,
		cfg.Daily.Reward,
		cfg.Daily.CooldownHours,
	)

	transferService := service.NewTransferService(userRepo, txRepo)

	rankingService := service.NewRankingService(userRepo, txRepo, time.Local)

	// Initialize user lock
	userLock := lock.NewUserLock()

	// Initialize game registry and register games
	gameRegistry := game.NewRegistry()

	// Register dice game
	diceGame := dice.New(&dice.Config{
		MaxBet:   cfg.Games.Dice.MaxBet,
		Cooldown: cfg.Games.Dice.CooldownSeconds,
	})
	if err := gameRegistry.Register(diceGame); err != nil {
		log.Fatal().Err(err).Msg("Failed to register dice game")
	}

	// Register slot game
	slotGame := slot.New(&slot.Config{
		Cooldown: cfg.Games.Slot.CooldownSeconds,
	})
	if err := gameRegistry.Register(slotGame); err != nil {
		log.Fatal().Err(err).Msg("Failed to register slot game")
	}

	// Initialize SicBo game (multiplayer)
	sicboGame := sicbo.New()

	// Initialize Rob game
	robGame := rob.NewRobGame(userRepo, txRepo, userLock)

	// Initialize Shop service
	shopService := service.NewShopService(userRepo, txRepo, inventoryRepo, userLock)

	// Connect shop service to rob game for item effects
	robGame.SetItemChecker(shopService)

	log.Info().
		Int("game_count", gameRegistry.Count()).
		Strs("games", gameRegistry.Commands()).
		Msg("Games registered")

	// Create bot dependencies
	deps := &bot.Dependencies{
		Config:          cfg,
		AccountService:  accountService,
		TransferService: transferService,
		RankingService:  rankingService,
		ShopService:     shopService,
		GameRegistry:    gameRegistry,
		SicBoGame:       sicboGame,
		RobGame:         robGame,
		UserLock:        userLock,
	}

	// Initialize bot
	telegramBot, err := bot.New(deps)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create bot")
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start bot in a goroutine
	go func() {
		log.Info().Msg("Bot is starting...")
		telegramBot.Start()
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Graceful shutdown
	telegramBot.Stop()
	log.Info().Msg("Bot stopped gracefully")
}

// runMigrations executes database migrations.
// Requirements: 8.4 - Implement database migrations for schema management
func runMigrations(ctx context.Context, pool *db.Pool) error {
	log.Info().Msg("Running database migrations...")

	// Migration 1: Create users table
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			telegram_id BIGINT PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			balance BIGINT NOT NULL DEFAULT 1000,
			last_daily_claim BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_users_balance ON users(balance DESC);
	`)
	if err != nil {
		return err
	}
	log.Info().Msg("Migration 1: users table created")

	// Migration 2: Create transactions table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS transactions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
			amount BIGINT NOT NULL,
			type VARCHAR(50) NOT NULL,
			description TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_transactions_user_time ON transactions(user_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_transactions_type_time ON transactions(type, created_at DESC);
	`)
	if err != nil {
		return err
	}
	log.Info().Msg("Migration 2: transactions table created")

	// Migration 3: Create daily stats view
	_, err = pool.Exec(ctx, `
		CREATE OR REPLACE VIEW daily_game_stats AS
		SELECT 
			user_id,
			SUM(amount) as net_profit,
			DATE(created_at) as game_date
		FROM transactions
		WHERE type IN ('dice', 'slot', 'sicbo_win', 'sicbo_bet', 'rob', 'robbed')
		GROUP BY user_id, DATE(created_at);
	`)
	if err != nil {
		return err
	}
	log.Info().Msg("Migration 3: daily_game_stats view created")

	// Migration 4: Create shop system tables
	// user_items - stores stackable items like handcuffs
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_items (
			user_id BIGINT NOT NULL,
			item_type VARCHAR(50) NOT NULL,
			quantity INT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, item_type)
		);
	`)
	if err != nil {
		return err
	}
	log.Info().Msg("Migration 4a: user_items table created")

	// user_effects - stores time-based effects (shield, thorn armor, bloodthirst sword)
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_effects (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			effect_type VARCHAR(50) NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_user_effects_user ON user_effects(user_id);
		CREATE INDEX IF NOT EXISTS idx_user_effects_expires ON user_effects(expires_at);
	`)
	if err != nil {
		return err
	}
	log.Info().Msg("Migration 4b: user_effects table created")

	// handcuff_locks - stores users locked by handcuffs
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS handcuff_locks (
			target_id BIGINT PRIMARY KEY,
			locked_by BIGINT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_handcuff_locks_expires ON handcuff_locks(expires_at);
	`)
	if err != nil {
		return err
	}
	log.Info().Msg("Migration 4c: handcuff_locks table created")

	log.Info().Msg("All migrations completed successfully")
	return nil
}
