package form

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/screen"
)

var errCanceled = errors.New("you canceled new, so it wrote no file")

var keys = struct {
	next, previous, down, up, listDown, listUp, enter, space, quit, back, write, move, listMove, edit, choose, selectFeature, scroll key.Binding
}{
	next:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
	previous: key.NewBinding(key.WithKeys("shift+tab")),
	down:     key.NewBinding(key.WithKeys("down")),
	up:       key.NewBinding(key.WithKeys("up")),
	listDown: key.NewBinding(key.WithKeys("j")),
	listUp:   key.NewBinding(key.WithKeys("k")),
	enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "next")),
	space:    key.NewBinding(key.WithKeys(" ")),
	quit:     key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	write:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "write")),
	move:     key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "move")),
	listMove: key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "move")),
	// help.ShortHelpView skips a binding with no keys, and no key in the form matches this binding.
	edit:          key.NewBinding(key.WithKeys("runes"), key.WithHelp("type", "to edit")),
	choose:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "choose")),
	selectFeature: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
	scroll:        key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "scroll")),
}

var _ tea.Model = Model{}

type refusedText struct {
	text    string
	problem string
}

type Model struct {
	draft     draft
	focus     row
	cursor    int
	input     textinput.Model
	refused   map[row]refusedText
	message   string
	offset    int
	reviewing bool
	preview   string
	canWrite  bool
	review    viewport.Model
	help      help.Model
	width     int
	height    int
	confirmed bool
}

func New(c catalog.Catalog, starterName, name string, given map[string]string, givenByFeature map[string]map[string]string) (Model, error) {
	d, err := newDraft(c, starterName, name, given, givenByFeature)
	if err != nil {
		return Model{}, err
	}
	m := Model{draft: d, help: help.New()}
	return m.focused(firstFocusable(d.fields()), 1).laidOut(), nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.laidOut(), nil
	case tea.KeyMsg:
		if m.reviewing {
			return m.updateReview(msg)
		}
		return m.updateForm(msg)
	}
	if m.reviewing || m.focusedField().control != textControl {
		return m, nil
	}
	m, cmd := m.typed(msg)
	return m.laidOut(), cmd
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if !screen.Fits(m.width, m.height) {
		return screen.TooSmall("new", m.width)
	}
	bodyHeight := m.height - 2
	var body []string
	if m.reviewing {
		pane := screen.Pane{Title: "Preview", Lines: strings.Split(m.review.View(), "\n"), Width: m.width, Height: bodyHeight, Border: screen.FocusedBorder}
		if m.review.TotalLineCount() > m.review.Height {
			pane.Note = fmt.Sprintf("%d%%", int(m.review.ScrollPercent()*100))
		}
		body = strings.Split(pane.View(), "\n")
	} else {
		lines, _, _ := m.formLines()
		body = lines[min(m.offset, len(lines)):min(m.offset+bodyHeight, len(lines))]
	}
	for len(body) < bodyHeight {
		body = append(body, "")
	}
	header := screen.Header("feature-compose new", screen.Counted(m.draft.featureCount(), "feature"), m.width)
	return header + "\n" + strings.Join(body, "\n") + "\n" + m.footer()
}

func (m Model) Confirmed() (catalog.Selection, error) {
	if !m.confirmed {
		return catalog.Selection{}, errCanceled
	}
	return m.draft.selection, nil
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.message = ""
	f := m.focusedField()
	var cmd tea.Cmd
	switch {
	case key.Matches(msg, keys.quit):
		return m, tea.Quit
	case key.Matches(msg, keys.next):
		m = m.moved(1)
	case key.Matches(msg, keys.previous):
		m = m.moved(-1)
	case key.Matches(msg, keys.down):
		m = m.down(f)
	case key.Matches(msg, keys.up):
		m = m.up(f)
	case key.Matches(msg, keys.enter):
		m = m.entered(f)
	case f.control == textControl:
		m, cmd = m.typed(msg)
	case f.isList() && key.Matches(msg, keys.listDown):
		m.cursor = min(m.cursor+1, len(f.items)-1)
	case f.isList() && key.Matches(msg, keys.listUp):
		m.cursor = max(m.cursor-1, 0)
	case key.Matches(msg, keys.space):
		m = m.spaced(f)
	}
	return m.laidOut(), cmd
}

