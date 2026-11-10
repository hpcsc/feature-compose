package prompt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/catalog"
)

var errCanceled = errors.New("you canceled new, so it wrote no file")

var keys = struct {
	scrollDown, scrollUp, choose, confirm, cancel key.Binding
}{
	scrollDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d/u", "scroll")),
	scrollUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
	choose:     key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "choose")),
	confirm:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	cancel:     key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
}

var titleStyle = lipgloss.NewStyle().Bold(true)

var _ tea.Model = Model{}

type Model struct {
	selection catalog.Selection
	preview   viewport.Model
	form      *huh.Form
	write     *bool
	help      help.Model
	width     int
	height    int
	confirmed bool
}

func New(c catalog.Catalog, starterName, name string, given map[string]string, givenByFeature map[string]map[string]string) (Model, error) {
	starter, err := c.Starter(starterName)
	if err != nil {
		return Model{}, err
	}
	selection, err := starter.Select(name, given, givenByFeature, catalog.Answers{})
	if err != nil {
		return Model{}, err
	}
	return Model{selection: selection, help: help.New()}.atConfirm()
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.cancel):
			return m, tea.Quit
		case key.Matches(msg, keys.scrollDown):
			m.preview.HalfPageDown()
			return m, nil
		case key.Matches(msg, keys.scrollUp):
			m.preview.HalfPageUp()
			return m, nil
		}
	}
	form, cmd := m.form.Update(msg)
	m.form = form.(*huh.Form)
	if m.form.State == huh.StateCompleted {
		m.confirmed = *m.write
		return m, tea.Quit
	}
	return m.laidOut(), cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	percent := ""
	if m.preview.TotalLineCount() > m.preview.Height {
		percent = fmt.Sprintf(" %d%% ·", int(m.preview.ScrollPercent()*100))
	}
	m.help.Width = m.width - lipgloss.Width(percent) - 1
	footer := percent + " " + m.help.ShortHelpView([]key.Binding{keys.scrollDown, keys.choose, keys.confirm, keys.cancel})
	return strings.Join([]string{titleStyle.Render("feature-compose new"), m.preview.View(), "", m.form.View(), footer}, "\n")
}

func (m Model) Confirmed() (catalog.Selection, error) {
	if !m.confirmed {
		return catalog.Selection{}, errCanceled
	}
	return m.selection, nil
}

func (m Model) atConfirm() (Model, error) {
	preview, err := m.selection.Preview()
	if err != nil {
		return Model{}, err
	}
	m.preview.SetContent(strings.TrimSuffix(preview, "\n"))
	write := true
	m.write = &write
	m.form = huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title("Write the repo?").Affirmative("Yes").Negative("No").Value(m.write),
	)).WithShowHelp(false)
	return m, nil
}

func (m Model) laidOut() Model {
	m.preview.Width = m.width
	m.preview.Height = max(m.height-lipgloss.Height(m.form.View())-3, 1)
	return m
}
