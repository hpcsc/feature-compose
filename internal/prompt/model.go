package prompt

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
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
	scrollDown, scrollUp, move, moveButtons, toggle, next, back, choose, confirm, cancel key.Binding
}{
	scrollDown:  key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d/u", "scroll")),
	scrollUp:    key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
	move:        key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "move")),
	moveButtons: key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "move")),
	toggle:      key.NewBinding(key.WithKeys("x", " "), key.WithHelp("x", "select")),
	next:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "next")),
	back:        key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
	choose:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "choose")),
	confirm:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	cancel:      key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	messageStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D2003F", Dark: "#FF5F87"})
)

type step struct {
	form         *huh.Form
	keys         []key.Binding
	showsPreview bool
	check        func() error
	done         func(Model) (tea.Model, tea.Cmd)
}

var _ tea.Model = Model{}

type Model struct {
	catalog        catalog.Catalog
	definitions    catalog.Definitions
	starter        catalog.Starter
	features       map[string]catalog.FeatureDefinition
	name           string
	given          map[string]string
	givenByFeature map[string]map[string]string
	answers        catalog.Answers
	selection      catalog.Selection
	step           step
	preview        viewport.Model
	help           help.Model
	width          int
	height         int
	confirmed      bool
	err            error
}

func New(c catalog.Catalog, starterName, name string, given map[string]string, givenByFeature map[string]map[string]string) (Model, error) {
	definitions, err := c.Definitions()
	if err != nil {
		return Model{}, err
	}
	m := Model{catalog: c, definitions: definitions, name: name, given: given, givenByFeature: givenByFeature, help: help.New()}
	if starterName == "" {
		return m.atStarters(), nil
	}
	m, err = m.withStarter(starterName)
	if err != nil {
		return Model{}, err
	}
	if name == "" {
		return m.atName(), nil
	}
	return m.afterName()
}

func (m Model) Init() tea.Cmd {
	return m.step.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.cancel):
			return m, tea.Quit
		// huh's Select has no switch to turn its filter off, and its filter takes esc, which cancels the TUI.
		case msg.String() == "/" && m.atList():
			return m, nil
		case m.step.showsPreview && key.Matches(msg, keys.scrollDown):
			m.preview.HalfPageDown()
			return m, nil
		case m.step.showsPreview && key.Matches(msg, keys.scrollUp):
			m.preview.HalfPageUp()
			return m, nil
		}
	}
	form, cmd := m.step.form.Update(msg)
	m.step.form = form.(*huh.Form)
	if m.step.form.State == huh.StateCompleted {
		return m.step.done(m)
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
	footer := percent + " " + m.help.ShortHelpView(m.step.keys)
	if m.step.check != nil {
		if err := m.step.check(); err != nil {
			footer = messageStyle.Width(m.width).Render(err.Error()) + "\n" + footer
		}
	}
	if !m.step.showsPreview {
		return strings.Join([]string{titleStyle.Render("feature-compose new"), "", m.step.form.View(), footer}, "\n")
	}
	return strings.Join([]string{titleStyle.Render("feature-compose new"), m.preview.View(), "", m.step.form.View(), footer}, "\n")
}

func (m Model) Confirmed() (catalog.Selection, error) {
	if !m.confirmed {
		return catalog.Selection{}, cmp.Or(m.err, errCanceled)
	}
	return m.selection, nil
}

func (m Model) atList() bool {
	_, isList := m.step.form.GetFocusedField().(interface{ GetFiltering() bool })
	return isList
}

func (m Model) atStarters() Model {
	starterName := ""
	options := make([]huh.Option[string], 0, len(m.definitions.Starters))
	for _, name := range slices.Sorted(maps.Keys(m.definitions.Starters)) {
		options = append(options, huh.NewOption(name+" — "+m.definitions.Starters[name].Summary, name))
	}
	m.step = step{
		form: huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().Title("Select the starter").Value(&starterName).Options(options...),
		)).WithShowHelp(false),
		keys: []key.Binding{keys.move, keys.choose, keys.cancel},
		done: func(m Model) (tea.Model, tea.Cmd) {
			next, err := m.withStarter(starterName)
			if err != nil {
				return m.ended(err)
			}
			return next.atName().started()
		},
	}
	return m
}

