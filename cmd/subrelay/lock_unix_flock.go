//go:build !windows

// Package main (lock_unix_flock.go) provides the flock syscall wrapper
// for the single-instance lock on Unix systems.
package main

import (
	"os"
	"syscall"
)

// tryFlock attempts a non-blocking exclusive lock on the file.
func tryFlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// releaseFlock releases the lock on the file.
func releaseFlock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
