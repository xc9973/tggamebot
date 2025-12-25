// Package bot provides middleware for the Telegram bot.
// Property-based tests for middleware functions.
// **Feature: go-telegram-bot, Property 11: Admin Permission Check**
// **Feature: go-telegram-bot, Property 12: Whitelist Enforcement**
// **Validates: Requirements 6.4, 7.1**
package bot

import (
	"testing"

	"pgregory.net/rapid"

	"telegram-game-bot/internal/config"
)

// TestAdminPermissionCheckProperty tests the admin permission check logic.
// Property 11: Admin Permission Check
// *For any* admin command execution:
// - If user_id NOT IN admin_ids, command SHALL fail with permission error
// - If user_id IN admin_ids, command SHALL execute
// **Validates: Requirements 6.4**
func TestAdminPermissionCheckProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a list of admin IDs (1-10 admins)
		numAdmins := rapid.IntRange(1, 10).Draw(t, "numAdmins")
		adminIDs := make([]int64, numAdmins)
		for i := 0; i < numAdmins; i++ {
			adminIDs[i] = rapid.Int64Range(1, 1000000000).Draw(t, "adminID")
		}

		// Create config with these admin IDs
		cfg := &config.Config{
			Admin: config.AdminConfig{
				IDs: adminIDs,
			},
		}

		// Generate a user ID to test
		userID := rapid.Int64Range(1, 1000000000).Draw(t, "userID")

		// Check if user is admin using the config method
		isAdmin := cfg.IsAdmin(userID)

		// Verify the property: user should be admin if and only if their ID is in the admin list
		expectedIsAdmin := false
		for _, id := range adminIDs {
			if id == userID {
				expectedIsAdmin = true
				break
			}
		}

		if isAdmin != expectedIsAdmin {
			t.Fatalf("Admin check mismatch: userID=%d, adminIDs=%v, expected=%v, got=%v",
				userID, adminIDs, expectedIsAdmin, isAdmin)
		}
	})
}

// TestAdminPermissionCheckWithKnownAdminProperty tests that known admins are always recognized.
// **Validates: Requirements 6.4**
func TestAdminPermissionCheckWithKnownAdminProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a list of admin IDs (1-10 admins)
		numAdmins := rapid.IntRange(1, 10).Draw(t, "numAdmins")
		adminIDs := make([]int64, numAdmins)
		for i := 0; i < numAdmins; i++ {
			adminIDs[i] = rapid.Int64Range(1, 1000000000).Draw(t, "adminID")
		}

		// Create config with these admin IDs
		cfg := &config.Config{
			Admin: config.AdminConfig{
				IDs: adminIDs,
			},
		}

		// Pick a random admin from the list
		adminIndex := rapid.IntRange(0, numAdmins-1).Draw(t, "adminIndex")
		knownAdminID := adminIDs[adminIndex]

		// This admin should always be recognized
		if !cfg.IsAdmin(knownAdminID) {
			t.Fatalf("Known admin ID %d should be recognized as admin, adminIDs=%v", knownAdminID, adminIDs)
		}
	})
}

// TestAdminPermissionCheckWithNonAdminProperty tests that non-admins are never recognized as admins.
// **Validates: Requirements 6.4**
func TestAdminPermissionCheckWithNonAdminProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a list of admin IDs (1-10 admins)
		numAdmins := rapid.IntRange(1, 10).Draw(t, "numAdmins")
		adminIDs := make([]int64, numAdmins)
		adminSet := make(map[int64]bool)
		for i := 0; i < numAdmins; i++ {
			adminIDs[i] = rapid.Int64Range(1, 1000000000).Draw(t, "adminID")
			adminSet[adminIDs[i]] = true
		}

		// Create config with these admin IDs
		cfg := &config.Config{
			Admin: config.AdminConfig{
				IDs: adminIDs,
			},
		}

		// Generate a user ID that is NOT in the admin list
		var nonAdminID int64
		for {
			nonAdminID = rapid.Int64Range(1, 1000000000).Draw(t, "nonAdminID")
			if !adminSet[nonAdminID] {
				break
			}
		}

		// This user should NOT be recognized as admin
		if cfg.IsAdmin(nonAdminID) {
			t.Fatalf("Non-admin ID %d should NOT be recognized as admin, adminIDs=%v", nonAdminID, adminIDs)
		}
	})
}

