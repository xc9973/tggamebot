// Package service provides business logic implementations.
package service

import (
	"context"
	"errors"
	"time"

	"telegram-game-bot/internal/model"
	"telegram-game-bot/internal/pkg/lock"
	"telegram-game-bot/internal/repository"
	"telegram-game-bot/internal/shop"
)

// Shop service errors
var (
	ErrItemNotFound       = errors.New("道具不存在")
	ErrNoHandcuff         = errors.New("没有手铐道具")
	ErrNoKey              = errors.New("没有钥匙道具")
	ErrSelfHandcuff       = errors.New("不能对自己使用手铐")
	ErrTargetNotFound     = errors.New("目标用户未找到")
	ErrAlreadyLocked      = errors.New("目标已被锁定")
	ErrNotLocked          = errors.New("你没有被锁定")
	ErrDailyLimitReached  = errors.New("今日购买次数已达上限")
)

// UserInventory represents a user's complete inventory
type UserInventory struct {
	HandcuffCount int
	Items         []repository.UserItem
}

// ShopService handles shop-related business logic
type ShopService struct {
	userRepo      *repository.UserRepository
	txRepo        *repository.TransactionRepository
	inventoryRepo *repository.InventoryRepository
	userLock      *lock.UserLock
}

// NewShopService creates a new ShopService instance
func NewShopService(
	userRepo *repository.UserRepository,
	txRepo *repository.TransactionRepository,
	inventoryRepo *repository.InventoryRepository,
	userLock *lock.UserLock,
) *ShopService {
	return &ShopService{
		userRepo:      userRepo,
		txRepo:        txRepo,
		inventoryRepo: inventoryRepo,
		userLock:      userLock,
	}
}

// GetShopItems returns all available shop items
func (s *ShopService) GetShopItems() []shop.ItemConfig {
	return shop.GetAllItems()
}

// PurchaseItem handles item purchase
// Requirements: 12.3, 12.4 - Check daily limit before purchase
func (s *ShopService) PurchaseItem(ctx context.Context, userID int64, itemType shop.ItemType) error {
	// Get item config
	item, ok := shop.GetItem(itemType)
	if !ok {
		return ErrItemNotFound
	}

	// Lock user for balance operation
	s.userLock.Lock(userID)
	defer s.userLock.Unlock(userID)

	// Check daily purchase limit if applicable
	// Requirements: 2.3, 2.9, 3.3, 3.8, 7.3, 7.8, 12.3, 12.4
	if item.HasDailyLimit() {
		purchaseCount, err := s.inventoryRepo.GetDailyPurchaseCount(ctx, userID, string(itemType))
		if err != nil {
			return err
		}
		if purchaseCount >= item.DailyLimit {
			return ErrDailyLimitReached
		}
	}

	// Check balance
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Balance < item.Price {
		return ErrInsufficientBalance
	}

	// Deduct balance
	desc := "购买" + item.Name
	_, err = s.userRepo.UpdateBalance(ctx, userID, -item.Price)
	if err != nil {
		return err
	}

	// Record transaction
	s.txRepo.Create(ctx, userID, -item.Price, model.TxTypeShopPurchase, &desc)

	// Add item to inventory with use count
	err = s.inventoryRepo.AddItem(ctx, userID, string(itemType), item.UseCount)
	if err != nil {
		return err
	}

	// Increment daily purchase count if item has daily limit
	if item.HasDailyLimit() {
		err = s.inventoryRepo.IncrementDailyPurchase(ctx, userID, string(itemType))
		if err != nil {
			return err
		}
	}

	return nil
}

