package form

import (
	"cmp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hpcsc/feature-compose/internal/screen"
)

type control int

const (
	fixedControl control = iota
	textControl
	selectControl
	checklistControl
	buttonControl
)

const writeButton = " Write the repo "

var writeField = field{row: row{kind: writeKind}, control: buttonControl}

type panel struct {
	title  string
	count  string
	fields []field
}

type item struct {
	name    string
	checked bool
	mark    string
	summary string
	hint    string
}

type field struct {
	row     row
	control control
	label   string
	value   string
	mark    string
	missing bool
	summary string
	hint    string
	items   []item
}

type fieldState struct {
	focused  bool
	cursor   int
	text     string
	input    string
	problems []string
}

func (f field) canFocus() bool {
	return f.control != fixedControl
}

func (f field) isList() bool {
	return f.control == selectControl || f.control == checklistControl
}

func (f field) checkedItem() int {
	for i, choice := range f.items {
		if choice.checked {
			return i
		}
	}
	return -1
}

func (f field) lines(state fieldState, width int) (lines []string, focusStart, focusEnd int) {
	if f.control == checklistControl {
		return f.checklistLines(state, width)
	}
	controlColumn := 2 + lipgloss.Width(f.label)
	switch {
	case f.control == buttonControl:
		lines = []string{f.buttonLine(state, width)}
	case f.control == selectControl && state.focused:
		lines = f.listLines(state, width)
	case f.control == textControl && state.focused:
		lines = []string{ansi.Truncate("▌ "+screen.FocusStyle.Render(f.label)+state.input+f.notes(state), width, "…")}
	default:
		lines = []string{screen.RowLine("", " "+screen.LabelStyle.Render(f.label)+f.valueView(state)+f.notes(state), width)}
	}
	if state.focused && f.hint != "" {
		lines = append(lines, under(screen.MarkStyle, f.hint, controlColumn, width)...)
	}
	for _, problem := range state.problems {
		lines = append(lines, under(screen.ProblemStyle, problem, controlColumn, width)...)
	}
	focusEnd = len(lines) - 1
	if f.control == selectControl && state.focused {
		focusStart = state.cursor
		if state.cursor < len(f.items)-1 && len(state.problems) == 0 {
			focusEnd = state.cursor
		}
	}
	return lines, focusStart, focusEnd
}

func (f field) valueView(state fieldState) string {
	if f.control == textControl {
		return state.text
	}
	return f.value
}

func (f field) notes(state fieldState) string {
	var notes []string
	if f.mark != "" {
		notes = append(notes, screen.MarkStyle.Render(f.mark))
	}
	if f.missing && len(state.problems) == 0 {
		notes = append(notes, screen.ProblemStyle.Render("missing"))
	}
	if len(notes) == 0 && f.summary != "" {
		notes = append(notes, screen.MarkStyle.Render(f.summary))
	}
	if len(notes) == 0 {
		return ""
	}
	return "  " + strings.Join(notes, screen.MarkStyle.Render(", "))
}

func (f field) listLines(state fieldState, width int) []string {
	widestName := f.widestItem()
	blank := strings.Repeat(" ", len(f.label))
	var lines, cursorSummary []string
	for i, choice := range f.items {
		label, styledLabel := blank, blank
		if i == 0 {
			label, styledLabel = f.label, screen.FocusStyle.Render(f.label)
		}
		radio := "○ "
		if choice.checked {
			radio = "● "
		}
		summary := ""
		if choice.summary != "" {
			summary = "  " + choice.summary
		}
		if i == state.cursor {
			row := " " + label + "› " + radio + padded(choice.name, widestName) + summary
			lines = append(lines, screen.SelectedRowLine(row, width))
			if lipgloss.Width(row) > width-1 {
				cursorSummary = under(screen.MarkStyle, choice.summary, 6+len(label), width)
			}
			continue
		}
		name := radio + choice.name
		if choice.checked {
			name = screen.NameStyle.Render(name)
		}
		lines = append(lines, screen.RowLine("", " "+styledLabel+"  "+name+strings.Repeat(" ", widestName-len(choice.name))+screen.MarkStyle.Render(summary), width))
	}
	return append(lines, cursorSummary...)
}

func (f field) checklistLines(state fieldState, width int) (lines []string, focusStart, focusEnd int) {
	widestName := f.widestItem()
	nameColumn := 2 + len("[x] ")
	for i, choice := range f.items {
		box := "[ ] "
		if choice.checked {
			box = "[x] "
		}
		if !state.focused || i != state.cursor {
			style := screen.NotSelectedStyle
			if choice.checked {
				style = screen.NameStyle
			}
			row := " " + style.Render(box+choice.name) + strings.Repeat(" ", widestName-len(choice.name))
			if note := cmp.Or(choice.mark, choice.summary); note != "" {
				row += "  " + screen.MarkStyle.Render(note)
			}
			lines = append(lines, screen.RowLine("", row, width))
			continue
		}
		aboutColumn := nameColumn + widestName + 2
		if width-aboutColumn < 20 {
			aboutColumn = nameColumn
		}
		note := choice.mark
		var about []string
		if choice.summary != "" {
			wrapped := screen.Wrapped(choice.summary, width-aboutColumn)
			if note == "" && aboutColumn > nameColumn {
				note, wrapped = wrapped[0], wrapped[1:]
			}
			about = indented(screen.MarkStyle, wrapped, aboutColumn)
		}
		if choice.hint != "" {
			about = append(about, under(screen.MarkStyle, choice.hint, aboutColumn, width)...)
		}
		row := " " + box + padded(choice.name, widestName)
		if note != "" {
			row += "  " + note
		}
		focusStart = len(lines)
		lines = append(lines, screen.SelectedRowLine(row, width))
		lines = append(lines, about...)
		for _, problem := range state.problems {
			lines = append(lines, under(screen.ProblemStyle, problem, 2, width)...)
		}
		focusEnd = len(lines) - 1
	}
	return lines, focusStart, focusEnd
}

func (f field) widestItem() int {
	names := make([]string, 0, len(f.items))
	for _, choice := range f.items {
		names = append(names, choice.name)
	}
	return widest(names)
}

func (f field) buttonLine(state fieldState, width int) string {
	button := " " + screen.ButtonStyle.Render(writeButton) + " "
	if state.focused {
		button = screen.FocusStyle.Render("▐") + screen.FocusButtonStyle.Render(writeButton) + screen.FocusStyle.Render("▌")
	}
	return strings.Repeat(" ", max(width-lipgloss.Width(button)-1, 0)) + button
}

func under(style lipgloss.Style, text string, column, width int) []string {
	return indented(style, screen.Wrapped(text, max(width-column, 1)), column)
}

func indented(style lipgloss.Style, lines []string, column int) []string {
	indented := make([]string, 0, len(lines))
	for _, line := range lines {
		indented = append(indented, strings.Repeat(" ", column)+style.Render(line))
	}
	return indented
}
