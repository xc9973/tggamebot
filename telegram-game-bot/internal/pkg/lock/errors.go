package lock

import "errors"

// Lock-related errors.
var (
	// ErrLockTimeout is returned when a lock cannot be acquired within the timeout period.
	ErrLockTimeout = errors.New("lock acquisition timeout")
)
