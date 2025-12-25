// Package handler provides Telegram bot command handlers.
// Requirements: 3.1, 4.1, 5.1, 5.5 - Game functionality
package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	tele "gopkg.in/telebot.v3"

	"telegram-game-bot/internal/config"
	"telegram-game-bot/internal/game"
	"telegram-game-bot/internal/game/dice"
	"telegram-game-bot/internal/game/rob"
	"telegram-game-bot/internal/game/sicbo"
	"telegram-game-bot/internal/game/slot"
	"telegram-game-bot/internal/model"
	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/service"
)

const (
	// MessageDeleteInterval is the interval for auto-deleting bot messages (30 minutes)
	MessageDeleteInterval = 30 * time.Minute
)

// BetTier represents a balance tier with its max bet limit
type BetTier struct {
	MinBalance int64 // Minimum balance for this tier
	MaxBet     int64 // Maximum bet allowed for this tier
}

// BetTiers defines the tiered betting limits based on balance
// Higher balance = higher max bet allowed
var BetTiers = []BetTier{
	{MinBalance: 500000, MaxBet: 10000}, // 50万+ 余额: 最大下注 1万
	{MinBalance: 100000, MaxBet: 5000},  // 10万-50万 余额: 最大下注 5千
	{MinBalance: 0, MaxBet: 3000},       // 10万以下: 最大下注 3千
}

// TrackedMessage represents a message to be deleted later
type TrackedMessage struct {
	ChatID    int64
	MessageID int
	SentAt    time.Time
}

// GameHandler handles game-related commands.
type GameHandler struct {
	cfg             *config.Config
	accountService  *service.AccountService
	gameRegistry    *game.Registry
	sicboGame       *sicbo.SicBoGame
	robGame         *rob.RobGame
	userLock        *lock.UserLock
	cooldowns       sync.Map // map[string]time.Time - key: "userID:game"
	trackedMessages []TrackedMessage
	messagesMu      sync.Mutex
	sicboPanels     sync.Map // map[int64]int - chatID -> panelMessageID
}

// NewGameHandler creates a new GameHandler.
func NewGameHandler(
	cfg *config.Config,
	accountService *service.AccountService,
	gameRegistry *game.Registry,
	sicboGame *sicbo.SicBoGame,
	robGame *rob.RobGame,
	userLock *lock.UserLock,
) *GameHandler {
	h := &GameHandler{
		cfg:             cfg,
		accountService:  accountService,
		gameRegistry:    gameRegistry,
		sicboGame:       sicboGame,
		robGame:         robGame,
		userLock:        userLock,
		trackedMessages: make([]TrackedMessage, 0),
	}
	return h
}

// StartMessageCleaner starts the background goroutine to delete old messages.
func (h *GameHandler) StartMessageCleaner(bot *tele.Bot) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
		defer ticker.Stop()

		for range ticker.C {
			h.cleanOldMessages(bot)
		}
	}()
}

// cleanOldMessages deletes messages older than MessageDeleteInterval.
func (h *GameHandler) cleanOldMessages(bot *tele.Bot) {
	h.messagesMu.Lock()
	defer h.messagesMu.Unlock()

	now := time.Now()
	remaining := make([]TrackedMessage, 0)

	for _, msg := range h.trackedMessages {
		if now.Sub(msg.SentAt) >= MessageDeleteInterval {
			// Try to delete the message
			err := bot.Delete(&tele.Message{
				ID:   msg.MessageID,
				Chat: &tele.Chat{ID: msg.ChatID},
			})
			if err != nil {
				log.Debug().Err(err).Int("msg_id", msg.MessageID).Msg("Failed to delete old message")
			}
		} else {
			remaining = append(remaining, msg)
		}
	}

	h.trackedMessages = remaining
}

// trackMessage adds a message to the tracking list for later deletion.
func (h *GameHandler) trackMessage(chatID int64, messageID int) {
	h.messagesMu.Lock()
	defer h.messagesMu.Unlock()

	h.trackedMessages = append(h.trackedMessages, TrackedMessage{
		ChatID:    chatID,
		MessageID: messageID,
		SentAt:    time.Now(),
	})
}

