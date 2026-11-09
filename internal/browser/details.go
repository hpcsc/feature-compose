package browser

import (
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hpcsc/feature-compose/internal/catalog"
)

const narrowestValue = 20

type field struct {
	label  string
	values []string
}

type entry struct {
	name    string
	summary string
	fields  []field
}

type section struct {
	heading string
	entries []entry
	none    string
}

type declaredExtensionPoint struct {
	catalog.ExtensionPointDefinition
	declarer string
}

func extensionPointsOf(narrowed catalog.Definitions, starterName string) map[string]declaredExtensionPoint {
	extensionPoints := map[string]declaredExtensionPoint{}
	for name, extensionPoint := range narrowed.Starters[starterName].ExtensionPoints {
		extensionPoints[name] = declaredExtensionPoint{ExtensionPointDefinition: extensionPoint, declarer: "the starter"}
	}
	for _, featureName := range slices.Sorted(maps.Keys(narrowed.Features)) {
		for name, extensionPoint := range narrowed.Features[featureName].ExtensionPoints {
			extensionPoints[name] = declaredExtensionPoint{ExtensionPointDefinition: extensionPoint, declarer: "the feature " + featureName}
		}
	}
	return extensionPoints
}

func detailsOf(definitions catalog.Definitions, r row, width int) string {
	narrowed, _ := definitions.ForStarter(r.starter)
	switch {
	case r.isStarter():
		return starterDetails(narrowed.Starters[r.starter], width)
	case r.isGroup() && r.group == featuresGroup:
		return featuresDetails(narrowed, width)
	case r.isGroup():
		return extensionPointsDetails(narrowed, r.starter, width)
	case r.group == featuresGroup:
		return featureDetails(narrowed.Features[r.name], width)
	}
	return extensionPointDetails(narrowed, r.starter, r.name, width)
}

func starterDetails(starter catalog.StarterDefinition, width int) string {
	folders := section{heading: "Folders", none: "none"}
	for _, name := range slices.Sorted(maps.Keys(starter.Folders)) {
		folder := starter.Folders[name]
		folders.entries = append(folders.entries, entry{name: name, fields: []field{
			{label: "path", values: []string{folder.Path}},
			{label: "after_render", values: folder.AfterRender},
			{label: "test", values: folder.Test},
		}})
	}
	return details(starter.Summary, width, folders, optionsSection(starter.Options))
}

func featuresDetails(narrowed catalog.Definitions, width int) string {
	features := section{heading: "Features", none: "none"}
	for _, name := range slices.Sorted(maps.Keys(narrowed.Features)) {
		features.entries = append(features.entries, entry{name: name, summary: narrowed.Features[name].Summary})
	}
	return details("", width, features)
}

func extensionPointsDetails(narrowed catalog.Definitions, starterName string, width int) string {
	extensionPoints := extensionPointsOf(narrowed, starterName)
	s := section{heading: "Extension points", none: "none"}
	for _, name := range slices.Sorted(maps.Keys(extensionPoints)) {
		s.entries = append(s.entries, entry{name: name, fields: []field{
			{label: "item", values: []string{strings.Join(extensionPoints[name].Item, ", ")}},
			{label: "declared by", values: []string{extensionPoints[name].declarer}},
		}})
	}
	return details("", width, s)
}

func featureDetails(feature catalog.FeatureDefinition, width int) string {
	needs := section{heading: "Needs", none: "none"}
	for _, name := range feature.Needs {
		needs.entries = append(needs.entries, entry{name: name})
	}

	declared := section{heading: "Extension points that it declares", none: "none"}
	for _, name := range slices.Sorted(maps.Keys(feature.ExtensionPoints)) {
		extensionPoint := feature.ExtensionPoints[name]
		declared.entries = append(declared.entries, entry{name: name, summary: extensionPoint.Summary, fields: []field{
			{label: "item", values: []string{strings.Join(extensionPoint.Item, ", ")}},
		}})
	}

	addedTo := section{heading: "Extension points that it adds items to", none: "none"}
	for _, name := range slices.Sorted(maps.Keys(feature.AddTo)) {
		addedTo.entries = append(addedTo.entries, entry{name: name})
	}

	return details(feature.Summary, width, needs, optionsSection(feature.Options), declared, addedTo)
}

