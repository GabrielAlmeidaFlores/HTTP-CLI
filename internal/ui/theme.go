package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/user/http-cli/internal/config"
)

func panelBorderStyle(focused bool, theme config.ThemeConfig) lipgloss.Style {
	color := theme.BlurBorder
	if focused {
		color = theme.FocusBorder
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color))
}

func methodColor(method string, theme config.ThemeConfig) string {
	switch method {
	case "GET":
		return theme.MethodGet
	case "POST":
		return theme.MethodPost
	case "PUT":
		return theme.MethodPut
	case "DELETE":
		return theme.MethodDelete
	case "PATCH":
		return theme.MethodPatch
	default:
		return "#ffffff"
	}
}

func accentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#00d7ff"))
}

func dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
}

func errorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#d70000"))
}

func successStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#00d700"))
}

func modalBorderStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color))
}

func modalWidth(screenW int) int {
	w := screenW * 3 / 4
	if w > 100 {
		w = 100
	}
	if w < 40 {
		w = 40
	}
	return w
}