// TestWhitelistEnforcementProperty tests the whitelist enforcement logic.
// Property 12: Whitelist Enforcement
// *For any* command in a group chat:
// - If chat_id NOT IN allowed_chats, command SHALL be ignored
// - If chat_id IN allowed_chats, command SHALL be processed
// **Validates: Requirements 7.1**
func TestWhitelistEnforcementProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a list of whitelisted chat IDs (1-10 chats)
		numChats := rapid.IntRange(1, 10).Draw(t, "numChats")
		chatIDs := make([]int64, numChats)
		for i := 0; i < numChats; i++ {
			// Group chat IDs are typically negative
			chatIDs[i] = -rapid.Int64Range(1, 1000000000).Draw(t, "chatID")
		}

		// Create config with these whitelisted chats
		cfg := &config.Config{
			Whitelist: config.WhitelistConfig{
				Chats: chatIDs,
			},
		}

		// Generate a chat ID to test
		testChatID := -rapid.Int64Range(1, 1000000000).Draw(t, "testChatID")

		// Check if chat is allowed using the config method
		isAllowed := cfg.IsChatAllowed(testChatID)

		// Verify the property: chat should be allowed if and only if its ID is in the whitelist
		expectedIsAllowed := false
		for _, id := range chatIDs {
			if id == testChatID {
				expectedIsAllowed = true
				break
			}
		}

		if isAllowed != expectedIsAllowed {
			t.Fatalf("Whitelist check mismatch: chatID=%d, whitelistedChats=%v, expected=%v, got=%v",
				testChatID, chatIDs, expectedIsAllowed, isAllowed)
		}
	})
}

// TestWhitelistEnforcementWithKnownChatProperty tests that known whitelisted chats are always allowed.
// **Validates: Requirements 7.1**
func TestWhitelistEnforcementWithKnownChatProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a list of whitelisted chat IDs (1-10 chats)
		numChats := rapid.IntRange(1, 10).Draw(t, "numChats")
		chatIDs := make([]int64, numChats)
		for i := 0; i < numChats; i++ {
			chatIDs[i] = -rapid.Int64Range(1, 1000000000).Draw(t, "chatID")
		}

		// Create config with these whitelisted chats
		cfg := &config.Config{
			Whitelist: config.WhitelistConfig{
				Chats: chatIDs,
			},
		}

		// Pick a random chat from the whitelist
		chatIndex := rapid.IntRange(0, numChats-1).Draw(t, "chatIndex")
		knownChatID := chatIDs[chatIndex]

		// This chat should always be allowed
		if !cfg.IsChatAllowed(knownChatID) {
			t.Fatalf("Known whitelisted chat ID %d should be allowed, whitelistedChats=%v", knownChatID, chatIDs)
		}
	})
}

// TestWhitelistEnforcementWithNonWhitelistedChatProperty tests that non-whitelisted chats are rejected.
// **Validates: Requirements 7.1**
func TestWhitelistEnforcementWithNonWhitelistedChatProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a list of whitelisted chat IDs (1-10 chats)
		numChats := rapid.IntRange(1, 10).Draw(t, "numChats")
		chatIDs := make([]int64, numChats)
		chatSet := make(map[int64]bool)
		for i := 0; i < numChats; i++ {
			chatIDs[i] = -rapid.Int64Range(1, 1000000000).Draw(t, "chatID")
			chatSet[chatIDs[i]] = true
		}

		// Create config with these whitelisted chats
		cfg := &config.Config{
			Whitelist: config.WhitelistConfig{
				Chats: chatIDs,
			},
		}

		// Generate a chat ID that is NOT in the whitelist
		var nonWhitelistedChatID int64
		for {
			nonWhitelistedChatID = -rapid.Int64Range(1, 1000000000).Draw(t, "nonWhitelistedChatID")
			if !chatSet[nonWhitelistedChatID] {
				break
			}
		}

		// This chat should NOT be allowed
		if cfg.IsChatAllowed(nonWhitelistedChatID) {
			t.Fatalf("Non-whitelisted chat ID %d should NOT be allowed, whitelistedChats=%v", nonWhitelistedChatID, chatIDs)
		}
	})
}

// TestEmptyWhitelistAllowsAllChatsProperty tests that an empty whitelist allows all chats.
// This is a special case in the implementation.
// **Validates: Requirements 7.1**
func TestEmptyWhitelistAllowsAllChatsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create config with empty whitelist
		cfg := &config.Config{
			Whitelist: config.WhitelistConfig{
				Chats: []int64{},
			},
		}

		// Generate any chat ID
		chatID := -rapid.Int64Range(1, 1000000000).Draw(t, "chatID")

		// With empty whitelist, all chats should be allowed
		if !cfg.IsChatAllowed(chatID) {
			t.Fatalf("With empty whitelist, chat ID %d should be allowed", chatID)
		}
	})
}

// TestPrivateUserCacheProperty tests the private user cache functionality.
// **Validates: Requirements 7.2**
func TestPrivateUserCacheProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a user ID
		userID := rapid.Int64Range(1, 1000000000).Draw(t, "userID")

		// Initially, user should not be in cache (unless added by previous test)
		// We test the round-trip: add user, then check

		// Add user to cache
		AllowPrivateUser(userID)

		// User should now be allowed
		if !IsPrivateUserAllowed(userID) {
			t.Fatalf("User %d should be allowed after being added to private user cache", userID)
		}
	})
}
