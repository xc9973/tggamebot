// Package handler provides Telegram bot command handlers.
// Requirements: 6.1, 6.2, 6.3, 6.4, 6.5 - Admin functionality
package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/model"
	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/service"
)

// AdminHandler handles admin-related commands.
type AdminHandler struct {
	accountService *service.AccountService
	userLock       *lock.UserLock
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(accountService *service.AccountService, userLock *lock.UserLock) *AdminHandler {
	return &AdminHandler{
		accountService: accountService,
		userLock:       userLock,
	}
}

// HandleAdminAdd handles the /admin_add command.
// Format: /admin_add <user_id> <amount>
// Requirements: 6.1, 6.5
func (h *AdminHandler) HandleAdminAdd(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse arguments
	targetID, amount, err := h.parseAdminArgs(c)
	if err != nil {
		return c.Reply(err.Error())
	}

	if amount <= 0 {
		return c.Reply("❌ 金额必须大于 0")
	}

	// Acquire lock for target user
	h.userLock.Lock(targetID)
	defer h.userLock.Unlock(targetID)

	// Add balance
	desc := fmt.Sprintf("管理员 %d 添加", sender.ID)
	user, err := h.accountService.UpdateBalance(ctx, targetID, amount, model.TxTypeAdminAdd, &desc)
	if err != nil {
		return c.Reply("❌ 操作失败，用户可能不存在")
	}

	// Log admin operation (Requirements: 6.5)
	log.Info().
		Int64("admin_id", sender.ID).
		Int64("target_id", targetID).
		Int64("amount", amount).
		Str("operation", "admin_add").
		Msg("Admin operation executed")

	displayName := user.Username
	if displayName == "" {
		displayName = fmt.Sprintf("%d", targetID)
	}

	return c.Reply(fmt.Sprintf(
		"✅ 操作成功\n\n"+
			"👤 用户: %s (ID: %d)\n"+
			"➕ 添加: %d 金币\n"+
			"💰 当前余额: %d 金币",
		displayName, targetID, amount, user.Balance,
	))
}

// HandleAdminSub handles the /admin_sub command.
// Format: /admin_sub <user_id> <amount>
// Requirements: 6.2, 6.5
func (h *AdminHandler) HandleAdminSub(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse arguments
	targetID, amount, err := h.parseAdminArgs(c)
	if err != nil {
		return c.Reply(err.Error())
	}

	if amount <= 0 {
		return c.Reply("❌ 金额必须大于 0")
	}

	// Acquire lock for target user
	h.userLock.Lock(targetID)
	defer h.userLock.Unlock(targetID)

	// Subtract balance (negative amount)
	desc := fmt.Sprintf("管理员 %d 扣除", sender.ID)
	user, err := h.accountService.UpdateBalance(ctx, targetID, -amount, model.TxTypeAdminSub, &desc)
	if err != nil {
		return c.Reply("❌ 操作失败，用户可能不存在")
	}

	// Log admin operation (Requirements: 6.5)
	log.Info().
		Int64("admin_id", sender.ID).
		Int64("target_id", targetID).
		Int64("amount", amount).
		Str("operation", "admin_sub").
		Msg("Admin operation executed")

	displayName := user.Username
	if displayName == "" {
		displayName = fmt.Sprintf("%d", targetID)
	}

	return c.Reply(fmt.Sprintf(
		"✅ 操作成功\n\n"+
			"👤 用户: %s (ID: %d)\n"+
			"➖ 扣除: %d 金币\n"+
			"💰 当前余额: %d 金币",
		displayName, targetID, amount, user.Balance,
	))
}

// HandleAdminSet handles the /admin_set command.
// Format: /admin_set <user_id> <amount>
// Requirements: 6.3, 6.5
func (h *AdminHandler) HandleAdminSet(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse arguments
	targetID, newBalance, err := h.parseAdminArgs(c)
	if err != nil {
		return c.Reply(err.Error())
	}

	if newBalance < 0 {
		return c.Reply("❌ 余额不能为负数")
	}

	// Acquire lock for target user
	h.userLock.Lock(targetID)
	defer h.userLock.Unlock(targetID)

	// Get current balance
	currentBalance, err := h.accountService.GetBalance(ctx, targetID)
	if err != nil {
		return c.Reply("❌ 用户不存在")
	}

	// Calculate difference and update
	diff := newBalance - currentBalance
	desc := fmt.Sprintf("管理员 %d 设置余额", sender.ID)
	user, err := h.accountService.UpdateBalance(ctx, targetID, diff, model.TxTypeAdminSet, &desc)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Log admin operation (Requirements: 6.5)
	log.Info().
		Int64("admin_id", sender.ID).
		Int64("target_id", targetID).
		Int64("old_balance", currentBalance).
		Int64("new_balance", newBalance).
		Str("operation", "admin_set").
		Msg("Admin operation executed")

	displayName := user.Username
	if displayName == "" {
		displayName = fmt.Sprintf("%d", targetID)
	}

	return c.Reply(fmt.Sprintf(
		"✅ 操作成功\n\n"+
			"👤 用户: %s (ID: %d)\n"+
			"📝 原余额: %d 金币\n"+
			"💰 新余额: %d 金币",
		displayName, targetID, currentBalance, user.Balance,
	))
}

// parseAdminArgs parses admin command arguments.
// Format: <user_id> <amount>
// Returns targetID, amount, error
func (h *AdminHandler) parseAdminArgs(c tele.Context) (int64, int64, error) {
	args := c.Args()
	if len(args) < 2 {
		return 0, 0, fmt.Errorf("❌ 用法: /admin_add <用户ID> <金额>\n例如: /admin_add 123456789 100")
	}

	// Parse target user ID
	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("❌ 用户ID格式错误，请输入数字")
	}

	// Parse amount
	amount, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("❌ 金额格式错误，请输入整数")
	}

	return targetID, amount, nil
}

// HandleAdminGiftAll handles the /admin_gift_all command.
// Format: /admin_gift_all amount
// Adds the specified amount to ALL users' balances.
func (h *AdminHandler) HandleAdminGiftAll(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse arguments
	args := c.Args()
	if len(args) < 1 {
		return c.Reply("❌ 用法: /admin_gift_all 金额\n例如: /admin_gift_all 100")
	}

	amount, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || amount <= 0 {
		return c.Reply("❌ 金额必须是大于 0 的整数")
	}

	// Add balance to all users
	count, err := h.accountService.AddBalanceToAllUsers(ctx, amount)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Log admin operation
	log.Info().
		Int64("admin_id", sender.ID).
		Int64("amount", amount).
		Int64("user_count", count).
		Str("operation", "admin_gift_all").
		Msg("Admin gift all operation executed")

	return c.Reply(fmt.Sprintf(
		"✅ 赠送成功\n\n"+
			"🎁 赠送金额: %d 金币\n"+
			"👥 受益用户: %d 人",
		amount, count,
	))
}
