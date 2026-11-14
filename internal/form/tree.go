package form

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/screen"
)

const missingMark = "missing"

type node struct {
	row
	label     string
	text      string
	textStyle lipgloss.Style
	marks     []string
	children  []node
}

func (n node) canOpen() bool {
	return len(n.children) > 0
}

func (n node) plain() string {
	return n.withMarks(n.label+n.text, strings.Join(n.marks, ", "))
}

func (n node) styled() string {
	marks := make([]string, 0, len(n.marks))
	for _, mark := range n.marks {
		style := screen.MarkStyle
		if mark == missingMark {
			style = screen.ProblemStyle
		}
		marks = append(marks, style.Render(mark))
	}
	label := n.label
	if label != "" {
		label = screen.LabelStyle.Render(label)
	}
	return n.withMarks(label+n.textStyle.Render(n.text), strings.Join(marks, screen.MarkStyle.Render(", ")))
}

func (n node) withMarks(text, marks string) string {
	if len(n.marks) == 0 {
		return text
	}
	return text + "  " + marks
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
	return screen.RowLine(r.prefix, r.styled(), width)
}

func (r treeRow) plain() string {
	return r.prefix + r.node.plain()
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

func (t tree) revealed(r row) tree {
	for _, parent := range t.parentsOf(r, t.nodes) {
		t = t.withOpen(parent, true)
	}
	t.selected = r
	return t
}

func (t tree) parentsOf(r row, nodes []node) []row {
	for _, n := range nodes {
		if n.row == r {
			return []row{}
		}
		if parents := t.parentsOf(r, n.children); parents != nil {
			return append([]row{n.row}, parents...)
		}
	}
	return nil
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