// getEffectiveMaxBet returns the max bet based on user's balance using tiered limits.
// Tiered limits take priority over config max bet.
func (h *GameHandler) getEffectiveMaxBet(balance int64, configMaxBet int64) int64 {
	// Find the appropriate tier based on balance
	for _, tier := range BetTiers {
		if balance >= tier.MinBalance {
			return tier.MaxBet
		}
	}
	// Fallback to config max bet if no tier matches
	return configMaxBet
}

// getBalanceTierInfo returns the current tier's max bet and threshold for error messages
func getBalanceTierInfo(balance int64) (maxBet int64, threshold int64) {
	for _, tier := range BetTiers {
		if balance >= tier.MinBalance {
			return tier.MaxBet, tier.MinBalance
		}
	}
	return BetTiers[len(BetTiers)-1].MaxBet, 0
}

// checkCooldown checks if user is in cooldown for a game.
// Returns remaining seconds if in cooldown, 0 otherwise.
func (h *GameHandler) checkCooldown(userID int64, gameName string, cooldownSecs int) int {
	key := fmt.Sprintf("%d:%s", userID, gameName)
	if lastTime, ok := h.cooldowns.Load(key); ok {
		elapsed := time.Since(lastTime.(time.Time))
		remaining := time.Duration(cooldownSecs)*time.Second - elapsed
		if remaining > 0 {
			return int(remaining.Seconds()) + 1
		}
	}
	return 0
}

// setCooldown sets the cooldown for a user and game.
func (h *GameHandler) setCooldown(userID int64, gameName string) {
	key := fmt.Sprintf("%d:%s", userID, gameName)
	h.cooldowns.Store(key, time.Now())
}

