package ui

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
)

var clipboardCopy = copyToClipboard

func copyToClipboard(value string) error {
	switch runtime.GOOS {
	case "darwin":
		return runClipboardCmd(value, "pbcopy")
	case "windows":
		return runClipboardCmd(value, "clip")
	default:
		if err := runClipboardCmd(value, "wl-copy"); err == nil {
			return nil
		}
		if err := runClipboardCmd(value, "xclip", "-selection", "clipboard"); err == nil {
			return nil
		}
		if err := runClipboardCmd(value, "xsel", "--clipboard", "--input"); err == nil {
			return nil
		}
		return errors.New("no clipboard command available (tried wl-copy, xclip, xsel)")
	}
}

func runClipboardCmd(value, name string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	if _, err := stdin.Write([]byte(value)); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return err
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	return cmd.Wait()
}
