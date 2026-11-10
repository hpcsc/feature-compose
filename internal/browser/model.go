package browser

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/screen"
)

const narrowestTree = 20

var keys = struct {
	down, up, open, close, filter, keepFilter, clearFilter, scrollDown, scrollUp, quit key.Binding
}{
	down:        key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/k", "move")),
	up:          key.NewBinding(key.WithKeys("k", "up")),
	open:        key.NewBinding(key.WithKeys("l", "enter", "right"), key.WithHelp("l", "open")),
	close:       key.NewBinding(key.WithKeys("h", "left", "esc"), key.WithHelp("h", "close")),
	filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	keepFilter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "keep filter")),
	clearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
	scrollDown:  key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d/u", "scroll details")),
	scrollUp:    key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
	quit:        key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
}

var _ tea.Model = Model{}

type Model struct {
	definitions catalog.Definitions
	tree        tree
	treeOffset  int
	details     viewport.Model
	filter      textinput.Model
	typing      bool
	help        help.Model
	width       int
	height      int
}

func New(definitions catalog.Definitions) Model {
	filter := textinput.New()
	filter.Prompt = "/"
	return Model{definitions: definitions, tree: newTree(definitions), details: viewport.New(0, 0), filter: filter, help: help.New()}
}

func NewAtStarter(definitions catalog.Definitions, starterName string) (Model, error) {
	if _, err := definitions.ForStarter(starterName); err != nil {
		return Model{}, err
	}
	m := New(definitions)
	m.tree.selected = row{starter: starterName}
	return m, nil
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
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.typing {
			return m.updateFilter(msg)
		}
		return m.updateTree(msg)
	}
	if !m.typing {
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	return m, cmd
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
	case m.tree.filter != "" && key.Matches(msg, keys.clearFilter):
		m.filter.Reset()
		m.tree = m.tree.filtered("")
	case key.Matches(msg, keys.close):
		m.tree = m.tree.closed()
	case key.Matches(msg, keys.scrollDown):
		m.details.HalfPageDown()
	case key.Matches(msg, keys.scrollUp):
		m.details.HalfPageUp()
	case key.Matches(msg, keys.filter):
		m.typing = true
		cmd := m.filter.Focus()
		return m.laidOut(before), cmd
	}
	return m.laidOut(before), nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	before := m.tree.selected
	switch {
	case key.Matches(msg, keys.keepFilter):
		m.typing = false
		m.filter.Blur()
		return m.laidOut(before), nil
	case key.Matches(msg, keys.clearFilter):
		m.typing = false
		m.filter.Blur()
		m.filter.Reset()
		m.tree = m.tree.filtered("")
		return m.laidOut(before), nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.tree = m.tree.filtered(m.filter.Value())
	return m.laidOut(before), cmd
}

func (m Model) laidOut(before row) Model {
	m.details.Width = max(m.width-m.treeWidth()-4, 0)
	m.details.Height = max(m.height-4, 0)
	m.details.SetContent(m.detailsText())
	if m.tree.selected != before {
		m.details.GotoTop()
	}
	treeHeight := max(m.height-4, 1)
	m.treeOffset = max(min(screen.OffsetShowing(m.treeOffset, m.tree.selectedIndex(), treeHeight), len(m.tree.rows())-treeHeight), 0)
	return m
}

func (m Model) treeWidth() int {
	return min(max(m.tree.widestRow()+5, narrowestTree), m.width*2/5)
}

func (m Model) detailsText() string {
	if m.tree.selectedIndex() < 0 {
		return ""
	}
	return detailsOf(m.definitions, m.tree.selected, m.details.Width)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if !screen.Fits(m.width, m.height) {
		return screen.TooSmall("show", m.width)
	}
	paneHeight := m.height - 2
	treePane := screen.Pane{Title: "Catalog", Width: m.treeWidth(), Height: paneHeight, Border: screen.FocusedBorder}
	if m.tree.filter != "" {
		treePane.Title += " /" + m.tree.filter
	}
	treePane.Lines = m.tree.lines(m.treeOffset, treePane.InnerWidth(), paneHeight-2)
	switch {
	case len(m.tree.starterNames) == 0:
		treePane.Lines = screen.Wrapped(screen.NoneStyle.Render("The catalog has no starter."), treePane.InnerWidth())
	case len(treePane.Lines) == 0:
		treePane.Lines = screen.Wrapped(screen.NoneStyle.Render("No name matches the filter."), treePane.InnerWidth())
	}
	detailsPane := screen.Pane{Title: m.detailsTitle(), Lines: strings.Split(m.details.View(), "\n"), Width: m.width - treePane.Width, Height: paneHeight, Border: screen.PlainBorder}
	if m.details.TotalLineCount() > m.details.Height {
		detailsPane.Note = fmt.Sprintf("%d%%", int(m.details.ScrollPercent()*100))
	}
	return screen.Header("feature-compose show", screen.Counted(len(m.definitions.Starters), "starter")+" · "+screen.Counted(len(m.definitions.Features), "feature"), m.width) + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, treePane.View(), detailsPane.View()) + "\n" + m.footer()
}

func (m Model) detailsTitle() string {
	selected := m.tree.selected
	switch {
	case m.tree.selectedIndex() < 0:
		return "Details"
	case selected.isStarter():
		return selected.starter
	case selected.isGroup():
		return selected.starter + " › " + string(selected.group)
	}
	return selected.name
}

func (m Model) footer() string {
	m.help.Width = m.width - 1
	if m.typing {
		return " " + m.filter.View() + "  " + m.help.ShortHelpView([]key.Binding{keys.keepFilter, keys.clearFilter})
	}
	bindings := []key.Binding{keys.down, keys.open, keys.close, keys.filter, keys.scrollDown, keys.quit}
	if m.tree.filter != "" {
		bindings = append([]key.Binding{keys.clearFilter}, bindings...)
	}
	return " " + m.help.ShortHelpView(bindings)
}
