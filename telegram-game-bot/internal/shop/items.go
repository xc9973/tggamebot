// Package shop provides shop system for purchasing items.
// Requirements: Shop System - Allow users to purchase items with game coins
package shop

import (
	"time"
)

// ItemType represents the type of shop item
type ItemType string

// Item types - easily extensible for future items
const (
	ItemHandcuff         ItemType = "handcuff"        // 手铐 - 锁定目标
	ItemKey              ItemType = "key"             // 钥匙 - 解除手铐锁定
	ItemShield           ItemType = "shield"          // 保护罩 - 防止被打劫
	ItemThornArmor       ItemType = "thorn_armor"     // 荆棘刺甲 - 被打劫时反伤
	ItemBloodthirstSword ItemType = "bloodthirst"     // 饮血剑 - 提升打劫成功率
	ItemBluntKnife       ItemType = "blunt_knife"     // 钝刀 - 无视防御，打劫1-100
	ItemGreatSword       ItemType = "great_sword"     // 大宝剑 - 无视防御，0.01%打劫90%
	ItemGoldenCassock    ItemType = "golden_cassock"  // 紫金袈裟 - 攻击者失去防御道具
	ItemEmperorClothes   ItemType = "emperor_clothes" // 皇帝的新衣 - 免疫所有攻击
)

// ItemCategory represents the category of an item
type ItemCategory string

const (
	CategoryAttack  ItemCategory = "attack"  // 攻击型道具
	CategoryDefense ItemCategory = "defense" // 防御型道具
	CategoryPassive ItemCategory = "passive" // 被动型道具
)

// ItemConfig holds the configuration for a shop item
type ItemConfig struct {
	Type           ItemType      // 道具类型
	Name           string        // 显示名称
	Emoji          string        // 图标
	Price          int64         // 价格（金币）
	UseCount       int           // 使用次数
	EffectDuration time.Duration // 效果持续时间（用于手铐锁定目标的时间）
	Description    string        // 描述
	Category       ItemCategory  // 分类
	DailyLimit     int           // 每日购买限制（0表示无限制）
	BypassDefense  bool          // 是否无视普通防御（保护罩、荆棘刺甲）
	ImmuneBypass   bool          // 是否免疫无视防御攻击
}

// ShopItems contains all available shop items
// Easily extensible - just add new items to this map
var ShopItems = map[ItemType]ItemConfig{
	ItemHandcuff: {
		Type:           ItemHandcuff,
		Name:           "手铐",
		Emoji:          "🔗",
		Price:          500,
		UseCount:       1,
		EffectDuration: 30 * time.Minute, // 锁定目标30分钟
		Description:    "锁定目标30分钟，使其无法打劫",
		Category:       CategoryAttack,
		DailyLimit:     5,
	},
	ItemKey: {
		Type:        ItemKey,
		Name:        "钥匙",
		Emoji:       "🔑",
		Price:       300,
		UseCount:    1,
		Description: "解除自己身上的手铐锁定",
		Category:    CategoryDefense,
	},
	ItemShield: {
		Type:        ItemShield,
		Name:        "保护罩",
		Emoji:       "🛡️",
		Price:       500,
		UseCount:    10,
		Description: "防止被打劫10次",
		Category:    CategoryDefense,
		DailyLimit:  2,
	},
	ItemThornArmor: {
		Type:        ItemThornArmor,
		Name:        "荆棘刺甲",
		Emoji:       "🌵",
		Price:       500,
		UseCount:    5,
		Description: "被打劫成功时攻击方扣双倍（5次）",
		Category:    CategoryPassive,
	},
	ItemBloodthirstSword: {
		Type:        ItemBloodthirstSword,
		Name:        "饮血剑",
		Emoji:       "🗡️",
		Price:       1000,
		UseCount:    10,
		Description: "打劫成功率提升到80%（10次）",
		Category:    CategoryAttack,
	},
	ItemBluntKnife: {
		Type:          ItemBluntKnife,
		Name:          "钝刀",
		Emoji:         "🔪",
		Price:         1000,
		UseCount:      10,
		Description:   "无视防御，打劫1-100随机（10次）",
		Category:      CategoryAttack,
		BypassDefense: true,
	},
	ItemGreatSword: {
		Type:          ItemGreatSword,
		Name:          "大宝剑",
		Emoji:         "⚔️",
		Price:         10000,
		UseCount:      3,
		Description:   "无视防御，1%打劫90%（3次）",
		Category:      CategoryAttack,
		DailyLimit:    1,
		BypassDefense: true,
	},
	ItemGoldenCassock: {
		Type:        ItemGoldenCassock,
		Name:        "紫金袈裟",
		Emoji:       "👘",
		Price:       10000,
		UseCount:    3,
		Description: "攻击者失去所有防御道具（3次）",
		Category:    CategoryDefense,
	},
	ItemEmperorClothes: {
		Type:         ItemEmperorClothes,
		Name:         "皇帝的新衣",
		Emoji:        "👑",
		Price:        5000,
		UseCount:     3,
		Description:  "免疫所有攻击（3次）",
		Category:     CategoryDefense,
		ImmuneBypass: true,
	},
}

// GetAllItems returns all shop items in display order
func GetAllItems() []ItemConfig {
	// Define display order - 9 items total
	order := []ItemType{
		ItemHandcuff,
		ItemKey,
		ItemShield,
		ItemThornArmor,
		ItemBloodthirstSword,
		ItemBluntKnife,
		ItemGreatSword,
		ItemGoldenCassock,
		ItemEmperorClothes,
	}

	items := make([]ItemConfig, 0, len(order))
	for _, itemType := range order {
		if item, ok := ShopItems[itemType]; ok {
			items = append(items, item)
		}
	}
	return items
}

// GetItem returns the item config for a given type
func GetItem(itemType ItemType) (ItemConfig, bool) {
	item, ok := ShopItems[itemType]
	return item, ok
}

// GetItemsByCategory returns all items of a specific category
func GetItemsByCategory(category ItemCategory) []ItemConfig {
	var items []ItemConfig
	for _, item := range GetAllItems() {
		if item.Category == category {
			items = append(items, item)
		}
	}
	return items
}

// HasDailyLimit returns true if the item has a daily purchase limit
func (c ItemConfig) HasDailyLimit() bool {
	return c.DailyLimit > 0
}

// CanBypassDefense returns true if the item can bypass normal defenses
func (c ItemConfig) CanBypassDefense() bool {
	return c.BypassDefense
}

// IsImmuneToBypass returns true if the item is immune to bypass attacks
func (c ItemConfig) IsImmuneToBypass() bool {
	return c.ImmuneBypass
}

// FormatDuration returns a human-readable duration string
func FormatDuration(d time.Duration) string {
	if d >= time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1小时"
		}
		return string(rune('0'+hours)) + "小时"
	}
	if d >= time.Minute {
		mins := int(d.Minutes())
		return string(rune('0'+mins/10)) + string(rune('0'+mins%10)) + "分钟"
	}
	return "即时"
}
