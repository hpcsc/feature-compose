package screen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type Pane struct {
	Title  string
	Lines  []string
	Note   string
	Width  int
	Height int
	Border lipgloss.Style
}

func (p Pane) InnerWidth() int {
	return p.Width - 4
}

func (p Pane) View() string {
	title := ansi.Truncate(p.Title, max(p.Width-6, 0), "…")
	rendered := make([]string, 0, p.Height)
	rendered = append(rendered, p.Border.Render("╭─")+" "+paneTitleStyle.Render(title)+" "+p.Border.Render(strings.Repeat("─", max(p.Width-5-lipgloss.Width(title), 0))+"╮"))
	for i := range p.Height - 2 {
		line := ""
		if i < len(p.Lines) {
			line = ansi.Truncate(p.Lines[i], p.InnerWidth(), "…")
		}
		rendered = append(rendered, p.Border.Render("│")+" "+padded(line, p.InnerWidth())+" "+p.Border.Render("│"))
	}
	bottom := "╰" + strings.Repeat("─", max(p.Width-2, 0)) + "╯"
	if p.Note != "" {
		bottom = "╰" + strings.Repeat("─", max(p.Width-5-lipgloss.Width(p.Note), 0)) + " " + p.Note + " ─╯"
	}
	rendered = append(rendered, p.Border.Render(bottom))
	return strings.Join(rendered, "\n")
}
