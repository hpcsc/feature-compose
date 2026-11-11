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

func (n node) canOpen() bool {
	return len(n.children) > 0
}

func (n node) styledText() string {
	if n.kind == starterKind {
		return screen.NameStyle.Render(n.text)
	}
	return n.text
}

func (n node) withMark(text, mark string) string {
	if n.mark == "" {
		return text
	}
	return text + "  " + mark
}

type treeRow struct {
	node
	prefix string
	parent row
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

type tree struct {
	nodes    []node
	selected row
	open     map[row]bool
}

func (t tree) rows() []treeRow {
	var rows []treeRow
	for _, n := range t.nodes {
		rows = append(rows, t.rowsOf(n, n.row, "", "  ")...)
	}
	return rows
}

func (t tree) rowsOf(n node, parent row, prefix, childIndent string) []treeRow {
	rows := []treeRow{{node: n, prefix: prefix + screen.Arrow(n.canOpen(), t.isOpen(n)), parent: parent}}
	if t.isOpen(n) {
		for i, child := range n.children {
			last := i == len(n.children)-1
			rows = append(rows, t.rowsOf(child, n.row, childIndent+screen.Branch(last), childIndent+screen.ChildIndent(last))...)
		}
	}
	return rows
}

func (t tree) isOpen(n node) bool {
	isOpen, ok := t.open[n.row]
	return n.canOpen() && (!ok || isOpen)
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
	case t.isOpen(rows[i].node):
		t.selected = rows[i+1].row
	case rows[i].canOpen():
		return t.withOpen(rows[i].row, true)
	}
	return t
}

func (t tree) closed() tree {
	r := t.rows()[t.selectedIndex()]
	if t.isOpen(r.node) {
		return t.withOpen(r.row, false)
	}
	t.selected = r.parent
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
