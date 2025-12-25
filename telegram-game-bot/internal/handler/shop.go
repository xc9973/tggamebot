// Package handler provides Telegram bot command handlers.
package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/service"
	"telegram-game-bot/internal/shop"
)

// Shop banner image file ID
const ShopBannerFileID = "AgACAgUAAxkBAAIXnWlMyQYxJ7Pj1TY_YkM0sv0VCVDkAAKDC2sbh7RoVmNP_zn_fF-lAQADAgADeQADNgQ"

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

	// Send shop panel with photo
	photo := &tele.Photo{File: tele.File{FileID: ShopBannerFileID}}
	photo.Caption = shop.FormatShopMessage(balance)
	markup := shop.BuildShopPanel()
	return c.Send(photo, markup)
}

// editShopPhoto deletes old message and sends new photo message
func (h *ShopHandler) editShopPhoto(c tele.Context, caption string, markup *tele.ReplyMarkup) error {
	// Delete old message
	c.Delete()
	
	// Send new photo message
	photo := &tele.Photo{File: tele.File{FileID: ShopBannerFileID}}
	photo.Caption = caption
	return c.Send(photo, markup)
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
	// Telebot v3 may add a \f prefix to callback data
	if strings.HasPrefix(data, "\f") {
		data = strings.TrimPrefix(data, "\f")
	}

	// Handle home - back to main menu
	if data == shop.CallbackShopHome {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		caption := shop.FormatShopMessage(balance)
		markup := shop.BuildShopPanel()
		if err := h.editShopPhoto(c, caption, markup); err != nil {
			log.Error().Err(err).Msg("Failed to edit shop photo")
		}
		return c.Respond()
	}

	// Handle refresh
	if data == shop.CallbackShopRefresh {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		caption := shop.FormatShopMessage(balance)
		markup := shop.BuildShopPanel()
		if err := h.editShopPhoto(c, caption, markup); err != nil {
			log.Error().Err(err).Msg("Failed to edit shop photo")
		}
		return c.Respond()
	}

	// Handle goods category view
	if data == shop.CallbackShopGoods {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		caption := shop.FormatGoodsCategoryMessage(balance)
		markup := shop.BuildGoodsCategoryPanel()
		if err := h.editShopPhoto(c, caption, markup); err != nil {
			log.Error().Err(err).Msg("Failed to edit shop photo")
		}
		return c.Respond()
	}

	// Handle attack items view
	if data == shop.CallbackShopAttack {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		caption := shop.FormatAttackItemsMessage(balance)
		markup := shop.BuildAttackItemsPanel()
		if err := h.editShopPhoto(c, caption, markup); err != nil {
			log.Error().Err(err).Msg("Failed to edit shop photo")
		}
		return c.Respond()
	}

	// Handle defense items view
	if data == shop.CallbackShopDefense {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		caption := shop.FormatDefenseItemsMessage(balance)
		markup := shop.BuildDefenseItemsPanel()
		if err := h.editShopPhoto(c, caption, markup); err != nil {
			log.Error().Err(err).Msg("Failed to edit shop photo")
		}
		return c.Respond()
	}

	// Handle bag view
	if data == shop.CallbackShopBag {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		inventory, err := h.shopService.GetUserInventory(ctx, sender.ID)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "❌ 获取背包失败", ShowAlert: true})
		}

		// Convert items to display format (use count based)
		var effects []shop.EffectInfo
		for _, item := range inventory.Items {
			// Skip handcuffs as they are shown separately
			if item.ItemType == string(shop.ItemHandcuff) {
				continue
			}
			effects = append(effects, shop.EffectInfo{
				EffectType:   item.ItemType,
				RemainingStr: shop.FormatUseCount(item.UseCount),
			})
		}

		caption := shop.FormatInventoryMessage(balance, inventory.HandcuffCount, effects)
		markup := shop.BuildBagPanel()
		if err := h.editShopPhoto(c, caption, markup); err != nil {
			log.Error().Err(err).Msg("Failed to edit shop photo")
		}
		return c.Respond()
	}

	// Handle cancel - back to shop (legacy, keep for compatibility)
	if data == shop.CallbackShopCancel {
		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		caption := shop.FormatShopMessage(balance)
		markup := shop.BuildShopPanel()
		if err := h.editShopPhoto(c, caption, markup); err != nil {
			log.Error().Err(err).Msg("Failed to edit shop photo")
		}
		return c.Respond()
	}

	// Handle item selection
	// Requirements: 1.2 - Show item detail with daily purchase count
	if strings.HasPrefix(data, shop.CallbackShopItem) {
		itemTypeStr := strings.TrimPrefix(data, shop.CallbackShopItem)
		itemType := shop.ItemType(itemTypeStr)
		
		item, ok := shop.GetItem(itemType)
		if !ok {
			return c.Respond(&tele.CallbackResponse{Text: "❌ 道具不存在"})
		}

		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		
		// Get daily purchase count for items with daily limit
		var caption string
		if item.HasDailyLimit() {
			_, dailyCount, _ := h.shopService.CheckDailyLimit(ctx, sender.ID, itemType)
			caption = shop.FormatItemDetailWithDailyCount(item, balance, dailyCount)
		} else {
			caption = shop.FormatItemDetail(item, balance)
		}
		
		markup := shop.BuildConfirmPanel(itemType)
		if err := h.editShopPhoto(c, caption, markup); err != nil {
			log.Error().Err(err).Msg("Failed to edit shop photo")
		}
		return c.Respond()
	}

	// Handle purchase
	// Requirements: 2.9, 3.8, 7.8 - Check daily limit and show error message
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
			if errors.Is(err, service.ErrDailyLimitReached) {
				return c.Respond(&tele.CallbackResponse{
					Text:      "❌ 今日购买次数已达上限",
					ShowAlert: true,
				})
			}
			log.Error().Err(err).Int64("user_id", sender.ID).Str("item", string(itemType)).Msg("Purchase failed")
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ 购买失败，请稍后重试",
				ShowAlert: true,
			})
		}

		// Success - go back to the category the item belongs to
		c.Respond(&tele.CallbackResponse{
			Text: "✅ 购买成功！" + item.Emoji + " " + item.Name,
		})

		balance, _ := h.accountService.GetBalance(ctx, sender.ID)
		
		// Return to the appropriate category
		if item.Category == shop.CategoryAttack {
			caption := shop.FormatAttackItemsMessage(balance)
			markup := shop.BuildAttackItemsPanel()
			h.editShopPhoto(c, caption, markup)
		} else {
			caption := shop.FormatDefenseItemsMessage(balance)
			markup := shop.BuildDefenseItemsPanel()
			h.editShopPhoto(c, caption, markup)
		}
		return nil
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

	// Convert items to display format (use count based)
	var effects []shop.EffectInfo
	for _, item := range inventory.Items {
		// Skip handcuffs as they are shown separately
		if item.ItemType == string(shop.ItemHandcuff) {
			continue
		}
		effects = append(effects, shop.EffectInfo{
			EffectType:   item.ItemType,
			RemainingStr: shop.FormatUseCount(item.UseCount),
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
