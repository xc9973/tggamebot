// Package bot provides middleware for the Telegram bot.
// Requirements: 6.4 - Admin permission verification
// Requirements: 7.1 - Whitelist enforcement
// Requirements: 7.2 - Private chat access control
package bot

import (
	"sync"

	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/config"
)

// privateUserCache tracks users who have used the bot in whitelisted groups.
// This allows them to use the bot in private chat.
// Requirements: 7.2
var (
	privateUserCache = make(map[int64]bool)
	privateUserMu    sync.RWMutex
)

// AllowPrivateUser marks a user as allowed to use private chat.
func AllowPrivateUser(userID int64) {
	privateUserMu.Lock()
	defer privateUserMu.Unlock()
	privateUserCache[userID] = true
}

// IsPrivateUserAllowed checks if a user is allowed to use private chat.
func IsPrivateUserAllowed(userID int64) bool {
	privateUserMu.RLock()
	defer privateUserMu.RUnlock()
	return privateUserCache[userID]
}

// WhitelistMiddleware creates a middleware that checks if the chat is whitelisted.
// Requirements: 7.1, 7.2
func WhitelistMiddleware(cfg *config.Config) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			chat := c.Chat()
			sender := c.Sender()

			if chat == nil || sender == nil {
				return nil
			}

			// Check if it's a private chat
			if chat.Type == tele.ChatPrivate {
				// Allow if user has previously used bot in whitelisted group
				// Requirements: 7.2
				if IsPrivateUserAllowed(sender.ID) {
					return next(c)
				}

				// If whitelist is empty, allow all private chats
				if len(cfg.Whitelist.Chats) == 0 {
					return next(c)
				}

				// Otherwise, ignore private chat from unknown users
				log.Debug().
					Int64("user_id", sender.ID).
					Msg("Ignoring private chat from user not in whitelist cache")
				return nil
			}

			// For group chats, check whitelist
			// Requirements: 7.1
			if !cfg.IsChatAllowed(chat.ID) {
				log.Debug().
					Int64("chat_id", chat.ID).
					Msg("Ignoring command from non-whitelisted chat")
				return nil
			}

			// Mark user as allowed for private chat
			// Requirements: 7.2
			AllowPrivateUser(sender.ID)

			return next(c)
		}
	}
}

// AdminMiddleware creates a middleware that checks if the user is an admin.
// Requirements: 6.4
func AdminMiddleware(cfg *config.Config) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			if sender == nil {
				return nil
			}

			// Check if user is admin
			// Requirements: 6.4
			if !cfg.IsAdmin(sender.ID) {
				log.Warn().
					Int64("user_id", sender.ID).
					Str("command", c.Text()).
					Msg("Non-admin attempted admin command")
				return c.Reply("❌ 权限不足：需要管理员权限")
			}

			return next(c)
		}
	}
}

// LoggingMiddleware creates a middleware that logs all incoming messages.
func LoggingMiddleware() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			chat := c.Chat()

			logEvent := log.Debug()
			if sender != nil {
				logEvent = logEvent.
					Int64("user_id", sender.ID).
					Str("username", sender.Username)
			}
			if chat != nil {
				logEvent = logEvent.
					Int64("chat_id", chat.ID).
					Str("chat_type", string(chat.Type))
			}
			logEvent.
				Str("text", c.Text()).
				Msg("Received message")

			return next(c)
		}
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics.
func RecoveryMiddleware() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Interface("panic", r).
						Msg("Recovered from panic in handler")
					_ = c.Reply("❌ 发生内部错误，请稍后重试")
				}
			}()
			return next(c)
		}
	}
}
