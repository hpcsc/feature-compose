package catalogscreen

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/hpcsc/feature-compose/internal/index"
	"github.com/hpcsc/feature-compose/internal/registry"
	"github.com/hpcsc/feature-compose/internal/screen"
)

const narrowestList = 20

var keys = struct {
	down, up, sync, syncAll, ref, remove, add, choose, cancel, yes, quit, stop key.Binding
}{
	down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/k", "move")),
	up:      key.NewBinding(key.WithKeys("k", "up")),
	sync:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sync")),
	syncAll: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync all")),
	ref:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "change the ref")),
	remove:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "remove")),
	add:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "add")),
	choose:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "choose")),
	cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	yes:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "remove")),
	quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	stop:    key.NewBinding(key.WithKeys("ctrl+c")),
}

type knownMsg struct {
	file index.File
	err  error
}

type doneMsg struct {
	origins []catalog.Origin
	listed  bool
	message string
	err     error
}

type refsMsg struct {
	refs []string
	err  error
}

type knownCatalog struct {
	name  string
	entry index.Entry
}

type selected struct {
	origin *catalog.Origin
	known  *knownCatalog
}

var _ tea.Model = Model{}

type Model struct {
	ctx        context.Context
	registry   *registry.Instance
	fetch      func(context.Context) (index.File, error)
	origins    []catalog.Origin
	index      index.File
	indexErr   error
	indexReady bool
	cursor     int
	refs       []string
	refCursor  int
	confirming bool
	running    string
	message    string
	failed     bool
	help       help.Model
	width      int
	height     int
}

func New(ctx context.Context, r *registry.Instance, fetch func(context.Context) (index.File, error)) (Model, error) {
	origins, err := r.List(ctx)
	if err != nil {
		return Model{}, err
	}
	return Model{ctx: ctx, registry: r, fetch: fetch, origins: origins, help: help.New()}, nil
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		file, err := m.fetch(m.ctx)
		return knownMsg{file: file, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case knownMsg:
		m.index, m.indexErr, m.indexReady = msg.file, msg.err, true
	case doneMsg:
		m.running, m.message, m.failed = "", msg.message, msg.err != nil
		if msg.err != nil {
			m.message = msg.err.Error()
		}
		if msg.listed {
			m.origins = msg.origins
		}
		m.cursor = min(m.cursor, max(len(m.known())+len(m.origins)-1, 0))
	case refsMsg:
		m.running = ""
		if msg.err != nil {
			m.message, m.failed = msg.err.Error(), true
			return m, nil
		}
		m.refs = append([]string{""}, msg.refs...)
		m.refCursor = max(slices.Index(m.refs, m.selected().origin.Ref), 0)
	case tea.KeyMsg:
		return m.pressed(msg)
	}
	return m, nil
}

func (m Model) pressed(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keys.stop) {
		return m, tea.Quit
	}
	if m.running != "" {
		return m, nil
	}
	if m.confirming {
		m.confirming = false
		name := m.selected().origin.Name
		if !key.Matches(msg, keys.yes) {
			m.message, m.failed = "Kept the catalog "+name+".", false
			return m, nil
		}
		return m.started("Remove the catalog "+name, func() (string, error) {
			return "Removed the catalog " + name + ".", m.registry.Remove(name)
		})
	}
	if m.refs != nil {
		return m.pressedInRefs(msg)
	}
	chosen := m.selected()
	switch {
	case key.Matches(msg, keys.quit):
		return m, tea.Quit
	case key.Matches(msg, keys.down):
		m.cursor = min(m.cursor+1, max(len(m.origins)+len(m.known())-1, 0))
	case key.Matches(msg, keys.up):
		m.cursor = max(m.cursor-1, 0)
	case key.Matches(msg, keys.syncAll) && len(m.origins) > 0:
		return m.synced(nil)
	case chosen.origin != nil && key.Matches(msg, keys.sync):
		return m.synced([]string{chosen.origin.Name})
	case chosen.origin != nil && key.Matches(msg, keys.ref) && config.IsGitURL(chosen.origin.Source):
		name := chosen.origin.Name
		m.running = "Read the refs of the catalog " + name
		return m, func() tea.Msg {
			refs, err := m.registry.Refs(m.ctx, name)
			return refsMsg{refs: refs, err: err}
		}
	case chosen.origin != nil && key.Matches(msg, keys.remove):
		m.confirming = true
		m.message, m.failed = "Remove the catalog "+chosen.origin.Name+"? Press y to remove it, or another key to keep it.", false
	case chosen.known != nil && key.Matches(msg, keys.add):
		known := *chosen.known
		return m.started("Add the catalog "+known.name, func() (string, error) {
			origin, err := m.registry.Add(m.ctx, known.name, known.entry.ConfigEntry(), catalog.NoSteps{})
			return fmt.Sprintf("Added the catalog %s at the commit %s.", known.name, registry.ShortCommit(origin.Commit)), err
		})
	}
	return m, nil
}