func (m Model) withStarter(starterName string) (Model, error) {
	forStarter, err := m.definitions.ForStarter(starterName)
	if err != nil {
		return Model{}, err
	}
	starter, err := m.catalog.Starter(starterName)
	if err != nil {
		return Model{}, err
	}
	m.starter, m.features = starter, forStarter.Features
	return m, nil
}

func (m Model) atName() Model {
	name := ""
	m.step = step{
		form: huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Enter the name of the repo").Value(&name).Validate(catalog.CheckRepoName),
		)).WithShowHelp(false).WithShowErrors(false),
		keys:  []key.Binding{keys.next, keys.cancel},
		check: func() error { return catalog.CheckRepoName(name) },
		done: func(m Model) (tea.Model, tea.Cmd) {
			m.name = name
			next, err := m.afterName()
			if err != nil {
				return m.ended(err)
			}
			return next.started()
		},
	}
	return m
}

func (m Model) afterName() (Model, error) {
	selection, err := m.starter.Select(m.name, m.given, m.givenByFeature, catalog.Answers{})
	if err != nil {
		return Model{}, err
	}
	m.selection = selection
	switch {
	case len(m.givenByFeature) == 0 && len(m.features) > 0:
		return m.atFeatures(), nil
	case len(selection.Questions()) > 0:
		return m.atOptions(), nil
	}
	return m.atConfirm()
}

func (m Model) atFeatures() Model {
	answered := slices.Clone(m.answers.Features)
	options := make([]huh.Option[string], 0, len(m.features))
	for _, featureName := range slices.Sorted(maps.Keys(m.features)) {
		options = append(options, huh.NewOption(featureName+" — "+m.features[featureName].Summary, featureName))
	}
	m.step = step{
		form: huh.NewForm(huh.NewGroup(
			huh.NewMultiSelect[string]().Title("Select the features").Value(&answered).Options(options...),
		)).WithShowHelp(false),
		keys: []key.Binding{keys.move, keys.toggle, keys.next, keys.cancel},
		done: func(m Model) (tea.Model, tea.Cmd) { return m.withFeatures(answered) },
	}
	return m
}

func (m Model) withFeatures(answered []string) (tea.Model, tea.Cmd) {
	m.answers.Features = slices.Clone(answered)
	selection, err := m.starter.Select(m.name, m.given, m.givenByFeature, m.answers)
	if err != nil {
		return m.ended(err)
	}
	m.selection = selection
	if len(selection.AddedByNeeds()) > 0 {
		return m.atNeeds().started()
	}
	return m.atOptionsOrConfirm()
}

func (m Model) atNeeds() Model {
	needs := m.selection.AddedByNeeds()
	lines := make([]string, 0, len(needs))
	for _, need := range needs {
		lines = append(lines, need.String())
	}
	goOn := true
	m.step = step{
		form: huh.NewForm(huh.NewGroup(
			// huh scrolls the list to the option of the value when Options runs, so Value comes first.
			huh.NewSelect[bool]().
				Title("The features that you selected need these features").
				Description(strings.Join(lines, "\n")).
				Value(&goOn).
				Options(huh.NewOption("Continue", true), huh.NewOption("Change the features", false)),
		)).WithShowHelp(false),
		keys: []key.Binding{keys.move, keys.choose, keys.cancel},
		done: func(m Model) (tea.Model, tea.Cmd) {
			if goOn {
				return m.atOptionsOrConfirm()
			}
			return m.atFeatures().started()
		},
	}
	return m
}

func (m Model) atOptionsOrConfirm() (tea.Model, tea.Cmd) {
	if len(m.selection.Questions()) > 0 {
		return m.atOptions().started()
	}
	return m.atConfirmOrEnded()
}

