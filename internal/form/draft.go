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
	held, err := starter.Select(d.name, d.given, d.givenByFeature, catalog.Answers{Features: answers.Features})
	if err != nil {
		return answers, catalog.Selection{}, err
	}
	answers.FeatureValues = maps.Clone(answers.FeatureValues)
	maps.DeleteFunc(answers.FeatureValues, func(answeredFeature string, _ map[string]string) bool { return !held.Holds(answeredFeature) })
	selection, err := starter.Select(d.name, d.given, d.givenByFeature, answers)
	return answers, selection, err
}

func (d draft) toggled(featureName string) (draft, string) {
	if d.isFixed(row{kind: featureKind, feature: featureName}) {
		return d, featureName + " stays selected: --feature selects it."
	}
	if neededBy := d.selection.NeededBy(featureName); len(neededBy) > 0 {
		return d, featureName + " stays selected: it is needed by " + strings.Join(neededBy, ", ") + "."
	}
	answers := d.answers
	answers.Features = slices.DeleteFunc(slices.Clone(answers.Features), func(answered string) bool { return answered == featureName })
	if len(answers.Features) == len(d.answers.Features) {
		answers.Features = append(answers.Features, featureName)
	}
	answers, selection, err := d.selected(d.starter, answers)
	if err != nil {
		return d, err.Error()
	}
	d.answers, d.selection = answers, selection
	return d, ""
}

func (d draft) withAnswer(r row, value string) (draft, error) {
	name, answers := d.name, d.answers
	switch r.kind {
	case starterKind:
		return d.withKeptStarter(value)
	case nameKind:
		if d.starterName == "" {
			d.name = value
			return d, nil
		}
		name = value
	case optionKind:
		answers = answers.WithValue(r.feature, r.option, value)
	default:
		panic(fmt.Sprintf("the form keeps no answer for the row kind %d", r.kind))
	}
	selection, err := d.starter.Select(name, d.given, d.givenByFeature, answers)
	if err != nil {
		return d, err
	}
	d.name, d.answers, d.selection = name, answers, selection
	return d, nil
}

func (d draft) withKeptStarter(starterName string) (draft, error) {
	if starterName == d.starterName {
		return d, nil
	}
	doesNotTarget := func(featureName string) bool {
		return !slices.Contains(d.definitions.Features[featureName].Targets, starterName)
	}
	kept := d
	kept.answers = catalog.Answers{
		Features:      slices.DeleteFunc(slices.Clone(d.answers.Features), doesNotTarget),
		FeatureValues: d.answers.FeatureValues,
	}
	kept, err := kept.withStarter(starterName)
	if err != nil {
		return d, err
	}
	return kept, nil
}

func (d draft) isFixed(r row) bool {
	switch r.kind {
	case nameKind:
		return d.nameGiven
	case starterKind:
		return d.starterGiven
	case featureKind:
		_, isGiven := d.givenByFeature[r.feature]
		return isGiven
	case optionKind:
		return d.question(r).Fixed()
	}
	panic(fmt.Sprintf("the form has no source for the row kind %d", r.kind))
}

func (d draft) editOf(r row) (edit, string) {
	if message := d.fixedMessage(r); message != "" {
		return edit{}, message
	}
	if r.kind == starterKind && len(d.definitions.Starters) == 0 {
		return edit{}, "The catalog has no starter."
	}
	if choices := d.choices(r); len(choices) > 0 {
		return newListEdit(r, choices, d.summaries(r), d.value(r)), ""
	}
	return newTextEdit(r, d.value(r)), ""
}

func (d draft) fixedMessage(r row) string {
	switch {
	case !d.isFixed(r):
		return ""
	case r.kind == nameKind:
		return "The name is fixed: the command line gives it."
	case r.kind == starterKind:
		return "The starter is fixed: the command line gives it."
	}
	q := d.question(r)
	switch {
	case q.Given && len(q.NeedOf) > 0:
		return "The option " + r.option + " is fixed: the command line and the need of " + strings.Join(q.NeedOf, ", ") + " give it."
	case len(q.NeedOf) > 0:
		return "The option " + r.option + " is fixed: the need of " + strings.Join(q.NeedOf, ", ") + " sets it."
	}
	return "The option " + r.option + " is fixed: the command line gives it."
}

