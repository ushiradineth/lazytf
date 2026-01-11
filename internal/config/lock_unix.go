//go:build !windows

package config

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func lockFilePlatform(file *os.File) error {
	if file == nil {
		return errors.New("lock file is nil")
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock config: %w", err)
	}
	return nil
}

func unlockFilePlatform(file *os.File) error {
	if file == nil {
		return nil
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("unlock config: %w", err)
	}
	return nil
}
