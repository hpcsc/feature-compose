package catalog

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
)

type feature struct {
	name              string
	targets           []string
	templates         []template
	options           options
	extensionPoints   map[string]extensionPoint
	addTo             map[string][]item
	needs             []string
	needValues        map[string]map[string]string
	whenParseMistakes []string
}

func readFeature(fsys fs.FS, featureName string) (feature, error) {
	parsed, err := readFeatureYML(fsys, featureName)
	if err != nil {
		return feature{}, err
	}
	return newFeature(fsys, featureName, parsed)
}

func readFeatureFor(fsys fs.FS, featureName, starterName string) (feature, bool, error) {
	parsed, err := readFeatureYML(fsys, featureName)
	if err != nil {
		return feature{}, false, err
	}
	if !slices.Contains(parsed.Targets, starterName) {
		return feature{}, false, nil
	}
	f, err := newFeature(fsys, featureName, parsed)
	if err != nil {
		return feature{}, false, err
	}
	if len(f.whenParseMistakes) > 0 {
		return feature{}, false, fmt.Errorf("read %s: %s", featureYMLPath(featureName), f.whenParseMistakes[0])
	}
	return f, true, nil
}

func newFeature(fsys fs.FS, featureName string, parsed featureYML) (feature, error) {
	root := path.Join("features", featureName)
	ymlPath := featureYMLPath(featureName)
	var rules []delimitersRule
	for _, declared := range parsed.Delimiters {
		rule, err := newDelimitersRule(declared.Glob, delimiters{left: declared.Left, right: declared.Right})
		if err != nil {
			return feature{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
		rules = append(rules, rule)
	}
	var templates []template
	entryRoots := map[string]bool{}
	var whenParseMistakes []string
	for _, entry := range parsed.Files {
		entryRoot := path.Join(root, entry.Path)
		if !strings.HasPrefix(entryRoot, root+"/") {
			return feature{}, fmt.Errorf("read %s: the files entry %q is outside the folder of the feature", ymlPath, entry.Path)
		}
		if entryRoots[entryRoot] {
			return feature{}, fmt.Errorf("read %s: files names the path %q two times", ymlPath, entry.Path)
		}
		entryRoots[entryRoot] = true
		read, err := readTemplates(fsys, entryRoot, featureName, rules)
		if err != nil {
			return feature{}, err
		}
		var when expression
		if entry.When != "" {
			when, err = newExpression(entry.When)
			if err != nil {
				whenParseMistakes = append(whenParseMistakes, fmt.Sprintf("the when %q of the files entry %q: %s", entry.When, entry.Path, err))
			}
		}
		for i := range read {
			read[i].entry, read[i].when = entry.Path, when
		}
		templates = append(templates, read...)
	}
	declaredOptions := options{declarer: "the feature " + featureName, declared: make(map[string]option, len(parsed.Options))}
	for _, optionName := range slices.Sorted(maps.Keys(parsed.Options)) {
		declared := parsed.Options[optionName]
		read, err := newOption(optionName, declared.Default, declared.Pattern, declared.Values)
		if err != nil {
			return feature{}, fmt.Errorf("read %s: %w", ymlPath, err)
		}
		if declared.When != "" {
			read.when, err = newExpression(declared.When)
			if err != nil {
				whenParseMistakes = append(whenParseMistakes, fmt.Sprintf("the when %q of the option %q: %s", declared.When, optionName, err))
			}
		}
		declaredOptions.declared[optionName] = read
	}
	addTo := make(map[string][]item, len(parsed.AddTo))
	for _, name := range slices.Sorted(maps.Keys(parsed.AddTo)) {
		for i, declared := range parsed.AddTo[name] {
			added, err := newItem(featureName, declared)
			if err != nil {
				whenParseMistakes = append(whenParseMistakes, fmt.Sprintf("the when %q of the item %d of the extension point %q: %s", declared[whenKey], i+1, name, err))
			}
			addTo[name] = append(addTo[name], added)
		}
	}
	extensionPoints := make(map[string]extensionPoint, len(parsed.ExtensionPoints))
	for name, declared := range parsed.ExtensionPoints {
		extensionPoints[name] = newExtensionPoint(name, "the feature "+featureName, declared)
	}
	return feature{
		name: featureName, targets: parsed.Targets, templates: templates, options: declaredOptions,
		extensionPoints: extensionPoints, addTo: addTo, needs: parsed.needNames, needValues: parsed.Needs,
		whenParseMistakes: whenParseMistakes,
	}, nil
}

func (f feature) placedIn(folders map[string]folder) ([]template, error) {
	placed := make([]template, 0, len(f.templates))
	for _, t := range f.templates {
		inFolder, err := t.place(folders)
		if err != nil {
			return nil, err
		}
		placed = append(placed, inFolder)
	}
	return placed, nil
}

func (f feature) targetsStarter(starterName string) bool {
	return slices.Contains(f.targets, starterName)
}

func (f feature) needMistakes(starterName string, catalogFeatures map[string]feature) []string {
	var mistakes []string
	for _, needName := range f.needs {
		needed, ok := catalogFeatures[needName]
		switch {
		case !ok:
			mistakes = append(mistakes, fmt.Sprintf("the feature %s needs %s, which is not a feature of the catalog", f.name, needName))
		case !needed.targetsStarter(starterName):
			mistakes = append(mistakes, fmt.Sprintf("the feature %s needs %s, which does not target the starter", f.name, needName))
		}
	}
	for _, needName := range slices.Sorted(maps.Keys(f.needValues)) {
		needed, ok := catalogFeatures[needName]
		if !ok {
			continue
		}
		values := f.needValues[needName]
		for _, optionName := range slices.Sorted(maps.Keys(values)) {
			if err := needed.options.check(map[string]string{optionName: values[optionName]}); err != nil {
				mistakes = append(mistakes, fmt.Sprintf("the need of the feature %s: %s", f.name, err))
			}
		}
	}
	return mistakes
}

func (f feature) mistakes(starters []Starter, catalogFeatures map[string]feature) []string {
	ofFeature := slices.Concat(f.whenParseMistakes, f.readMistakes(starters, catalogFeatures))
	for _, name := range slices.Sorted(maps.Keys(f.extensionPoints)) {
		ofFeature = append(ofFeature, f.extensionPoints[name].whenFieldMistakes()...)
		if !f.extensionPoints[name].inGeneratedFile() {
			ofFeature = append(ofFeature, f.extensionPoints[name].fileMistakes(f.templates)...)
		}
	}
	mistakes := f.itemFieldsMistakes(starters)
	for _, mistake := range ofFeature {
		mistakes = append(mistakes, "the feature "+f.name+": "+mistake)
	}
	return mistakes
}

func (f feature) readMistakes(starters []Starter, catalogFeatures map[string]feature) []string {
	readable := f.readable(starters, catalogFeatures)
	checkedEntries := map[string]bool{}
	var entryWhenMistakes, templateReads, parseMistakes []string
	for _, t := range f.templates {
		if !checkedEntries[t.entry] {
			checkedEntries[t.entry] = true
			entryWhenMistakes = append(entryWhenMistakes, f.whenReadMistakes(t.when, fmt.Sprintf("the files entry %q", t.entry), starters, readable)...)
		}
		reads, err := t.readMistakes(readable, func(reference string) string { return f.whyCannotRead(reference, readable, starters, "the feature") })
		if err != nil {
			parseMistakes = append(parseMistakes, err.Error())
		}
		templateReads = append(templateReads, reads...)
	}
	return slices.Concat(entryWhenMistakes, f.optionWhenReadMistakes(starters, readable), f.itemWhenReadMistakes(starters, readable), templateReads, parseMistakes)
}

func (f feature) itemWhenReadMistakes(starters []Starter, readable map[string]bool) []string {
	var mistakes []string
	for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
		for i, added := range f.addTo[name] {
			mistakes = append(mistakes, f.whenReadMistakes(added.when, fmt.Sprintf("the item %d of the extension point %q", i+1, name), starters, readable)...)
		}
	}
	return mistakes
}

func (f feature) whenReadMistakes(when expression, subject string, starters []Starter, readable map[string]bool) []string {
	var mistakes []string
	for _, reference := range when.references() {
		if !readable[reference] {
			mistakes = append(mistakes, fmt.Sprintf("the when %q of %s reads %q, which %s",
				when.text, subject, reference, f.whyCannotRead(reference, readable, starters, "the feature")))
		}
	}
	return mistakes
}

func (f feature) optionWhenReadMistakes(starters []Starter, readable map[string]bool) []string {
	readableByWhen := maps.Clone(readable)
	for optionName, declared := range f.options.declared {
		if declared.hasWhen() {
			delete(readableByWhen, optionName)
		}
	}
	var mistakes []string
	for _, optionName := range slices.Sorted(maps.Keys(f.options.declared)) {
		when := f.options.declared[optionName].when
		for _, reference := range when.references() {
			switch {
			case readableByWhen[reference]:
			case f.options.declared[reference].hasWhen():
				mistakes = append(mistakes, fmt.Sprintf("the when %q of the option %q reads %q, which has a when of its own, so the when of an option cannot read it",
					when.text, optionName, reference))
			default:
				mistakes = append(mistakes, fmt.Sprintf("the when %q of the option %q reads %q, which %s",
					when.text, optionName, reference, f.whyCannotRead(reference, readableByWhen, starters, "the when of an option")))
			}
		}
	}
	return mistakes
}

func (f feature) readable(starters []Starter, catalogFeatures map[string]feature) map[string]bool {
	readable := f.options.references("")
	for _, needName := range f.needs {
		maps.Copy(readable, catalogFeatures[needName].options.references(needName+"."))
	}
	for _, s := range starters {
		if !f.targetsStarter(s.name) {
			continue
		}
		for optionName := range s.options.declared {
			if len(f.targetsThatDoNotDeclare(optionName, starters)) == 0 {
				readable[starterPrefix+optionName] = true
			}
		}
	}
	return readable
}

func (f feature) whyCannotRead(reference string, readable map[string]bool, starters []Starter, reader string) string {
	if optionName, isStarterOption := strings.CutPrefix(reference, starterPrefix); isStarterOption {
		switch notDeclaring := f.targetsThatDoNotDeclare(optionName, starters); len(notDeclaring) {
		case 0:
		case 1:
			return "the starter " + notDeclaring[0] + " does not declare"
		default:
			return "the starters " + strings.Join(notDeclaring, ", ") + " do not declare"
		}
	}
	if len(readable) == 0 {
		return reader + " cannot read: it can read no option"
	}
	return reader + " cannot read: it can read " + strings.Join(slices.Sorted(maps.Keys(readable)), ", ")
}

func (f feature) targetsThatDoNotDeclare(optionName string, starters []Starter) []string {
	var notDeclaring []string
	for _, s := range starters {
		if _, declared := s.options.declared[optionName]; f.targetsStarter(s.name) && !declared {
			notDeclaring = append(notDeclaring, s.name)
		}
	}
	return notDeclaring
}

func (f feature) itemFieldsMistakes(starters []Starter) []string {
	declaredByTarget := map[string]map[string]extensionPoint{}
	for _, s := range starters {
		if f.targetsStarter(s.name) {
			declaredByTarget[s.name] = s.extensionPointsDeclaredFor(f)
		}
	}
	var mistakes []string
	for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
		var declarations []string
		var first extensionPoint
		differ := false
		for _, s := range starters {
			declared, ok := declaredByTarget[s.name][name]
			if !ok {
				continue
			}
			if declarations == nil {
				first = declared
			}
			differ = differ || !declared.hasItemFields(first.itemFields)
			declarations = append(declarations, fmt.Sprintf("[%s] in the starter %s", strings.Join(declared.itemFields, ", "), s.name))
		}
		if differ {
			mistakes = append(mistakes, fmt.Sprintf("the feature %s adds to the extension point %q, which its targets declare with different item fields: %s", f.name, name, strings.Join(declarations, ", ")))
		}
	}
	return mistakes
}

func (f feature) addsToOneExtensionPointWith(other feature) bool {
	for _, name := range slices.Sorted(maps.Keys(f.addTo)) {
		if _, ok := other.addTo[name]; ok {
			return true
		}
	}
	return false
}
