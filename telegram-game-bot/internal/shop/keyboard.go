// Package shop provides shop system for purchasing items.
package shop

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

// Callback data prefixes
const (
	CallbackShopItem    = "shop_item:"    // shop_item:handcuff
	CallbackShopBuy     = "shop_buy:"     // shop_buy:handcuff
	CallbackShopCancel  = "shop_cancel"   // shop_cancel
	CallbackShopRefresh = "shop_refresh"  // shop_refresh
)

// BuildShopPanel creates the main shop panel with item buttons
func BuildShopPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	items := GetAllItems()
	var rows []tele.Row
	
	// Create a button for each item (2 per row)
	var currentRow []tele.Btn
	for i, item := range items {
		btn := markup.Data(
			fmt.Sprintf("%s %s (%d💰)", item.Emoji, item.Name, item.Price),
			CallbackShopItem+string(item.Type),
		)
		currentRow = append(currentRow, btn)
		
		// 2 buttons per row
		if len(currentRow) == 2 || i == len(items)-1 {
			rows = append(rows, markup.Row(currentRow...))
			currentRow = nil
		}
	}
	
	// Add refresh button
	refreshBtn := markup.Data("🔄 刷新", CallbackShopRefresh)
	rows = append(rows, markup.Row(refreshBtn))
	
	markup.Inline(rows...)
	return markup
}

// BuildConfirmPanel creates the purchase confirmation panel
func BuildConfirmPanel(itemType ItemType) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	buyBtn := markup.Data("✅ 购买", CallbackShopBuy+string(itemType))
	cancelBtn := markup.Data("❌ 取消", CallbackShopCancel)
	
	markup.Inline(
		markup.Row(buyBtn, cancelBtn),
	)
	return markup
}

// FormatShopMessage creates the shop welcome message
func FormatShopMessage(balance int64) string {
	msg := "🏪 欢迎来到商店\n"
	msg += "━━━━━━━━━━━━━━━\n"
	msg += fmt.Sprintf("💰 你的余额: %d 金币\n", balance)
	msg += "━━━━━━━━━━━━━━━\n"
	msg += "点击下方按钮查看商品详情："
	return msg
}

// FormatItemDetail creates the item detail message
func FormatItemDetail(item ItemConfig, balance int64) string {
	msg := fmt.Sprintf("%s %s\n", item.Emoji, item.Name)
	msg += "━━━━━━━━━━━━━━━\n"
	msg += fmt.Sprintf("💰 价格: %d 金币\n", item.Price)
	
	if item.IsTimeBased() {
		msg += fmt.Sprintf("⏱️ 时效: %s\n", FormatDuration(item.Duration))
	} else {
		msg += "⏱️ 类型: 一次性使用\n"
	}
	
	msg += fmt.Sprintf("📝 效果: %s\n", item.Description)
	msg += "━━━━━━━━━━━━━━━\n"
	msg += fmt.Sprintf("💰 你的余额: %d 金币\n", balance)
	
	if balance < item.Price {
		msg += "❌ 余额不足！"
	} else {
		msg += "确认购买吗？"
	}
	
	return msg
}

// FormatInventoryMessage creates the inventory display message
func FormatInventoryMessage(handcuffCount int, effects []EffectInfo) string {
	if handcuffCount == 0 && len(effects) == 0 {
		return "🎒 背包为空\n\n去商店购买道具吧！私聊我发送 /start"
	}
	
	msg := "🎒 我的背包\n"
	msg += "━━━━━━━━━━━━━━━\n"
	
	if handcuffCount > 0 {
		item, _ := GetItem(ItemHandcuff)
		msg += fmt.Sprintf("%s %s x%d\n", item.Emoji, item.Name, handcuffCount)
		msg += "   使用方法: 回复目标消息发送 /handcuff\n"
	}
	
	for _, effect := range effects {
		item, ok := GetItem(ItemType(effect.EffectType))
		if !ok {
			continue
		}
		msg += fmt.Sprintf("%s %s\n", item.Emoji, item.Name)
		msg += fmt.Sprintf("   剩余时间: %s\n", effect.RemainingStr)
	}
	
	return msg
}

// EffectInfo holds effect display information
type EffectInfo struct {
	EffectType   string
	RemainingStr string
}

// FormatRemainingTime formats remaining time for display
func FormatRemainingTime(remaining int64) string {
	if remaining <= 0 {
		return "已过期"
	}
	
	hours := remaining / 3600
	minutes := (remaining % 3600) / 60
	
	if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}