// HandleDice handles the /dice command.
// Requirements: 3.1
func (h *GameHandler) HandleDice(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse bet amount
	args := c.Args()
	if len(args) < 1 {
		return c.Reply("❌ 用法: /dice <金额>\n例如: /dice 100")
	}

	bet, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || bet <= 0 {
		return c.Reply("❌ 请输入有效的下注金额")
	}

	// Check cooldown (3 seconds)
	cooldownSecs := 3
	if remaining := h.checkCooldown(sender.ID, "dice", cooldownSecs); remaining > 0 {
		return c.Reply(fmt.Sprintf("⏰ 请等待 %d 秒后再玩", remaining))
	}

	// Ensure user exists
	username := sender.Username
	if username == "" {
		username = sender.FirstName
	}
	_, _, err = h.accountService.EnsureUser(ctx, sender.ID, username)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Acquire lock
	h.userLock.Lock(sender.ID)
	defer h.userLock.Unlock(sender.ID)

	// Check balance
	balance, err := h.accountService.GetBalance(ctx, sender.ID)
	if err != nil {
		return c.Reply("❌ 获取余额失败")
	}

	// Check max bet based on balance
	maxBet := h.getEffectiveMaxBet(balance, h.cfg.Games.Dice.MaxBet)
	if bet > maxBet {
		tierMaxBet, tierThreshold := getBalanceTierInfo(balance)
		if tierThreshold > 0 {
			return c.Reply(fmt.Sprintf("❌ 余额超过 %d，单次下注上限为 %d", tierThreshold, tierMaxBet))
		}
		return c.Reply(fmt.Sprintf("❌ 最大下注金额为 %d", maxBet))
	}

	if balance < bet {
		return c.Reply("❌ 余额不足")
	}

	// Deduct bet first
	desc := fmt.Sprintf("骰子游戏下注 %d", bet)
	_, err = h.accountService.UpdateBalance(ctx, sender.ID, -bet, model.TxTypeDice, &desc)
	if err != nil {
		return c.Reply("❌ 扣款失败，请稍后重试")
	}

	// Send two dice
	dice1Msg, err := c.Bot().Send(c.Chat(), tele.Cube)
	if err != nil {
		// Refund on error
		h.accountService.UpdateBalance(ctx, sender.ID, bet, model.TxTypeDice, nil)
		return c.Reply("❌ 发送骰子失败")
	}
	h.trackMessage(c.Chat().ID, dice1Msg.ID)

	// Wait a bit before sending second dice
	time.Sleep(500 * time.Millisecond)

	dice2Msg, err := c.Bot().Send(c.Chat(), tele.Cube)
	if err != nil {
		// Refund on error
		h.accountService.UpdateBalance(ctx, sender.ID, bet, model.TxTypeDice, nil)
		return c.Reply("❌ 发送骰子失败")
	}
	h.trackMessage(c.Chat().ID, dice2Msg.ID)

	// Get dice values
	dice1Val := dice1Msg.Dice.Value
	dice2Val := dice2Msg.Dice.Value

	// Calculate payout
	payout := dice.CalculatePayout(dice1Val, dice2Val, bet)
	total := dice1Val + dice2Val

	// Set cooldown
	h.setCooldown(sender.ID, "dice")

	// Process result asynchronously to avoid blocking
	go func() {
		// Wait for dice animation
		time.Sleep(3 * time.Second)

		// Credit winnings (payout is net, so add bet back + payout)
		if payout >= 0 {
			// Win or push - credit bet + payout
			creditAmount := bet + payout
			if creditAmount > 0 {
				h.userLock.Lock(sender.ID)
				desc := fmt.Sprintf("骰子游戏赢得 %d", payout)
				h.accountService.UpdateBalance(ctx, sender.ID, creditAmount, model.TxTypeDice, &desc)
				h.userLock.Unlock(sender.ID)
			}
		}
		// If payout < 0, bet was already deducted, nothing more to do

		// Get new balance
		newBalance, _ := h.accountService.GetBalance(ctx, sender.ID)

		// Build result message with @username
		var resultMsg string
		switch {
		case payout > bet:
			resultMsg = fmt.Sprintf("@%s 🎲🎲 %d + %d = %d\n🎊 JACKPOT! 赢得 %d 金币！\n💰 余额: %d", username, dice1Val, dice2Val, total, payout, newBalance)
		case payout > 0:
			resultMsg = fmt.Sprintf("@%s 🎲🎲 %d + %d = %d\n🎉 赢得 %d 金币！\n💰 余额: %d", username, dice1Val, dice2Val, total, payout, newBalance)
		case payout == 0:
			resultMsg = fmt.Sprintf("@%s 🎲🎲 %d + %d = %d\n😐 平局，返还下注\n💰 余额: %d", username, dice1Val, dice2Val, total, newBalance)
		default:
			resultMsg = fmt.Sprintf("@%s 🎲🎲 %d + %d = %d\n😢 输了 %d 金币\n💰 余额: %d", username, dice1Val, dice2Val, total, bet, newBalance)
		}

		replyMsg, err := c.Bot().Send(c.Chat(), resultMsg)
		if err == nil && replyMsg != nil {
			h.trackMessage(c.Chat().ID, replyMsg.ID)
		}
	}()

	return nil
}

