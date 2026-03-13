package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type selectBox struct {
	options []string
	current int
	open    bool
}

func newSelectBox(options []string, initial string) selectBox {
	sb := selectBox{options: options}
	for i, o := range options {
		if o == initial {
			sb.current = i
			break
		}
	}
	return sb
}

func (s *selectBox) value() string {
	if len(s.options) == 0 {
		return ""
	}
	return s.options[s.current]
}

func (s *selectBox) set(val string) {
	for i, o := range s.options {
		if o == val {
			s.current = i
			return
		}
	}
}

func (s *selectBox) next() {
	if len(s.options) > 0 {
		s.current = (s.current + 1) % len(s.options)
	}
}

func (s *selectBox) prev() {
	if len(s.options) > 0 {
		s.current = (s.current - 1 + len(s.options)) % len(s.options)
	}
}

func (s *selectBox) handleKey(key string, action string) (bool, bool) {
	if s.open {
		switch key {
		case "down":
			prev := s.current
			s.next()
			return true, s.current != prev
		case "up":
			prev := s.current
			s.prev()
			return true, s.current != prev
		case "enter", " ":
			s.open = false
			return true, false
		case "esc":
			s.open = false
			return true, false
		}
		return false, false
	}
	switch {
	case key == "enter" || key == " " || action == "insert_mode":
		s.open = true
		return true, false
	case key == "right":
		prev := s.current
		s.next()
		return true, s.current != prev
	case key == "left":
		prev := s.current
		s.prev()
		return true, s.current != prev
	}
	return false, false
}

func (s *selectBox) isOpen() bool {
	return s.open
}

func (s *selectBox) renderInline(focused bool) string {
	val := s.value()
	if !focused {
		return "[" + val + " ▾]"
	}
	if !s.open {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00d7ff")).
			Render("[" + val + " ▾]")
	}
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d7ff")).
		Render("[" + val + " ▾]")
	var items []string
	for i, opt := range s.options {
		if i == s.current {
			items = append(items, lipgloss.NewStyle().
				Background(lipgloss.Color("#00d7ff")).
				Foreground(lipgloss.Color("#000000")).
				Render("▶ "+opt+" "))
		} else {
			items = append(items, "  "+opt+" ")
		}
	}
	dropdown := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00d7ff")).
		Render(strings.Join(items, "\n"))
	return header + "\n" + dropdown
}