func (m Model) pressedInRefs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.cancel):
		m.refs = nil
	case key.Matches(msg, keys.down):
		m.refCursor = min(m.refCursor+1, len(m.refs)-1)
	case key.Matches(msg, keys.up):
		m.refCursor = max(m.refCursor-1, 0)
	case key.Matches(msg, keys.choose):
		name, ref := m.selected().origin.Name, m.refs[m.refCursor]
		m.refs = nil
		return m.started("Set the ref of the catalog "+name, func() (string, error) {
			origin, err := m.registry.Set(m.ctx, name, registry.Change{Ref: &ref}, catalog.NoSteps{})
			return fmt.Sprintf("The catalog %s has the ref %s, at the commit %s.", name, m.refText(ref), registry.ShortCommit(origin.Commit)), err
		})
	}
	return m, nil
}

func (m Model) synced(names []string) (tea.Model, tea.Cmd) {
	label := "Sync each catalog"
	if len(names) == 1 {
		label = "Sync the catalog " + names[0]
	}
	return m.started(label, func() (string, error) {
		synced, err := m.registry.Sync(m.ctx, names, catalog.NoSteps{})
		if err != nil {
			return "", err
		}
		var lines []string
		for _, s := range synced {
			line := s.String()
			if s.Err != nil {
				line += ": " + s.Err.Error()
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n"), nil
	})
}

func (m Model) started(label string, run func() (string, error)) (tea.Model, tea.Cmd) {
	m.running, m.message, m.failed = label, "", false
	return m, func() tea.Msg {
		message, err := run()
		origins, listErr := m.registry.List(m.ctx)
		if err == nil {
			err = listErr
		}
		return doneMsg{origins: origins, listed: listErr == nil, message: message, err: err}
	}
}

func (m Model) known() []knownCatalog {
	if !m.indexReady || m.indexErr != nil {
		return nil
	}
	var known []knownCatalog
	for _, name := range m.index.Names() {
		if slices.ContainsFunc(m.origins, func(o catalog.Origin) bool { return o.Name == name }) {
			continue
		}
		entry, err := m.index.Entry(name)
		if err == nil {
			known = append(known, knownCatalog{name: name, entry: entry})
		}
	}
	return known
}

func (m Model) selected() selected {
	if m.cursor < len(m.origins) {
		return selected{origin: &m.origins[m.cursor]}
	}
	known := m.known()
	if i := m.cursor - len(m.origins); i < len(known) {
		return selected{known: &known[i]}
	}
	return selected{}
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if !screen.Fits(m.width, m.height) {
		return screen.TooSmall("catalog", m.width)
	}
	paneHeight := m.height - 3
	list := screen.Pane{Title: "Catalogs", Lines: m.listLines(), Width: m.listWidth(), Height: paneHeight, Border: screen.FocusedBorder}
	list.Lines = list.Lines[:min(len(list.Lines), max(paneHeight-2, 0))]
	title, detailLines := m.details(m.width - list.Width - 4)
	details := screen.Pane{Title: title, Lines: detailLines, Width: m.width - list.Width, Height: paneHeight, Border: screen.PlainBorder}
	counts := screen.Counted(len(m.origins), "catalog") + " · " + screen.Counted(len(m.known()), "known catalog")
	return screen.Header("feature-compose catalog", counts, m.width) + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, list.View(), details.View()) + "\n" + m.status() + "\n" + m.footer()
}

func (m Model) listLines() []string {
	width := m.listWidth() - 4
	var lines []string
	if len(m.origins) == 0 {
		lines = append(lines, screen.Wrapped(screen.NoneStyle.Render("The config file has no catalog."), width)...)
	}
	for i, origin := range m.origins {
		lines = append(lines, m.rowLine(i, origin.Name, width))
	}
	lines = append(lines, "", screen.HeadingStyle.Render("Known catalogs"))
	known := m.known()
	switch {
	case !m.indexReady:
		lines = append(lines, screen.Wrapped(screen.NoneStyle.Render("Getting the index…"), width)...)
	case m.indexErr != nil:
		lines = append(lines, screen.Wrapped(screen.NoneStyle.Render("The tool cannot get the index."), width)...)
	case len(known) == 0:
		lines = append(lines, screen.Wrapped(screen.NoneStyle.Render("The config file has each known catalog."), width)...)
	}
	for i, k := range known {
		lines = append(lines, m.rowLine(len(m.origins)+i, k.name, width))
	}
	return lines
}

func (m Model) rowLine(i int, name string, width int) string {
	if i == m.cursor {
		return screen.SelectedRowLine(" "+name, width)
	}
	return screen.RowLine(" ", screen.NameStyle.Render(name), width)
}

func (m Model) listWidth() int {
	widest := lipgloss.Width("Known catalogs")
	for _, origin := range m.origins {
		widest = max(widest, lipgloss.Width(origin.Name)+2)
	}
	for _, k := range m.known() {
		widest = max(widest, lipgloss.Width(k.name)+2)
	}
	return min(max(widest+5, narrowestList), m.width*2/5)
}

func (m Model) details(width int) (string, []string) {
	chosen := m.selected()
	switch {
	case chosen.origin != nil && m.refs != nil:
		return "the refs of " + chosen.origin.Name, m.refLines(width)
	case chosen.origin != nil:
		return "the catalog " + chosen.origin.Name, m.originLines(*chosen.origin, width)
	case chosen.known != nil:
		return "the known catalog " + chosen.known.name, m.knownLines(chosen.known.entry, width)
	case m.indexErr != nil:
		return "Details", strings.Split(screen.Message(m.indexErr.Error(), width), "\n")
	}
	return "Details", nil
}

func (m Model) originLines(origin catalog.Origin, width int) []string {
	path, ref, commit := origin.Path, m.refText(origin.Ref), origin.Commit
	if path == "" {
		path = "the root of the source"
	}
	switch {
	case !config.IsGitURL(origin.Source):
		ref, commit = "none: a local folder", "none: a local folder"
	case commit == "":
		commit = "no clone: press s to clone the source again"
	}
	return screen.Entry{Fields: []screen.Field{
		{Label: "source", Values: []string{origin.Source}},
		{Label: "path", Values: []string{path}},
		{Label: "ref", Values: []string{ref}},
		{Label: "commit", Values: []string{commit}},
	}}.Lines(width)
}

func (m Model) knownLines(entry index.Entry, width int) []string {
	path := entry.Path
	if path == "" {
		path = "the root of the source"
	}
	lines := append(screen.Wrapped(entry.Summary, width), "")
	lines = append(lines, screen.Entry{Fields: []screen.Field{
		{Label: "source", Values: []string{entry.Source}},
		{Label: "path", Values: []string{path}},
		{Label: "ref", Values: []string{m.refText(entry.Ref)}},
	}}.Lines(width)...)
	return append(append(lines, ""), screen.Wrapped("Press enter to add it to the config file.", width)...)
}

func (m Model) refLines(width int) []string {
	lines := screen.Wrapped("Choose the ref of the catalog. The list has the branches and the tags of its clone.", width)
	lines = append(lines, "")
	for i, ref := range m.refs {
		if i == m.refCursor {
			lines = append(lines, screen.SelectedRowLine(" "+m.refText(ref), width))
			continue
		}
		lines = append(lines, screen.RowLine(" ", m.refText(ref), width))
	}
	return lines
}

func (m Model) refText(ref string) string {
	if ref == "" {
		return "the default branch"
	}
	return ref
}

func (m Model) status() string {
	message := strings.ReplaceAll(m.message, "\n", " · ")
	if width := m.width - 2; lipgloss.Width(message) > width {
		message = ansi.Truncate(message, width, "…")
	}
	switch {
	case m.running != "":
		return " " + screen.MarkStyle.Render(m.running+"…")
	case m.failed:
		return " " + screen.ProblemStyle.Render(message)
	}
	return " " + message
}

func (m Model) footer() string {
	m.help.Width = m.width - 1
	chosen := m.selected()
	var bindings []key.Binding
	switch {
	case m.running != "":
		return ""
	case m.confirming:
		bindings = []key.Binding{keys.yes}
	case m.refs != nil:
		bindings = []key.Binding{keys.down, keys.choose, keys.cancel}
	case chosen.origin != nil:
		bindings = []key.Binding{keys.down, keys.sync, keys.syncAll}
		if config.IsGitURL(chosen.origin.Source) {
			bindings = append(bindings, keys.ref)
		}
		bindings = append(bindings, keys.remove, keys.quit)
	case chosen.known != nil:
		bindings = []key.Binding{keys.down, keys.add, keys.quit}
	default:
		bindings = []key.Binding{keys.down, keys.quit}
	}
	return " " + m.help.ShortHelpView(bindings)
}
