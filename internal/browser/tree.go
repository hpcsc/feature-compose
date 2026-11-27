package browser

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/screen"
)

type group string

const (
	featuresGroup        group = "features"
	extensionPointsGroup group = "extension points"
)

var groups = []group{featuresGroup, extensionPointsGroup}

type row struct {
	catalog string
	starter string
	group   group
	name    string
}

func (r row) catalogName() string {
	if r.isCatalog() {
		return r.catalog
	}
	catalogName, _ := catalog.SplitFullName(r.starter)
	return catalogName
}

func (r row) isCatalog() bool {
	return r.starter == ""
}

func (r row) isStarter() bool {
	return r.starter != "" && r.group == ""
}

func (r row) isGroup() bool {
	return r.group != "" && r.name == ""
}

func (r row) parent() row {
	switch {
	case r.isCatalog():
		return r
	case r.isStarter():
		if r.catalogName() == "" {
			return r
		}
		return row{catalog: r.catalogName()}
	case r.isGroup():
		return row{starter: r.starter}
	}
	return row{starter: r.starter, group: r.group}
}

type treeRow struct {
	row
	prefix  string
	canOpen bool
	isOpen  bool
}

type tree struct {
	catalogNames []string
	starterNames []string
	children     map[string]map[group][]string
	open         map[row]bool
	filter       string
	filterOpen   map[row]bool
	selected     row
}

func newTree(definitions catalog.Definitions) tree {
	t := tree{starterNames: slices.Sorted(maps.Keys(definitions.Starters)), children: map[string]map[group][]string{}, open: map[row]bool{}}
	for _, starterName := range t.starterNames {
		narrowed, _ := definitions.ForStarter(starterName)
		t.children[starterName] = map[group][]string{
			featuresGroup:        slices.Sorted(maps.Keys(narrowed.Features)),
			extensionPointsGroup: slices.Sorted(maps.Keys(extensionPointsOf(narrowed, starterName))),
		}
		if catalogName := (row{starter: starterName}).catalogName(); catalogName != "" && !slices.Contains(t.catalogNames, catalogName) {
			t.catalogNames = append(t.catalogNames, catalogName)
		}
	}
	if len(t.starterNames) > 0 {
		t.selected = row{starter: t.starterNames[0]}
	}
	return t
}

func (t tree) rows() []treeRow {
	if len(t.catalogNames) == 0 {
		return t.starterRows("", t.starterNames)
	}
	var rows []treeRow
	for _, catalogName := range t.catalogNames {
		shown := t.shownStartersOf(catalogName)
		if len(shown) == 0 {
			continue
		}
		catalogRow := row{catalog: catalogName}
		rows = append(rows, treeRow{row: catalogRow, prefix: screen.Arrow(true, t.isOpen(catalogRow)), canOpen: true, isOpen: t.isOpen(catalogRow)})
		if t.isOpen(catalogRow) {
			rows = append(rows, t.starterRows("  ", shown)...)
		}
	}
	return rows
}

func (t tree) starterRows(indent string, starterNames []string) []treeRow {
	var rows []treeRow
	for _, starterName := range starterNames {
		shown := t.shownGroupsOf(starterName)
		if !t.matchesStarter(starterName) && len(shown) == 0 {
			continue
		}
		starter := row{starter: starterName}
		rows = append(rows, treeRow{row: starter, prefix: indent + screen.Arrow(len(shown) > 0, t.isOpen(starter)), canOpen: len(shown) > 0, isOpen: t.isOpen(starter) && len(shown) > 0})
		if !t.isOpen(starter) {
			continue
		}
		for i, g := range shown {
			lastGroup := i == len(shown)-1
			groupRow := row{starter: starterName, group: g}
			rows = append(rows, treeRow{row: groupRow, prefix: indent + "  " + screen.Branch(lastGroup) + screen.Arrow(true, t.isOpen(groupRow)), canOpen: true, isOpen: t.isOpen(groupRow)})
			if !t.isOpen(groupRow) {
				continue
			}
			names := t.shownNamesOf(starterName, g)
			for j, name := range names {
				rows = append(rows, treeRow{row: row{starter: starterName, group: g, name: name}, prefix: indent + "  " + screen.ChildIndent(lastGroup) + screen.Branch(j == len(names)-1)})
			}
		}
	}
	return rows
}

func (t tree) startersOf(catalogName string) []string {
	var starterNames []string
	for _, starterName := range t.starterNames {
		if (row{starter: starterName}).catalogName() == catalogName {
			starterNames = append(starterNames, starterName)
		}
	}
	return starterNames
}

func (t tree) shownStartersOf(catalogName string) []string {
	var shown []string
	for _, starterName := range t.startersOf(catalogName) {
		if t.matchesStarter(starterName) || len(t.shownGroupsOf(starterName)) > 0 {
			shown = append(shown, starterName)
		}
	}
	return shown
}

func (t tree) shownGroupsOf(starterName string) []group {
	var shown []group
	for _, g := range groups {
		if len(t.shownNamesOf(starterName, g)) > 0 {
			shown = append(shown, g)
		}
	}
	return shown
}

