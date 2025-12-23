// Package slot tests for the slot machine game.
// Requirements: 4.2, 4.4
package slot

import (
	"context"
	"testing"

	"pgregory.net/rapid"
)

func TestDecodeSlot(t *testing.T) {
	tests := []struct {
		name      string
		slotValue int
		wantLeft  int
		wantMid   int
		wantRight int
	}{
		{"value 1 (1,1,1)", 1, 1, 1, 1},
		{"value 22 (2,2,2)", 22, 2, 2, 2},
		{"value 43 (3,3,3)", 43, 3, 3, 3},
		{"value 64 (4,4,4)", 64, 4, 4, 4},
		{"value 2 (2,1,1)", 2, 2, 1, 1},
		{"value 5 (1,2,1)", 5, 1, 2, 1},
		{"value 17 (1,1,2)", 17, 1, 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, middle, right := DecodeSlot(tt.slotValue)
			if left != tt.wantLeft || middle != tt.wantMid || right != tt.wantRight {
				t.Errorf("DecodeSlot(%d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.slotValue, left, middle, right, tt.wantLeft, tt.wantMid, tt.wantRight)
			}
		})
	}
}

func TestEncodeSlot(t *testing.T) {
	tests := []struct {
		name  string
		left  int
		mid   int
		right int
		want  int
	}{
		{"(1,1,1) = 1", 1, 1, 1, 1},
		{"(2,2,2) = 22", 2, 2, 2, 22},
		{"(3,3,3) = 43", 3, 3, 3, 43},
		{"(4,4,4) = 64", 4, 4, 4, 64},
		{"(2,1,1) = 2", 2, 1, 1, 2},
		{"(1,2,1) = 5", 1, 2, 1, 5},
		{"(1,1,2) = 17", 1, 1, 2, 17},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeSlot(tt.left, tt.mid, tt.right)
			if got != tt.want {
				t.Errorf("EncodeSlot(%d, %d, %d) = %d, want %d",
					tt.left, tt.mid, tt.right, got, tt.want)
			}
		})
	}
}

func TestDecodeEncodeRoundTrip(t *testing.T) {
	// Test round-trip for all valid slot values (1-64)
	for v := 1; v <= 64; v++ {
		left, middle, right := DecodeSlot(v)
		encoded := EncodeSlot(left, middle, right)
		if encoded != v {
			t.Errorf("Round-trip failed: DecodeSlot(%d) = (%d, %d, %d), EncodeSlot = %d",
				v, left, middle, right, encoded)
		}
	}
}

func TestDecodeSlotSymbolRange(t *testing.T) {
	// All decoded symbols should be in range [1, 4]
	for v := 1; v <= 64; v++ {
		left, middle, right := DecodeSlot(v)
		if left < 1 || left > 4 {
			t.Errorf("DecodeSlot(%d): left = %d, want [1,4]", v, left)
		}
		if middle < 1 || middle > 4 {
			t.Errorf("DecodeSlot(%d): middle = %d, want [1,4]", v, middle)
		}
		if right < 1 || right > 4 {
			t.Errorf("DecodeSlot(%d): right = %d, want [1,4]", v, right)
		}
	}
}

func TestCalculatePayout_ThreeMatches(t *testing.T) {
	tests := []struct {
		name string
		bet  int64
		want int64
	}{
		{"small bet 3x", 100, 300},
		{"medium bet 2x", 5000, 10000},
		{"large bet 1.5x", 50000, 75000},
		{"huge bet 1x", 200000, 200000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All three symbols match (e.g., 1,1,1)
			got := CalculatePayout(1, 1, 1, tt.bet)
			if got != tt.want {
				t.Errorf("CalculatePayout(1,1,1,%d) = %d, want %d", tt.bet, got, tt.want)
			}
		})
	}
}

func TestCalculatePayout_TwoMatches(t *testing.T) {
	tests := []struct {
		name   string
		left   int
		middle int
		right  int
	}{
		{"left-middle match", 1, 1, 2},
		{"middle-right match", 1, 2, 2},
		{"left-right match", 1, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePayout(tt.left, tt.middle, tt.right, 100)
			if got != 0 {
				t.Errorf("CalculatePayout(%d,%d,%d,100) = %d, want 0 (push)",
					tt.left, tt.middle, tt.right, got)
			}
		})
	}
}

func TestCalculatePayout_NoMatch(t *testing.T) {
	// All different symbols
	bet := int64(100)
	got := CalculatePayout(1, 2, 3, bet)
	if got != -bet {
		t.Errorf("CalculatePayout(1,2,3,%d) = %d, want %d", bet, got, -bet)
	}
}

