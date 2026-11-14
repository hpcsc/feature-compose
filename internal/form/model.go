package form

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/screen"
)

var errCanceled = errors.New("you canceled new, so it wrote no file")

var keys = struct {
	down, up, open, enter, close, toggle, write, scrollDown, scrollUp, quit, keep, drop, quitWhileTyping, confirm, back key.Binding
}{
	down:            key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/k", "move")),
	up:              key.NewBinding(key.WithKeys("k", "up")),
	open:            key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "open")),
	enter:           key.NewBinding(key.WithKeys("enter")),
	close:           key.NewBinding(key.WithKeys("h", "left", "esc"), key.WithHelp("h", "close")),
	toggle:          key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
	write:           key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "write")),
	scrollDown:      key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d/u", "scroll details")),
	scrollUp:        key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
	quit:            key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	keep:            key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "keep")),
	drop:            key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "drop")),
	quitWhileTyping: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	confirm:         key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "write")),
	back:            key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("esc", "back")),
}

var _ tea.Model = Model{}

type Model struct {
	draft        draft
	tree         tree
	treeOffset   int
	details      viewport.Model
	detailsTitle string
	message      string
	editing      bool
	edit         edit
	previewing   bool
	preview      string
	canWrite     bool
	help         help.Model
	width        int
	height       int
	confirmed    bool
}

func New(c catalog.Catalog, starterName, name string, given map[string]string, givenByFeature map[string]map[string]string) (Model, error) {
	d, err := newDraft(c, starterName, name, given, givenByFeature)
	if err != nil {
		return Model{}, err
	}
	return Model{draft: d, help: help.New()}.laidOut(row{kind: nameKind}), nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.laidOut(m.tree.selected), nil
	case tea.KeyMsg:
		if m.previewing {
			return m.updatePreview(msg)
		}
		if m.editing {
			return m.updateEdit(msg)
		}
		return m.updateTree(msg)
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if !screen.Fits(m.width, m.height) {
		return screen.TooSmall("new", m.width)
	}
	paneHeight := m.height - 2
	treePane := screen.Pane{Title: "Repo", Width: m.treeWidth(), Height: paneHeight, Border: screen.FocusedBorder}
	treePane.Lines = m.tree.lines(m.treeOffset, treePane.InnerWidth(), paneHeight-2)
	detailsPane := screen.Pane{Title: m.detailsTitle, Lines: strings.Split(m.details.View(), "\n"), Width: m.width - treePane.Width, Height: paneHeight, Border: screen.PlainBorder}
	if m.previewing {
		treePane.Border, detailsPane.Border = screen.PlainBorder, screen.FocusedBorder
	}
	if m.details.TotalLineCount() > m.details.Height {
		detailsPane.Note = fmt.Sprintf("%d%%", int(m.details.ScrollPercent()*100))
	}
	header := screen.Header("feature-compose new", screen.Counted(m.draft.featureCount(), "feature"), m.width)
	return header + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, treePane.View(), detailsPane.View()) + "\n" + m.footer()
}

func (m Model) Confirmed() (catalog.Selection, error) {
	if !m.confirmed {
		return catalog.Selection{}, errCanceled
	}
	return m.draft.selection, nil
}

func (m Model) updateTree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	before := m.tree.selected
	m.message = ""
	switch {
	case key.Matches(msg, keys.quit):
		return m, tea.Quit
	case key.Matches(msg, keys.down):
		m.tree = m.tree.moved(1)
	case key.Matches(msg, keys.up):
		m.tree = m.tree.moved(-1)
	case key.Matches(msg, keys.open):
		m.tree = m.tree.opened()
	case key.Matches(msg, keys.enter):
		m = m.entered()
	case key.Matches(msg, keys.close):
		m.tree = m.tree.closed()
	case key.Matches(msg, keys.toggle) && m.tree.selected.kind == featureKind:
		m.draft, m.message = m.draft.toggled(m.tree.selected.feature)
	case key.Matches(msg, keys.write):
		m = m.previewed()
	case key.Matches(msg, keys.scrollDown):
		m.details.HalfPageDown()
	case key.Matches(msg, keys.scrollUp):
		m.details.HalfPageUp()
	}
	return m.laidOut(before), nil
}

func (m Model) previewed() Model {
	if r, message, isMissing := m.draft.missingAnswer(); isMissing {
		m.tree = m.tree.revealed(r)
		m.message = message
		return m
	}
	preview, err := m.draft.selection.Preview()
	m.previewing, m.preview, m.canWrite = true, strings.TrimSuffix(preview, "\n"), err == nil
	if err != nil {
		m.preview = err.Error()
	}
	m.details.GotoTop()
	return m
}

