package form

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/screen"
)

type draft struct {
	catalog        catalog.Catalog
	definitions    catalog.Definitions
	starter        catalog.Starter
	starterName    string
	starterGiven   bool
	featureNames   []string
	name           string
	nameGiven      bool
	given          map[string]string
	givenByFeature map[string]map[string]string
	answers        catalog.Answers
	selection      catalog.Selection
}

func newDraft(c catalog.Catalog, starterName, name string, given map[string]string, givenByFeature map[string]map[string]string) (draft, error) {
	if name != "" {
		if err := catalog.CheckRepoName(name); err != nil {
			return draft{}, err
		}
	}
	definitions, err := c.Definitions()
	if err != nil {
		return draft{}, err
	}
	d := draft{catalog: c, definitions: definitions, starterGiven: starterName != "", name: name, nameGiven: name != "", given: given, givenByFeature: givenByFeature}
	if starterName == "" {
		return d, nil
	}
	return d.withStarter(starterName)
}

func (d draft) withStarter(starterName string) (draft, error) {
	starter, err := d.catalog.Starter(starterName)
	if err != nil {
		return draft{}, err
	}
	featureNames, err := starter.FeatureNames()
	if err != nil {
		return draft{}, err
	}
	answers, selection, err := d.selected(starter, d.answers)
	if err != nil {
		return draft{}, err
	}
	d.starter, d.starterName, d.featureNames, d.answers, d.selection = starter, starterName, featureNames, answers, selection
	return d, nil
}

func (d draft) selected(starter catalog.Starter, answers catalog.Answers) (catalog.Answers, catalog.Selection, error) {
	selection, err := starter.Select(d.name, d.given, d.givenByFeature, answers)
	return answers, selection, err
}

func (d draft) rows(t tree) []treeRow {
	nameMark := ""
	switch {
	case d.nameGiven:
		nameMark = "fixed"
	case d.name == "":
		nameMark = "missing"
	}
	rows := []treeRow{{row: row{kind: nameKind}, prefix: screen.Arrow(false, false), text: "name     " + d.name, mark: nameMark}}

	starterMark := ""
	switch {
	case d.starterGiven:
		starterMark = "fixed"
	case d.starterName == "":
		starterMark = "missing"
	}
	starterRow := row{kind: starterKind}
	children := d.featureRows()
	canOpen := len(children) > 0
	isOpen := t.isOpen(starterRow, canOpen)
	rows = append(rows, treeRow{row: starterRow, prefix: screen.Arrow(canOpen, isOpen), text: "starter  " + d.starterName, mark: starterMark, canOpen: canOpen})
	if isOpen {
		rows = append(rows, children...)
	}
	return rows
}

func (d draft) featureRows() []treeRow {
	rows := make([]treeRow, 0, len(d.featureNames))
	for i, featureName := range d.featureNames {
		box := "[ ] "
		if d.selection.Holds(featureName) {
			box = "[x] "
		}
		var marks []string
		if _, isGiven := d.givenByFeature[featureName]; isGiven {
			marks = append(marks, "fixed")
		}
		if neededBy := d.selection.NeededBy(featureName); len(neededBy) > 0 {
			marks = append(marks, "needed by "+strings.Join(neededBy, ", "))
		}
		rows = append(rows, treeRow{row: row{kind: featureKind, feature: featureName}, prefix: "  " + screen.Branch(i == len(d.featureNames)-1), text: screen.Arrow(false, false) + box + featureName, mark: strings.Join(marks, ", ")})
	}
	return rows
}

func (d draft) featureCount() int {
	count := 0
	for _, featureName := range d.featureNames {
		if d.selection.Holds(featureName) {
			count++
		}
	}
	return count
}

func (d draft) details(r row, width int) (title, text string) {
	switch r.kind {
	case nameKind:
		return "name", d.nameDetails(width)
	case starterKind:
		return cmp.Or(d.starterName, "starter"), d.starterDetails(width)
	case featureKind:
		return r.feature, d.featureDetails(r.feature, width)
	}
	panic(fmt.Sprintf("the form has no details for the row kind %d", r.kind))
}

func (d draft) nameDetails(width int) string {
	if d.name == "" {
		return strings.Join(screen.Wrapped(catalog.CheckRepoName("").Error(), width), "\n")
	}
	source := "answered"
	if d.nameGiven {
		source = "command line"
	}
	return strings.Join(screen.Entry{Fields: []screen.Field{{Label: "value", Values: []string{d.name}}, {Label: "source", Values: []string{source}}}}.Lines(width), "\n")
}

func (d draft) starterDetails(width int) string {
	if d.starterName == "" {
		starters := screen.Section{Heading: "Starters", None: "none"}
		for _, starterName := range slices.Sorted(maps.Keys(d.definitions.Starters)) {
			starters.Entries = append(starters.Entries, screen.Entry{Name: starterName, Summary: d.definitions.Starters[starterName].Summary})
		}
		return screen.Details("No starter is chosen.", width, starters)
	}
	source := "answered"
	if d.starterGiven {
		source = "command line"
	}
	return d.withSource(screen.Details(d.definitions.Starters[d.starterName].Summary, width), source, width)
}

func (d draft) featureDetails(featureName string, width int) string {
	feature := d.definitions.Features[featureName]
	needs := screen.Section{Heading: "Needs", None: "none"}
	for _, need := range feature.Needs {
		needs.Entries = append(needs.Entries, screen.Entry{Name: need})
	}
	return d.withSource(screen.Details(feature.Summary, width, needs), d.featureSource(featureName), width)
}

func (d draft) featureSource(featureName string) string {
	var sources []string
	if _, isGiven := d.givenByFeature[featureName]; isGiven {
		sources = append(sources, "command line")
	}
	if neededBy := d.selection.NeededBy(featureName); len(neededBy) > 0 {
		sources = append(sources, "needed by "+strings.Join(neededBy, ", "))
	}
	if len(sources) == 0 {
		return "not selected"
	}
	return strings.Join(sources, ", ")
}

func (d draft) withSource(details, source string, width int) string {
	lines := screen.Entry{Fields: []screen.Field{{Label: "source", Values: []string{source}}}}.Lines(width)
	if details == "" {
		return strings.Join(lines, "\n")
	}
	return details + "\n\n" + strings.Join(lines, "\n")
}
