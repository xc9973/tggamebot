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
	ItemHandcuff        ItemType = "handcuff"      // 手铐 - 锁定目标
	ItemShield          ItemType = "shield"        // 保护罩 - 防止被打劫
	ItemThornArmor      ItemType = "thorn_armor"   // 荆棘刺甲 - 被打劫时反伤
	ItemBloodthirstSword ItemType = "bloodthirst"  // 饮血剑 - 提升打劫成功率
	// Future items can be added here
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
	Type        ItemType      // 道具类型
	Name        string        // 显示名称
	Emoji       string        // 图标
	Price       int64         // 价格（金币）
	Duration    time.Duration // 时效（0表示一次性使用）
	EffectDuration time.Duration // 效果持续时间（用于手铐锁定目标的时间）
	Description string        // 描述
	Category    ItemCategory  // 分类
	Stackable   bool          // 是否可叠加数量
}

// ShopItems contains all available shop items
// Easily extensible - just add new items to this map
var ShopItems = map[ItemType]ItemConfig{
	ItemHandcuff: {
		Type:           ItemHandcuff,
		Name:           "手铐",
		Emoji:          "🔗",
		Price:          500,
		Duration:       0, // 一次性使用
		EffectDuration: 30 * time.Minute, // 锁定目标30分钟
		Description:    "锁定目标30分钟，使其无法打劫",
		Category:       CategoryAttack,
		Stackable:      true, // 可以购买多个
	},
	ItemShield: {
		Type:        ItemShield,
		Name:        "保护罩",
		Emoji:       "🛡️",
		Price:       500,
		Duration:    6 * time.Hour,
		Description: "6小时内无法被打劫",
		Category:    CategoryDefense,
		Stackable:   false,
	},
	ItemThornArmor: {
		Type:        ItemThornArmor,
		Name:        "荆棘刺甲",
		Emoji:       "🌵",
		Price:       500,
		Duration:    3 * time.Hour,
		Description: "3小时内被打劫成功时，攻击方扣双倍",
		Category:    CategoryPassive,
		Stackable:   false,
	},
	ItemBloodthirstSword: {
		Type:        ItemBloodthirstSword,
		Name:        "饮血剑",
		Emoji:       "🗡️",
		Price:       1000,
		Duration:    30 * time.Minute,
		Description: "30分钟内打劫成功率提升到80%",
		Category:    CategoryAttack,
		Stackable:   false,
	},
}

// GetAllItems returns all shop items in display order
func GetAllItems() []ItemConfig {
	// Define display order
	order := []ItemType{
		ItemHandcuff,
		ItemShield,
		ItemThornArmor,
		ItemBloodthirstSword,
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

// IsTimeBased returns true if the item has a duration (not one-time use)
func (c ItemConfig) IsTimeBased() bool {
	return c.Duration > 0
}

// IsOneTimeUse returns true if the item is consumed on use
func (c ItemConfig) IsOneTimeUse() bool {
	return c.Duration == 0
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
