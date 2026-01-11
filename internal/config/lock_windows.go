//go:build windows

package config

import "os"

func lockFilePlatform(file *os.File) error {
	return nil
}

func unlockFilePlatform(file *os.File) error {
	return nil
}