func (m Model) down(f field) Model {
	if f.isList() && m.cursor < len(f.items)-1 {
		m.cursor++
		return m
	}
	return m.moved(1)
}

func (m Model) up(f field) Model {
	if f.isList() && m.cursor > 0 {
		m.cursor--
		return m
	}
	return m.moved(-1)
}

func (m Model) entered(f field) Model {
	switch f.control {
	case selectControl:
		m = m.chosen(f)
		if m.message != "" {
			return m
		}
	case buttonControl:
		return m.reviewed()
	}
	return m.moved(1)
}

func (m Model) spaced(f field) Model {
	switch f.control {
	case selectControl:
		m = m.chosen(f)
	case checklistControl:
		m.draft, m.message = m.draft.toggled(f.items[m.cursor].name)
	case buttonControl:
		m = m.reviewed()
	}
	return m
}

func (m Model) chosen(f field) Model {
	d, err := m.draft.withAnswer(f.row, f.items[m.cursor].name)
	if err != nil {
		m.message = err.Error()
		return m
	}
	m.draft = d
	return m
}

func (m Model) typed(msg tea.Msg) (Model, tea.Cmd) {
	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() == before {
		return m, cmd
	}
	return m.withText(m.focus, m.input.Value()), cmd
}

func (m Model) withText(r row, text string) Model {
	m.refused = maps.Clone(m.refused)
	if m.refused == nil {
		m.refused = map[row]refusedText{}
	}
	delete(m.refused, r)
	if err := m.draft.check(r, text); err != nil {
		m.refused[r] = refusedText{text: text, problem: err.Error()}
		return m
	}
	d, err := m.draft.withAnswer(r, text)
	if err != nil {
		m.refused[r] = refusedText{text: text, problem: err.Error()}
		return m
	}
	m.draft = d
	return m
}

func (m Model) reviewed() Model {
	for _, f := range m.draft.fields() {
		if _, isRefused := m.refused[f.row]; isRefused && f.canFocus() {
			return m.focused(f.row, 1)
		}
	}
	if r, message, isMissing := m.draft.missingAnswer(); isMissing {
		if slices.ContainsFunc(m.draft.fields(), func(f field) bool { return f.row == r && f.canFocus() }) {
			m = m.focused(r, 1)
		}
		m.message = message
		return m
	}
	preview, err := m.draft.selection.Preview()
	m.reviewing, m.preview, m.canWrite = true, strings.TrimSuffix(preview, "\n"), err == nil
	if err != nil {
		m.preview = err.Error()
	}
	m.review.GotoTop()
	return m
}

func (m Model) updateReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.quit):
		return m, tea.Quit
	case m.canWrite && key.Matches(msg, keys.write):
		m.confirmed = true
		return m, tea.Quit
	case key.Matches(msg, keys.back):
		m.reviewing = false
		return m.laidOut(), nil
	}
	var cmd tea.Cmd
	m.review, cmd = m.review.Update(msg)
	return m, cmd
}

func (m Model) moved(by int) Model {
	var focusable []row
	for _, f := range m.draft.fields() {
		if f.canFocus() {
			focusable = append(focusable, f.row)
		}
	}
	i := slices.Index(focusable, m.focus)
	next := min(max(i+by, 0), len(focusable)-1)
	if next == i {
		return m
	}
	return m.focused(focusable[next], by)
}

func (m Model) focused(r row, by int) Model {
	m.focus = r
	f := m.focusedField()
	switch {
	case f.control == textControl:
		m.input = textinput.New()
		m.input.Prompt = ""
		m.input.Cursor.SetMode(cursor.CursorStatic)
		m.input.SetValue(m.textOf(r))
		m.input.Focus()
	case !f.isList():
	case f.control == selectControl && f.checkedItem() >= 0:
		m.cursor = f.checkedItem()
	case by < 0:
		m.cursor = len(f.items) - 1
	default:
		m.cursor = 0
	}
	return m
}

func (m Model) focusedField() field {
	for _, f := range m.draft.fields() {
		if f.row == m.focus {
			return f
		}
	}
	return writeField
}

func firstFocusable(fields []field) row {
	for _, f := range fields {
		if f.canFocus() {
			return f.row
		}
	}
	return writeField.row
}

