// Package main (lock.go) provides the single-instance lock interface.
// The platform-specific implementation is selected via build constraints.
package main

// lockHandle is the platform-specific lock resource.
type lockHandle interface {
	release()
}

// acquireLock obtains the single-instance lock.
//
// Returns:
//   - A lockHandle on success.
//
// Errors:
//   - Returns an error when another instance holds the lock.
func acquireLock() (lockHandle, error) {
	return acquireLockPlatform()
}

// releaseLock releases the single-instance lock.
func releaseLock(h lockHandle) {
	if h != nil {
		h.release()
	}
}
