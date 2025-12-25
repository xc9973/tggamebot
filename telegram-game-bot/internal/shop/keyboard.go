// Package shop provides shop system for purchasing items.
package shop

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

// Callback data prefixes
const (
	CallbackShopItem     = "shop_item:"     // shop_item:handcuff
	CallbackShopBuy      = "shop_buy:"      // shop_buy:handcuff
	CallbackShopCancel   = "shop_cancel"    // shop_cancel
	CallbackShopRefresh  = "shop_refresh"   // shop_refresh
	CallbackShopBag      = "shop_bag"       // shop_bag - view inventory
	CallbackShopGoods    = "shop_goods"     // shop_goods - view goods categories
	CallbackShopAttack   = "shop_attack"    // shop_attack - attack items
	CallbackShopDefense  = "shop_defense"   // shop_defense - defense items
	CallbackShopHome     = "shop_home"      // shop_home - back to main menu
)

// BuildShopPanel creates the main shop panel (first level: Bag | Goods)
// Requirements: 1.1, 1.2 - Display main menu with bag and goods options
func BuildShopPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	markup.InlineKeyboard = [][]tele.InlineButton{
		{
			{Text: "🎒 我的背包", Data: CallbackShopBag},
			{Text: "🛒 商品", Data: CallbackShopGoods},
		},
		{
			{Text: "🔄 刷新", Data: CallbackShopRefresh},
		},
	}
	return markup
}

// BuildGoodsCategoryPanel creates the goods category panel (second level: Attack | Defense)
func BuildGoodsCategoryPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	markup.InlineKeyboard = [][]tele.InlineButton{
		{
			{Text: "⚔️ 攻击道具", Data: CallbackShopAttack},
			{Text: "🛡️ 防御道具", Data: CallbackShopDefense},
		},
		{
			{Text: "🔙 返回", Data: CallbackShopHome},
		},
	}
	return markup
}

// BuildAttackItemsPanel creates the attack items panel
func BuildAttackItemsPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	items := GetItemsByCategory(CategoryAttack)
	var rows [][]tele.InlineButton
	
	// Create a button for each item (2 per row)
	var currentRow []tele.InlineButton
	for i, item := range items {
		btn := tele.InlineButton{
			Text: fmt.Sprintf("%s %s (%d💰)", item.Emoji, item.Name, item.Price),
			Data: CallbackShopItem + string(item.Type),
		}
		currentRow = append(currentRow, btn)
		
		if len(currentRow) == 2 || i == len(items)-1 {
			rows = append(rows, currentRow)
			currentRow = nil
		}
	}
	
	// Add back button
	rows = append(rows, []tele.InlineButton{
		{Text: "🔙 返回", Data: CallbackShopGoods},
	})
	
	markup.InlineKeyboard = rows
	return markup
}

// BuildDefenseItemsPanel creates the defense items panel
func BuildDefenseItemsPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	// Get defense and passive items
	defenseItems := GetItemsByCategory(CategoryDefense)
	passiveItems := GetItemsByCategory(CategoryPassive)
	items := append(defenseItems, passiveItems...)
	
	var rows [][]tele.InlineButton
	
	// Create a button for each item (2 per row)
	var currentRow []tele.InlineButton
	for i, item := range items {
		btn := tele.InlineButton{
			Text: fmt.Sprintf("%s %s (%d💰)", item.Emoji, item.Name, item.Price),
			Data: CallbackShopItem + string(item.Type),
		}
		currentRow = append(currentRow, btn)
		
		if len(currentRow) == 2 || i == len(items)-1 {
			rows = append(rows, currentRow)
			currentRow = nil
		}
	}
	
	// Add back button
	rows = append(rows, []tele.InlineButton{
		{Text: "🔙 返回", Data: CallbackShopGoods},
	})
	
	markup.InlineKeyboard = rows
	return markup
}

// BuildConfirmPanel creates the purchase confirmation panel
func BuildConfirmPanel(itemType ItemType) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	
	// Determine which category to go back to
	item, ok := GetItem(itemType)
	backData := CallbackShopGoods
	if ok {
		if item.Category == CategoryAttack {
			backData = CallbackShopAttack
		} else {
			backData = CallbackShopDefense
		}
	}
	
	markup.InlineKeyboard = [][]tele.InlineButton{
		{
			{Text: "✅ 购买", Data: CallbackShopBuy + string(itemType)},
			{Text: "❌ 取消", Data: backData},
		},
	}
	return markup
}

// FormatShopMessage creates the shop welcome message (main menu)
func FormatShopMessage(balance int64) string {
	msg := fmt.Sprintf("🏪 游戏商店\n💰 余额: %d 金币\n\n", balance)
	msg += "欢迎来到游戏商店！\n"
	msg += "请选择要查看的内容：\n\n"
	msg += "🎒 背包 - 查看已购买的道具\n"
	msg += "🛒 商品 - 浏览和购买道具"
	return msg
}

// FormatGoodsCategoryMessage creates the goods category message
func FormatGoodsCategoryMessage(balance int64) string {
	msg := fmt.Sprintf("🛒 商品分类\n💰 余额: %d 金币\n\n", balance)
	msg += "请选择道具类型：\n\n"
	msg += "⚔️ 攻击道具 - 用于打劫的道具\n"
	msg += "🛡️ 防御道具 - 用于防御的道具"
	return msg
}

// FormatAttackItemsMessage creates the attack items list message
func FormatAttackItemsMessage(balance int64) string {
	msg := fmt.Sprintf("⚔️ 攻击道具\n💰 余额: %d 金币\n\n", balance)
	
	items := GetItemsByCategory(CategoryAttack)
	for _, item := range items {
		msg += fmt.Sprintf("%s %s - %d💰\n", item.Emoji, item.Name, item.Price)
		msg += fmt.Sprintf("   📦 %d次", item.UseCount)
		if item.HasDailyLimit() {
			msg += fmt.Sprintf(" | 🔒 限购%d/日", item.DailyLimit)
		}
		msg += "\n"
	}
	
	msg += "\n👇 点击按钮查看详情"
	return msg
}

// FormatDefenseItemsMessage creates the defense items list message
func FormatDefenseItemsMessage(balance int64) string {
	msg := fmt.Sprintf("🛡️ 防御道具\n💰 余额: %d 金币\n\n", balance)
	
	// Get defense and passive items
	defenseItems := GetItemsByCategory(CategoryDefense)
	passiveItems := GetItemsByCategory(CategoryPassive)
	items := append(defenseItems, passiveItems...)
	
	for _, item := range items {
		msg += fmt.Sprintf("%s %s - %d💰\n", item.Emoji, item.Name, item.Price)
		msg += fmt.Sprintf("   📦 %d次", item.UseCount)
		if item.HasDailyLimit() {
			msg += fmt.Sprintf(" | 🔒 限购%d/日", item.DailyLimit)
		}
		msg += "\n"
	}
	
	msg += "\n👇 点击按钮查看详情"
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
			{Text: "🔙 返回", Data: CallbackShopHome},
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
