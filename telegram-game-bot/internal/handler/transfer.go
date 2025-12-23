// Package handler provides Telegram bot command handlers.
// Requirements: 2.1, 2.2, 2.3, 2.4, 2.5 - Transfer functionality
package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/service"
)

// TransferHandler handles transfer-related commands.
type TransferHandler struct {
	accountService  *service.AccountService
	transferService *service.TransferService
	userLock        *lock.UserLock
}

// NewTransferHandler creates a new TransferHandler.
func NewTransferHandler(
	accountService *service.AccountService,
	transferService *service.TransferService,
	userLock *lock.UserLock,
) *TransferHandler {
	return &TransferHandler{
		accountService:  accountService,
		transferService: transferService,
		userLock:        userLock,
	}
}

// HandlePay handles the /pay command.
// Format: /pay @username amount
// Requirements: 2.1, 2.2, 2.3, 2.4, 2.5
func (h *TransferHandler) HandlePay(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse arguments
	args := c.Args()
	if len(args) < 2 {
		return c.Reply("❌ 用法: /pay @用户名 金额\n例如: /pay @alice 100")
	}

	// Parse target user
	targetStr := args[0]
	if !strings.HasPrefix(targetStr, "@") {
		return c.Reply("❌ 请使用 @用户名 格式指定收款人")
	}
	targetUsername := strings.TrimPrefix(targetStr, "@")

	// Parse amount
	amount, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return c.Reply("❌ 金额格式错误，请输入正整数")
	}

	// Validate amount (Requirements: 2.3)
	if amount <= 0 {
		return c.Reply("❌ 转账金额必须大于 0")
	}

	// Get target user by username from message mention or reply
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

	// If still no target found, we need to look up by username
	// This is a limitation - Telegram doesn't allow looking up users by username
	if targetID == 0 {
		return c.Reply("❌ 找不到用户 @" + targetUsername + "\n请确保该用户已使用过本机器人，或回复该用户的消息进行转账")
	}

	// Prevent self-transfer (Requirements: 2.4)
	if sender.ID == targetID {
		return c.Reply("❌ 不能给自己转账")
	}

	// Ensure both users exist
	senderUsername := sender.Username
	if senderUsername == "" {
		senderUsername = sender.FirstName
	}
	_, _, err = h.accountService.EnsureUser(ctx, sender.ID, senderUsername)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Acquire lock for sender
	h.userLock.Lock(sender.ID)
	defer h.userLock.Unlock(sender.ID)

	// Execute transfer (Requirements: 2.1, 2.2, 2.5)
	err = h.transferService.Transfer(ctx, sender.ID, targetID, amount)
	if err != nil {
		if errors.Is(err, service.ErrInsufficientBalance) {
			return c.Reply("❌ 余额不足")
		}
		if errors.Is(err, service.ErrInvalidAmount) {
			return c.Reply("❌ 转账金额必须大于 0")
		}
		if errors.Is(err, service.ErrSelfTransfer) {
			return c.Reply("❌ 不能给自己转账")
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Reply("❌ 收款用户不存在")
		}
		return c.Reply("❌ 转账失败，请稍后重试")
	}

	// Get updated balance
	newBalance, _ := h.accountService.GetBalance(ctx, sender.ID)

	return c.Reply(fmt.Sprintf(
		"✅ 转账成功！\n\n"+
			"💸 已向 @%s 转账 %d 金币\n"+
			"💰 当前余额: %d 金币",
		targetUsername, amount, newBalance,
	))
}

// HandlePayReply handles transfer via reply to a message.
// Format: /pay amount (as reply to target user's message)
func (h *TransferHandler) HandlePayReply(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Check if this is a reply
	if c.Message() == nil || c.Message().ReplyTo == nil {
		return nil
	}

	replyTo := c.Message().ReplyTo
	if replyTo.Sender == nil {
		return c.Reply("❌ 无法获取收款人信息")
	}

	targetID := replyTo.Sender.ID
	targetUsername := replyTo.Sender.Username
	if targetUsername == "" {
		targetUsername = replyTo.Sender.FirstName
	}

	// Parse amount from args
	args := c.Args()
	if len(args) < 1 {
		return c.Reply("❌ 请指定转账金额\n用法: /pay 金额 (回复对方消息)")
	}

	amount, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Reply("❌ 金额格式错误，请输入正整数")
	}

	if amount <= 0 {
		return c.Reply("❌ 转账金额必须大于 0")
	}

	if sender.ID == targetID {
		return c.Reply("❌ 不能给自己转账")
	}

	// Ensure sender exists
	senderUsername := sender.Username
	if senderUsername == "" {
		senderUsername = sender.FirstName
	}
	_, _, err = h.accountService.EnsureUser(ctx, sender.ID, senderUsername)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Acquire lock for sender
	h.userLock.Lock(sender.ID)
	defer h.userLock.Unlock(sender.ID)

	// Execute transfer
	err = h.transferService.Transfer(ctx, sender.ID, targetID, amount)
	if err != nil {
		if errors.Is(err, service.ErrInsufficientBalance) {
			return c.Reply("❌ 余额不足")
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Reply("❌ 收款用户不存在，请确保对方已使用过本机器人")
		}
		return c.Reply("❌ 转账失败，请稍后重试")
	}

	newBalance, _ := h.accountService.GetBalance(ctx, sender.ID)

	return c.Reply(fmt.Sprintf(
		"✅ 转账成功！\n\n"+
			"💸 已向 @%s 转账 %d 金币\n"+
			"💰 当前余额: %d 金币",
		targetUsername, amount, newBalance,
	))
}
