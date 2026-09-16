//go:build windows

package app

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

// acquireTokenRefreshLock acquires an exclusive lock on lockPath using
// LockFileEx, polling until the ctx deadline. The returned function releases
// the lock and closes the file.
func acquireTokenRefreshLock(ctx context.Context, lockPath string) (func(), error) {
	pathp, err := windows.UTF16PtrFromString(lockPath)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		pathp,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	var ol windows.Overlapped
	for {
		err = windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
			0, 1, 0, &ol)
		if err == nil {
			return func() {
				_ = windows.UnlockFileEx(handle, 0, 1, 0, &ol)
				_ = windows.CloseHandle(handle)
			}, nil
		}
		// ERROR_LOCK_VIOLATION means held by another process; poll again.
		if err != windows.ERROR_LOCK_VIOLATION {
			windows.CloseHandle(handle)
			return nil, fmt.Errorf("acquire lock: %w", err)
		}

		select {
		case <-ctx.Done():
			windows.CloseHandle(handle)
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
