package browser

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hpcsc/feature-compose/internal/catalog"
)

type group string

const (
	featuresGroup        group = "features"
	extensionPointsGroup group = "extension points"
)

var groups = []group{featuresGroup, extensionPointsGroup}

type row struct {
	starter string
	group   group
	name    string
}

func (r row) isStarter() bool {
	return r.group == ""
}

func (r row) isGroup() bool {
	return r.group != "" && r.name == ""
}

func (r row) parent() row {
	if r.isGroup() {
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
	}
	if len(t.starterNames) > 0 {
		t.selected = row{starter: t.starterNames[0]}
	}
	return t
}

func (t tree) rows() []treeRow {
	var rows []treeRow
	for _, starterName := range t.starterNames {
		shown := t.shownGroupsOf(starterName)
		if !t.matches(starterName) && len(shown) == 0 {
			continue
		}
		starterRow := row{starter: starterName}
		rows = append(rows, treeRow{row: starterRow, prefix: marker(len(shown) > 0, t.isOpen(starterRow)), canOpen: len(shown) > 0, isOpen: t.isOpen(starterRow) && len(shown) > 0})
		if !t.isOpen(starterRow) {
			continue
		}
		for i, g := range shown {
			lastGroup := i == len(shown)-1
			groupRow := row{starter: starterName, group: g}
			rows = append(rows, treeRow{row: groupRow, prefix: "  " + branch(lastGroup) + marker(true, t.isOpen(groupRow)), canOpen: true, isOpen: t.isOpen(groupRow)})
			if !t.isOpen(groupRow) {
				continue
			}
			names := t.shownNamesOf(starterName, g)
			for j, name := range names {
				rows = append(rows, treeRow{row: row{starter: starterName, group: g, name: name}, prefix: "  " + childIndent(lastGroup) + branch(j == len(names)-1)})
			}
		}
	}
	return rows
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
	if t.matches(starterName) {
		return t.children[starterName][g]
	}
	var shown []string
	for _, name := range t.children[starterName][g] {
		if t.matches(name) {
			shown = append(shown, name)
		}
	}
	return shown
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
	if !t.selected.isStarter() && !t.selected.isGroup() {
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
		if text == "" || t.matches(starterName) {
			continue
		}
		for _, g := range t.shownGroupsOf(starterName) {
			t.filterOpen[row{starter: starterName}] = true
			t.filterOpen[row{starter: starterName, group: g}] = true
		}
	}
	for t.selectedIndex() < 0 && !t.selected.isStarter() {
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
	widest := 0
	for _, starterName := range t.starterNames {
		widest = max(widest, lipgloss.Width(marker(true, true)+starterName))
		for _, g := range groups {
			widest = max(widest, lipgloss.Width("  "+branch(false)+marker(true, true)+string(g)))
			for _, name := range t.children[starterName][g] {
				widest = max(widest, lipgloss.Width("  "+childIndent(false)+branch(false)+name))
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
	name := r.name
	switch {
	case r.isStarter():
		name = r.starter
	case r.isGroup():
		name = string(r.group)
	}
	if selected {
		text := ansi.Truncate(r.prefix+name, width-1, "…")
		return "▌" + selectedRowStyle.Render(text+strings.Repeat(" ", max(width-1-lipgloss.Width(text), 0)))
	}
	switch {
	case r.isStarter():
		name = nameStyle.Render(name)
	case r.isGroup():
		name = groupStyle.Render(name)
	}
	return " " + ansi.Truncate(treeLineStyle.Render(r.prefix)+name, width-1, "…")
}

func marker(canOpen, isOpen bool) string {
	switch {
	case !canOpen:
		return "  "
	case isOpen:
		return "▾ "
	}
	return "▸ "
}

func branch(last bool) string {
	if last {
		return "└─ "
	}
	return "├─ "
}

func childIndent(lastGroup bool) string {
	if lastGroup {
		return "   "
	}
	return "│  "
}
