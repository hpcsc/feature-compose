package catalog

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var repoName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Starter struct {
	name            string
	options         options
	folders         map[string]folder
	templates       []template
	features        map[string]feature
	extensionPoints map[string]extensionPoint
}

func readStarter(fsys fs.FS, starterName string, features map[string]feature) (Starter, error) {
	root := path.Join("starters", starterName)
	ymlPath := starterYMLPath(starterName)
	parsed, err := readStarterYML(fsys, starterName)
	if err != nil {
		return Starter{}, err
	}
	declaredOptions := options{declarer: "the starter " + starterName, declared: make(map[string]option, len(parsed.Options))}
	for optionName, declared := range parsed.Options {
		declaredOptions.declared[optionName], err = newOption(optionName, declared.Default, declared.Pattern, declared.Values)
		if err != nil {
			return Starter{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
	}
	folders := make(map[string]folder, len(parsed.Folders))
	for folderName, declared := range parsed.Folders {
		if !filepath.IsLocal(filepath.FromSlash(declared.Path)) {
			return Starter{}, fmt.Errorf("read %s: the path %q of the folder %s is outside the repo", ymlPath, declared.Path, folderName)
		}
		folders[folderName] = folder{name: folderName, path: declared.Path, afterRender: declared.AfterRender, test: declared.Test}
	}
	var rules []delimitersRule
	for _, declared := range parsed.Delimiters {
		rule, err := newDelimitersRule(declared.Glob, delimiters{left: declared.Left, right: declared.Right})
		if err != nil {
			return Starter{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
		rules = append(rules, rule)
	}
	read, err := readTemplates(fsys, path.Join(root, "files"), "")
	if err != nil {
		return Starter{}, err
	}
	templates := make([]template, 0, len(read))
	for _, t := range read {
		placed, err := t.place(folders, rules)
		if err != nil {
			return Starter{}, err
		}
		templates = append(templates, placed)
	}
	extensionPoints := make(map[string]extensionPoint, len(parsed.ExtensionPoints))
	for name, declared := range parsed.ExtensionPoints {
		extensionPoints[name] = extensionPoint{name: name, declarer: "the starter " + starterName, itemFields: declared.Item}
	}
	return Starter{name: starterName, options: declaredOptions, folders: folders, templates: templates, features: features, extensionPoints: extensionPoints}, nil
}

func (s Starter) Select(name string, given map[string]string, givenByFeature map[string]map[string]string, answers Answers) (Selection, error) {
	if name != "" {
		if err := CheckRepoName(name); err != nil {
			return Selection{}, err
		}
	}
	starterValues, err := s.options.withDefaults(given, answers.StarterValues, name)
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
	features := make([]selectedFeature, 0, len(selected))
	for _, f := range selected {
		values, err := f.options.withDefaults(givenByFeature[f.name], answers.FeatureValues[f.name], name)
		if err != nil {
			return Selection{}, err
		}
		_, isGiven := givenByFeature[f.name]
		features = append(features, selectedFeature{
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
	return Selection{
		folders:         s.folders,
		name:            name,
		starter:         selectedStarter{name: s.name, options: s.options, values: starterValues},
		features:        features,
		templates:       templates,
		extensionPoints: extensionPoints,
	}, nil
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
	for _, values := range s.options.otherValuesOfOptionsWithValues() {
		combinations = append(combinations, Combination{StarterName: s.name, StarterValues: values})
	}
	withFeatures := func(featureValues map[string]map[string]string) Combination {
		return Combination{StarterName: s.name, StarterValues: maps.Clone(starterDefaults), FeatureValues: featureValues}
	}
	featureNames := slices.Sorted(maps.Keys(s.features))
	for _, featureName := range featureNames {
		declared := s.features[featureName].options
		combinations = append(combinations, withFeatures(map[string]map[string]string{featureName: declared.defaultsOfOptionsWithValues()}))
		for _, values := range declared.otherValuesOfOptionsWithValues() {
			combinations = append(combinations, withFeatures(map[string]map[string]string{featureName: values}))
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
			return nil, s.unknownFeature(featureName)
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
			added, err := declared.add(f.name, f.addTo[name])
			if err != nil {
				return nil, err
			}
			extensionPoints[name] = added
		}
	}
	return extensionPoints, nil
}

func (s Starter) unknownFeature(featureName string) error {
	if len(s.features) == 0 {
		return fmt.Errorf("unknown feature %q: no feature targets the starter %s", featureName, s.name)
	}
	return fmt.Errorf("unknown feature %q: the features that target the starter %s are %s",
		featureName, s.name, strings.Join(slices.Sorted(maps.Keys(s.features)), ", "))
}

func CheckRepoName(name string) error {
	if !repoName.MatchString(name) {
		return fmt.Errorf("the name %q of the repo must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'", name)
	}
	return nil
}
