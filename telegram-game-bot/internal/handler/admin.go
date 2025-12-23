// Package handler provides Telegram bot command handlers.
// Requirements: 6.1, 6.2, 6.3, 6.4, 6.5 - Admin functionality
package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
// Format: /admin_add @username amount
// Requirements: 6.1, 6.5
func (h *AdminHandler) HandleAdminAdd(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse arguments
	targetID, targetUsername, amount, err := h.parseAdminArgs(c)
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
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Log admin operation (Requirements: 6.5)
	log.Info().
		Int64("admin_id", sender.ID).
		Int64("target_id", targetID).
		Str("target_username", targetUsername).
		Int64("amount", amount).
		Str("operation", "admin_add").
		Msg("Admin operation executed")

	return c.Reply(fmt.Sprintf(
		"✅ 操作成功\n\n"+
			"👤 用户: @%s\n"+
			"➕ 添加: %d 金币\n"+
			"💰 当前余额: %d 金币",
		targetUsername, amount, user.Balance,
	))
}

// HandleAdminSub handles the /admin_sub command.
// Format: /admin_sub @username amount
// Requirements: 6.2, 6.5
func (h *AdminHandler) HandleAdminSub(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse arguments
	targetID, targetUsername, amount, err := h.parseAdminArgs(c)
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
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Log admin operation (Requirements: 6.5)
	log.Info().
		Int64("admin_id", sender.ID).
		Int64("target_id", targetID).
		Str("target_username", targetUsername).
		Int64("amount", amount).
		Str("operation", "admin_sub").
		Msg("Admin operation executed")

	return c.Reply(fmt.Sprintf(
		"✅ 操作成功\n\n"+
			"👤 用户: @%s\n"+
			"➖ 扣除: %d 金币\n"+
			"💰 当前余额: %d 金币",
		targetUsername, amount, user.Balance,
	))
}

// HandleAdminSet handles the /admin_set command.
// Format: /admin_set @username amount
// Requirements: 6.3, 6.5
func (h *AdminHandler) HandleAdminSet(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse arguments
	targetID, targetUsername, newBalance, err := h.parseAdminArgs(c)
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
		Str("target_username", targetUsername).
		Int64("old_balance", currentBalance).
		Int64("new_balance", newBalance).
		Str("operation", "admin_set").
		Msg("Admin operation executed")

	return c.Reply(fmt.Sprintf(
		"✅ 操作成功\n\n"+
			"👤 用户: @%s\n"+
			"📝 原余额: %d 金币\n"+
			"💰 新余额: %d 金币",
		targetUsername, currentBalance, user.Balance,
	))
}

// parseAdminArgs parses admin command arguments.
// Returns targetID, targetUsername, amount, error
func (h *AdminHandler) parseAdminArgs(c tele.Context) (int64, string, int64, error) {
	args := c.Args()
	if len(args) < 2 {
		return 0, "", 0, fmt.Errorf("❌ 用法: %s @用户名 金额", c.Text())
	}

	// Parse target user
	targetStr := args[0]
	if !strings.HasPrefix(targetStr, "@") {
		return 0, "", 0, fmt.Errorf("❌ 请使用 @用户名 格式指定用户")
	}
	targetUsername := strings.TrimPrefix(targetStr, "@")

	// Parse amount
	amount, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return 0, "", 0, fmt.Errorf("❌ 金额格式错误，请输入整数")
	}

	// Get target user ID from message mention or reply
	var targetID int64

	// Check if message has entities (mentions)
	if c.Message() != nil && len(c.Message().Entities) > 0 {
		for _, entity := range c.Message().Entities {
			if entity.Type == tele.EntityMention && entity.User != nil {
				if entity.User.Username == targetUsername {
					targetID = entity.User.ID
					break
				}
			}
		}
	}

	// If no mention found, try to find user by reply
	if targetID == 0 && c.Message() != nil && c.Message().ReplyTo != nil {
		replyUser := c.Message().ReplyTo.Sender
		if replyUser != nil && replyUser.Username == targetUsername {
			targetID = replyUser.ID
		}
	}

	if targetID == 0 {
		return 0, "", 0, fmt.Errorf("❌ 找不到用户 @%s\n请确保该用户已使用过本机器人，或回复该用户的消息", targetUsername)
	}

	return targetID, targetUsername, amount, nil
}
