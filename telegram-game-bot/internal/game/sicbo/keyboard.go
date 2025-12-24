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
	
	// Handle special actions with underscores
	if strings.HasPrefix(content, "early_settle") {
		return "early_settle", ""
	}
	
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

// BuildMainPanelWithSettle builds the main betting panel keyboard with early settle button.
// Only shown to the session starter.
func (kb *KeyboardBuilder) BuildMainPanelWithSettle() *tele.ReplyMarkup {
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

	// Row 4: Early settle button [🎲 提前开奖]
	settleRow := []tele.InlineButton{
		{
			Text: "🎲 提前开奖",
			Data: EncodeCallback("early_settle", ""),
		},
	}

	markup.InlineKeyboard = [][]tele.InlineButton{
		bigSmallRow,
		singleRow1,
		singleRow2,
		settleRow,
	}

	return markup
}

// FormatPanelMessage formats the betting panel message with odds and probabilities.
func FormatPanelMessage(remainingTime int, playerCount int, totalBetAmount int64) string {
	msg := "🎲 骰宝 - 下注中\n"
	msg += "┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄\n"
	msg += fmt.Sprintf("⏰ 剩余 %d 秒 | 👥 %d 人 | 💰 %d\n", remainingTime, playerCount, totalBetAmount)
	msg += "┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄\n"
	msg += "📊 赔率说明:\n"
	msg += "• 押大/小: 1:1 (48.6%)\n"
	msg += "• 押单数: 1出现1次=1:1, 2次=2:1, 3次=3:1\n"
	msg += "  (单数出现概率: 42.1%)\n"
	msg += "┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄\n"
	msg += fmt.Sprintf("💰 每次下注: %d 金币", FixedBetAmount)
	return msg
}

// FormatSettlementMessage formats the settlement result message.
func FormatSettlementMessage(dice [3]int, playerResults map[int64]PlayerResult) string {
	total := dice[0] + dice[1] + dice[2]
	isTriple := IsTriple(dice)

	// Header
	msg := "🎰 骰宝开奖\n\n"
	
	// Dice display
	msg += fmt.Sprintf("🎲 %d   🎲 %d   🎲 %d\n", dice[0], dice[1], dice[2])
	
	// Result
	if isTriple {
		msg += fmt.Sprintf("点数 %d 【围骰】\n", total)
	} else if total >= 11 {
		msg += fmt.Sprintf("点数 %d 【大】\n", total)
	} else {
		msg += fmt.Sprintf("点数 %d 【小】\n", total)
	}

	if len(playerResults) == 0 {
		msg += "\n😴 本局无人下注"
		return msg
	}

	// Find top winner and calculate stats
	var topWinner PlayerResult
	var hasWinner bool
	var totalWinners, totalLosers int

	for _, result := range playerResults {
		if result.TotalPayout > 0 {
			totalWinners++
			if result.TotalPayout > topWinner.TotalPayout {
				topWinner = result
				hasWinner = true
			}
		} else if result.TotalPayout < 0 {
			totalLosers++
		}
	}

	// Show top winner
	if hasWinner {
		displayName := topWinner.Username
		if displayName == "" {
			displayName = fmt.Sprintf("%d", topWinner.UserID)
		}
		if !strings.HasPrefix(displayName, "@") {
			displayName = "@" + displayName
		}
		msg += fmt.Sprintf("\n🏆 最大赢家 %s +%d\n", displayName, topWinner.TotalPayout)
	}

	// Player results
	msg += "\n📋 结算:\n"
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
			msg += fmt.Sprintf("🟢 %s +%d\n", displayName, net)
		} else if net < 0 {
			msg += fmt.Sprintf("🔴 %s %d\n", displayName, net)
		} else {
			msg += fmt.Sprintf("⚪ %s ±0\n", displayName)
		}
	}

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