// UseHandcuff uses a handcuff on a target user
func (s *ShopService) UseHandcuff(ctx context.Context, userID, targetID int64) error {
	// Can't handcuff yourself
	if userID == targetID {
		return ErrSelfHandcuff
	}

	// Check if target exists
	exists, err := s.userRepo.Exists(ctx, targetID)
	if err != nil || !exists {
		return ErrTargetNotFound
	}

	// Check if user has handcuffs
	count, err := s.inventoryRepo.GetItemCount(ctx, userID, string(shop.ItemHandcuff))
	if err != nil {
		return err
	}
	if count <= 0 {
		return ErrNoHandcuff
	}

	// Check if target is already locked
	locked, _, _, err := s.inventoryRepo.IsHandcuffed(ctx, targetID)
	if err != nil {
		return err
	}
	if locked {
		return ErrAlreadyLocked
	}

	// Consume handcuff
	success, err := s.inventoryRepo.DecrementItem(ctx, userID, string(shop.ItemHandcuff))
	if err != nil || !success {
		return ErrNoHandcuff
	}

	// Lock target
	item, _ := shop.GetItem(shop.ItemHandcuff)
	expiresAt := time.Now().Add(item.EffectDuration)
	return s.inventoryRepo.AddHandcuffLock(ctx, targetID, userID, expiresAt)
}

