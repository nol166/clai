package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf16"
)

func copyToClipboard(text string) error {
	text = strings.TrimSpace(text)
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	case "windows":
		return clipExe("clip", text)
	case "linux":
		if cmd := exec.Command("xclip", "-selection", "clipboard"); tryPipe(cmd, text) == nil {
			return nil
		}
		if cmd := exec.Command("xsel", "--clipboard", "--input"); tryPipe(cmd, text) == nil {
			return nil
		}
		// WSL: Windows interop puts clip.exe on PATH
		if clipExe("clip.exe", text) == nil {
			return nil
		}
		return fmt.Errorf("xclip, xsel, or clip.exe (WSL) required on Linux")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
}

// clipExe pipes text to Windows clip as UTF-16LE — clip assumes the
// system code page for plain bytes, garbling non-ASCII UTF-8. No BOM:
// clip auto-detects UTF-16 but would keep a BOM as literal text.
// ponytail: detection can misread strings with no ASCII at all (pure
// CJK); prepend a BOM-stripping layer only if that ever bites.
func clipExe(name, text string) error {
	cmd := exec.Command(name)
	cmd.Stdin = bytes.NewReader(encodeForClip(text))
	return cmd.Run()
}

func encodeForClip(text string) []byte {
	codes := utf16.Encode([]rune(text))
	b := make([]byte, 0, len(codes)*2)
	for _, c := range codes {
		b = append(b, byte(c), byte(c>>8))
	}
	return b
}

func tryPipe(cmd *exec.Cmd, text string) error {
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
