//go:build !windows

package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"syscall"
)

func lockFilePlatform(file *os.File) error {
	if file == nil {
		return errors.New("lock file is nil")
	}
	fd := file.Fd()
	if fd > uintptr(math.MaxInt) {
		return errors.New("lock file descriptor out of range")
	}
	if err := syscall.Flock(int(fd), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock config: %w", err)
	}
	return nil
}

func unlockFilePlatform(file *os.File) error {
	if file == nil {
		return nil
	}
	fd := file.Fd()
	if fd > uintptr(math.MaxInt) {
		return errors.New("lock file descriptor out of range")
	}
	if err := syscall.Flock(int(fd), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("unlock config: %w", err)
	}
	return nil
}
