package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/astgrep"
)

var repoName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Starter struct {
	name              string
	options           options
	folders           folders
	templates         []template
	features          map[string]feature
	otherFeatureNames []string
	extensionPoints   map[string]extensionPoint
	removals          []removal
}

func readStarter(fsys fs.FS, starterName string, features map[string]feature, otherFeatureNames []string) (Starter, error) {
	root := path.Join("starters", starterName)
	ymlPath := starterYMLPath(starterName)
	parsed, err := readStarterYML(fsys, starterName)
	if err != nil {
		return Starter{}, err
	}
	declaredOptions := options{declarer: "the starter " + starterName, declared: make(map[string]option, len(parsed.Options))}
	for _, optionName := range slices.Sorted(maps.Keys(parsed.Options)) {
		declared := parsed.Options[optionName]
		if declared.When != "" {
			return Starter{}, fmt.Errorf("read %s: the option %s has a when, and only an option of a feature can have a when", ymlPath, optionName)
		}
		declaredOptions.declared[optionName], err = newOption(optionName, declared.Default, declared.Pattern, declared.Values)
		if err != nil {
			return Starter{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
	}
	starterFolders := make(folders, len(parsed.Folders))
	for folderName, declared := range parsed.Folders {
		if !filepath.IsLocal(filepath.FromSlash(declared.Path)) {
			return Starter{}, fmt.Errorf("read %s: the path %q of the folder %s is outside the repo", ymlPath, declared.Path, folderName)
		}
		starterFolders[folderName] = folder{name: folderName, path: declared.Path, generate: declared.Generate, afterRender: declared.AfterRender, test: declared.Test}
	}
	var rules []delimitersRule
	for _, declared := range parsed.Delimiters {
		rule, err := newDelimitersRule(declared.Glob, delimiters{left: declared.Left, right: declared.Right})
		if err != nil {
			return Starter{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
		rules = append(rules, rule)
	}
	read, err := readTemplates(fsys, path.Join(root, "files"), "", rules)
	if err != nil && !(starterFolders.hasGenerator() && errors.Is(err, fs.ErrNotExist)) {
		return Starter{}, err
	}
	templates := make([]template, 0, len(read))
	for _, t := range read {
		placed, err := t.place(starterFolders)
		if err != nil {
			return Starter{}, err
		}
		templates = append(templates, placed)
	}
	extensionPoints := make(map[string]extensionPoint, len(parsed.ExtensionPoints))
	for name, declared := range parsed.ExtensionPoints {
		extensionPoints[name] = newExtensionPoint(name, "the starter "+starterName, declared)
	}
	for _, name := range slices.Sorted(maps.Keys(parsed.AddTo)) {
		declared, ok := extensionPoints[name]
		if !ok {
			return Starter{}, fmt.Errorf("read %s: the starter adds to the extension point %q, which it does not declare", ymlPath, name)
		}
		items := make([]item, 0, len(parsed.AddTo[name]))
		for _, fields := range parsed.AddTo[name] {
			added, err := newItem("", fields)
			if err != nil {
				return Starter{}, fmt.Errorf("read %s: an item of the extension point %q: %w", ymlPath, name, err)
			}
			items = append(items, added)
		}
		withItems, err := declared.add(items)
		if err != nil {
			return Starter{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
		extensionPoints[name] = withItems
	}
	removals := make([]removal, 0, len(parsed.Remove))
	for _, declared := range parsed.Remove {
		removals = append(removals, newRemoval(declared))
	}
	return Starter{
		name: starterName, options: declaredOptions, folders: starterFolders, templates: templates, features: features,
		otherFeatureNames: otherFeatureNames, extensionPoints: extensionPoints, removals: removals,
	}, nil
}

func (s Starter) Select(name string, given map[string]string, givenByFeature map[string]map[string]string, answers Answers) (Selection, error) {
	if name != "" {
		if err := CheckRepoName(name); err != nil {
			return Selection{}, err
		}
	}
	starterValues, err := s.options.withDefaults(given, valuesSetByNeeds{}, answers.StarterValues, name, map[string]string{})
	if err != nil {
		return Selection{}, err
	}
	selected, err := s.featuresWithNeeds(givenByFeature, answers.Features)
	if err != nil {
		return Selection{}, err
	}
	if err := s.checkAnsweredFeatures(selected, answers.FeatureValues); err != nil {
		return Selection{}, err
	}
	setByNeeds, err := s.setByNeeds(selected)
	if err != nil {
		return Selection{}, err
	}
	selection := Selection{folders: s.folders, removals: s.removals, name: name, starter: selectedStarter{name: s.name, options: s.options, values: starterValues}}
	for _, f := range selected {
		values, err := f.options.withDefaults(givenByFeature[f.name], setByNeeds[f.name], answers.FeatureValues[f.name], name, selection.readableOfNeedsAndStarter(f.needs))
		if err != nil {
			return Selection{}, err
		}
		_, isGiven := givenByFeature[f.name]
		selection.features = append(selection.features, selectedFeature{
			name:     f.name,
			given:    isGiven,
			answered: slices.Contains(answers.Features, f.name),
			options:  f.options,
			values:   values,
			needs:    f.needs,
			neededBy: s.neededBy(f.name, selected),
		})
	}
	templates, err := s.templatesOf(selected)
	if err != nil {
		return Selection{}, err
	}
	extensionPoints, err := s.extensionPointsOf(selected)
	if err != nil {
		return Selection{}, err
	}
	selection.templates, selection.extensionPoints = templates, extensionPoints
	return selection, nil
}

func (s Starter) FeatureNames() ([]string, error) {
	ordered, err := s.inNeedsOrder(s.features)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(ordered))
	for _, f := range ordered {
		names = append(names, f.name)
	}
	return names, nil
}

func (s Starter) combinations() []Combination {
	starterDefaults := s.options.defaultsOfOptionsWithValues()
	combinations := []Combination{{StarterName: s.name, StarterValues: starterDefaults}}
	for _, values := range s.options.otherValuesOfOptionsWithValues(nil) {
		combinations = append(combinations, Combination{StarterName: s.name, StarterValues: values})
	}
	withFeatures := func(featureValues map[string]map[string]string) Combination {
		return Combination{StarterName: s.name, StarterValues: maps.Clone(starterDefaults), FeatureValues: featureValues}
	}
	featureNames := slices.Sorted(maps.Keys(s.features))
	for _, featureName := range featureNames {
		f := s.features[featureName]
		featureDefaults := f.options.defaultsOfOptionsWithValues()
		combinations = append(combinations, withFeatures(map[string]map[string]string{featureName: featureDefaults}))
		for _, values := range f.options.otherValuesOfOptionsWithValues(nil) {
			combinations = append(combinations, withFeatures(map[string]map[string]string{featureName: values}))
		}
		for _, needName := range slices.Sorted(slices.Values(f.needs)) {
			setByNeeds, err := s.setByNeeds(s.withAllNeeds(featureName))
			if err != nil {
				continue
			}
			for _, values := range s.features[needName].options.otherValuesOfOptionsWithValues(setByNeeds[needName].byOption) {
				combinations = append(combinations, withFeatures(map[string]map[string]string{featureName: maps.Clone(featureDefaults), needName: values}))
			}
		}
	}
	for i, first := range featureNames {
		for _, second := range featureNames[i+1:] {
			if s.features[first].addsToOneExtensionPointWith(s.features[second]) && !s.oneNeedsTheOther(first, second) {
				combinations = append(combinations, withFeatures(map[string]map[string]string{
					first:  s.features[first].options.defaultsOfOptionsWithValues(),
					second: s.features[second].options.defaultsOfOptionsWithValues(),
				}))
			}
		}
	}
	return combinations
}

func (s Starter) mistakes(ctx context.Context, catalogFeatures map[string]feature) []string {
	var mistakes []string
	for _, featureName := range slices.Sorted(maps.Keys(s.features)) {
		f := s.features[featureName]
		mistakes = append(mistakes, s.addToMistakes(f)...)
		mistakes = append(mistakes, s.folderMistakes(f)...)
		mistakes = append(mistakes, f.needMistakes(s.name, catalogFeatures)...)
		for _, name := range slices.Sorted(maps.Keys(f.extensionPoints)) {
			if e := f.extensionPoints[name]; e.inGeneratedFile() {
				for _, mistake := range e.generatedFileMistakes(s.folders) {
					mistakes = append(mistakes, "the feature "+f.name+": "+mistake)
				}
			}
		}
	}
	mistakes = append(mistakes, s.cycleMistakes()...)
	mistakes = append(mistakes, s.templateMistakes()...)
	for _, name := range slices.Sorted(maps.Keys(s.extensionPoints)) {
		mistakes = append(mistakes, s.extensionPoints[name].whenFieldMistakes()...)
		if s.extensionPoints[name].inGeneratedFile() {
			mistakes = append(mistakes, s.extensionPoints[name].generatedFileMistakes(s.folders)...)
			continue
		}
		mistakes = append(mistakes, s.extensionPoints[name].fileMistakes(s.templates)...)
	}
	mistakes = append(mistakes, s.removalMistakes()...)
	mistakes = append(mistakes, s.queryMistakes(ctx)...)
	mistakes = append(mistakes, s.generatedTemplateMistakes(catalogFeatures)...)
	mistakes = append(mistakes, s.ownerMistakes()...)
	for i, mistake := range mistakes {
		mistakes[i] = "the starter " + s.name + ": " + mistake
	}
	return mistakes
}

func (s Starter) removalMistakes() []string {
	var mistakes []string
	for i, r := range s.removals {
		for _, mistake := range r.mistakes(s.folders) {
			mistakes = append(mistakes, fmt.Sprintf("the removal %d %s", i+1, mistake))
		}
	}
	return mistakes
}

func (s Starter) generatedTemplateMistakes(catalogFeatures map[string]feature) []string {
	var mistakes []string
	check := func(subject, text, reader string, readable map[string]bool) {
		parsed, err := calls{}.with(subject, text, defaultDelimiters)
		if err != nil {
			mistakes = append(mistakes, fmt.Sprintf("%s does not parse: %s", subject, err))
			return
		}
		for _, reference := range parsed.argumentsOf("option") {
			if !readable[reference] {
				mistakes = append(mistakes, fmt.Sprintf("%s reads %q, which %s cannot read", subject, reference, reader))
			}
		}
	}
	readable := s.options.references(starterPrefix)
	for _, name := range slices.Sorted(maps.Keys(s.extensionPoints)) {
		e := s.extensionPoints[name]
		if !e.inGeneratedFile() {
			continue
		}
		check(fmt.Sprintf("the file of the extension point %q", name), e.file, "the starter", readable)
		check(fmt.Sprintf("the insert of the extension point %q", name), e.generatedFile.insert, "the starter", readable)
		for i, added := range e.items {
			for _, field := range slices.Sorted(maps.Keys(added.fields)) {
				check(fmt.Sprintf("the field %s of the item %d that the starter adds to the extension point %q", field, i+1, name), added.fields[field], "the starter", readable)
			}
		}
	}
	for i, r := range s.removals {
		check(fmt.Sprintf("the file of the removal %d", i+1), r.file, "the starter", readable)
	}
	for _, featureName := range slices.Sorted(maps.Keys(s.features)) {
		f := s.features[featureName]
		declared := s.extensionPointsDeclaredFor(f)
		featureReadable := f.readable([]Starter{s}, catalogFeatures)
		for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
			if e, ok := declared[name]; !ok || !e.inGeneratedFile() {
				continue
			}
			for i, added := range f.addTo[name] {
				for _, field := range slices.Sorted(maps.Keys(added.fields)) {
					check(fmt.Sprintf("the field %s of the item %d that the feature %s adds to the extension point %q", field, i+1, featureName, name), added.fields[field], "the feature "+featureName, featureReadable)
				}
			}
		}
	}
	return mistakes
}

func (s Starter) queryMistakes(ctx context.Context) []string {
	type labeledQuery struct {
		label string
		query astgrep.Query
	}
	var queries []labeledQuery
	for _, name := range slices.Sorted(maps.Keys(s.extensionPoints)) {
		if e := s.extensionPoints[name]; e.inGeneratedFile() && e.generatedFile.query.Language != "" {
			queries = append(queries, labeledQuery{fmt.Sprintf("the extension point %q", name), e.generatedFile.query})
		}
	}
	for _, featureName := range slices.Sorted(maps.Keys(s.features)) {
		extensionPoints := s.features[featureName].extensionPoints
		for _, name := range slices.Sorted(maps.Keys(extensionPoints)) {
			if e := extensionPoints[name]; e.inGeneratedFile() && e.generatedFile.query.Language != "" {
				queries = append(queries, labeledQuery{fmt.Sprintf("the extension point %q of the feature %s", name, featureName), e.generatedFile.query})
			}
		}
	}
	for i, r := range s.removals {
		if !r.query.Empty() && r.query.Language != "" {
			queries = append(queries, labeledQuery{fmt.Sprintf("the removal %d", i+1), r.query})
		}
	}
	if len(queries) == 0 {
		return nil
	}
	if err := astgrep.Installed(); err != nil {
		return []string{"check cannot read the patterns and the rules: " + err.Error()}
	}
	sample, err := s.Select("check", nil, nil, Answers{})
	if err != nil {
		return []string{"check cannot render the patterns and the rules with the default values of the options: " + err.Error()}
	}
	var mistakes []string
	for _, q := range queries {
		rendered, err := sample.renderedQuery(q.query, q.label)
		if err != nil {
			mistakes = append(mistakes, err.Error())
			continue
		}
		if err := astgrep.Check(ctx, rendered); err != nil {
			mistakes = append(mistakes, q.label+": "+err.Error())
		}
	}
	return mistakes
}

func (s Starter) addToMistakes(f feature) []string {
	var mistakes []string
	declared := s.extensionPointsDeclaredFor(f)
	for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
		extensionPoint, ok := declared[name]
		if !ok {
			mistakes = append(mistakes, fmt.Sprintf("the feature %s adds to the extension point %q, which the starter, the feature and its needs do not declare", f.name, name))
			continue
		}
		for i, added := range f.addTo[name] {
			if err := extensionPoint.checkItem(i+1, added); err != nil {
				mistakes = append(mistakes, err.Error())
			}
		}
	}
	return mistakes
}

func (s Starter) folderMistakes(f feature) []string {
	var mistakes []string
	missing := map[string]bool{}
	for _, t := range f.templates {
		folderName, inFolder := t.folderIn(s.folders)
		if inFolder {
			continue
		}
		switch {
		case folderName == "":
			mistakes = append(mistakes, t.description()+" is not in a folder")
		case !missing[folderName]:
			missing[folderName] = true
			mistakes = append(mistakes, fmt.Sprintf("the feature %s adds files to the folder %s, which the starter does not have", f.name, folderName))
		}
	}
	return mistakes
}

func (s Starter) templateMistakes() []string {
	readable := s.options.references(starterPrefix)
	canRead := "it can read no option"
	if len(readable) > 0 {
		canRead = "it can read " + strings.Join(slices.Sorted(maps.Keys(readable)), ", ")
	}
	var reads, parseMistakes []string
	for _, t := range s.templates {
		templateReads, err := t.readMistakes(readable, func(string) string { return "the starter cannot read: " + canRead })
		if err != nil {
			parseMistakes = append(parseMistakes, err.Error())
		}
		reads = append(reads, templateReads...)
	}
	return append(reads, parseMistakes...)
}

func (s Starter) ownerMistakes() []string {
	templates := slices.Clone(s.templates)
	for _, featureName := range slices.Sorted(maps.Keys(s.features)) {
		for _, t := range s.features[featureName].templates {
			if placed, err := t.place(s.folders); err == nil {
				templates = append(templates, placed)
			}
		}
	}
	owners := map[string][]string{}
	for _, t := range templates {
		if !slices.Contains(owners[t.pathBeforeRender()], t.owner) {
			owners[t.pathBeforeRender()] = append(owners[t.pathBeforeRender()], t.owner)
		}
	}
	var mistakes []string
	for _, ownedPath := range slices.Sorted(maps.Keys(owners)) {
		pathOwners := owners[ownedPath]
		for i, first := range pathOwners {
			for _, second := range pathOwners[i+1:] {
				mistakes = append(mistakes, fmt.Sprintf("%s and %s both write %s", first, second, ownedPath))
			}
		}
	}
	return mistakes
}

func (s Starter) cycleMistakes() []string {
	var mistakes []string
	inReportedCycle := map[string]bool{}
	for _, featureName := range slices.Sorted(maps.Keys(s.features)) {
		if inReportedCycle[featureName] {
			continue
		}
		cycle := s.pathOfNeeds(featureName, featureName, map[string]bool{})
		if cycle == nil {
			continue
		}
		for _, inCycle := range cycle {
			inReportedCycle[inCycle] = true
		}
		mistakes = append(mistakes, "the needs of the features form a cycle: "+strings.Join(cycle, " needs "))
	}
	return mistakes
}

func (s Starter) pathOfNeeds(from, to string, visited map[string]bool) []string {
	visited[from] = true
	for _, need := range s.features[from].needs {
		if need == to {
			return []string{from, to}
		}
		if visited[need] {
			continue
		}
		if rest := s.pathOfNeeds(need, to, visited); rest != nil {
			return append([]string{from}, rest...)
		}
	}
	return nil
}

func (s Starter) extensionPointsDeclaredFor(f feature) map[string]extensionPoint {
	declared := maps.Clone(s.extensionPoints)
	declarers := append([]string{f.name}, slices.Sorted(maps.Keys(s.allNeedsOf(f.name)))...)
	for _, declarer := range declarers {
		extensionPoints := s.features[declarer].extensionPoints
		for _, name := range slices.Sorted(maps.Keys(extensionPoints)) {
			if _, ok := declared[name]; !ok {
				declared[name] = extensionPoints[name]
			}
		}
	}
	return declared
}

func (s Starter) withAllNeeds(featureName string) []feature {
	features := []feature{s.features[featureName]}
	for needName := range s.allNeedsOf(featureName) {
		features = append(features, s.features[needName])
	}
	return features
}

func (s Starter) oneNeedsTheOther(first, second string) bool {
	return s.allNeedsOf(first)[second] || s.allNeedsOf(second)[first]
}

func (s Starter) allNeedsOf(featureName string) map[string]bool {
	needed := map[string]bool{}
	s.addNeeds(needed, featureName)
	return needed
}

func (s Starter) addNeeds(needed map[string]bool, featureName string) {
	for _, need := range s.features[featureName].needs {
		if !needed[need] {
			needed[need] = true
			s.addNeeds(needed, need)
		}
	}
}

func (s Starter) featuresWithNeeds(givenByFeature map[string]map[string]string, answered []string) ([]feature, error) {
	requested := slices.AppendSeq(slices.Clone(answered), maps.Keys(givenByFeature))
	slices.Sort(requested)
	requested = slices.Compact(requested)
	for _, featureName := range requested {
		if _, ok := s.features[featureName]; !ok {
			return nil, s.unselectableFeature(featureName)
		}
	}
	selected := map[string]feature{}
	for _, featureName := range requested {
		if err := s.selectWithNeeds(selected, s.features[featureName]); err != nil {
			return nil, err
		}
	}
	return s.inNeedsOrder(selected)
}

func (s Starter) selectWithNeeds(selected map[string]feature, f feature) error {
	if _, ok := selected[f.name]; ok {
		return nil
	}
	selected[f.name] = f
	for _, need := range f.needs {
		needed, ok := s.features[need]
		if !ok {
			return fmt.Errorf("the feature %s needs %q, which is not a feature that targets the starter %s: the features that target it are %s",
				f.name, need, s.name, strings.Join(slices.Sorted(maps.Keys(s.features)), ", "))
		}
		if err := s.selectWithNeeds(selected, needed); err != nil {
			return err
		}
	}
	return nil
}

func (s Starter) inNeedsOrder(selected map[string]feature) ([]feature, error) {
	names := slices.Sorted(maps.Keys(selected))
	ordered := make([]feature, 0, len(selected))
	placed := map[string]bool{}
	for len(ordered) < len(selected) {
		next := slices.IndexFunc(names, func(featureName string) bool {
			return !placed[featureName] && !slices.ContainsFunc(selected[featureName].needs, func(need string) bool { return s.isPending(need, selected, placed) })
		})
		if next < 0 {
			return nil, s.needsCycle(selected, placed)
		}
		ordered = append(ordered, selected[names[next]])
		placed[names[next]] = true
	}
	return ordered, nil
}

func (s Starter) needsCycle(selected map[string]feature, placed map[string]bool) error {
	unplaced := slices.DeleteFunc(slices.Sorted(maps.Keys(selected)), func(featureName string) bool { return placed[featureName] })
	cycle := s.cycleFrom([]string{unplaced[0]}, selected, placed)
	return fmt.Errorf("the needs of the features form a cycle: %s", strings.Join(cycle, " needs "))
}

func (s Starter) cycleFrom(path []string, selected map[string]feature, placed map[string]bool) []string {
	needs := selected[path[len(path)-1]].needs
	next := needs[slices.IndexFunc(needs, func(need string) bool { return s.isPending(need, selected, placed) })]
	if start := slices.Index(path, next); start >= 0 {
		return append(path[start:], next)
	}
	return s.cycleFrom(append(path, next), selected, placed)
}

func (s Starter) isPending(need string, selected map[string]feature, placed map[string]bool) bool {
	_, isSelected := selected[need]
	return isSelected && !placed[need]
}

func (s Starter) checkAnsweredFeatures(selected []feature, featureValues map[string]map[string]string) error {
	for _, featureName := range slices.Sorted(maps.Keys(featureValues)) {
		if !slices.ContainsFunc(selected, func(f feature) bool { return f.name == featureName }) {
			return fmt.Errorf("the answers give values to the feature %s, which is not selected", featureName)
		}
	}
	return nil
}

func (s Starter) setByNeeds(selected []feature) (map[string]valuesSetByNeeds, error) {
	setByNeeds := map[string]valuesSetByNeeds{}
	for _, f := range slices.SortedFunc(slices.Values(selected), func(a, b feature) int { return strings.Compare(a.name, b.name) }) {
		for _, needName := range slices.Sorted(maps.Keys(f.needValues)) {
			values := f.needValues[needName]
			if err := s.features[needName].options.check(values); err != nil {
				return nil, fmt.Errorf("the need of %s: %w", f.name, err)
			}
			set, ok := setByNeeds[needName]
			if !ok {
				set = newValuesSetByNeeds(needName)
			}
			added, err := set.add(f.name, values)
			if err != nil {
				return nil, err
			}
			setByNeeds[needName] = added
		}
	}
	return setByNeeds, nil
}

func (s Starter) neededBy(featureName string, selected []feature) []string {
	var neededBy []string
	for _, other := range selected {
		if slices.Contains(other.needs, featureName) {
			neededBy = append(neededBy, other.name)
		}
	}
	slices.Sort(neededBy)
	return neededBy
}

func (s Starter) templatesOf(selected []feature) ([]template, error) {
	templates := slices.Clone(s.templates)
	for _, f := range selected {
		placed, err := f.placedIn(s.folders)
		if err != nil {
			return nil, err
		}
		templates = append(templates, placed...)
	}
	return templates, nil
}

func (s Starter) extensionPointsOf(selected []feature) (map[string]extensionPoint, error) {
	extensionPoints := maps.Clone(s.extensionPoints)
	for _, f := range selected {
		for _, name := range slices.Sorted(maps.Keys(f.extensionPoints)) {
			if other, ok := extensionPoints[name]; ok {
				return nil, fmt.Errorf("%s and %s both declare the extension point %q", other.declarer, f.extensionPoints[name].declarer, name)
			}
			extensionPoints[name] = f.extensionPoints[name]
		}
	}
	for _, f := range selected {
		for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
			declared, ok := extensionPoints[name]
			if !ok {
				return nil, fmt.Errorf("the feature %s adds to the extension point %q, which neither the starter nor a selected feature declares", f.name, name)
			}
			added, err := declared.add(f.addTo[name])
			if err != nil {
				return nil, err
			}
			extensionPoints[name] = added
		}
	}
	return extensionPoints, nil
}

func (s Starter) unselectableFeature(featureName string) error {
	reason := fmt.Sprintf("unknown feature %q", featureName)
	if slices.Contains(s.otherFeatureNames, featureName) {
		reason = fmt.Sprintf("the feature %s does not target the starter %s", featureName, s.name)
	}
	if len(s.features) == 0 {
		return fmt.Errorf("%s: no feature targets the starter %s", reason, s.name)
	}
	return fmt.Errorf("%s: the features that target the starter %s are %s", reason, s.name, strings.Join(slices.Sorted(maps.Keys(s.features)), ", "))
}

func CheckRepoName(name string) error {
	if !repoName.MatchString(name) {
		return fmt.Errorf("the name %q of the repo must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'", name)
	}
	return nil
}
