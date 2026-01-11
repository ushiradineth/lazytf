package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type fileLock struct {
	file *os.File
}

var (
	lockFilePlatformFunc   = lockFilePlatform
	unlockFilePlatformFunc = unlockFilePlatform
)

func lockFile(path string) (*fileLock, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := lockFilePlatformFunc(file); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			// Best effort close after lock failure.
			_ = closeErr
		}
		return nil, err
	}
	return &fileLock{file: file}, nil
}

func (l *fileLock) Unlock() (err error) {
	if l == nil || l.file == nil {
		return nil
	}

	// Ensure file is closed even if panic occurs
	defer func() {
		if r := recover(); r != nil {
			if err := l.file.Close(); err != nil {
				// Best effort close on panic.
				_ = err
			}
			panic(r)
		}
	}()

	// Ensure we always attempt to close the file
	unlockErr := unlockFilePlatformFunc(l.file)
	closeErr := l.file.Close()

	// Return both errors if both failed
	if unlockErr != nil && closeErr != nil {
		return fmt.Errorf("unlock/close failed: %w", errors.Join(unlockErr, closeErr))
	}
	if unlockErr != nil {
		return fmt.Errorf("unlock failed: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close lock file failed: %w", closeErr)
	}
	return nil
}