func TestSlotGame_Interface(t *testing.T) {
	sg := New(nil)

	if sg.Name() != "Slot Machine" {
		t.Errorf("Name() = %s, want Slot Machine", sg.Name())
	}
	if sg.Command() != "slot" {
		t.Errorf("Command() = %s, want slot", sg.Command())
	}
	if sg.MaxBet() != DefaultMaxBet {
		t.Errorf("MaxBet() = %d, want %d", sg.MaxBet(), DefaultMaxBet)
	}
	if sg.Cooldown() != DefaultCooldown {
		t.Errorf("Cooldown() = %d, want %d", sg.Cooldown(), DefaultCooldown)
	}
}

func TestSlotGame_ValidateBet(t *testing.T) {
	sg := New(&Config{MaxBet: 1000})

	tests := []struct {
		name    string
		bet     int64
		wantErr bool
	}{
		{"valid bet", 100, false},
		{"zero bet", 0, true},
		{"negative bet", -100, true},
		{"bet too high", 2000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sg.ValidateBet(tt.bet, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBet(%d) error = %v, wantErr %v", tt.bet, err, tt.wantErr)
			}
		})
	}
}

func TestSlotGame_Play(t *testing.T) {
	sg := New(nil)
	ctx := context.Background()

	tests := []struct {
		name       string
		bet        int64
		slotValue  int
		wantPayout int64
		wantErr    bool
	}{
		{"three matches (1,1,1)", 100, 1, 300, false},
		{"three matches (2,2,2)", 100, 22, 300, false},
		{"two matches", 100, 2, 0, false},      // (2,1,1)
		{"no match", 100, 7, -100, false},      // (3,2,1) - all different
		{"missing slot value", 100, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{}
			if tt.slotValue > 0 {
				params["slot_value"] = tt.slotValue
			}

			result, err := sg.Play(ctx, 12345, tt.bet, params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Play() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && result.Payout != tt.wantPayout {
				t.Errorf("Play() payout = %d, want %d", result.Payout, tt.wantPayout)
			}
		})
	}
}


// TestSlotDecodeCorrectnessProperty tests the slot decode/encode round-trip property.
// **Feature: go-telegram-bot, Property 7: Slot Decode Correctness**
// *For any* slot value V ∈ [1,64]:
// - DecodeSlot(V) produces (left, middle, right) where each ∈ [1,4]
// - EncodeSlot(left, middle, right) = V (round-trip)
// **Validates: Requirements 4.4**
func TestSlotDecodeCorrectnessProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random slot value in valid range [1,64]
		slotValue := rapid.IntRange(1, 64).Draw(t, "slotValue")

		// Decode the slot value
		left, middle, right := DecodeSlot(slotValue)

		// Property 7a: Each symbol must be in range [1,4]
		if left < 1 || left > 4 {
			t.Fatalf("DecodeSlot(%d): left symbol %d not in range [1,4]", slotValue, left)
		}
		if middle < 1 || middle > 4 {
			t.Fatalf("DecodeSlot(%d): middle symbol %d not in range [1,4]", slotValue, middle)
		}
		if right < 1 || right > 4 {
			t.Fatalf("DecodeSlot(%d): right symbol %d not in range [1,4]", slotValue, right)
		}

		// Property 7b: Round-trip - EncodeSlot(DecodeSlot(V)) = V
		encoded := EncodeSlot(left, middle, right)
		if encoded != slotValue {
			t.Fatalf("Round-trip failed: DecodeSlot(%d) = (%d, %d, %d), EncodeSlot = %d, expected %d",
				slotValue, left, middle, right, encoded, slotValue)
		}
	})
}

// TestSlotEncodeDecodeRoundTripProperty tests the encode/decode round-trip from symbols.
// **Feature: go-telegram-bot, Property 7: Slot Decode Correctness**
// *For any* symbols (left, middle, right) where each ∈ [1,4]:
// - DecodeSlot(EncodeSlot(left, middle, right)) = (left, middle, right)
// **Validates: Requirements 4.4**
func TestSlotEncodeDecodeRoundTripProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random symbols in valid range [1,4]
		left := rapid.IntRange(1, 4).Draw(t, "left")
		middle := rapid.IntRange(1, 4).Draw(t, "middle")
		right := rapid.IntRange(1, 4).Draw(t, "right")

		// Encode then decode
		encoded := EncodeSlot(left, middle, right)
		decodedLeft, decodedMiddle, decodedRight := DecodeSlot(encoded)

		// Round-trip should preserve symbols
		if decodedLeft != left || decodedMiddle != middle || decodedRight != right {
			t.Fatalf("Round-trip failed: EncodeSlot(%d, %d, %d) = %d, DecodeSlot = (%d, %d, %d)",
				left, middle, right, encoded, decodedLeft, decodedMiddle, decodedRight)
		}

		// Encoded value should be in valid range [1,64]
		if encoded < 1 || encoded > 64 {
			t.Fatalf("EncodeSlot(%d, %d, %d) = %d, not in range [1,64]",
				left, middle, right, encoded)
		}
	})
}

