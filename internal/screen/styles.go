package screen

import "github.com/charmbracelet/lipgloss"

var (
	accent     = lipgloss.AdaptiveColor{Light: "#5B3FD0", Dark: "#A78BFA"}
	faint      = lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#7A7A7A"}
	borderLine = lipgloss.AdaptiveColor{Light: "#C4C4C4", Dark: "#4A4A4A"}
	problem    = lipgloss.AdaptiveColor{Light: "#D2003F", Dark: "#FF5F87"}
	onAccent   = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1B1530"}

	FocusedBorder    = lipgloss.NewStyle().Foreground(accent)
	PlainBorder      = lipgloss.NewStyle().Foreground(borderLine)
	NameStyle        = lipgloss.NewStyle().Bold(true)
	GroupStyle       = lipgloss.NewStyle().Foreground(accent)
	NoneStyle        = lipgloss.NewStyle().Foreground(faint).Italic(true)
	MarkStyle        = lipgloss.NewStyle().Foreground(faint)
	LabelStyle       = lipgloss.NewStyle().Foreground(faint)
	ProblemStyle     = lipgloss.NewStyle().Foreground(problem)
	NotSelectedStyle = lipgloss.NewStyle().Foreground(faint)
	HeadingStyle     = lipgloss.NewStyle().Bold(true).Foreground(accent)
	FocusStyle       = lipgloss.NewStyle().Bold(true).Foreground(accent)
	ButtonStyle      = lipgloss.NewStyle().Background(borderLine)
	FocusButtonStyle = lipgloss.NewStyle().Bold(true).Foreground(onAccent).Background(accent)

	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(onAccent).Background(accent).Padding(0, 1)
	countStyle       = lipgloss.NewStyle().Foreground(faint)
	paneTitleStyle   = lipgloss.NewStyle().Bold(true)
	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(accent).Background(lipgloss.AdaptiveColor{Light: "#ECE7FF", Dark: "#2B2440"})
	treeLineStyle    = lipgloss.NewStyle().Foreground(faint)
)