// HandleSlot handles the /slot command.
// Requirements: 4.1
func (h *GameHandler) HandleSlot(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	// Parse bet amount
	args := c.Args()
	if len(args) < 1 {
		return c.Reply("❌ 用法: /slot <金额>\n例如: /slot 100")
	}

	bet, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || bet <= 0 {
		return c.Reply("❌ 请输入有效的下注金额")
	}

	// Check cooldown (3 seconds)
	cooldownSecs := 3
	if remaining := h.checkCooldown(sender.ID, "slot", cooldownSecs); remaining > 0 {
		return c.Reply(fmt.Sprintf("⏰ 请等待 %d 秒后再玩", remaining))
	}

	// Ensure user exists
	username := sender.Username
	if username == "" {
		username = sender.FirstName
	}
	_, _, err = h.accountService.EnsureUser(ctx, sender.ID, username)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Acquire lock
	h.userLock.Lock(sender.ID)
	defer h.userLock.Unlock(sender.ID)

	// Check balance
	balance, err := h.accountService.GetBalance(ctx, sender.ID)
	if err != nil {
		return c.Reply("❌ 获取余额失败")
	}

	// Check max bet based on balance (use dice max bet as default)
	maxBet := h.getEffectiveMaxBet(balance, h.cfg.Games.Dice.MaxBet)
	if bet > maxBet {
		tierMaxBet, tierThreshold := getBalanceTierInfo(balance)
		if tierThreshold > 0 {
			return c.Reply(fmt.Sprintf("❌ 余额超过 %d，单次下注上限为 %d", tierThreshold, tierMaxBet))
		}
		return c.Reply(fmt.Sprintf("❌ 最大下注金额为 %d", maxBet))
	}

	if balance < bet {
		return c.Reply("❌ 余额不足")
	}

	// Deduct bet first
	desc := fmt.Sprintf("老虎机下注 %d", bet)
	_, err = h.accountService.UpdateBalance(ctx, sender.ID, -bet, model.TxTypeSlot, &desc)
	if err != nil {
		return c.Reply("❌ 扣款失败，请稍后重试")
	}

	// Send slot machine
	slotMsg, err := c.Bot().Send(c.Chat(), tele.Slot)
	if err != nil {
		// Refund on error
		h.accountService.UpdateBalance(ctx, sender.ID, bet, model.TxTypeSlot, nil)
		return c.Reply("❌ 发送老虎机失败")
	}
	h.trackMessage(c.Chat().ID, slotMsg.ID)

	// Get slot value
	slotValue := slotMsg.Dice.Value

	// Decode and calculate payout
	left, middle, right := slot.DecodeSlot(slotValue)
	payout := slot.CalculatePayout(left, middle, right, bet)

	// Set cooldown
	h.setCooldown(sender.ID, "slot")

	// Process result asynchronously to avoid blocking
	go func() {
		// Wait for slot animation
		time.Sleep(3 * time.Second)

		// Credit winnings
		if payout >= 0 {
			creditAmount := bet + payout
			if creditAmount > 0 {
				h.userLock.Lock(sender.ID)
				desc := fmt.Sprintf("老虎机赢得 %d", payout)
				h.accountService.UpdateBalance(ctx, sender.ID, creditAmount, model.TxTypeSlot, &desc)
				h.userLock.Unlock(sender.ID)
			}
		}

		// Get new balance
		newBalance, _ := h.accountService.GetBalance(ctx, sender.ID)

		// Build result message with @username
		symbols := []string{slot.SymbolNames[left], slot.SymbolNames[middle], slot.SymbolNames[right]}
		slotDisplay := strings.Join(symbols, " ")

		var resultMsg string
		switch {
		case payout > 0:
			resultMsg = fmt.Sprintf("@%s 🎰 %s\n🎊 三连！赢得 %d 金币！\n💰 余额: %d", username, slotDisplay, payout, newBalance)
		case payout == 0:
			resultMsg = fmt.Sprintf("@%s 🎰 %s\n😐 两连，返还下注\n💰 余额: %d", username, slotDisplay, newBalance)
		default:
			resultMsg = fmt.Sprintf("@%s 🎰 %s\n😢 没中，输了 %d 金币\n💰 余额: %d", username, slotDisplay, bet, newBalance)
		}

		replyMsg, err := c.Bot().Send(c.Chat(), resultMsg)
		if err == nil && replyMsg != nil {
			h.trackMessage(c.Chat().ID, replyMsg.ID)
		}
	}()

	return nil
}


