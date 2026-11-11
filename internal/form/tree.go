package form

import (
	"maps"
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/screen"
)

type node struct {
	row
	text     string
	mark     string
	children []node
}

type treeRow struct {
	row
	prefix  string
	text    string
	mark    string
	canOpen bool
}

func (r treeRow) line(selected bool, width int) string {
	if selected {
		return screen.SelectedRowLine(r.plain(), width)
	}
	return screen.RowLine(r.prefix, r.withMark(r.styledText(), screen.MarkStyle.Render(r.mark)), width)
}

func (r treeRow) plain() string {
	return r.prefix + r.withMark(r.text, r.mark)
}

func (r treeRow) styledText() string {
	if r.kind == starterKind {
		return screen.NameStyle.Render(r.text)
	}
	return r.text
}

func (r treeRow) withMark(text, mark string) string {
	if r.mark == "" {
		return text
	}
	return text + "  " + mark
}

type tree struct {
	nodes    []node
	selected row
	open     map[row]bool
}

func (t tree) rows() []treeRow {
	return t.rowsOf(t.nodes, "", false)
}

func (t tree) rowsOf(nodes []node, indent string, areChildren bool) []treeRow {
	var rows []treeRow
	for i, n := range nodes {
		branch, childIndent := "", "  "
		if areChildren {
			branch, childIndent = screen.Branch(i == len(nodes)-1), indent+screen.ChildIndent(i == len(nodes)-1)
		}
		canOpen := len(n.children) > 0
		isOpen := t.isOpen(n.row, canOpen)
		rows = append(rows, treeRow{row: n.row, prefix: indent + branch + screen.Arrow(canOpen, isOpen), text: n.text, mark: n.mark, canOpen: canOpen})
		if isOpen {
			rows = append(rows, t.rowsOf(n.children, childIndent, true)...)
		}
	}
	return rows
}

func (t tree) isOpen(r row, canOpen bool) bool {
	isOpen, ok := t.open[r]
	return canOpen && (!ok || isOpen)
}

func (t tree) withOpen(r row, isOpen bool) tree {
	t.open = maps.Clone(t.open)
	if t.open == nil {
		t.open = map[row]bool{}
	}
	t.open[r] = isOpen
	return t
}

func (t tree) selectedIndex() int {
	return slices.IndexFunc(t.rows(), func(r treeRow) bool { return r.row == t.selected })
}

func (t tree) moved(by int) tree {
	rows := t.rows()
	t.selected = rows[min(max(t.selectedIndex()+by, 0), len(rows)-1)].row
	return t
}

func (t tree) opened() tree {
	rows := t.rows()
	i := t.selectedIndex()
	switch {
	case t.isOpen(rows[i].row, rows[i].canOpen):
		t.selected = rows[i+1].row
	case rows[i].canOpen:
		return t.withOpen(rows[i].row, true)
	}
	return t
}

func (t tree) closed() tree {
	r := t.rows()[t.selectedIndex()]
	if t.isOpen(r.row, r.canOpen) {
		return t.withOpen(r.row, false)
	}
	t.selected = t.selected.parent()
	return t
}

func (t tree) widestRow() int {
	widest := 0
	for _, r := range t.rows() {
		widest = max(widest, lipgloss.Width(r.plain()))
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