func (d draft) value(r row) string {
	switch r.kind {
	case nameKind:
		return d.name
	case starterKind:
		return d.starterName
	case optionKind:
		return d.question(r).Value
	}
	panic(fmt.Sprintf("the form has no value for the row kind %d", r.kind))
}

func (d draft) choices(r row) []string {
	switch r.kind {
	case nameKind:
		return nil
	case starterKind:
		return slices.Sorted(maps.Keys(d.definitions.Starters))
	case optionKind:
		return d.question(r).Accepted
	}
	panic(fmt.Sprintf("the form has no choices for the row kind %d", r.kind))
}

func (d draft) summaries(r row) []string {
	var summaries []string
	if r.kind == starterKind {
		for _, starterName := range d.choices(r) {
			summaries = append(summaries, d.definitions.Starters[starterName].Summary)
		}
	}
	return summaries
}

func (d draft) check(r row, value string) error {
	switch r.kind {
	case starterKind:
		return nil
	case nameKind:
		return catalog.CheckRepoName(value)
	case optionKind:
		return d.question(r).Check(value)
	}
	panic(fmt.Sprintf("the form has no check for the row kind %d", r.kind))
}

func (d draft) nodes() []node {
	var nameMarks []string
	switch {
	case d.nameGiven:
		nameMarks = []string{"fixed"}
	case d.name == "":
		nameMarks = []string{missingMark}
	}
	var starterMarks []string
	switch {
	case d.starterGiven:
		starterMarks = []string{"fixed"}
	case d.starterName == "":
		starterMarks = []string{missingMark}
	}
	return []node{
		{row: row{kind: nameKind}, label: "name     ", text: d.name, textStyle: screen.PlainStyle, marks: nameMarks},
		{row: row{kind: starterKind}, label: "starter  ", text: d.starterName, textStyle: screen.NameStyle, marks: starterMarks, children: d.groupNodes()},
	}
}

func (d draft) groupNodes() []node {
	var nodes []node
	if options := d.optionNodes(""); len(options) > 0 {
		nodes = append(nodes, node{row: row{kind: groupKind, group: optionsGroup}, text: string(optionsGroup), textStyle: screen.GroupStyle, children: options})
	}
	if features := d.featureNodes(); len(features) > 0 {
		count := fmt.Sprintf("%d of %d", d.featureCount(), len(d.featureNames))
		nodes = append(nodes, node{row: row{kind: groupKind, group: featuresGroup}, text: string(featuresGroup), textStyle: screen.GroupStyle, marks: []string{count}, children: features})
	}
	return nodes
}

func (d draft) featureNodes() []node {
	nodes := make([]node, 0, len(d.featureNames))
	for _, featureName := range d.featureNames {
		box, style := "[ ] ", screen.NotSelectedStyle
		if d.selection.Holds(featureName) {
			box, style = "[x] ", screen.NameStyle
		}
		var marks []string
		if _, isGiven := d.givenByFeature[featureName]; isGiven {
			marks = append(marks, "fixed")
		}
		if neededBy := d.selection.NeededBy(featureName); len(neededBy) > 0 {
			marks = append(marks, "needed by "+strings.Join(neededBy, ", "))
		}
		nodes = append(nodes, node{row: row{kind: featureKind, feature: featureName}, text: box + featureName, textStyle: style, marks: marks, children: d.optionNodes(featureName)})
	}
	return nodes
}