// HandleSicBoStart handles the /sicbo command to start a new game session.
// Requirements: 5.1
func (h *GameHandler) HandleSicBoStart(c tele.Context) error {
	ctx := context.Background()
	chat := c.Chat()
	sender := c.Sender()

	if chat == nil || sender == nil {
		return nil
	}

	// Only allow in group chats
	if chat.Type == tele.ChatPrivate {
		return c.Reply("❌ 骰宝游戏只能在群组中进行")
	}

	// Check if session already exists
	if h.sicboGame.IsSessionActive(chat.ID) {
		remaining := h.sicboGame.GetSessionTimeRemaining(chat.ID)
		return c.Reply(fmt.Sprintf("❌ 当前已有进行中的游戏，剩余 %d 秒", remaining))
	}

	// Start new session with starter ID
	duration := h.cfg.Games.SicBo.BettingDurationSeconds
	if duration <= 0 {
		duration = 60 // Default to 60 seconds if not configured
		log.Warn().Msg("SicBo betting duration not configured, using default 60 seconds")
	}

	log.Info().
		Int64("chat_id", chat.ID).
		Int64("starter_id", sender.ID).
		Int("duration", duration).
		Msg("Starting SicBo session")

	err := h.sicboGame.StartSession(ctx, chat.ID, sender.ID, duration)
	if err != nil {
		if errors.Is(err, sicbo.ErrSessionExists) {
			return c.Reply("❌ 当前已有进行中的游戏")
		}
		return c.Reply("❌ 启动游戏失败，请稍后重试")
	}

	// Build keyboard with early settle button (only starter sees it)
	kb := sicbo.NewKeyboardBuilder()
	markup := kb.BuildMainPanelWithSettle()

	// Send betting panel
	msg := sicbo.FormatPanelMessage(duration, 0, 0)
	panelMsg, err := c.Bot().Send(chat, msg, markup)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send sicbo panel")
	} else {
		h.trackMessage(chat.ID, panelMsg.ID)
		// Store panel message ID for periodic refresh
		h.sicboPanels.Store(chat.ID, panelMsg.ID)
	}

	// Schedule periodic panel refresh (every 15 seconds)
	go h.scheduleSicBoPanelRefresh(chat.ID, duration, c.Bot())

	// Schedule auto-settle (3 seconds before end time to show dice animation)
	go h.scheduleSicBoSettle(chat.ID, duration, c.Bot())

	return nil
}

// scheduleSicBoSettle schedules automatic settlement after betting phase ends.
func (h *GameHandler) scheduleSicBoSettle(chatID int64, durationSecs int, bot *tele.Bot) {
	// Ensure minimum duration to prevent immediate settlement
	if durationSecs < 10 {
		durationSecs = 60 // Default to 60 seconds if invalid
		log.Warn().Int64("chat_id", chatID).Msg("Invalid betting duration, using default 60 seconds")
	}

	// Wait until 3 seconds before end time (for dice animation)
	waitTime := durationSecs - 3
	
	log.Info().
		Int64("chat_id", chatID).
		Int("duration_secs", durationSecs).
		Int("wait_time", waitTime).
		Msg("Scheduling SicBo auto-settle")

	time.Sleep(time.Duration(waitTime) * time.Second)

	// Check if session still exists (might have been manually settled)
	if !h.sicboGame.IsSessionActive(chatID) {
		log.Debug().Int64("chat_id", chatID).Msg("Session already settled, skipping auto-settle")
		return
	}

	ctx := context.Background()
	h.settleSicBoWithAnimation(ctx, chatID, bot)
}

// scheduleSicBoPanelRefresh periodically refreshes the sicbo panel every 15 seconds.
func (h *GameHandler) scheduleSicBoPanelRefresh(chatID int64, durationSecs int, bot *tele.Bot) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check if session still exists
		if !h.sicboGame.IsSessionActive(chatID) {
			// Clean up panel reference
			h.sicboPanels.Delete(chatID)
			return
		}

		// Get panel message ID
		panelMsgID, ok := h.sicboPanels.Load(chatID)
		if !ok {
			return
		}

		// Get current stats
		remaining := h.sicboGame.GetSessionTimeRemaining(chatID)
		playerCount, totalBetAmount, _ := h.sicboGame.GetSessionStats(chatID)

		// Build updated message
		kb := sicbo.NewKeyboardBuilder()
		markup := kb.BuildMainPanelWithSettle()
		msg := sicbo.FormatPanelMessage(remaining, playerCount, totalBetAmount)

		// Edit the panel message
		editMsg := &tele.Message{
			ID:   panelMsgID.(int),
			Chat: &tele.Chat{ID: chatID},
		}
		_, err := bot.Edit(editMsg, msg, markup)
		if err != nil {
			log.Debug().Err(err).Int64("chat_id", chatID).Msg("Failed to refresh sicbo panel")
		}
	}
}

