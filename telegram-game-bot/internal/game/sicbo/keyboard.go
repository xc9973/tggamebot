// Package sicbo implements the Sic Bo keyboard builder for Telegram inline keyboards.
// Requirements: 5.6
package sicbo

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

const (
	// CallbackPrefix is the prefix for all SicBo callback data
	CallbackPrefix = "sicbo_"
)

// SingleNumbers are the numbers available for single number bets
var SingleNumbers = []int{1, 2, 3, 4, 5, 6}

// KeyboardBuilder builds Telegram inline keyboards for SicBo game.
type KeyboardBuilder struct{}

// NewKeyboardBuilder creates a new KeyboardBuilder instance.
func NewKeyboardBuilder() *KeyboardBuilder {
	return &KeyboardBuilder{}
}

// EncodeCallback encodes an action and parameter into callback data.
func EncodeCallback(action string, param string) string {
	if param != "" {
		return fmt.Sprintf("%s%s_%s", CallbackPrefix, action, param)
	}
	return fmt.Sprintf("%s%s", CallbackPrefix, action)
}

// DecodeCallback decodes callback data into action and parameter.
func DecodeCallback(data string) (action string, param string) {
	if !strings.HasPrefix(data, CallbackPrefix) {
		return "", ""
	}

	content := strings.TrimPrefix(data, CallbackPrefix)
	parts := strings.SplitN(content, "_", 2)
	action = parts[0]
	if len(parts) > 1 {
		param = parts[1]
	}
	return action, param
}

// BuildMainPanel builds the main betting panel keyboard.
// Layout:
//   - Row 1: [押大] [押小]
//   - Row 2: [押1] [押2] [押3]
//   - Row 3: [押4] [押5] [押6]
//
// Requirements: 5.6
func (kb *KeyboardBuilder) BuildMainPanel() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	// Row 1: Big/Small [押大] [押小]
	bigSmallRow := []tele.InlineButton{
		{
			Text: "押大",
			Data: EncodeCallback("big", ""),
		},
		{
			Text: "押小",
			Data: EncodeCallback("small", ""),
		},
	}

	// Row 2: Single numbers [押1] [押2] [押3]
	singleRow1 := []tele.InlineButton{
		{
			Text: "押1",
			Data: EncodeCallback("single", "1"),
		},
		{
			Text: "押2",
			Data: EncodeCallback("single", "2"),
		},
		{
			Text: "押3",
			Data: EncodeCallback("single", "3"),
		},
	}

	// Row 3: Single numbers [押4] [押5] [押6]
	singleRow2 := []tele.InlineButton{
		{
			Text: "押4",
			Data: EncodeCallback("single", "4"),
		},
		{
			Text: "押5",
			Data: EncodeCallback("single", "5"),
		},
		{
			Text: "押6",
			Data: EncodeCallback("single", "6"),
		},
	}

	markup.InlineKeyboard = [][]tele.InlineButton{
		bigSmallRow,
		singleRow1,
		singleRow2,
	}

	return markup
}

// FormatPanelMessage formats the betting panel message.
func FormatPanelMessage(remainingTime int, playerCount int, totalBetAmount int64) string {
	msg := "🎲 骰宝 - 下注中\n"
	msg += fmt.Sprintf("⏰ 剩余 %d 秒 | 👥 %d 人 | 💰 %d\n", remainingTime, playerCount, totalBetAmount)
	msg += "\n"
	msg += fmt.Sprintf("点击按钮下注 (每次 %d 金币)", FixedBetAmount)
	return msg
}

// FormatSettlementMessage formats the settlement result message.
func FormatSettlementMessage(dice [3]int, playerResults map[int64]PlayerResult) string {
	diceStr := fmt.Sprintf("🎲%d 🎲%d 🎲%d", dice[0], dice[1], dice[2])
	total := dice[0] + dice[1] + dice[2]
	isTriple := IsTriple(dice)

	msg := "🎰 骰宝结算\n"
	msg += "━━━━━━━━━━━━━━━\n"
	msg += fmt.Sprintf("骰子: %s = %d", diceStr, total)

	if isTriple {
		msg += " (围骰)\n"
	} else if total >= 11 {
		msg += " (大)\n"
	} else {
		msg += " (小)\n"
	}

	msg += "━━━━━━━━━━━━━━━\n"

	if len(playerResults) == 0 {
		msg += "本局无人下注\n"
	} else {
		for _, result := range playerResults {
			net := result.TotalPayout
			displayName := result.Username
			if displayName == "" {
				displayName = fmt.Sprintf("%d", result.UserID)
			}
			if !strings.HasPrefix(displayName, "@") {
				displayName = "@" + displayName
			}

			if net > 0 {
				msg += fmt.Sprintf("🎉 %s +%d\n", displayName, net)
			} else if net < 0 {
				msg += fmt.Sprintf("😢 %s %d\n", displayName, net)
			} else {
				msg += fmt.Sprintf("😐 %s ±0\n", displayName)
			}
		}
	}

	msg += "━━━━━━━━━━━━━━━\n"
	msg += "游戏结束"

	return msg
}

// PlayerResult represents a player's result in a SicBo game.
type PlayerResult struct {
	UserID      int64
	Username    string
	TotalBet    int64
	TotalPayout int64
}

// FormatMyBets formats a user's bet list.
func FormatMyBets(bets map[string]int64) string {
	if len(bets) == 0 {
		return "您还没有下注"
	}

	msg := "📋 您的押注:\n"
	msg += "━━━━━━━━━━━━━━━\n"

	var totalAmount int64
	for key, amount := range bets {
		betName := formatBetKey(key)
		msg += fmt.Sprintf("• %s: %d 金币\n", betName, amount)
		totalAmount += amount
	}

	msg += "━━━━━━━━━━━━━━━\n"
	msg += fmt.Sprintf("💰 总计: %d 金币", totalAmount)

	return msg
}

// formatBetKey converts a bet key to a display name.
func formatBetKey(key string) string {
	switch key {
	case "big":
		return "大"
	case "small":
		return "小"
	default:
		// Check for single_N format
		var num int
		if _, err := fmt.Sscanf(key, "single_%d", &num); err == nil {
			return fmt.Sprintf("单一数字 %d", num)
		}
		return key
	}
}
