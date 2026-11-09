package browser

import "github.com/charmbracelet/lipgloss"

var (
	accent     = lipgloss.AdaptiveColor{Light: "#5B3FD0", Dark: "#A78BFA"}
	faint      = lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#7A7A7A"}
	borderLine = lipgloss.AdaptiveColor{Light: "#C4C4C4", Dark: "#4A4A4A"}

	appTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1B1530"}).Background(accent).Padding(0, 1)
	countStyle       = lipgloss.NewStyle().Foreground(faint)
	focusedBorder    = lipgloss.NewStyle().Foreground(accent)
	plainBorder      = lipgloss.NewStyle().Foreground(borderLine)
	panelTitleStyle  = lipgloss.NewStyle().Bold(true)
	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(accent).Background(lipgloss.AdaptiveColor{Light: "#ECE7FF", Dark: "#2B2440"})
	treeLineStyle    = lipgloss.NewStyle().Foreground(faint)
	groupStyle       = lipgloss.NewStyle().Foreground(accent)
	headingStyle     = lipgloss.NewStyle().Bold(true).Foreground(accent)
	nameStyle        = lipgloss.NewStyle().Bold(true)
	labelStyle       = lipgloss.NewStyle().Foreground(faint)
	noneStyle        = lipgloss.NewStyle().Foreground(faint).Italic(true)
)
