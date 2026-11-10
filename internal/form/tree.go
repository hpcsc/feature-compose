package form

import (
	"maps"
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/screen"
)

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
	rows     []treeRow
	selected row
	open     map[row]bool
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
	return slices.IndexFunc(t.rows, func(r treeRow) bool { return r.row == t.selected })
}

func (t tree) moved(by int) tree {
	t.selected = t.rows[min(max(t.selectedIndex()+by, 0), len(t.rows)-1)].row
	return t
}

func (t tree) opened() tree {
	i := t.selectedIndex()
	r := t.rows[i]
	switch {
	case t.isOpen(r.row, r.canOpen):
		t.selected = t.rows[i+1].row
	case r.canOpen:
		return t.withOpen(r.row, true)
	}
	return t
}

func (t tree) closed() tree {
	r := t.rows[t.selectedIndex()]
	if t.isOpen(r.row, r.canOpen) {
		return t.withOpen(r.row, false)
	}
	t.selected = t.selected.parent()
	return t
}

func (t tree) widestRow() int {
	widest := 0
	for _, r := range t.rows {
		widest = max(widest, lipgloss.Width(r.plain()))
	}
	return widest
}

func (t tree) lines(offset, width, height int) []string {
	lines := make([]string, 0, height)
	for _, r := range t.rows[min(offset, len(t.rows)):min(offset+height, len(t.rows))] {
		lines = append(lines, r.line(r.row == t.selected, width))
	}
	return lines
}