// TestSlotPayoutCalculationProperty tests the slot payout calculation property.
// **Feature: go-telegram-bot, Property 8: Slot Payout Calculation**
// *For any* slot result and bet B:
// - If left == middle == right: payout > 0 (tiered by bet amount)
// - If exactly 2 symbols match: payout = 0
// - If no symbols match: payout = -B
// **Validates: Requirements 4.2**
func TestSlotPayoutCalculationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random symbols in valid range [1,4]
		left := rapid.IntRange(1, 4).Draw(t, "left")
		middle := rapid.IntRange(1, 4).Draw(t, "middle")
		right := rapid.IntRange(1, 4).Draw(t, "right")

		// Generate random positive bet amount
		bet := rapid.Int64Range(1, 100000).Draw(t, "bet")

		// Calculate payout
		payout := CalculatePayout(left, middle, right, bet)

		// Count matches
		threeMatch := left == middle && middle == right
		twoMatch := (left == middle || middle == right || left == right) && !threeMatch
		noMatch := left != middle && middle != right && left != right

		// Verify the property based on match count
		switch {
		case threeMatch:
			// Three matches: payout should be positive (tiered multiplier)
			if payout <= 0 {
				t.Fatalf("Three matches (%d, %d, %d) with bet %d should have positive payout, got %d",
					left, middle, right, bet, payout)
			}
			// Verify tiered multiplier
			var expectedMultiplier float64
			switch {
			case bet <= 1000:
				expectedMultiplier = 3.0
			case bet <= 10000:
				expectedMultiplier = 2.0
			case bet <= 100000:
				expectedMultiplier = 1.5
			default:
				expectedMultiplier = 1.0
			}
			expectedPayout := int64(float64(bet) * expectedMultiplier)
			if payout != expectedPayout {
				t.Fatalf("Three matches (%d, %d, %d) with bet %d: expected payout %d (%.1fx), got %d",
					left, middle, right, bet, expectedPayout, expectedMultiplier, payout)
			}

		case twoMatch:
			// Two matches: payout should be 0 (push)
			if payout != 0 {
				t.Fatalf("Two matches (%d, %d, %d) with bet %d should push (payout=0), got %d",
					left, middle, right, bet, payout)
			}

		case noMatch:
			// No matches: payout should be -bet (lose)
			if payout != -bet {
				t.Fatalf("No matches (%d, %d, %d) with bet %d should lose (payout=%d), got %d",
					left, middle, right, bet, -bet, payout)
			}

		default:
			t.Fatalf("Unexpected match state for symbols (%d, %d, %d)", left, middle, right)
		}
	})
}

// TestSlotPayoutTieredMultiplierProperty tests that payout multiplier is correctly tiered.
// **Feature: go-telegram-bot, Property 8: Slot Payout Calculation**
// *For any* three matching symbols and bet B:
// - bet <= 1000: multiplier = 3x
// - bet 1001-10000: multiplier = 2x
// - bet 10001-100000: multiplier = 1.5x
// - bet > 100000: multiplier = 1x
// **Validates: Requirements 4.2**
func TestSlotPayoutTieredMultiplierProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a matching symbol (all three same)
		symbol := rapid.IntRange(1, 4).Draw(t, "symbol")

		// Generate bet in different tiers
		tier := rapid.IntRange(1, 4).Draw(t, "tier")
		var bet int64
		var expectedMultiplier float64

		switch tier {
		case 1:
			bet = rapid.Int64Range(1, 1000).Draw(t, "bet")
			expectedMultiplier = 3.0
		case 2:
			bet = rapid.Int64Range(1001, 10000).Draw(t, "bet")
			expectedMultiplier = 2.0
		case 3:
			bet = rapid.Int64Range(10001, 100000).Draw(t, "bet")
			expectedMultiplier = 1.5
		case 4:
			bet = rapid.Int64Range(100001, 200000).Draw(t, "bet")
			expectedMultiplier = 1.0
		}

		// Calculate payout for three matching symbols
		payout := CalculatePayout(symbol, symbol, symbol, bet)
		expectedPayout := int64(float64(bet) * expectedMultiplier)

		if payout != expectedPayout {
			t.Fatalf("Three matches (symbol=%d) with bet %d (tier %d): expected payout %d (%.1fx), got %d",
				symbol, bet, tier, expectedPayout, expectedMultiplier, payout)
		}
	})
}
