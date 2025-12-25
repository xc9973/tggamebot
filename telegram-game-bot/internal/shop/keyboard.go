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
func BuildShopPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	items := GetAllItems()
	var rows [][]tele.InlineButton
	
	// Create a button for each item (2 per row)
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

// FormatShopMessage creates the shop welcome message with ASCII art header
func FormatShopMessage(balance int64) string {
	msg := "┏━━━━━━━━━━━━━━━━┓\n"
	msg += "┃    🛒 游戏商店    ┃\n"
	msg += "┗━━━━━━━━━━━━━━━━┛\n\n"
	msg += fmt.Sprintf("💰 余额: %d 金币\n\n", balance)
	msg += "👇 选择要购买的道具"
	return msg
}

// FormatItemDetail creates the item detail message
func FormatItemDetail(item ItemConfig, balance int64) string {
	msg := "┏━━━━━━━━━━━━━━━━┓\n"
	msg += fmt.Sprintf("┃  %s %s\n", item.Emoji, item.Name)
	msg += "┗━━━━━━━━━━━━━━━━┛\n\n"
	msg += fmt.Sprintf("💰 价格: %d 金币\n", item.Price)
	
	if item.IsTimeBased() {
		msg += fmt.Sprintf("⏱ 时效: %s\n", FormatDuration(item.Duration))
	} else {
		msg += "📦 类型: 一次性道具\n"
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

// FormatInventoryMessage creates the inventory display message
func FormatInventoryMessage(balance int64, handcuffCount int, effects []EffectInfo) string {
	msg := "┏━━━━━━━━━━━━━━━━┓\n"
	msg += "┃    🎒 我的背包    ┃\n"
	msg += "┗━━━━━━━━━━━━━━━━┛\n\n"
	msg += fmt.Sprintf("💰 余额: %d 金币\n\n", balance)
	
	if handcuffCount == 0 && len(effects) == 0 {
		msg += "📭 背包空空如也~"
	} else {
		msg += "📦 道具列表:\n"
		if handcuffCount > 0 {
			item, _ := GetItem(ItemHandcuff)
			msg += fmt.Sprintf("  • %s %s ×%d\n", item.Emoji, item.Name, handcuffCount)
			msg += "    用法: 回复消息 /handcuff\n"
		}
		
		for _, effect := range effects {
			item, ok := GetItem(ItemType(effect.EffectType))
			if !ok {
				continue
			}
			msg += fmt.Sprintf("  • %s %s (%s)\n", item.Emoji, item.Name, effect.RemainingStr)
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