func (m Model) textOf(r row) string {
	if refused, isRefused := m.refused[r]; isRefused {
		return refused.text
	}
	return m.draft.value(r)
}

func (m Model) laidOut() Model {
	fields := m.draft.fields()
	m.refused = maps.Clone(m.refused)
	maps.DeleteFunc(m.refused, func(r row, _ refusedText) bool {
		return !slices.ContainsFunc(fields, func(f field) bool { return f.row == r })
	})
	if !slices.ContainsFunc(fields, func(f field) bool { return f.row == m.focus && f.canFocus() }) {
		m = m.focused(firstFocusable(fields), 1)
	}
	if f := m.focusedField(); f.isList() {
		m.cursor = min(max(m.cursor, 0), len(f.items)-1)
	}
	m.review.Width, m.review.Height = max(m.width-4, 0), max(m.height-4, 0)
	if m.reviewing {
		m.review.SetContent(m.reviewText())
	}
	lines, first, last := m.formLines()
	bodyHeight := max(m.height-2, 1)
	offset := min(screen.OffsetShowing(m.offset, last, bodyHeight), first)
	m.offset = max(min(offset, len(lines)-bodyHeight), 0)
	return m
}

func (m Model) formLines() (lines []string, first, last int) {
	atTop := m.focus == firstFocusable(m.draft.fields())
	for _, p := range m.draft.panels() {
		var inner []string
		border := screen.PlainBorder
		focusStart, focusEnd := -1, -1
		for _, f := range p.fields {
			state := m.stateOf(f)
			fieldLines, start, end := f.lines(state, m.width-4)
			if state.focused {
				border = screen.FocusedBorder
				atTop = atTop && start == 0
				focusStart, focusEnd = len(inner)+start, len(inner)+end
			}
			inner = append(inner, fieldLines...)
		}
		top := len(lines)
		pane := screen.Pane{Title: p.title, Count: p.count, Lines: inner, Width: m.width, Height: len(inner) + 2, Border: border}
		lines = append(lines, strings.Split(pane.View(), "\n")...)
		if focusStart < 0 {
			continue
		}
		first, last = top+1+focusStart, top+1+focusEnd
		if focusStart == 0 {
			first = top
		}
		if focusEnd == len(inner)-1 {
			last = top + len(inner) + 1
		}
	}
	state := m.stateOf(writeField)
	buttonLines, _, _ := writeField.lines(state, m.width)
	if state.focused {
		first, last = len(lines), len(lines)+len(buttonLines)-1
	}
	lines = append(lines, buttonLines...)
	if atTop {
		first = 0
	}
	return lines, first, last
}

func (m Model) stateOf(f field) fieldState {
	state := fieldState{focused: f.row == m.focus, text: f.value}
	if f.control == textControl {
		state.text = m.textOf(f.row)
	}
	if state.focused {
		state.cursor, state.input = m.cursor, m.input.View()
	}
	if refused, isRefused := m.refused[f.row]; isRefused {
		state.problems = append(state.problems, refused.problem)
	}
	if state.focused && m.message != "" {
		state.problems = append(state.problems, m.message)
	}
	return state
}

func (m Model) reviewText() string {
	width := max(m.width-4, 1)
	if !m.canWrite {
		return screen.Message(m.preview, width)
	}
	return screen.HeadingStyle.Render("Write the repo?") + "\n\n" + strings.Join(screen.Wrapped(m.preview, width), "\n")
}

func (m Model) footer() string {
	m.help.Width = m.width - 1
	return " " + m.help.ShortHelpView(m.footerKeys())
}

func (m Model) footerKeys() []key.Binding {
	if m.reviewing {
		if m.canWrite {
			return []key.Binding{keys.write, keys.back, keys.scroll, keys.quit}
		}
		return []key.Binding{keys.back, keys.scroll, keys.quit}
	}
	switch m.focusedField().control {
	case textControl:
		return []key.Binding{keys.edit, keys.move, keys.enter, keys.quit}
	case selectControl:
		return []key.Binding{keys.listMove, keys.choose, keys.next, keys.quit}
	case checklistControl:
		return []key.Binding{keys.listMove, keys.selectFeature, keys.next, keys.quit}
	}
	return []key.Binding{keys.write, keys.move, keys.quit}
}
