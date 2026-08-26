//go:build !windows

// Package main (lock_unix.go) implements the single-instance lock on
// Linux and other Unix-like systems using a file lock in the data
// directory.
package main

import (
	"fmt"
	"os"

	"subrelay/internal/paths"
)

// unixLock holds the lock file handle.
type unixLock struct {
	file *os.File
}

func acquireLockPlatform() (lockHandle, error) {
	path, err := paths.LockFile()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	// Try a non-blocking exclusive lock via flock.
	if err := tryFlock(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("another instance is already running: %w", err)
	}
	return &unixLock{file: f}, nil
}

func (l *unixLock) release() {
	if l.file != nil {
		releaseFlock(l.file)
		l.file.Close()
		l.file = nil
	}
}