// settleSicBoWithAnimation sends dice animation and then settles the game.
func (h *GameHandler) settleSicBoWithAnimation(ctx context.Context, chatID int64, bot *tele.Bot) error {
	chat := &tele.Chat{ID: chatID}

	// Send 3 dice animation
	for i := 0; i < 3; i++ {
		diceMsg, err := bot.Send(chat, tele.Cube)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to send sicbo dice animation")
		} else {
			h.trackMessage(chatID, diceMsg.ID)
		}
		if i < 2 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	// Wait for dice animation to complete
	time.Sleep(3 * time.Second)

	// Now settle the game
	return h.settleSicBo(ctx, chatID, bot)
}

// HandleSicBoSettle handles the /sicbo_settle command to manually settle the game.
func (h *GameHandler) HandleSicBoSettle(c tele.Context) error {
	ctx := context.Background()
	chat := c.Chat()

	if chat == nil {
		return nil
	}

	if !h.sicboGame.IsSessionActive(chat.ID) {
		return c.Reply("❌ 当前没有进行中的游戏")
	}

	return h.settleSicBo(ctx, chat.ID, c.Bot())
}

// settleSicBo settles the SicBo game and sends results.
func (h *GameHandler) settleSicBo(ctx context.Context, chatID int64, bot *tele.Bot) error {
	// Get all bets before settling
	bets, err := h.sicboGame.GetSessionBets(ctx, chatID)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("Failed to get session bets")
		return err
	}

	// Get starter info before settling (session will be deleted after settle)
	starterID := h.sicboGame.GetSessionStarterID(chatID)
	starterUsername := ""
	if starterID != 0 {
		starterUser, err := h.accountService.GetUser(ctx, starterID)
		if err == nil && starterUser != nil {
			starterUsername = starterUser.Username
		}
	}

	// Settle the game
	payouts, details, err := h.sicboGame.Settle(ctx, chatID)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("Failed to settle sicbo game")
		return err
	}

	// Get dice results
	diceArr, ok := details["dice"].([3]int)
	if !ok {
		log.Error().Msg("Invalid dice result type")
		return errors.New("invalid dice result")
	}

	// Process payouts and build results
	playerResults := make(map[int64]sicbo.PlayerResult)
	for userID, netPayout := range payouts {
		// Calculate total bet for this user
		var totalBet int64
		if userBets, ok := bets[userID]; ok {
			for _, amount := range userBets {
				totalBet += amount
			}
		}

		// Get username (we'll need to look this up)
		user, err := h.accountService.GetUser(ctx, userID)
		username := ""
		if err == nil && user != nil {
			username = user.Username
		}

		playerResults[userID] = sicbo.PlayerResult{
			UserID:      userID,
			Username:    username,
			TotalBet:    totalBet,
			TotalPayout: netPayout,
		}

		// Update user balance
		// Note: Bet amount was already deducted when placing the bet
		// netPayout is the net result: positive = win, negative = loss
		// For wins: we need to credit (bet + winnings) = totalBet + netPayout
		// For losses: netPayout is negative, but bet was already deducted, so we don't deduct again
		// 
		// Example: User bets 100 on "big", dice shows 12 (big wins)
		//   - At bet time: -100 deducted
		//   - netPayout = +100 (1:1 payout)
		//   - Credit amount = totalBet + netPayout = 100 + 100 = 200
		//   - Final: -100 + 200 = +100 net gain ✓
		//
		// Example: User bets 100 on "big", dice shows 8 (big loses)
		//   - At bet time: -100 deducted
		//   - netPayout = -100 (loss)
		//   - Since netPayout < 0, we don't credit anything (bet already lost)
		//   - Final: -100 net loss ✓
		
		if netPayout > 0 {
			// User won - credit bet amount + winnings
			creditAmount := totalBet + netPayout
			h.userLock.Lock(userID)
			desc := fmt.Sprintf("骰宝赢得 %d (本金 %d + 盈利 %d)", creditAmount, totalBet, netPayout)
			h.accountService.UpdateBalance(ctx, userID, creditAmount, model.TxTypeSicBoWin, &desc)
			h.userLock.Unlock(userID)
		}
		// If netPayout <= 0, user lost - bet was already deducted, nothing more to do
	}

	// Format and send settlement message
	msg := sicbo.FormatSettlementMessage(diceArr, playerResults, starterUsername)

	// Send result to chat
	if bot != nil {
		chat := &tele.Chat{ID: chatID}
		_, err = bot.Send(chat, msg)
		if err != nil {
			log.Error().Err(err).Int64("chat_id", chatID).Msg("Failed to send sicbo settlement message")
		}
	}

	log.Info().
		Int64("chat_id", chatID).
		Interface("dice", diceArr).
		Interface("payouts", payouts).
		Msg("SicBo game settled")

	return nil
}

