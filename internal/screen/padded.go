package screen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func padded(text string, width int) string {
	return text + strings.Repeat(" ", max(width-lipgloss.Width(text), 0))
}
