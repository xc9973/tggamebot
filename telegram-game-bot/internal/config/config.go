// Package config provides configuration management using viper.
// It supports loading from YAML files and environment variable overrides.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Bot       BotConfig       `mapstructure:"bot"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Admin     AdminConfig     `mapstructure:"admin"`
	Whitelist WhitelistConfig `mapstructure:"whitelist"`
	Daily     DailyConfig     `mapstructure:"daily"`
	Games     GamesConfig     `mapstructure:"games"`
}

// BotConfig holds Telegram bot configuration.
type BotConfig struct {
	Token string `mapstructure:"token"`
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	PoolSize        int           `mapstructure:"pool_size"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

// AdminConfig holds admin user configuration.
type AdminConfig struct {
	IDs []int64 `mapstructure:"ids"`
}

// WhitelistConfig holds chat whitelist configuration.
type WhitelistConfig struct {
	Chats []int64 `mapstructure:"chats"`
}

// DailyConfig holds daily reward configuration.
type DailyConfig struct {
	Reward        int64 `mapstructure:"reward"`
	CooldownHours int   `mapstructure:"cooldown_hours"`
}


// GamesConfig holds game-specific configuration.
type GamesConfig struct {
	Dice  DiceConfig  `mapstructure:"dice"`
	Slot  SlotConfig  `mapstructure:"slot"`
	SicBo SicBoConfig `mapstructure:"sicbo"`
}

// DiceConfig holds dice game configuration.
type DiceConfig struct {
	MaxBet          int64 `mapstructure:"max_bet"`
	CooldownSeconds int   `mapstructure:"cooldown_seconds"`
}

// SlotConfig holds slot game configuration.
type SlotConfig struct {
	CooldownSeconds int `mapstructure:"cooldown_seconds"`
}

// SicBoConfig holds sic bo game configuration.
type SicBoConfig struct {
	BettingDurationSeconds int   `mapstructure:"betting_duration_seconds"`
	FixedBetAmount         int64 `mapstructure:"fixed_bet_amount"`
}

// DSN returns the PostgreSQL connection string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		d.User, d.Password, d.Host, d.Port, d.Name,
	)
}

// Load reads configuration from file and environment variables.
// It looks for config.yaml in the config directory.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure viper
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Enable environment variable override
	// Environment variables use underscore separator and uppercase
	// e.g., BOT_TOKEN, DATABASE_HOST, DATABASE_PORT
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (optional - env vars can provide all config)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK - we can use env vars
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "gamebot")
	v.SetDefault("database.name", "gamebot")
	v.SetDefault("database.pool_size", 20)
	v.SetDefault("database.connect_timeout", "10s")
	v.SetDefault("database.max_conn_lifetime", "1h")
	v.SetDefault("database.max_conn_idle_time", "30m")

	// Daily reward defaults
	v.SetDefault("daily.reward", 500)
	v.SetDefault("daily.cooldown_hours", 24)

	// Game defaults
	v.SetDefault("games.dice.max_bet", 1000)
	v.SetDefault("games.dice.cooldown_seconds", 3)
	v.SetDefault("games.slot.cooldown_seconds", 5)
	v.SetDefault("games.sicbo.betting_duration_seconds", 60)
	v.SetDefault("games.sicbo.fixed_bet_amount", 100)
}

// IsAdmin checks if a user ID is in the admin list.
func (c *Config) IsAdmin(userID int64) bool {
	for _, id := range c.Admin.IDs {
		if id == userID {
			return true
		}
	}
	return false
}

// IsChatAllowed checks if a chat ID is in the whitelist.
func (c *Config) IsChatAllowed(chatID int64) bool {
	// Empty whitelist means all chats are allowed
	if len(c.Whitelist.Chats) == 0 {
		return true
	}
	for _, id := range c.Whitelist.Chats {
		if id == chatID {
			return true
		}
	}
	return false
}