// HandleSicBoCallback handles SicBo inline button callbacks.
// Requirements: 5.2, 5.6, 5.8
func (h *GameHandler) HandleSicBoCallback(c tele.Context) error {
	ctx := context.Background()
	callback := c.Callback()
	sender := c.Sender()
	chat := c.Chat()

	if callback == nil || sender == nil || chat == nil {
		return nil
	}

	// Parse callback data
	action, param := sicbo.DecodeCallback(callback.Data)
	
	// Debug logging
	log.Debug().
		Str("raw_data", callback.Data).
		Str("action", action).
		Str("param", param).
		Int64("user_id", sender.ID).
		Int64("chat_id", chat.ID).
		Msg("SicBo callback received")
	
	if action == "" {
		return c.Respond(&tele.CallbackResponse{
			Text: "❌ 无效操作",
		})
	}

	// Handle early settle action
	if action == "early_settle" {
		// Check if user is the session starter
		starterID := h.sicboGame.GetSessionStarterID(chat.ID)
		
		// Debug logging for starter check
		log.Debug().
			Int64("starter_id", starterID).
			Int64("sender_id", sender.ID).
			Int64("chat_id", chat.ID).
			Bool("is_starter", starterID == sender.ID).
			Msg("Early settle check")
		
		if starterID != sender.ID {
			return c.Respond(&tele.CallbackResponse{
				Text:      fmt.Sprintf("❌ 只有发起者可以提前开奖 (发起者ID: %d, 你的ID: %d)", starterID, sender.ID),
				ShowAlert: true,
			})
		}

		// Check if session is active
		if !h.sicboGame.IsSessionActive(chat.ID) {
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ 游戏已结束",
				ShowAlert: true,
			})
		}

		// Respond immediately
		c.Respond(&tele.CallbackResponse{
			Text: "🎲 开始开奖...",
		})

		// Settle with animation
		go h.settleSicBoWithAnimation(ctx, chat.ID, c.Bot())
		return nil
	}

	// Check if session is active
	if !h.sicboGame.IsSessionActive(chat.ID) {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ 游戏已结束",
			ShowAlert: true,
		})
	}

	// Determine bet type
	var betType string
	switch action {
	case "single":
		betType = param // "1", "2", etc.
	case "big":
		betType = "big"
	case "small":
		betType = "small"
	default:
		return c.Respond(&tele.CallbackResponse{
			Text: "❌ 无效操作",
		})
	}

	// Ensure user exists
	username := sender.Username
	if username == "" {
		username = sender.FirstName
	}
	_, _, err := h.accountService.EnsureUser(ctx, sender.ID, username)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ 操作失败",
			ShowAlert: true,
		})
	}

	// Check balance
	betAmount := sicbo.FixedBetAmount
	h.userLock.Lock(sender.ID)
	balance, err := h.accountService.GetBalance(ctx, sender.ID)
	if err != nil {
		h.userLock.Unlock(sender.ID)
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ 获取余额失败",
			ShowAlert: true,
		})
	}

	if balance < betAmount {
		h.userLock.Unlock(sender.ID)
		return c.Respond(&tele.CallbackResponse{
			Text:      fmt.Sprintf("❌ 下注失败，余额不足（需要 %d，当前 %d）", betAmount, balance),
			ShowAlert: true,
		})
	}

	// Deduct bet amount
	desc := fmt.Sprintf("骰宝下注 %s", betType)
	_, err = h.accountService.UpdateBalance(ctx, sender.ID, -betAmount, model.TxTypeSicBoBet, &desc)
	h.userLock.Unlock(sender.ID)

	if err != nil {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ 扣款失败",
			ShowAlert: true,
		})
	}

	// Place bet
	err = h.sicboGame.PlaceBet(ctx, chat.ID, sender.ID, betType, betAmount)
	if err != nil {
		// Refund on error
		h.userLock.Lock(sender.ID)
		h.accountService.UpdateBalance(ctx, sender.ID, betAmount, model.TxTypeSicBoBet, nil)
		h.userLock.Unlock(sender.ID)

		if errors.Is(err, sicbo.ErrBettingEnded) {
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ 下注时间已结束",
				ShowAlert: true,
			})
		}
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ 下注失败",
			ShowAlert: true,
		})
	}

	// Get bet display name
	betName := betType
	switch betType {
	case "big":
		betName = "大"
	case "small":
		betName = "小"
	}

	// Don't refresh panel on every bet - let the 15s timer handle it
	// This reduces API calls and makes the UI less jumpy

	return c.Respond(&tele.CallbackResponse{
		Text: fmt.Sprintf("✅ 已下注 %s: %d 金币", betName, betAmount),
	})
}

