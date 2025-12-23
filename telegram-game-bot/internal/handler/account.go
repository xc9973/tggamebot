// Package handler provides Telegram bot command handlers.
// Requirements: 1.1, 1.2, 1.3, 1.4, 1.5 - User account management
// Requirements: 9.1, 9.2 - Per-user locks for balance operations
package handler

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/service"
)

// AccountHandler handles account-related commands.
type AccountHandler struct {
	accountService *service.AccountService
	rankingService *service.RankingService
	userLock       *lock.UserLock
}

// NewAccountHandler creates a new AccountHandler.
func NewAccountHandler(accountService *service.AccountService, rankingService *service.RankingService, userLock *lock.UserLock) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		rankingService: rankingService,
		userLock:       userLock,
	}
}

// HandleStart handles the /start command.
// Creates a new account with 1000 initial coins if user doesn't exist.
// Requirements: 1.1, 9.1
func (h *AccountHandler) HandleStart(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	username := sender.Username
	if username == "" {
		username = sender.FirstName
	}

	// Acquire lock before balance-modifying operation
	// Requirements: 9.1
	h.userLock.Lock(sender.ID)
	defer h.userLock.Unlock(sender.ID)

	user, created, err := h.accountService.EnsureUser(ctx, sender.ID, username)
	if err != nil {
		return c.Reply("❌ 创建账户失败，请稍后重试")
	}

	if created {
		return c.Reply(fmt.Sprintf(
			"🎉 欢迎 @%s！\n\n"+
				"您的账户已创建，初始金币: %d\n\n"+
				"可用命令:\n"+
				"/balance - 查看余额\n"+
				"/daily - 每日签到\n"+
				"/top - 富豪榜\n"+
				"/dice <金额> - 骰子游戏\n"+
				"/slot <金额> - 老虎机\n"+
				"/pay @用户 <金额> - 转账",
			username, user.Balance,
		))
	}

	return c.Reply(fmt.Sprintf(
		"👋 欢迎回来 @%s！\n\n"+
			"当前余额: %d 金币",
		username, user.Balance,
	))
}

// HandleBalance handles the /balance command.
// Displays the user's current balance.
// Requirements: 1.2
func (h *AccountHandler) HandleBalance(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	balance, err := h.accountService.GetBalance(ctx, sender.ID)
	if err != nil {
		// User might not exist, try to create
		username := sender.Username
		if username == "" {
			username = sender.FirstName
		}
		user, _, err := h.accountService.EnsureUser(ctx, sender.ID, username)
		if err != nil {
			return c.Reply("❌ 获取余额失败，请稍后重试")
		}
		balance = user.Balance
	}

	return c.Reply(fmt.Sprintf("💰 当前余额: %d 金币", balance))
}

// HandleMy handles the /my command.
// Displays the user's account information.
func (h *AccountHandler) HandleMy(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	user, err := h.accountService.GetUser(ctx, sender.ID)
	if err != nil {
		// User might not exist, try to create
		username := sender.Username
		if username == "" {
			username = sender.FirstName
		}
		user, _, err = h.accountService.EnsureUser(ctx, sender.ID, username)
		if err != nil {
			return c.Reply("❌ 获取账户信息失败，请稍后重试")
		}
	}

	// Get daily profit
	dailyProfit, _ := h.rankingService.GetUserDailyProfit(ctx, sender.ID)

	profitStr := fmt.Sprintf("%d", dailyProfit)
	if dailyProfit > 0 {
		profitStr = "+" + profitStr
	}

	return c.Reply(fmt.Sprintf(
		"📊 账户信息\n"+
			"━━━━━━━━━━━━━━━\n"+
			"👤 用户: @%s\n"+
			"💰 余额: %d 金币\n"+
			"📈 今日盈亏: %s\n"+
			"━━━━━━━━━━━━━━━",
		user.Username, user.Balance, profitStr,
	))
}

// HandleDaily handles the /daily command.
// Grants 500 coins if 24 hours have passed since last claim.
// Requirements: 1.3, 1.4, 9.1
func (h *AccountHandler) HandleDaily(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Ensure user exists first (outside lock to avoid nested locking)
	username := sender.Username
	if username == "" {
		username = sender.FirstName
	}

	// Acquire lock before balance-modifying operation
	// Requirements: 9.1
	h.userLock.Lock(sender.ID)
	defer h.userLock.Unlock(sender.ID)

	_, _, err := h.accountService.EnsureUser(ctx, sender.ID, username)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Try to claim daily reward
	success, msg, err := h.accountService.ClaimDaily(ctx, sender.ID)
	if err != nil {
		return c.Reply("❌ 签到失败，请稍后重试")
	}

	if success {
		return c.Reply(fmt.Sprintf("✅ %s", msg))
	}

	return c.Reply(fmt.Sprintf("⏰ %s", msg))
}

// HandleTop handles the /top command.
// Displays the top 10 users by balance.
// Requirements: 1.5
func (h *AccountHandler) HandleTop(c tele.Context) error {
	ctx := context.Background()

	users, err := h.rankingService.GetTopUsers(ctx, 10)
	if err != nil {
		return c.Reply("❌ 获取排行榜失败，请稍后重试")
	}

	if len(users) == 0 {
		return c.Reply("📊 暂无排行数据")
	}

	msg := "🏆 富豪榜 TOP 10\n"
	msg += "━━━━━━━━━━━━━━━\n"

	medals := []string{"🥇", "🥈", "🥉"}
	for i, user := range users {
		rank := fmt.Sprintf("%d.", i+1)
		if i < 3 {
			rank = medals[i]
		}

		displayName := user.Username
		if displayName == "" {
			displayName = fmt.Sprintf("User%d", user.TelegramID)
		}

		msg += fmt.Sprintf("%s @%s: %d\n", rank, displayName, user.Balance)
	}

	msg += "━━━━━━━━━━━━━━━"

	return c.Reply(msg)
}
