package screen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func RowLine(prefix, text string, width int) string {
	return " " + ansi.Truncate(treeLineStyle.Render(prefix)+text, width-1, "…")
}

func SelectedRowLine(text string, width int) string {
	truncated := ansi.Truncate(text, width-1, "…")
	return "▌" + selectedRowStyle.Render(truncated+strings.Repeat(" ", max(width-1-lipgloss.Width(truncated), 0)))
}
