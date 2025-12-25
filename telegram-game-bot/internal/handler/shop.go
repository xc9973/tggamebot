// Package handler provides Telegram bot command handlers.
package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/service"
	"telegram-game-bot/internal/shop"
)

// ShopHandler handles shop-related commands
type ShopHandler struct {
	shopService    *service.ShopService
	accountService *service.AccountService
}

// NewShopHandler creates a new ShopHandler
func NewShopHandler(shopService *service.ShopService, accountService *service.AccountService) *ShopHandler {
	return &ShopHandler{
		shopService:    shopService,
		accountService: accountService,
	}
}

// HandleShopStart handles /start in private chat to show shop
func (h *ShopHandler) HandleShopStart(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	chat := c.Chat()

	if sender == nil || chat == nil {
		return nil
	}

	// Only show shop in private chat
	if chat.Type != tele.ChatPrivate {
		return nil // Let other handlers handle group /start
	}

	// Ensure user exists
	username := sender.Username
	if username == "" {
		username = sender.FirstName
	}
	_, _, err := h.accountService.EnsureUser(ctx, sender.ID, username)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Get balance
	balance, err := h.accountService.GetBalance(ctx, sender.ID)
	if err != nil {
		balance = 0
	}

	// Send shop panel
	msg := shop.FormatShopMessage(balance)
	markup := shop.BuildShopPanel()
	return c.Send(msg, markup)
}

// HandleShopCallback handles shop button callbacks
func (h *ShopHandler) HandleShopCallback(c tele.Context) error {
	ctx := context.Background()
	callback := c.Callback()
	sender := c.Sender()

	if callback == nil || sender == nil {
		return nil
	}

	data := callback.Data

	// Handle refresh
	if data == shop.CallbackShopRefresh {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		msg := shop.FormatShopMessage(balance)
		markup := shop.BuildShopPanel()
		return c.Edit(msg, markup)
	}

	// Handle bag view
	if data == shop.CallbackShopBag {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		inventory, err := h.shopService.GetUserInventory(ctx, sender.ID)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "❌ 获取背包失败", ShowAlert: true})
		}

		// Convert effects to display format
		var effects []shop.EffectInfo
		for _, effect := range inventory.Effects {
			remaining := time.Until(effect.ExpiresAt).Seconds()
			effects = append(effects, shop.EffectInfo{
				EffectType:   effect.EffectType,
				RemainingStr: shop.FormatRemainingTime(int64(remaining)),
			})
		}

		msg := shop.FormatInventoryMessage(balance, inventory.HandcuffCount, effects)
		markup := shop.BuildBagPanel()
		return c.Edit(msg, markup)
	}

	// Handle cancel
	if data == shop.CallbackShopCancel {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		msg := shop.FormatShopMessage(balance)
		markup := shop.BuildShopPanel()
		return c.Edit(msg, markup)
	}

	// Handle item selection
	if strings.HasPrefix(data, shop.CallbackShopItem) {
		itemTypeStr := strings.TrimPrefix(data, shop.CallbackShopItem)
		itemType := shop.ItemType(itemTypeStr)
		
		item, ok := shop.GetItem(itemType)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "❌ 道具不存在"})
		}

		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		msg := shop.FormatItemDetail(item, balance)
		markup := shop.BuildConfirmPanel(itemType)
		return c.Edit(msg, markup)
	}

	// Handle purchase
	if strings.HasPrefix(data, shop.CallbackShopBuy) {
		itemTypeStr := strings.TrimPrefix(data, shop.CallbackShopBuy)
		itemType := shop.ItemType(itemTypeStr)

		item, ok := shop.GetItem(itemType)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "❌ 道具不存在", ShowAlert: true})
		}

		err := h.shopService.PurchaseItem(ctx, sender.ID, itemType)
		if err != nil {
			if errors.Is(err, service.ErrInsufficientBalance) {
				return c.Respond(&tele.CallbackResponse{
					Text:      "❌ 余额不足！",
					ShowAlert: true,
				})
			}
			log.Error().Err(err).Int64("user_id", sender.ID).Str("item", string(itemType)).Msg("Purchase failed")
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ 购买失败，请稍后重试",
				ShowAlert: true,
			})
		}

		// Success - show updated shop
		c.Respond(&tele.CallbackResponse{
			Text: "✅ 购买成功！" + item.Emoji + " " + item.Name,
		})

		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		msg := shop.FormatShopMessage(balance)
		markup := shop.BuildShopPanel()
		return c.Edit(msg, markup)
	}

	return nil
}

// HandleBag handles /bag command to show inventory
func (h *ShopHandler) HandleBag(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()

	if sender == nil {
		return nil
	}

	balance, _ := h.accountService.GetBalance(ctx, sender.ID)
	inventory, err := h.shopService.GetUserInventory(ctx, sender.ID)
	if err != nil {
		return c.Reply("❌ 获取背包失败")
	}

	// Convert effects to display format
	var effects []shop.EffectInfo
	for _, effect := range inventory.Effects {
		remaining := time.Until(effect.ExpiresAt).Seconds()
		effects = append(effects, shop.EffectInfo{
			EffectType:   effect.EffectType,
			RemainingStr: shop.FormatRemainingTime(int64(remaining)),
		})
	}

	msg := shop.FormatInventoryMessage(balance, inventory.HandcuffCount, effects)
	return c.Reply(msg)
}

// HandleHandcuff handles /handcuff command
func (h *ShopHandler) HandleHandcuff(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()

	if sender == nil {
		return nil
	}

	// Check if user has handcuffs (silent fail if not)
	if !h.shopService.HasHandcuff(ctx, sender.ID) {
		return nil // Silent ignore per requirements
	}

	// Get target from reply
	var targetID int64
	var targetName string

	if c.Message().ReplyTo != nil && c.Message().ReplyTo.Sender != nil {
		targetID = c.Message().ReplyTo.Sender.ID
		targetName = c.Message().ReplyTo.Sender.Username
		if targetName == "" {
			targetName = c.Message().ReplyTo.Sender.FirstName
		}
	} else {
		return c.Reply("❌ 请回复目标用户的消息来使用手铐")
	}

	// Use handcuff
	err := h.shopService.UseHandcuff(ctx, sender.ID, targetID)
	if err != nil {
		if errors.Is(err, service.ErrSelfHandcuff) {
			return c.Reply("❌ 不能对自己使用手铐")
		}
		if errors.Is(err, service.ErrTargetNotFound) {
			return c.Reply("❌ 目标用户未注册")
		}
		if errors.Is(err, service.ErrAlreadyLocked) {
			return c.Reply("❌ 目标已被锁定")
		}
		if errors.Is(err, service.ErrNoHandcuff) {
			return nil // Silent ignore
		}
		log.Error().Err(err).Msg("Handcuff failed")
		return c.Reply("❌ 使用失败，请稍后重试")
	}

	// Get username
	username := sender.Username
	if username == "" {
		username = sender.FirstName
	}

	return c.Reply("🔗 " + username + " 对 " + targetName + " 使用了手铐！\n⏱️ 锁定时间: 30分钟\n🚫 " + targetName + " 无法打劫任何人")
}
