// Package main (lock_windows.go) implements the single-instance lock on
// Windows using a named mutex.
package main

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// winLock holds the Windows mutex handle.
type winLock struct {
	handle windows.Handle
}

// mutexName is the global mutex name used to detect a running instance.
const mutexName = "Global\\SubrelaySingleInstance"

func acquireLockPlatform() (lockHandle, error) {
	name, _ := windows.UTF16PtrFromString(mutexName)
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		// CreateMutex returns ERROR_ALREADY_EXISTS when another
		// instance holds the mutex; the handle is still valid and
		// must be closed.
		if err == windows.ERROR_ALREADY_EXISTS {
			windows.CloseHandle(handle)
			return nil, fmt.Errorf("another instance is already running")
		}
		return nil, fmt.Errorf("create mutex: %w", err)
	}
	return &winLock{handle: handle}, nil
}

func (l *winLock) release() {
	if l.handle != 0 {
		windows.CloseHandle(l.handle)
		l.handle = 0
	}
}
