// Package handler provides Telegram bot command handlers.
// Requirements: 11.1, 11.3 - Daily ranking functionality
package handler

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/service"
)

// RankingHandler handles ranking-related commands.
type RankingHandler struct {
	rankingService *service.RankingService
}

// NewRankingHandler creates a new RankingHandler.
func NewRankingHandler(rankingService *service.RankingService) *RankingHandler {
	return &RankingHandler{
		rankingService: rankingService,
	}
}

// HandleDailyTop handles the /daily_top command.
// Displays today's top winners and losers.
// Requirements: 11.1, 11.3
func (h *RankingHandler) HandleDailyTop(c tele.Context) error {
	ctx := context.Background()

	// Get top winners
	winners, err := h.rankingService.GetDailyWinners(ctx, 10)
	if err != nil {
		return c.Reply("❌ 获取排行榜失败，请稍后重试")
	}

	// Get top losers
	losers, err := h.rankingService.GetDailyLosers(ctx, 10)
	if err != nil {
		return c.Reply("❌ 获取排行榜失败，请稍后重试")
	}

	msg := "📊 今日游戏榜\n"
	msg += "━━━━━━━━━━━━━━━\n"

	// Winners section
	msg += "🏆 赢家榜 TOP 10\n"
	if len(winners) == 0 {
		msg += "暂无数据\n"
	} else {
		medals := []string{"🥇", "🥈", "🥉"}
		for i, winner := range winners {
			rank := fmt.Sprintf("%d.", i+1)
			if i < 3 {
				rank = medals[i]
			}

			displayName := winner.Username
			if displayName == "" {
				displayName = fmt.Sprintf("User%d", winner.UserID)
			}

			msg += fmt.Sprintf("%s %s: +%d\n", rank, displayName, winner.NetProfit)
		}
	}

	msg += "\n━━━━━━━━━━━━━━━\n"

	// Losers section
	msg += "😢 输家榜 TOP 10\n"
	if len(losers) == 0 {
		msg += "暂无数据\n"
	} else {
		for i, loser := range losers {
			rank := fmt.Sprintf("%d.", i+1)

			displayName := loser.Username
			if displayName == "" {
				displayName = fmt.Sprintf("User%d", loser.UserID)
			}

			msg += fmt.Sprintf("%s %s: %d\n", rank, displayName, loser.NetProfit)
		}
	}

	msg += "━━━━━━━━━━━━━━━"

	return c.Reply(msg)
}
