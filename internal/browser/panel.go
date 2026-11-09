package browser

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type panel struct {
	title  string
	lines  []string
	note   string
	width  int
	height int
	border lipgloss.Style
}

func (p panel) innerWidth() int {
	return p.width - 4
}

func (p panel) view() string {
	title := ansi.Truncate(p.title, max(p.width-6, 0), "…")
	rendered := make([]string, 0, p.height)
	rendered = append(rendered, p.border.Render("╭─")+" "+panelTitleStyle.Render(title)+" "+p.border.Render(strings.Repeat("─", max(p.width-5-lipgloss.Width(title), 0))+"╮"))
	for i := range p.height - 2 {
		line := ""
		if i < len(p.lines) {
			line = ansi.Truncate(p.lines[i], p.innerWidth(), "…")
		}
		rendered = append(rendered, p.border.Render("│")+" "+padded(line, p.innerWidth())+" "+p.border.Render("│"))
	}
	bottom := "╰" + strings.Repeat("─", max(p.width-2, 0)) + "╯"
	if p.note != "" {
		bottom = "╰" + strings.Repeat("─", max(p.width-5-lipgloss.Width(p.note), 0)) + " " + p.note + " ─╯"
	}
	rendered = append(rendered, p.border.Render(bottom))
	return strings.Join(rendered, "\n")
}
