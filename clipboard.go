package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func copyToClipboard(text string) error {
	text = strings.TrimSpace(text)
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	case "linux":
		if cmd := exec.Command("xclip", "-selection", "clipboard"); tryPipe(cmd, text) == nil {
			return nil
		}
		if cmd := exec.Command("xsel", "--clipboard", "--input"); tryPipe(cmd, text) == nil {
			return nil
		}
		return fmt.Errorf("xclip or xsel required on Linux")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
}

func tryPipe(cmd *exec.Cmd, text string) error {
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
