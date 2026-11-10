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
	down, up, open, close, scrollDown, scrollUp, quit key.Binding
}{
	down:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/k", "move")),
	up:         key.NewBinding(key.WithKeys("k", "up")),
	open:       key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "open")),
	close:      key.NewBinding(key.WithKeys("h", "left", "esc"), key.WithHelp("h", "close")),
	scrollDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d/u", "scroll details")),
	scrollUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
	quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

var _ tea.Model = Model{}

type Model struct {
	draft        draft
	tree         tree
	treeOffset   int
	details      viewport.Model
	detailsTitle string
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
	switch {
	case key.Matches(msg, keys.quit):
		return m, tea.Quit
	case key.Matches(msg, keys.down):
		m.tree = m.tree.moved(1)
	case key.Matches(msg, keys.up):
		m.tree = m.tree.moved(-1)
	case key.Matches(msg, keys.open):
		m.tree = m.tree.opened()
	case key.Matches(msg, keys.close):
		m.tree = m.tree.closed()
	case key.Matches(msg, keys.scrollDown):
		m.details.HalfPageDown()
	case key.Matches(msg, keys.scrollUp):
		m.details.HalfPageUp()
	}
	return m.laidOut(before), nil
}

func (m Model) laidOut(before row) Model {
	m.tree.rows = m.draft.rows(m.tree)
	m.details.Width = max(m.width-m.treeWidth()-4, 0)
	m.details.Height = max(m.height-4, 0)
	var text string
	m.detailsTitle, text = m.draft.details(m.tree.selected, m.details.Width)
	m.details.SetContent(text)
	if m.tree.selected != before {
		m.details.GotoTop()
	}
	treeHeight := max(m.height-4, 1)
	m.treeOffset = max(min(screen.OffsetShowing(m.treeOffset, m.tree.selectedIndex(), treeHeight), len(m.tree.rows)-treeHeight), 0)
	return m
}

func (m Model) treeWidth() int {
	return min(m.tree.widestRow()+5, m.width*2/5)
}

func (m Model) footer() string {
	m.help.Width = m.width - 1
	return " " + m.help.ShortHelpView([]key.Binding{keys.down, keys.open, keys.close, keys.scrollDown, keys.quit})
}