func (t tree) shownNamesOf(starterName string, g group) []string {
	if t.matchesStarter(starterName) {
		return t.children[starterName][g]
	}
	var shown []string
	for _, name := range t.children[starterName][g] {
		if t.matches(displayed(g, name)) {
			shown = append(shown, name)
		}
	}
	return shown
}

func (t tree) matchesStarter(starterName string) bool {
	catalogName, name := catalog.SplitFullName(starterName)
	return t.matches(name) || (catalogName != "" && t.matches(catalogName))
}

func (t tree) matches(name string) bool {
	return strings.Contains(strings.ToLower(name), strings.ToLower(t.filter))
}

func (t tree) isOpen(r row) bool {
	if isOpen, ok := t.filterOpen[r]; ok && t.filter != "" {
		return isOpen
	}
	if isOpen, ok := t.open[r]; ok {
		return isOpen
	}
	return true
}

func (t tree) selectedIndex() int {
	return slices.IndexFunc(t.rows(), func(r treeRow) bool { return r.row == t.selected })
}

func (t tree) moved(by int) tree {
	rows := t.rows()
	if len(rows) == 0 {
		return t
	}
	t.selected = rows[min(max(t.selectedIndex()+by, 0), len(rows)-1)].row
	return t
}

func (t tree) opened() tree {
	rows := t.rows()
	i := t.selectedIndex()
	if i < 0 || !rows[i].canOpen {
		return t
	}
	if !rows[i].isOpen {
		return t.withOpen(t.selected, true)
	}
	t.selected = rows[i+1].row
	return t
}

func (t tree) closed() tree {
	if !t.selected.isCatalog() && !t.selected.isStarter() && !t.selected.isGroup() {
		t.selected = t.selected.parent()
		return t
	}
	if t.isOpen(t.selected) {
		return t.withOpen(t.selected, false)
	}
	t.selected = t.selected.parent()
	return t
}

func (t tree) filtered(text string) tree {
	t.filter = text
	t.filterOpen = map[row]bool{}
	for _, starterName := range t.starterNames {
		if text == "" {
			continue
		}
		starter := row{starter: starterName}
		if starter.catalogName() != "" && (t.matchesStarter(starterName) || len(t.shownGroupsOf(starterName)) > 0) {
			t.filterOpen[row{catalog: starter.catalogName()}] = true
		}
		if t.matchesStarter(starterName) {
			continue
		}
		for _, g := range t.shownGroupsOf(starterName) {
			t.filterOpen[starter] = true
			t.filterOpen[row{starter: starterName, group: g}] = true
		}
	}
	for t.selectedIndex() < 0 && t.selected.parent() != t.selected {
		t.selected = t.selected.parent()
	}
	if rows := t.rows(); t.selectedIndex() < 0 && len(rows) > 0 {
		t.selected = rows[0].row
	}
	return t
}

func (t tree) withOpen(r row, isOpen bool) tree {
	if t.filter != "" {
		t.filterOpen = maps.Clone(t.filterOpen)
		t.filterOpen[r] = isOpen
		return t
	}
	t.open = maps.Clone(t.open)
	t.open[r] = isOpen
	return t
}

func (t tree) widestRow() int {
	indent := ""
	widest := 0
	for _, catalogName := range t.catalogNames {
		indent = "  "
		widest = max(widest, lipgloss.Width(screen.Arrow(true, true)+catalogName))
	}
	for _, starterName := range t.starterNames {
		widest = max(widest, lipgloss.Width(indent+screen.Arrow(true, true)+shortName(starterName)))
		for _, g := range groups {
			widest = max(widest, lipgloss.Width(indent+"  "+screen.Branch(false)+screen.Arrow(true, true)+string(g)))
			for _, name := range t.children[starterName][g] {
				widest = max(widest, lipgloss.Width(indent+"  "+screen.ChildIndent(false)+screen.Branch(false)+displayed(g, name)))
			}
		}
	}
	return widest
}

func (t tree) lines(offset, width, height int) []string {
	rows := t.rows()
	lines := make([]string, 0, height)
	for _, r := range rows[min(offset, len(rows)):min(offset+height, len(rows))] {
		lines = append(lines, r.line(r.row == t.selected, width))
	}
	return lines
}

func (r treeRow) line(selected bool, width int) string {
	name := displayed(r.group, r.name)
	switch {
	case r.isCatalog():
		name = r.catalog
	case r.isStarter():
		name = shortName(r.starter)
	case r.isGroup():
		name = string(r.group)
	}
	if selected {
		return screen.SelectedRowLine(r.prefix+name, width)
	}
	switch {
	case r.isCatalog(), r.isStarter():
		name = screen.NameStyle.Render(name)
	case r.isGroup():
		name = screen.GroupStyle.Render(name)
	}
	return screen.RowLine(r.prefix, name, width)
}

func displayed(g group, name string) string {
	if g == featuresGroup {
		return shortName(name)
	}
	return name
}

func shortName(fullName string) string {
	_, name := catalog.SplitFullName(fullName)
	return name
}
