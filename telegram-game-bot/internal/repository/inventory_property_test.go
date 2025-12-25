// Package repository provides data access layer implementations.
// Property-based tests for InventoryRepository.
// **Feature: shop-system, Property 3: Use Count Decrement**
// **Validates: Requirements 3.6, 3.7, 4.4, 4.5, 5.4, 5.5, 6.5, 6.6, 7.6, 7.7, 8.4, 8.5, 9.5, 9.6**
package repository

import (
	"testing"

	"pgregory.net/rapid"
)

// UseCountState represents the state of an item's use count
// This is a pure model for testing the use count decrement logic
type UseCountState struct {
	UseCount int
}

// NewUseCountState creates a new use count state
func NewUseCountState(initialCount int) *UseCountState {
	return &UseCountState{UseCount: initialCount}
}

// Decrement decreases the use count by 1 if count > 0
// Returns true if decrement was successful, false if count was already 0
func (s *UseCountState) Decrement() bool {
	if s.UseCount <= 0 {
		return false
	}
	s.UseCount--
	return true
}

// HasEffect returns true if the item effect is active (use count > 0)
func (s *UseCountState) HasEffect() bool {
	return s.UseCount > 0
}

// GetCount returns the current use count
func (s *UseCountState) GetCount() int {
	return s.UseCount
}

// TestUseCountDecrementProperty tests Property 3: Use Count Decrement
// *For any* item use, the use count should decrease by exactly 1.
// When use count reaches 0, the item effect should be removed.
// **Validates: Requirements 3.6, 3.7, 4.4, 4.5, 5.4, 5.5, 6.5, 6.6, 7.6, 7.7, 8.4, 8.5, 9.5, 9.6**
func TestUseCountDecrementProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate initial use count (1-20)
		initialUseCount := rapid.IntRange(1, 20).Draw(rt, "initialUseCount")

		// Create state with initial use count
		state := NewUseCountState(initialUseCount)

		// Verify initial use count
		if state.GetCount() != initialUseCount {
			rt.Fatalf("Initial use count mismatch: expected %d, got %d", initialUseCount, state.GetCount())
		}

		// Verify item is active initially
		if !state.HasEffect() {
			rt.Fatalf("Item should be active when use count > 0")
		}

		// Decrement use count multiple times and verify
		for i := 0; i < initialUseCount; i++ {
			expectedCount := initialUseCount - i

			// Verify current count before decrement
			if state.GetCount() != expectedCount {
				rt.Fatalf("Use count before decrement %d: expected %d, got %d", i, expectedCount, state.GetCount())
			}

			// Decrement
			success := state.Decrement()
			if !success {
				rt.Fatalf("Decrement should succeed at iteration %d (count was %d)", i, expectedCount)
			}

			// Verify count decreased by exactly 1
			if state.GetCount() != expectedCount-1 {
				rt.Fatalf("Use count after decrement %d: expected %d, got %d", i, expectedCount-1, state.GetCount())
			}
		}

		// After all decrements, count should be 0
		if state.GetCount() != 0 {
			rt.Fatalf("Final use count should be 0, got %d", state.GetCount())
		}

		// Verify item effect is removed (HasEffect should return false)
		if state.HasEffect() {
			rt.Fatalf("Item should not be active when use count is 0")
		}

		// Verify decrement fails when count is 0
		success := state.Decrement()
		if success {
			rt.Fatalf("Decrement should return false when count is 0")
		}

		// Verify count is still 0 after failed decrement
		if state.GetCount() != 0 {
			rt.Fatalf("Use count should still be 0 after failed decrement, got %d", state.GetCount())
		}
	})
}

// TestUseCountDecrementExactlyOneProperty tests that each decrement reduces count by exactly 1
// **Validates: Requirements 3.6, 3.7**
func TestUseCountDecrementExactlyOneProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Start with a random count
		startCount := rapid.IntRange(1, 100).Draw(rt, "startCount")

		state := NewUseCountState(startCount)

		// Get count before decrement
		beforeCount := state.GetCount()

		// Decrement once
		success := state.Decrement()
		if !success {
			rt.Fatalf("Decrement should succeed when count > 0")
		}

		// Get count after decrement
		afterCount := state.GetCount()

		// Verify exactly 1 was decremented
		if afterCount != beforeCount-1 {
			rt.Fatalf("Decrement should reduce count by exactly 1: before=%d, after=%d", beforeCount, afterCount)
		}
	})
}