func (m Model) atOptions() Model {
	questions := m.selection.Questions()
	values := make([]string, len(questions))
	for i, q := range questions {
		values[i] = q.Default
	}
	return m.atOptionsOf(questions, values, 0)
}

func (m Model) atOptionsOf(questions []catalog.Question, values []string, start int) Model {
	end := start
	for end < len(questions) && questions[end].Declarer() == questions[start].Declarer() {
		end++
	}
	var fields []huh.Field
	for i := start; i < end; i++ {
		if len(questions[i].Accepted) > 0 {
			// huh scrolls the list to the option of the value when Options runs, so Value comes first.
			fields = append(fields, huh.NewSelect[string]().Title(questions[i].Option).Value(&values[i]).Options(huh.NewOptions(m.defaultFirst(questions[i])...)...).Validate(questions[i].Check))
		} else {
			fields = append(fields, huh.NewInput().Title(questions[i].Option).Value(&values[i]).Validate(questions[i].Check))
		}
	}
	// huh sizes every group of a form when its first group starts, so each declarer gets a form of its own.
	form := huh.NewForm(huh.NewGroup(fields...).Title("The options of " + questions[start].Declarer())).WithShowHelp(false).WithShowErrors(false)
	m.step = step{
		form: form,
		keys: []key.Binding{keys.move, keys.next, keys.back, keys.cancel},
		check: func() error {
			i := start + slices.Index(fields, form.GetFocusedField())
			return questions[i].Check(values[i])
		},
		done: func(m Model) (tea.Model, tea.Cmd) {
			if end < len(questions) {
				return m.atOptionsOf(questions, values, end).started()
			}
			return m.withOptions(questions, values)
		},
	}
	return m
}

// huh scrolls a list to its chosen option, which hides each value above it.
func (m Model) defaultFirst(q catalog.Question) []string {
	if !slices.Contains(q.Accepted, q.Default) {
		return q.Accepted
	}
	return append([]string{q.Default}, slices.DeleteFunc(slices.Clone(q.Accepted), func(value string) bool { return value == q.Default })...)
}

func (m Model) withOptions(questions []catalog.Question, values []string) (tea.Model, tea.Cmd) {
	m.answers = m.answers.WithValues(questions, values)
	selection, err := m.starter.Select(m.name, m.given, m.givenByFeature, m.answers)
	if err != nil {
		return m.ended(err)
	}
	m.selection = selection
	return m.atConfirmOrEnded()
}

func (m Model) atConfirmOrEnded() (tea.Model, tea.Cmd) {
	next, err := m.atConfirm()
	if err != nil {
		return m.ended(err)
	}
	return next.started()
}

func (m Model) atConfirm() (Model, error) {
	preview, err := m.selection.Preview()
	if err != nil {
		return Model{}, err
	}
	m.preview.SetContent(strings.TrimSuffix(preview, "\n"))
	write := true
	m.step = step{
		form: huh.NewForm(huh.NewGroup(
			huh.NewConfirm().Title("Write the repo?").Affirmative("Yes").Negative("No").Value(&write),
		)).WithShowHelp(false),
		keys:         []key.Binding{keys.scrollDown, keys.moveButtons, keys.confirm, keys.cancel},
		showsPreview: true,
		done: func(m Model) (tea.Model, tea.Cmd) {
			m.confirmed = write
			return m, tea.Quit
		},
	}
	return m, nil
}

func (m Model) started() (tea.Model, tea.Cmd) {
	cmd := m.step.form.Init()
	if m.width > 0 {
		m.step.form.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	}
	return m.laidOut(), cmd
}

func (m Model) ended(err error) (tea.Model, tea.Cmd) {
	m.err = err
	return m, tea.Quit
}

func (m Model) laidOut() Model {
	m.preview.Width = m.width
	m.preview.Height = max(m.height-lipgloss.Height(m.step.form.View())-3, 1)
	return m
}
