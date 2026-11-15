package form

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/catalog"
)

const (
	fromCommandLine = "from the command line"
	repoLabelWidth  = len("starter") + 2
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
	if _, isGiven := d.givenByFeature[featureName]; isGiven {
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

func (d draft) value(r row) string {
	switch r.kind {
	case nameKind:
		return d.name
	case optionKind:
		return d.question(r).Value
	}
	panic(fmt.Sprintf("the form has no text for the row kind %d", r.kind))
}

func (d draft) check(r row, value string) error {
	switch r.kind {
	case nameKind:
		return catalog.CheckRepoName(value)
	case optionKind:
		return d.question(r).Check(value)
	}
	panic(fmt.Sprintf("the form has no check for the row kind %d", r.kind))
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
		return row{kind: nameKind}, catalog.CheckRepoName("").Error(), true
	case d.starterName == "":
		return row{kind: starterKind}, "No starter is chosen.", true
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

func (d draft) panels() []panel {
	panels := []panel{{title: "Repo", fields: []field{d.nameField(), d.starterField()}}}
	if fields := d.optionFields(""); len(fields) > 0 {
		panels = append(panels, panel{title: "Options of " + d.starterName, fields: fields})
	}
	if len(d.featureNames) > 0 {
		panels = append(panels, panel{title: "Features", count: fmt.Sprintf("%d of %d", d.featureCount(), len(d.featureNames)), fields: []field{d.featuresField()}})
	}
	for _, featureName := range d.featureNames {
		if fields := d.optionFields(featureName); len(fields) > 0 {
			panels = append(panels, panel{title: "Options of " + featureName, fields: fields})
		}
	}
	return panels
}

func (d draft) fields() []field {
	var fields []field
	for _, p := range d.panels() {
		fields = append(fields, p.fields...)
	}
	return append(fields, writeField)
}

func (d draft) nameField() field {
	f := field{row: row{kind: nameKind}, label: padded("name", repoLabelWidth), control: textControl, value: d.name, missing: d.name == ""}
	if d.nameGiven {
		f.control, f.mark = fixedControl, fromCommandLine
	}
	return f
}

func (d draft) starterField() field {
	f := field{row: row{kind: starterKind}, label: padded("starter", repoLabelWidth), value: d.starterName}
	switch {
	case d.starterGiven:
		f.control, f.mark = fixedControl, fromCommandLine
		return f
	case len(d.definitions.Starters) == 0:
		f.control, f.mark = fixedControl, "the catalog has no starter"
		return f
	}
	f.control, f.missing = selectControl, d.starterName == ""
	f.summary = d.definitions.Starters[d.starterName].Summary
	for _, starterName := range slices.Sorted(maps.Keys(d.definitions.Starters)) {
		f.items = append(f.items, item{name: starterName, checked: starterName == d.starterName, summary: d.definitions.Starters[starterName].Summary})
	}
	return f
}

func (d draft) optionFields(featureName string) []field {
	questions := d.questionsOf(featureName)
	optionNames := make([]string, 0, len(questions))
	for _, q := range questions {
		optionNames = append(optionNames, q.Option)
	}
	fields := make([]field, 0, len(questions))
	for _, q := range questions {
		f := field{row: row{kind: optionKind, feature: featureName, option: q.Option}, label: padded(q.Option, widest(optionNames)+2), value: q.Value, missing: q.Missing, hint: hintOf(q)}
		switch {
		case q.Fixed():
			f.control, f.mark = fixedControl, sourceOfFixed(q)
		case len(q.Accepted) > 0:
			f.control = selectControl
			for _, value := range q.Accepted {
				f.items = append(f.items, item{name: value, checked: value == q.Value})
			}
		default:
			f.control = textControl
		}
		fields = append(fields, f)
	}
	return fields
}

func (d draft) featuresField() field {
	f := field{row: row{kind: featuresKind}, control: checklistControl}
	for _, featureName := range d.featureNames {
		var marks []string
		if _, isGiven := d.givenByFeature[featureName]; isGiven {
			marks = append(marks, fromCommandLine)
		}
		if neededBy := d.selection.NeededBy(featureName); len(neededBy) > 0 {
			marks = append(marks, "needed by "+strings.Join(neededBy, ", "))
		}
		feature := d.definitions.Features[featureName]
		choice := item{name: featureName, checked: d.selection.Holds(featureName), mark: strings.Join(marks, ", "), summary: feature.Summary}
		if len(feature.Needs) > 0 {
			choice.hint = "needs " + strings.Join(slices.Sorted(maps.Keys(feature.Needs)), ", ")
		}
		f.items = append(f.items, choice)
	}
	return f
}

func sourceOfFixed(q catalog.Question) string {
	switch {
	case q.Given && len(q.NeedOf) > 0:
		return "from the command line and the need of " + strings.Join(q.NeedOf, ", ")
	case len(q.NeedOf) > 0:
		return "from the need of " + strings.Join(q.NeedOf, ", ")
	}
	return fromCommandLine
}

func hintOf(q catalog.Question) string {
	var parts []string
	if q.Default != "" {
		parts = append(parts, "default "+q.Default)
	}
	if q.Pattern != "" {
		parts = append(parts, "pattern "+q.Pattern)
	}
	return strings.Join(parts, ", ")
}

func widest(texts []string) int {
	widest := 0
	for _, text := range texts {
		widest = max(widest, len(text))
	}
	return widest
}

func padded(text string, width int) string {
	return text + strings.Repeat(" ", max(width-len(text), 0))
}
