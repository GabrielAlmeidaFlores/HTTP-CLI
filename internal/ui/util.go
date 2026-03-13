package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width-1]) + "…"
}

func isPrintable(key string) bool {
	runes := []rune(key)
	if len(runes) != 1 {
		return false
	}
	r := runes[0]
	return r >= 32 && r != 127
}

func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/1024/1024)
	}
}

func insertAtCursor(val string, cursor int, text string) (string, int) {
	runes := []rune(val)
	textRunes := []rune(text)
	newRunes := make([]rune, len(runes)+len(textRunes))
	copy(newRunes, runes[:cursor])
	copy(newRunes[cursor:], textRunes)
	copy(newRunes[cursor+len(textRunes):], runes[cursor:])
	return string(newRunes), cursor + len(textRunes)
}

func overlayCenter(bg, fg string, w, h int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	fgH := len(fgLines)
	fgW := 0
	for _, l := range fgLines {
		if vw := lipgloss.Width(l); vw > fgW {
			fgW = vw
		}
	}

	startY := (h - fgH) / 2
	startX := (w - fgW) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	out := make([]string, len(bgLines))
	for i, bgLine := range bgLines {
		fgI := i - startY
		if fgI < 0 || fgI >= fgH {
			out[i] = bgLine
			continue
		}
		bgLineW := lipgloss.Width(bgLine)
		needed := startX + fgW
		if bgLineW < needed {
			bgLine += strings.Repeat(" ", needed-bgLineW)
		}
		left := ansi.Truncate(bgLine, startX, "")
		right := ansi.TruncateLeft(bgLine, startX+fgW, "")
		out[i] = left + fgLines[fgI] + right
	}
	return strings.Join(out, "\n")
}