// HandleMyBets handles the /mybets command to show user's current bets.
func (h *GameHandler) HandleMyBets(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	chat := c.Chat()

	if sender == nil || chat == nil {
		return nil
	}

	if !h.sicboGame.IsSessionActive(chat.ID) {
		return c.Reply("❌ 当前没有进行中的游戏")
	}

	bets, err := h.sicboGame.GetSessionBets(ctx, chat.ID)
	if err != nil {
		return c.Reply("❌ 获取下注信息失败")
	}

	userBets, ok := bets[sender.ID]
	if !ok || len(userBets) == 0 {
		return c.Reply("📋 您还没有下注")
	}

	msg := sicbo.FormatMyBets(userBets)
	return c.Reply(msg)
}

// HandleDajie handles the /dajie command for robbery game.
// Requirements: Rob Game - Allow users to rob coins from other users
func (h *GameHandler) HandleDajie(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	chat := c.Chat()

	if sender == nil || chat == nil {
		return nil
	}

	// Get robber's username
	robberName := sender.Username
	if robberName == "" {
		robberName = sender.FirstName
	}

	// Ensure robber exists
	_, _, err := h.accountService.EnsureUser(ctx, sender.ID, robberName)
	if err != nil {
		return c.Reply("❌ 操作失败，请稍后重试")
	}

	// Determine victim from reply or @mention
	var victimID int64
	var victimName string

	// Check if replying to a message
	if c.Message().ReplyTo != nil && c.Message().ReplyTo.Sender != nil {
		victimID = c.Message().ReplyTo.Sender.ID
		victimName = c.Message().ReplyTo.Sender.Username
		if victimName == "" {
			victimName = c.Message().ReplyTo.Sender.FirstName
		}
	} else {
		// Check for @mention in args
		args := c.Args()
		if len(args) < 1 {
			return c.Reply("❌ 用法: /dajie (回复消息) 或 /dajie @用户名")
		}

		// Parse @username
		mention := args[0]
		if !strings.HasPrefix(mention, "@") {
			return c.Reply("❌ 请使用 @用户名 格式")
		}

		// We need to find the user by username - this is tricky in Telegram
		// For now, we'll require reply-to-message method
		return c.Reply("❌ 请回复目标用户的消息来发起打劫")
	}

	// Execute robbery
	result, err := h.robGame.Rob(ctx, sender.ID, victimID, robberName, victimName)
	if err != nil {
		log.Error().Err(err).Int64("robber", sender.ID).Int64("victim", victimID).Msg("Robbery failed")
		return c.Reply("❌ 打劫失败，请稍后重试")
	}

	// Send result
	if result.Success {
		msg := result.Message + fmt.Sprintf("\n💰 你的余额: %d", result.NewBalance)
		return c.Reply(msg)
	}

	return c.Reply("❌ " + result.Message)
}
