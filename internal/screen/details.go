package screen

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const narrowestValue = 20

type Field struct {
	Label  string
	Values []string
}

type Entry struct {
	Name    string
	Summary string
	Fields  []Field
}

func (e Entry) Lines(width int) []string {
	return e.lines(width, e.labelWidth())
}

func (e Entry) labelWidth() int {
	widest := 0
	for _, f := range e.Fields {
		widest = max(widest, lipgloss.Width(f.Label))
	}
	return widest
}

func (e Entry) lines(width, labelWidth int) []string {
	var lines []string
	if e.Summary != "" {
		lines = append(lines, Wrapped(e.Summary, width)...)
	}
	for _, f := range e.Fields {
		for i, value := range f.Values {
			label := strings.Repeat(" ", labelWidth)
			if i == 0 {
				label = LabelStyle.Render(padded(f.Label, labelWidth))
			}
			for j, line := range Wrapped(value, max(width-labelWidth-2, 1)) {
				if j > 0 {
					label = strings.Repeat(" ", labelWidth)
				}
				lines = append(lines, label+"  "+line)
			}
		}
	}
	return lines
}

type Section struct {
	Heading string
	Entries []Entry
	None    string
}

func (s Section) lines(width int) []string {
	lines := Wrapped(HeadingStyle.Render(s.Heading), width)
	if len(s.Entries) == 0 {
		return append(lines, "  "+NoneStyle.Render(s.None))
	}
	nameWidth, labelWidth := 0, 0
	for _, e := range s.Entries {
		nameWidth = max(nameWidth, lipgloss.Width(e.Name))
		labelWidth = max(labelWidth, e.labelWidth())
	}
	valueWidth := width - 2 - nameWidth - 2
	for _, e := range s.Entries {
		if valueWidth < narrowestValue {
			lines = append(lines, "  "+NameStyle.Render(e.Name))
			for _, line := range e.lines(width-4, labelWidth) {
				lines = append(lines, "    "+line)
			}
			continue
		}
		entryLines := e.lines(valueWidth, labelWidth)
		if len(entryLines) == 0 {
			entryLines = []string{""}
		}
		for i, line := range entryLines {
			name := strings.Repeat(" ", nameWidth)
			if i == 0 {
				name = NameStyle.Render(padded(e.Name, nameWidth))
			}
			lines = append(lines, strings.TrimRight("  "+name+"  "+line, " "))
		}
	}
	return lines
}

func ValuesSection(heading, none string, valuesByName map[string]map[string]string) Section {
	s := Section{Heading: heading, None: none}
	for _, name := range slices.Sorted(maps.Keys(valuesByName)) {
		var fields []Field
		for _, label := range slices.Sorted(maps.Keys(valuesByName[name])) {
			fields = append(fields, Field{Label: label, Values: []string{valuesByName[name][label]}})
		}
		s.Entries = append(s.Entries, Entry{Name: name, Fields: fields})
	}
	return s
}

func Details(summary string, width int, sections ...Section) string {
	var lines []string
	if summary != "" {
		lines = append(lines, Wrapped(summary, width)...)
	}
	for _, s := range sections {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, s.lines(width)...)
	}
	return strings.Join(lines, "\n")
}