func (m Model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.quit):
		return m, tea.Quit
	case m.canWrite && key.Matches(msg, keys.confirm):
		m.confirmed = true
		return m, tea.Quit
	case key.Matches(msg, keys.back):
		m.previewing = false
		m.details.GotoTop()
	case key.Matches(msg, keys.scrollDown):
		m.details.HalfPageDown()
	case key.Matches(msg, keys.scrollUp):
		m.details.HalfPageUp()
	}
	return m.laidOut(m.tree.selected), nil
}

func (m Model) entered() Model {
	r := m.tree.selected
	if r.kind == groupKind || r.kind == featureKind {
		m.tree = m.tree.opened()
		return m
	}
	e, message := m.draft.editOf(r)
	if message != "" {
		m.message = message
		return m
	}
	m.editing, m.edit = true, e
	return m
}

func (m Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	before := m.tree.selected
	m.message = ""
	var cmd tea.Cmd
	switch {
	case key.Matches(msg, keys.quitWhileTyping):
		return m, tea.Quit
	case key.Matches(msg, keys.keep):
		m = m.kept()
	case key.Matches(msg, keys.drop):
		m.editing = false
	case m.edit.isList() && key.Matches(msg, keys.down):
		m.edit = m.edit.moved(1)
	case m.edit.isList() && key.Matches(msg, keys.up):
		m.edit = m.edit.moved(-1)
	case m.edit.isList() && key.Matches(msg, keys.quit):
		return m, tea.Quit
	case !m.edit.isList():
		m.edit, cmd = m.edit.typed(msg)
	}
	return m.laidOut(before), cmd
}

func (m Model) kept() Model {
	if m.draft.check(m.edit.row, m.edit.value()) != nil {
		return m
	}
	d, err := m.draft.withAnswer(m.edit.row, m.edit.value())
	if err != nil {
		m.message = err.Error()
		return m
	}
	m.draft, m.editing = d, false
	return m
}

func (m Model) laidOut(before row) Model {
	m.tree.nodes = m.draft.nodes()
	m.details.Width = max(m.width-m.treeWidth()-4, 0)
	m.details.Height = max(m.height-4, 0)
	var text string
	m.detailsTitle, text = m.draft.details(m.tree.selected, m.details.Width)
	if m.editing {
		text = m.edit.text(m.draft.check(m.edit.row, m.edit.value()), m.details.Width)
	}
	if m.previewing {
		m.detailsTitle, text = "Preview", strings.Join(screen.Wrapped(m.preview, m.details.Width), "\n")
		if m.canWrite {
			text = "Write the repo?\n\n" + text
		}
	}
	linesAbove := 0
	if m.message != "" {
		message := screen.Message(m.message, m.details.Width)
		text, linesAbove = message+"\n\n"+text, lipgloss.Height(message)+1
	}
	m.details.SetContent(text)
	if m.tree.selected != before {
		m.details.GotoTop()
	}
	if m.editing {
		m.details.SetYOffset(screen.OffsetShowing(m.details.YOffset, linesAbove+m.edit.chosen, m.details.Height))
	}
	treeHeight := max(m.height-4, 1)
	m.treeOffset = max(min(screen.OffsetShowing(m.treeOffset, m.tree.selectedIndex(), treeHeight), len(m.tree.rows())-treeHeight), 0)
	return m
}

func (m Model) treeWidth() int {
	return min(m.tree.widestRow()+5, m.width*2/5)
}

func (m Model) footer() string {
	m.help.Width = m.width - 1
	switch {
	case m.previewing && m.canWrite:
		return " " + m.help.ShortHelpView([]key.Binding{keys.confirm, keys.back, keys.scrollDown, keys.quit})
	case m.previewing:
		return " " + m.help.ShortHelpView([]key.Binding{keys.back, keys.scrollDown, keys.quit})
	case m.editing && m.edit.isList():
		return " " + m.help.ShortHelpView([]key.Binding{keys.down, keys.keep, keys.drop, keys.quit})
	case m.editing:
		return " " + m.help.ShortHelpView([]key.Binding{keys.keep, keys.drop, keys.quitWhileTyping})
	}
	return " " + m.help.ShortHelpView([]key.Binding{keys.down, keys.open, keys.close, keys.toggle, keys.write, keys.scrollDown, keys.quit})
}