func (d draft) optionNodes(featureName string) []node {
	questions := d.questionsOf(featureName)
	widest := 0
	for _, q := range questions {
		widest = max(widest, len(q.Option))
	}
	nodes := make([]node, 0, len(questions))
	for _, q := range questions {
		var marks []string
		if q.Fixed() {
			marks = append(marks, "fixed")
		}
		if q.Missing {
			marks = append(marks, missingMark)
		}
		nodes = append(nodes, node{row: row{kind: optionKind, feature: featureName, option: q.Option}, label: fmt.Sprintf("%-*s  ", widest, q.Option), text: q.Value, textStyle: screen.PlainStyle, marks: marks})
	}
	return nodes
}

func (d draft) questionsOf(featureName string) []catalog.Question {
	return slices.DeleteFunc(d.selection.Questions(), func(q catalog.Question) bool { return q.Feature != featureName })
}

func (d draft) question(r row) catalog.Question {
	for _, q := range d.questionsOf(r.feature) {
		if q.Option == r.option {
			return q
		}
	}
	return catalog.Question{}
}

func (d draft) missingAnswer() (row, string, bool) {
	switch {
	case d.name == "":
		return row{kind: nameKind}, "", true
	case d.starterName == "":
		return row{kind: starterKind}, "", true
	}
	for _, q := range d.selection.Questions() {
		if q.Missing {
			return row{kind: optionKind, feature: q.Feature, option: q.Option}, q.Check("").Error(), true
		}
	}
	return row{}, "", false
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
	case groupKind:
		return string(r.group), d.groupDetails(r.group, width)
	case featureKind:
		return r.feature, d.featureDetails(r.feature, width)
	case optionKind:
		return cmp.Or(r.feature, "starter") + " › " + r.option, d.optionDetails(r, width)
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

func (d draft) groupDetails(g group, width int) string {
	if g == optionsGroup {
		var fields []screen.Field
		for _, q := range d.questionsOf("") {
			fields = append(fields, screen.Field{Label: q.Option, Values: []string{q.Value}})
		}
		return strings.Join(screen.Entry{Fields: fields}.Lines(width), "\n")
	}
	selected := screen.Section{Heading: "Selected", None: "none"}
	notSelected := screen.Section{Heading: "Not selected", None: "none"}
	for _, featureName := range d.featureNames {
		entry := screen.Entry{Name: featureName, Summary: d.definitions.Features[featureName].Summary}
		if d.selection.Holds(featureName) {
			selected.Entries = append(selected.Entries, entry)
		} else {
			notSelected.Entries = append(notSelected.Entries, entry)
		}
	}
	return screen.Details("", width, selected, notSelected)
}

func (d draft) featureDetails(featureName string, width int) string {
	feature := d.definitions.Features[featureName]
	needs := screen.ValuesSection("Needs", "none", feature.Needs)
	return d.withSource(screen.Details(feature.Summary, width, needs), d.featureSource(featureName), width)
}

func (d draft) optionDetails(r row, width int) string {
	q := d.question(r)
	fields := []screen.Field{{Label: "value", Values: []string{q.Value}}, {Label: "default", Values: []string{q.Default}}}
	if len(q.Accepted) > 0 {
		fields = append(fields, screen.Field{Label: "values", Values: []string{strings.Join(q.Accepted, ", ")}})
	}
	if q.Pattern != "" {
		fields = append(fields, screen.Field{Label: "pattern", Values: []string{q.Pattern}})
	}
	fields = append(fields, screen.Field{Label: "source", Values: []string{q.Source}})
	return strings.Join(screen.Entry{Fields: fields}.Lines(width), "\n")
}

func (d draft) featureSource(featureName string) string {
	return cmp.Or(d.selection.FeatureSource(featureName), "not selected")
}

func (d draft) withSource(details, source string, width int) string {
	lines := screen.Entry{Fields: []screen.Field{{Label: "source", Values: []string{source}}}}.Lines(width)
	if details == "" {
		return strings.Join(lines, "\n")
	}
	return details + "\n\n" + strings.Join(lines, "\n")
}
