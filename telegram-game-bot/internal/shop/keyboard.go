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
	CallbackShopBag     = "shop_bag"      // shop_bag - view inventory
)

// BuildShopPanel creates the main shop panel with item buttons
// Requirements: 1.1, 1.2 - Display 8 items with use count and daily limit info
func BuildShopPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	items := GetAllItems()
	var rows [][]tele.InlineButton
	
	// Create a button for each item (2 per row)
	// Display: emoji name (price💰)
	var currentRow []tele.InlineButton
	for i, item := range items {
		btn := tele.InlineButton{
			Text: fmt.Sprintf("%s %s (%d💰)", item.Emoji, item.Name, item.Price),
			Data: CallbackShopItem + string(item.Type),
		}
		currentRow = append(currentRow, btn)
		
		// 2 buttons per row
		if len(currentRow) == 2 || i == len(items)-1 {
			rows = append(rows, currentRow)
			currentRow = nil
		}
	}
	
	// Add bag and refresh buttons
	rows = append(rows, []tele.InlineButton{
		{Text: "🎒 我的背包", Data: CallbackShopBag},
		{Text: "🔄 刷新", Data: CallbackShopRefresh},
	})
	
	markup.InlineKeyboard = rows
	return markup
}

// BuildConfirmPanel creates the purchase confirmation panel
func BuildConfirmPanel(itemType ItemType) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	markup.InlineKeyboard = [][]tele.InlineButton{
		{
			{Text: "✅ 购买", Data: CallbackShopBuy + string(itemType)},
			{Text: "❌ 取消", Data: CallbackShopCancel},
		},
	}
	return markup
}

// FormatShopMessage creates the shop welcome message
// Requirements: 1.1, 1.2 - Display all 8 items with name, price, use count, and description
func FormatShopMessage(balance int64) string {
	msg := fmt.Sprintf("🏪 游戏商店\n💰 余额: %d 金币\n\n", balance)
	
	// List all items with details
	items := GetAllItems()
	for _, item := range items {
		msg += fmt.Sprintf("%s %s - %d💰\n", item.Emoji, item.Name, item.Price)
		msg += fmt.Sprintf("   📦 %d次", item.UseCount)
		if item.HasDailyLimit() {
			msg += fmt.Sprintf(" | 🔒 限购%d/日", item.DailyLimit)
		}
		msg += "\n"
	}
	
	msg += "\n👇 点击按钮购买道具"
	return msg
}

// FormatItemDetail creates the item detail message
// Requirements: 1.2 - Show item name, price, use count, and daily limit info
func FormatItemDetail(item ItemConfig, balance int64) string {
	msg := fmt.Sprintf("%s %s\n\n", item.Emoji, item.Name)
	msg += fmt.Sprintf("💰 价格: %d 金币\n", item.Price)
	msg += fmt.Sprintf("📦 使用次数: %d次\n", item.UseCount)

	if item.HasDailyLimit() {
		msg += fmt.Sprintf("🔒 每日限购: %d次\n", item.DailyLimit)
	}

	msg += fmt.Sprintf("📝 %s\n\n", item.Description)
	msg += fmt.Sprintf("💰 你的余额: %d 金币\n\n", balance)

	if balance < item.Price {
		msg += "❌ 余额不足"
	} else {
		msg += "✅ 确认购买？"
	}

	return msg
}

// FormatItemDetailWithDailyCount creates the item detail message with daily purchase count
// Requirements: 1.2, 2.9, 3.8, 7.8 - Show daily limit and current purchase count
func FormatItemDetailWithDailyCount(item ItemConfig, balance int64, dailyCount int) string {
	msg := fmt.Sprintf("%s %s\n\n", item.Emoji, item.Name)
	msg += fmt.Sprintf("💰 价格: %d 金币\n", item.Price)
	msg += fmt.Sprintf("📦 使用次数: %d次\n", item.UseCount)

	if item.HasDailyLimit() {
		msg += fmt.Sprintf("🔒 每日限购: %d/%d次\n", dailyCount, item.DailyLimit)
	}

	msg += fmt.Sprintf("📝 %s\n\n", item.Description)
	msg += fmt.Sprintf("💰 你的余额: %d 金币\n\n", balance)

	// Check daily limit first
	if item.HasDailyLimit() && dailyCount >= item.DailyLimit {
		msg += "❌ 今日购买次数已达上限"
	} else if balance < item.Price {
		msg += "❌ 余额不足"
	} else {
		msg += "✅ 确认购买？"
	}

	return msg
}

// FormatInventoryMessage creates the inventory display message
// Requirements: 11.2 - Show item name, quantity (for Handcuffs), and remaining use count (for other items)
func FormatInventoryMessage(balance int64, handcuffCount int, effects []EffectInfo) string {
	msg := "🎒 我的背包\n\n"
	msg += fmt.Sprintf("💰 余额: %d 金币\n\n", balance)
	
	if handcuffCount == 0 && len(effects) == 0 {
		msg += "📭 背包空空如也~"
	} else {
		msg += "📦 道具列表:\n"
		msg += "─────────────\n"
		
		if handcuffCount > 0 {
			item, _ := GetItem(ItemHandcuff)
			msg += fmt.Sprintf("%s %s ×%d\n", item.Emoji, item.Name, handcuffCount)
			msg += "   └ 用法: 回复消息 /handcuff\n"
		}
		
		for _, effect := range effects {
			item, ok := GetItem(ItemType(effect.EffectType))
			if !ok {
				continue
			}
			msg += fmt.Sprintf("%s %s - %s\n", item.Emoji, item.Name, effect.RemainingStr)
		}
	}
	
	return msg
}

// BuildBagPanel creates the bag panel with back button
func BuildBagPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	markup.InlineKeyboard = [][]tele.InlineButton{
		{
			{Text: "🔙 返回商店", Data: CallbackShopCancel},
			{Text: "🔄 刷新", Data: CallbackShopBag},
		},
	}
	return markup
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

// FormatUseCount formats use count for display
func FormatUseCount(useCount int) string {
	if useCount <= 0 {
		return "已用完"
	}
	return fmt.Sprintf("剩余%d次", useCount)
}
