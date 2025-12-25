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
	ErrItemNotFound   = errors.New("道具不存在")
	ErrNoHandcuff     = errors.New("没有手铐道具")
	ErrSelfHandcuff   = errors.New("不能对自己使用手铐")
	ErrTargetNotFound = errors.New("目标用户未找到")
	ErrAlreadyLocked  = errors.New("目标已被锁定")
)

// UserInventory represents a user's complete inventory
type UserInventory struct {
	HandcuffCount int
	Effects       []repository.UserEffect
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
func (s *ShopService) PurchaseItem(ctx context.Context, userID int64, itemType shop.ItemType) error {
	// Get item config
	item, ok := shop.GetItem(itemType)
	if !ok {
		return ErrItemNotFound
	}

	// Lock user for balance operation
	s.userLock.Lock(userID)
	defer s.userLock.Unlock(userID)

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

	// Add item to inventory
	if item.IsOneTimeUse() {
		// Stackable item (like handcuffs)
		err = s.inventoryRepo.AddItem(ctx, userID, string(itemType), 1)
	} else {
		// Time-based effect
		expiresAt := time.Now().Add(item.Duration)
		err = s.inventoryRepo.AddEffect(ctx, userID, string(itemType), expiresAt)
	}

	return err
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

	// Get active effects
	effects, err := s.inventoryRepo.GetActiveEffects(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserInventory{
		HandcuffCount: handcuffCount,
		Effects:       effects,
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
func (s *ShopService) GetEffectExpiry(ctx context.Context, userID int64, effectType shop.ItemType) time.Time {
	expiry, _ := s.inventoryRepo.GetEffectExpiry(ctx, userID, string(effectType))
	return expiry
}