// GetUserInventory returns a user's complete inventory
func (s *ShopService) GetUserInventory(ctx context.Context, userID int64) (*UserInventory, error) {
	// Get handcuff count
	handcuffCount, err := s.inventoryRepo.GetItemCount(ctx, userID, string(shop.ItemHandcuff))
	if err != nil {
		return nil, err
	}

	// Get all items with use_count > 0
	items, err := s.inventoryRepo.GetAllItems(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserInventory{
		HandcuffCount: handcuffCount,
		Items:         items,
	}, nil
}

// HasHandcuff checks if user has at least one handcuff
func (s *ShopService) HasHandcuff(ctx context.Context, userID int64) bool {
	count, err := s.inventoryRepo.GetItemCount(ctx, userID, string(shop.ItemHandcuff))
	return err == nil && count > 0
}

// IsHandcuffed checks if a user is locked by handcuffs
// Returns (isLocked, remainingTime)
func (s *ShopService) IsHandcuffed(ctx context.Context, userID int64) (bool, time.Duration) {
	locked, remaining, _, err := s.inventoryRepo.IsHandcuffed(ctx, userID)
	if err != nil {
		return false, 0
	}
	return locked, remaining
}

// HasShield checks if user has active shield
func (s *ShopService) HasShield(ctx context.Context, userID int64) bool {
	has, err := s.inventoryRepo.HasActiveEffect(ctx, userID, string(shop.ItemShield))
	return err == nil && has
}

// HasThornArmor checks if user has active thorn armor
func (s *ShopService) HasThornArmor(ctx context.Context, userID int64) bool {
	has, err := s.inventoryRepo.HasActiveEffect(ctx, userID, string(shop.ItemThornArmor))
	return err == nil && has
}

// HasBloodthirstSword checks if user has active bloodthirst sword
func (s *ShopService) HasBloodthirstSword(ctx context.Context, userID int64) bool {
	has, err := s.inventoryRepo.HasActiveEffect(ctx, userID, string(shop.ItemBloodthirstSword))
	return err == nil && has
}

// GetEffectExpiry returns the expiry time of an effect
// Deprecated: Use GetEffectUseCount instead since we now use use-count based system
func (s *ShopService) GetEffectExpiry(ctx context.Context, userID int64, effectType shop.ItemType) time.Time {
	expiry, _ := s.inventoryRepo.GetEffectExpiry(ctx, userID, string(effectType))
	return expiry
}

// GetEffectUseCount returns the remaining use count of an effect
// Requirements: 6.3, 7.4, 8.3, 9.3 - Get remaining use count for items
func (s *ShopService) GetEffectUseCount(ctx context.Context, userID int64, effectType shop.ItemType) (int, error) {
	return s.inventoryRepo.GetUseCount(ctx, userID, string(effectType))
}

// DecrementUseCount decreases the use count of an item by 1
// Requirements: 3.6, 3.7, 4.4, 4.5, 5.4, 5.5, 6.5, 6.6, 7.6, 7.7, 8.4, 8.5, 9.5, 9.6
func (s *ShopService) DecrementUseCount(ctx context.Context, userID int64, effectType shop.ItemType) error {
	_, err := s.inventoryRepo.DecrementUseCount(ctx, userID, string(effectType))
	return err
}

// DecrementUseCountByString decreases the use count of an item by 1 (accepts string type)
// This method is used by the ItemEffectChecker interface
// Requirements: 6.5, 7.6, 8.4, 9.5 - Decrement use count after item use
func (s *ShopService) DecrementUseCountByString(ctx context.Context, userID int64, effectType string) error {
	_, err := s.inventoryRepo.DecrementUseCount(ctx, userID, effectType)
	return err
}

// HasEmperorClothes checks if user has active emperor clothes (highest priority defense)
// Requirements: 9.3, 9.4 - Emperor clothes immunity check
func (s *ShopService) HasEmperorClothes(ctx context.Context, userID int64) bool {
	has, err := s.inventoryRepo.HasActiveEffect(ctx, userID, string(shop.ItemEmperorClothes))
	return err == nil && has
}

// HasBluntKnife checks if user has active blunt knife
// Requirements: 6.3 - Blunt knife bypass defense check
func (s *ShopService) HasBluntKnife(ctx context.Context, userID int64) bool {
	has, err := s.inventoryRepo.HasActiveEffect(ctx, userID, string(shop.ItemBluntKnife))
	return err == nil && has
}

// HasGreatSword checks if user has active great sword
// Requirements: 7.4 - Great sword bypass defense check
func (s *ShopService) HasGreatSword(ctx context.Context, userID int64) bool {
	has, err := s.inventoryRepo.HasActiveEffect(ctx, userID, string(shop.ItemGreatSword))
	return err == nil && has
}

// HasGoldenCassock checks if user has active golden cassock
// Requirements: 8.3 - Golden cassock defense removal check
func (s *ShopService) HasGoldenCassock(ctx context.Context, userID int64) bool {
	has, err := s.inventoryRepo.HasActiveEffect(ctx, userID, string(shop.ItemGoldenCassock))
	return err == nil && has
}

// RemoveDefensiveItems removes all defensive items (Shield, Thorn Armor) from a user
// This is triggered by Golden Cassock effect
// Requirements: 8.4 - Remove attacker's defensive items
func (s *ShopService) RemoveDefensiveItems(ctx context.Context, userID int64) error {
	// Remove Shield
	err := s.inventoryRepo.RemoveItem(ctx, userID, string(shop.ItemShield))
	if err != nil {
		return err
	}
	// Remove Thorn Armor
	err = s.inventoryRepo.RemoveItem(ctx, userID, string(shop.ItemThornArmor))
	return err
}

// CheckDailyLimit checks if a user has reached the daily purchase limit for an item
// Returns (canPurchase, currentCount, error)
// Requirements: 12.3, 12.4 - Daily purchase limit check
func (s *ShopService) CheckDailyLimit(ctx context.Context, userID int64, itemType shop.ItemType) (bool, int, error) {
	item, ok := shop.GetItem(itemType)
	if !ok {
		return false, 0, ErrItemNotFound
	}

	// If no daily limit, always allow
	if !item.HasDailyLimit() {
		return true, 0, nil
	}

	purchaseCount, err := s.inventoryRepo.GetDailyPurchaseCount(ctx, userID, string(itemType))
	if err != nil {
		return false, 0, err
	}

	canPurchase := purchaseCount < item.DailyLimit
	return canPurchase, purchaseCount, nil
}

// UseKey uses a key to unlock self from handcuffs
func (s *ShopService) UseKey(ctx context.Context, userID int64) error {
	// Check if user is locked
	locked, _, _, err := s.inventoryRepo.IsHandcuffed(ctx, userID)
	if err != nil {
		return err
	}
	if !locked {
		return ErrNotLocked
	}

	// Check if user has key
	count, err := s.inventoryRepo.GetItemCount(ctx, userID, string(shop.ItemKey))
	if err != nil {
		return err
	}
	if count <= 0 {
		return ErrNoKey
	}

	// Consume key
	success, err := s.inventoryRepo.DecrementItem(ctx, userID, string(shop.ItemKey))
	if err != nil || !success {
		return ErrNoKey
	}

	// Remove handcuff lock
	_, err = s.inventoryRepo.RemoveHandcuffLock(ctx, userID)
	return err
}

// HasKey checks if user has at least one key
func (s *ShopService) HasKey(ctx context.Context, userID int64) bool {
	count, err := s.inventoryRepo.GetItemCount(ctx, userID, string(shop.ItemKey))
	return err == nil && count > 0
}
