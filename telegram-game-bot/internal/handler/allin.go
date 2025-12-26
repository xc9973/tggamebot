// Package handler provides Telegram bot command handlers.
package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/game/allin"
	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/service"
)

// AllInHandler handles all-in gambling commands.
type AllInHandler struct {
	accountService *service.AccountService
	allInGame      *allin.AllInGame
	userLock       *lock.UserLock
}

// NewAllInHandler creates a new AllInHandler.
func NewAllInHandler(
	accountService *service.AccountService,
	allInGame *allin.AllInGame,
	userLock *lock.UserLock,
) *AllInHandler {
	return &AllInHandler{
		accountService: accountService,
		allInGame:      allInGame,
		userLock:       userLock,
	}
}

// HandleAllInRob handles the /shdj command for all-in robbery.
func (h *AllInHandler) HandleAllInRob(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	chat := c.Chat()

	if sender == nil || chat == nil {
		return nil
	}

	// Get robber's username
	robberName := sender.Username
	if robberName == "" {
		robberName = sender.FirstName
	}

	// Ensure robber exists
	_, _, err := h.accountService.EnsureUser(ctx, sender.ID, robberName)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Determine victim from reply or @mention
	var victimID int64
	var victimName string

	// Check if replying to a message
	if c.Message().ReplyTo != nil && c.Message().ReplyTo.Sender != nil {
		victimID = c.Message().ReplyTo.Sender.ID
		victimName = c.Message().ReplyTo.Sender.Username
		if victimName == "" {
			victimName = c.Message().ReplyTo.Sender.FirstName
		}
	} else {
		return c.Reply("❌ 用法: 回复目标用户的消息，然后发送 /shdj")
	}

	// Ensure victim exists
	_, _, err = h.accountService.EnsureUser(ctx, victimID, victimName)
	if err != nil {
		return c.Reply("❌ 目标用户未注册")
	}

	// Execute all-in robbery
	result, err := h.allInGame.AllInRob(ctx, sender.ID, victimID, robberName, victimName)
	if err != nil {
		log.Error().Err(err).Int64("robber", sender.ID).Int64("victim", victimID).Msg("All-in robbery failed")
		return c.Reply("❌ " + err.Error())
	}

	return c.Reply(result.Message)
}

// HandleDuel handles the /duijue command for duel challenge.
func (h *AllInHandler) HandleDuel(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	chat := c.Chat()

	if sender == nil || chat == nil {
		return nil
	}

	// Get challenger's username
	challengerName := sender.Username
	if challengerName == "" {
		challengerName = sender.FirstName
	}

	// Ensure challenger exists
	_, _, err := h.accountService.EnsureUser(ctx, sender.ID, challengerName)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Determine target from reply
	var targetID int64
	var targetName string

	if c.Message().ReplyTo != nil && c.Message().ReplyTo.Sender != nil {
		targetID = c.Message().ReplyTo.Sender.ID
		targetName = c.Message().ReplyTo.Sender.Username
		if targetName == "" {
			targetName = c.Message().ReplyTo.Sender.FirstName
		}
	} else {
		return c.Reply("❌ 用法: 回复目标用户的消息，然后发送 /duijue")
	}

	// Ensure target exists
	_, _, err = h.accountService.EnsureUser(ctx, targetID, targetName)
	if err != nil {
		return c.Reply("❌ 目标用户未注册")
	}

	// Create duel challenge
	duel, err := h.allInGame.CreateDuel(ctx, sender.ID, targetID, challengerName, targetName, chat.ID)
	if err != nil {
		log.Error().Err(err).Int64("challenger", sender.ID).Int64("target", targetID).Msg("Create duel failed")
		return c.Reply("❌ " + err.Error())
	}

	// Build inline keyboard
	markup := &tele.ReplyMarkup{}
	btnAccept := markup.Data("✅ 接受", "duel_accept", fmt.Sprintf("%d", targetID))
	btnDecline := markup.Data("❌ 拒绝", "duel_decline", fmt.Sprintf("%d", targetID))
	markup.Inline(
		markup.Row(btnAccept, btnDecline),
	)

	// Send challenge message
	msg := fmt.Sprintf("⚔️ @%s 向 @%s 发起梭哈对决！\n\n💰 赌注: %d 金币\n⏰ 60秒内响应\n\n只有 @%s 可以接受或拒绝",
		challengerName, targetName, duel.Amount, targetName)

	sentMsg, err := c.Bot().Send(chat, msg, markup)
	if err != nil {
		return c.Reply("❌ 发送挑战失败")
	}

	// Store message ID for later update
	h.allInGame.SetDuelMessageID(targetID, sentMsg.ID)

	return nil
}

// HandleDuelCallback handles duel accept/decline button callbacks.
func (h *AllInHandler) HandleDuelCallback(c tele.Context) error {
	ctx := context.Background()
	callback := c.Callback()
	sender := c.Sender()

	if callback == nil || sender == nil {
		return nil
	}

	// Parse callback data
	data := callback.Data
	parts := strings.Split(data, "|")
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ 无效操作"})
	}

	action := parts[0]
	targetIDStr := parts[1]

	var targetID int64
	fmt.Sscanf(targetIDStr, "%d", &targetID)

	// Check if sender is the target
	if sender.ID != targetID {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ 这不是你的对决",
			ShowAlert: true,
		})
	}

	// Get pending duel
	duel := h.allInGame.GetPendingDuel(targetID)
	if duel == nil {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ 对决已过期或不存在",
			ShowAlert: true,
		})
	}

	switch action {
	case "duel_accept":
		// Accept and execute duel
		result, err := h.allInGame.AcceptDuel(ctx, targetID)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ " + err.Error(),
				ShowAlert: true,
			})
		}

		// Update message with result
		c.Edit(result.Message)
		return c.Respond(&tele.CallbackResponse{Text: "⚔️ 对决完成！"})

	case "duel_decline":
		// Decline duel
		err := h.allInGame.DeclineDuel(targetID)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ " + err.Error(),
				ShowAlert: true,
			})
		}

		// Update message
		c.Edit(fmt.Sprintf("❌ @%s 拒绝了 @%s 的对决挑战", duel.TargetName, duel.ChallengerName))
		return c.Respond(&tele.CallbackResponse{Text: "已拒绝对决"})
	}

	return nil
}

// HandleAllInDice handles the /shdice command for all-in dice.
func (h *AllInHandler) HandleAllInDice(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()

	if sender == nil {
		return nil
	}

	// Get username
	username := sender.Username
	if username == "" {
		username = sender.FirstName
	}

	// Ensure user exists
	_, _, err := h.accountService.EnsureUser(ctx, sender.ID, username)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Execute all-in dice
	result, err := h.allInGame.AllInDice(ctx, sender.ID, username)
	if err != nil {
		log.Error().Err(err).Int64("user", sender.ID).Msg("All-in dice failed")
		return c.Reply("❌ " + err.Error())
	}

	return c.Reply(result.Message)
}
