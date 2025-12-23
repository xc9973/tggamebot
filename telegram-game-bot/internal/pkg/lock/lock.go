// Package lock provides user-level locking for concurrent balance operations.
// Requirements: 9.1 - Per-user locks for balance operations
// Requirements: 9.2 - Prevent concurrent game sessions for same user
package lock

import (
	"context"
	"sync"
	"time"
)

// userMutex wraps a mutex with reference counting for cleanup.
type userMutex struct {
	mu       sync.Mutex
	refCount int
}

// UserLock provides per-user locking to prevent race conditions
// during balance operations and game sessions.
type UserLock struct {
	locks sync.Map // map[int64]*userMutex
	pool  sync.Pool
}

// NewUserLock creates a new UserLock instance.
func NewUserLock() *UserLock {
	return &UserLock{
		pool: sync.Pool{
			New: func() any {
				return &userMutex{}
			},
		},
	}
}

// getLock retrieves or creates a mutex for the given user ID.
func (ul *UserLock) getLock(userID int64) *userMutex {
	// Try to load existing lock
	if v, ok := ul.locks.Load(userID); ok {
		return v.(*userMutex)
	}

	// Create new lock from pool
	newLock := ul.pool.Get().(*userMutex)
	newLock.refCount = 0

	// Store or load existing (handles race condition)
	actual, loaded := ul.locks.LoadOrStore(userID, newLock)
	if loaded {
		// Another goroutine created the lock first, return ours to pool
		ul.pool.Put(newLock)
	}
	return actual.(*userMutex)
}

// Lock acquires the lock for a user.
// This should be called before any balance-modifying operation.
// Requirements: 9.1
func (ul *UserLock) Lock(userID int64) {
	lock := ul.getLock(userID)
	lock.mu.Lock()
	lock.refCount++
}

// Unlock releases the lock for a user.
// This should be called after balance-modifying operations complete.
// Requirements: 9.1
func (ul *UserLock) Unlock(userID int64) {
	if v, ok := ul.locks.Load(userID); ok {
		lock := v.(*userMutex)
		lock.refCount--
		lock.mu.Unlock()
	}
}

// TryLock attempts to acquire the lock without blocking.
// Returns true if the lock was acquired, false otherwise.
// Requirements: 9.2
func (ul *UserLock) TryLock(userID int64) bool {
	lock := ul.getLock(userID)
	if lock.mu.TryLock() {
		lock.refCount++
		return true
	}
	return false
}

// LockWithTimeout attempts to acquire the lock with a timeout.
// Returns true if the lock was acquired, false if timeout occurred.
// Requirements: 9.1, 9.2
func (ul *UserLock) LockWithTimeout(ctx context.Context, userID int64, timeout time.Duration) bool {
	lock := ul.getLock(userID)

	// Create a channel to signal lock acquisition
	done := make(chan struct{})

	go func() {
		lock.mu.Lock()
		close(done)
	}()

	// Create timeout context if not already set
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-done:
		lock.refCount++
		return true
	case <-timeoutCtx.Done():
		// Timeout occurred - we need to handle the goroutine that's waiting for the lock
		// It will eventually acquire and release, but we return false
		go func() {
			<-done // Wait for lock acquisition
			lock.mu.Unlock()
		}()
		return false
	}
}

// WithLock executes a function while holding the user's lock.
// This is a convenience method that ensures proper lock/unlock.
// Requirements: 9.1
func (ul *UserLock) WithLock(userID int64, fn func() error) error {
	ul.Lock(userID)
	defer ul.Unlock(userID)
	return fn()
}

// WithLockContext executes a function while holding the user's lock,
// with context support for cancellation.
// Requirements: 9.1
func (ul *UserLock) WithLockContext(ctx context.Context, userID int64, timeout time.Duration, fn func() error) error {
	if !ul.LockWithTimeout(ctx, userID, timeout) {
		return ErrLockTimeout
	}
	defer ul.Unlock(userID)

	// Check if context was cancelled while waiting for lock
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fn()
	}
}

// IsLocked checks if a user currently has an active lock.
// Note: This is a point-in-time check and may change immediately after.
// Requirements: 9.2
func (ul *UserLock) IsLocked(userID int64) bool {
	if v, ok := ul.locks.Load(userID); ok {
		lock := v.(*userMutex)
		// Try to acquire and immediately release to check if locked
		if lock.mu.TryLock() {
			lock.mu.Unlock()
			return false
		}
		return true
	}
	return false
}

// DefaultUserLock is the global user lock instance.
var DefaultUserLock = NewUserLock()

// Lock acquires the lock for a user using the default instance.
func Lock(userID int64) {
	DefaultUserLock.Lock(userID)
}

// Unlock releases the lock for a user using the default instance.
func Unlock(userID int64) {
	DefaultUserLock.Unlock(userID)
}

// TryLock attempts to acquire the lock using the default instance.
func TryLock(userID int64) bool {
	return DefaultUserLock.TryLock(userID)
}

// WithLock executes a function while holding the lock using the default instance.
func WithLock(userID int64, fn func() error) error {
	return DefaultUserLock.WithLock(userID, fn)
}
