package form

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/screen"
)

type edit struct {
	row     row
	input   textinput.Model
	choices []string
	chosen  int
}

func newTextEdit(r row, value string) edit {
	input := textinput.New()
	input.Cursor.SetMode(cursor.CursorStatic)
	input.SetValue(value)
	input.Focus()
	return edit{row: r, input: input}
}

func newListEdit(r row, choices []string, value string) edit {
	return edit{row: r, choices: choices, chosen: max(slices.Index(choices, value), 0)}
}

func (e edit) isList() bool {
	return len(e.choices) > 0
}

func (e edit) value() string {
	if e.isList() {
		return e.choices[e.chosen]
	}
	return e.input.Value()
}

func (e edit) moved(by int) edit {
	e.chosen = min(max(e.chosen+by, 0), len(e.choices)-1)
	return e
}

func (e edit) typed(msg tea.KeyMsg) (edit, tea.Cmd) {
	var cmd tea.Cmd
	e.input, cmd = e.input.Update(msg)
	return e, cmd
}

func (e edit) text(check error, width int) string {
	var lines []string
	for i, choice := range e.choices {
		if i == e.chosen {
			lines = append(lines, screen.SelectedRowLine(choice, width))
		} else {
			lines = append(lines, screen.RowLine("", choice, width))
		}
	}
	if !e.isList() {
		lines = append(lines, e.input.View())
	}
	if check != nil {
		lines = append(lines, "", screen.Message(check.Error(), width))
	}
	return strings.Join(lines, "\n")
}