func extensionPointDetails(narrowed catalog.Definitions, starterName, name string, width int) string {
	extensionPoint := extensionPointsOf(narrowed, starterName)[name]
	addedToBy := []string{}
	for _, featureName := range slices.Sorted(maps.Keys(narrowed.Features)) {
		if _, ok := narrowed.Features[featureName].AddTo[name]; ok {
			addedToBy = append(addedToBy, featureName)
		}
	}
	if len(addedToBy) == 0 {
		addedToBy = []string{noneStyle.Render("none")}
	}
	fields := entry{fields: []field{
		{label: "file", values: []string{extensionPoint.File}},
		{label: "item", values: []string{strings.Join(extensionPoint.Item, ", ")}},
		{label: "declared by", values: []string{extensionPoint.declarer}},
		{label: "added to by", values: addedToBy},
	}}
	var lines []string
	if extensionPoint.Summary != "" {
		lines = append(wrapped(extensionPoint.Summary, width), "")
	}
	return strings.Join(append(lines, fields.lines(width, fields.labelWidth())...), "\n")
}

func optionsSection(options map[string]catalog.OptionDefinition) section {
	s := section{heading: "Options", none: "none"}
	for _, name := range slices.Sorted(maps.Keys(options)) {
		option := options[name]
		fields := []field{{label: "default", values: []string{option.Default}}}
		if len(option.Values) > 0 {
			fields = append(fields, field{label: "values", values: []string{strings.Join(option.Values, ", ")}})
		}
		if option.Pattern != "" {
			fields = append(fields, field{label: "pattern", values: []string{option.Pattern}})
		}
		s.entries = append(s.entries, entry{name: name, fields: fields})
	}
	return s
}

func details(summary string, width int, sections ...section) string {
	var lines []string
	if summary != "" {
		lines = append(lines, wrapped(summary, width)...)
	}
	for _, s := range sections {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, s.lines(width)...)
	}
	return strings.Join(lines, "\n")
}

func (s section) lines(width int) []string {
	lines := wrapped(headingStyle.Render(s.heading), width)
	if len(s.entries) == 0 {
		return append(lines, "  "+noneStyle.Render(s.none))
	}
	nameWidth, labelWidth := 0, 0
	for _, e := range s.entries {
		nameWidth = max(nameWidth, lipgloss.Width(e.name))
		labelWidth = max(labelWidth, e.labelWidth())
	}
	valueWidth := width - 2 - nameWidth - 2
	for _, e := range s.entries {
		if valueWidth < narrowestValue {
			lines = append(lines, "  "+nameStyle.Render(e.name))
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
				name = nameStyle.Render(padded(e.name, nameWidth))
			}
			lines = append(lines, strings.TrimRight("  "+name+"  "+line, " "))
		}
	}
	return lines
}

func (e entry) labelWidth() int {
	widest := 0
	for _, f := range e.fields {
		widest = max(widest, lipgloss.Width(f.label))
	}
	return widest
}

func (e entry) lines(width, labelWidth int) []string {
	var lines []string
	if e.summary != "" {
		lines = append(lines, wrapped(e.summary, width)...)
	}
	for _, f := range e.fields {
		for i, value := range f.values {
			label := strings.Repeat(" ", labelWidth)
			if i == 0 {
				label = labelStyle.Render(padded(f.label, labelWidth))
			}
			for j, line := range wrapped(value, max(width-labelWidth-2, 1)) {
				if j > 0 {
					label = strings.Repeat(" ", labelWidth)
				}
				lines = append(lines, label+"  "+line)
			}
		}
	}
	return lines
}

func wrapped(text string, width int) []string {
	return strings.Split(ansi.Wrap(text, max(width, 1), ""), "\n")
}

func padded(text string, width int) string {
	return text + strings.Repeat(" ", max(width-lipgloss.Width(text), 0))
}
