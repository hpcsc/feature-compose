package screen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Header(title, counts string, width int) string {
	renderedTitle := titleStyle.Render(title)
	renderedCounts := countStyle.Render(counts)
	gap := width - lipgloss.Width(renderedTitle) - lipgloss.Width(renderedCounts) - 1
	if gap < 1 {
		return renderedTitle
	}
	return renderedTitle + strings.Repeat(" ", gap) + renderedCounts + " "
}