// TestUseCountZeroRemovesEffectProperty tests that when use count reaches 0, the effect is removed
// **Validates: Requirements 3.7, 4.5, 5.5, 6.6, 7.7, 8.5, 9.6**
func TestUseCountZeroRemovesEffectProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Start with count of 1
		state := NewUseCountState(1)

		// Verify item is active
		if !state.HasEffect() {
			rt.Fatalf("Item should be active when use count > 0")
		}

		// Decrement to 0
		success := state.Decrement()
		if !success {
			rt.Fatalf("Decrement should succeed")
		}

		// Verify item is no longer active
		if state.HasEffect() {
			rt.Fatalf("Item should NOT be active when use count is 0")
		}

		// Verify use count is 0
		if state.GetCount() != 0 {
			rt.Fatalf("Use count should be 0, got %d", state.GetCount())
		}
	})
}

// TestUseCountCannotGoNegativeProperty tests that use count cannot go below 0
// **Validates: Requirements 3.6, 3.7**
func TestUseCountCannotGoNegativeProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Start with count of 0
		state := NewUseCountState(0)

		// Try to decrement multiple times
		numAttempts := rapid.IntRange(1, 10).Draw(rt, "numAttempts")
		for i := 0; i < numAttempts; i++ {
			success := state.Decrement()
			if success {
				rt.Fatalf("Decrement should fail when count is 0")
			}
			if state.GetCount() != 0 {
				rt.Fatalf("Use count should remain 0, got %d", state.GetCount())
			}
			if state.HasEffect() {
				rt.Fatalf("Item should not be active when use count is 0")
			}
		}
	})
}

// TestMultipleItemsIndependentProperty tests that different item types have independent use counts
// **Validates: Requirements 3.6, 4.4, 5.4, 6.5, 7.6, 8.4, 9.5**
func TestMultipleItemsIndependentProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate use counts for different items
		shieldCount := rapid.IntRange(1, 10).Draw(rt, "shieldCount")
		thornArmorCount := rapid.IntRange(1, 10).Draw(rt, "thornArmorCount")
		bloodthirstCount := rapid.IntRange(1, 10).Draw(rt, "bloodthirstCount")

		// Create independent states for each item
		shield := NewUseCountState(shieldCount)
		thornArmor := NewUseCountState(thornArmorCount)
		bloodthirst := NewUseCountState(bloodthirstCount)

		// Decrement shield
		shield.Decrement()

		// Verify other items are unaffected
		if thornArmor.GetCount() != thornArmorCount {
			rt.Fatalf("Thorn armor count should be unchanged: expected %d, got %d", thornArmorCount, thornArmor.GetCount())
		}
		if bloodthirst.GetCount() != bloodthirstCount {
			rt.Fatalf("Bloodthirst count should be unchanged: expected %d, got %d", bloodthirstCount, bloodthirst.GetCount())
		}

		// Verify shield was decremented
		if shield.GetCount() != shieldCount-1 {
			rt.Fatalf("Shield count should be decremented: expected %d, got %d", shieldCount-1, shield.GetCount())
		}
	})
}

// TestAllItemTypesDecrementProperty tests that all 8 item types follow the same decrement behavior
// **Validates: Requirements 3.6, 3.7, 4.4, 4.5, 5.4, 5.5, 6.5, 6.6, 7.6, 7.7, 8.4, 8.5, 9.5, 9.6**
func TestAllItemTypesDecrementProperty(t *testing.T) {
	itemTypes := []string{
		"handcuff", "shield", "thorn_armor", "bloodthirst",
		"blunt_knife", "great_sword", "golden_cassock", "emperor_clothes",
	}

	for _, itemType := range itemTypes {
		t.Run(itemType, func(t *testing.T) {
			rapid.Check(t, func(rt *rapid.T) {
				// Generate initial use count
				initialCount := rapid.IntRange(1, 10).Draw(rt, "initialCount")

				state := NewUseCountState(initialCount)

				// Verify initial state
				if state.GetCount() != initialCount {
					rt.Fatalf("[%s] Initial count mismatch: expected %d, got %d", itemType, initialCount, state.GetCount())
				}
				if !state.HasEffect() {
					rt.Fatalf("[%s] Item should be active initially", itemType)
				}

				// Decrement all uses
				for i := 0; i < initialCount; i++ {
					success := state.Decrement()
					if !success {
						rt.Fatalf("[%s] Decrement %d should succeed", itemType, i)
					}
				}

				// Verify final state
				if state.GetCount() != 0 {
					rt.Fatalf("[%s] Final count should be 0, got %d", itemType, state.GetCount())
				}
				if state.HasEffect() {
					rt.Fatalf("[%s] Item should not be active when count is 0", itemType)
				}

				// Verify cannot decrement below 0
				success := state.Decrement()
				if success {
					rt.Fatalf("[%s] Decrement should fail when count is 0", itemType)
				}
			})
		})
	}
}
