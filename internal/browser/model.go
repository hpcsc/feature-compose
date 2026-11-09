package browser

import (
	"maps"
	"slices"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/catalog"
)

var _ tea.Model = Model{}

type Model struct {
	definitions catalog.Definitions
	starters    list.Model
	starterView *starterView
	featureView *featureView
}

func New(definitions catalog.Definitions) Model {
	names := slices.Sorted(maps.Keys(definitions.Starters))
	rows := make([]list.Item, 0, len(names))
	for _, name := range names {
		rows = append(rows, starterRow{name: name, summary: definitions.Starters[name].Summary})
	}
	starters := list.New(rows, list.NewDefaultDelegate(), 0, 0)
	starters.Title = "Starters"
	starters.SetStatusBarItemName("starter", "starters")
	starters.KeyMap.PrevPage = key.NewBinding(key.WithKeys("left", "pgup", "b", "u"), key.WithHelp("←/pgup", "prev page"))
	starters.KeyMap.NextPage = key.NewBinding(key.WithKeys("right", "pgdown", "f", "d"), key.WithHelp("→/pgdn", "next page"))
	starters.KeyMap.Quit = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	return Model{definitions: definitions, starters: starters}
}

func NewAtStarter(definitions catalog.Definitions, starterName string) (Model, error) {
	if _, err := definitions.ForStarter(starterName); err != nil {
		return Model{}, err
	}
	m := New(definitions)
	m.starters.Select(slices.IndexFunc(m.starters.Items(), func(row list.Item) bool { return row.(starterRow).name == starterName }))
	return m.withStarterView(starterName), nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.starters.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.featureView != nil {
			return m.updateFeatureView(msg)
		}
		if m.starterView != nil {
			return m.updateStarterView(msg)
		}
		if m.starters.FilterState() != list.Filtering && (msg.String() == "l" || msg.String() == "enter") {
			if selected, ok := m.starters.SelectedItem().(starterRow); ok {
				return m.withStarterView(selected.name), nil
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.starters, cmd = m.starters.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.featureView != nil {
		return m.featureView.view()
	}
	if m.starterView != nil {
		return m.starterView.view()
	}
	return m.starters.View()
}

func (m Model) updateStarterView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "h", "esc":
		m.starterView = nil
	case "j", "down":
		m.starterView = m.starterView.moved(1)
	case "k", "up":
		m.starterView = m.starterView.moved(-1)
	case "l", "enter":
		m.featureView = m.starterView.selectedFeature()
	}
	return m, nil
}

func (m Model) updateFeatureView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "h", "esc":
		m.featureView = nil
	}
	return m, nil
}

func (m Model) withStarterView(starterName string) Model {
	m.starterView = newStarterView(m.definitions, starterName)
	return m
}

var _ list.DefaultItem = starterRow{}

type starterRow struct {
	name    string
	summary string
}

func (r starterRow) Title() string {
	return r.name
}

func (r starterRow) Description() string {
	return r.summary
}

func (r starterRow) FilterValue() string {
	return r.name
}
