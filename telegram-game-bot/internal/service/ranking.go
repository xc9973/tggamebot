package service

import (
	"context"
	"time"

	"telegram-game-bot/internal/model"
	"telegram-game-bot/internal/repository"
)

// RankingService handles ranking and leaderboard operations.
// Requirements: 1.5, 11.1, 11.2, 11.3 - Ranking functionality
type RankingService struct {
	userRepo *repository.UserRepository
	txRepo   *repository.TransactionRepository
	timezone *time.Location
}

// NewRankingService creates a new RankingService instance.
func NewRankingService(
	userRepo *repository.UserRepository,
	txRepo *repository.TransactionRepository,
	timezone *time.Location,
) *RankingService {
	if timezone == nil {
		timezone = time.UTC
	}
	return &RankingService{
		userRepo: userRepo,
		txRepo:   txRepo,
		timezone: timezone,
	}
}

// GetTopUsers retrieves the top users by balance.
// Requirements: 1.5 - Display top 10 users by balance
func (s *RankingService) GetTopUsers(ctx context.Context, limit int) ([]*model.User, error) {
	return s.userRepo.GetTopUsers(ctx, limit)
}

// GetDailyWinners retrieves today's top winners (users with most profit).
// Requirements: 11.1, 11.3 - Show top 10 winners (most profit)
func (s *RankingService) GetDailyWinners(ctx context.Context, limit int) ([]*model.DailyRank, error) {
	today := time.Now().In(s.timezone)
	return s.txRepo.GetDailyWinners(ctx, today, limit)
}

// GetDailyLosers retrieves today's top losers (users with most loss).
// Requirements: 11.1, 11.3 - Show top 10 losers (most loss)
func (s *RankingService) GetDailyLosers(ctx context.Context, limit int) ([]*model.DailyRank, error) {
	today := time.Now().In(s.timezone)
	return s.txRepo.GetDailyLosers(ctx, today, limit)
}

// GetDailyWinnersForDate retrieves winners for a specific date.
func (s *RankingService) GetDailyWinnersForDate(ctx context.Context, date time.Time, limit int) ([]*model.DailyRank, error) {
	return s.txRepo.GetDailyWinners(ctx, date, limit)
}

// GetDailyLosersForDate retrieves losers for a specific date.
func (s *RankingService) GetDailyLosersForDate(ctx context.Context, date time.Time, limit int) ([]*model.DailyRank, error) {
	return s.txRepo.GetDailyLosers(ctx, date, limit)
}

// GetDailyStats retrieves all daily game statistics for today.
// Requirements: 11.2 - Track daily net profit/loss for each user
func (s *RankingService) GetDailyStats(ctx context.Context) ([]*model.DailyRank, error) {
	today := time.Now().In(s.timezone)
	return s.txRepo.GetDailyStats(ctx, today)
}

// GetUserDailyProfit retrieves a specific user's profit for today.
func (s *RankingService) GetUserDailyProfit(ctx context.Context, userID int64) (int64, error) {
	today := time.Now().In(s.timezone)
	return s.txRepo.GetUserDailyProfit(ctx, userID, today)
}
