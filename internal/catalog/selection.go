package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	texttemplate "text/template"

	"github.com/hpcsc/feature-compose/internal/github"
	"github.com/hpcsc/feature-compose/internal/textcase"
)

var shellWord = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

type Selection struct {
	folders         map[string]folder
	name            string
	starter         selectedStarter
	features        []selectedFeature
	templates       []template
	extensionPoints map[string]extensionPoint
}

func (s Selection) Name() string {
	return s.name
}

func (s Selection) StarterName() string {
	return s.starter.name
}

func (s Selection) AddedByNeeds() []Need {
	var added []Need
	for _, f := range s.features {
		if !f.requested() {
			added = append(added, f.asNeed())
		}
	}
	return added
}

func (s Selection) Holds(featureName string) bool {
	return slices.ContainsFunc(s.features, func(f selectedFeature) bool { return f.name == featureName })
}

func (s Selection) FeatureSource(featureName string) string {
	if !s.Holds(featureName) {
		return ""
	}
	return s.feature(featureName).source()
}

func (s Selection) NeededBy(featureName string) []string {
	return slices.Clone(s.feature(featureName).neededBy)
}

func (s Selection) Write(output string) error {
	files, err := s.files()
	if err != nil {
		return err
	}
	if err := s.checkEmpty(output); err != nil {
		return err
	}
	for _, f := range files {
		if err := f.write(output); err != nil {
			return err
		}
	}
	return nil
}

func (s Selection) Preview() (string, error) {
	files, err := s.files()
	if err != nil {
		return "", err
	}
	var preview strings.Builder
	fmt.Fprintf(&preview, "Repo: %s\nStarter: %s\n", s.name, s.starter.name)
	s.writeSection(&preview, "Features", s.featureLines())
	s.writeSection(&preview, "Options", s.optionLines())
	s.writeSection(&preview, "Files", s.fileLines(files))
	return preview.String(), nil
}

func (s Selection) Questions() []Question {
	questions := s.starter.questions()
	for _, f := range s.features {
		questions = append(questions, f.questions()...)
	}
	return questions
}

func (s Selection) Arguments(output string) string {
	words := []string{s.starter.name, s.name}
	if output != s.name {
		words = append(words, "--output-dir", output)
	}
	words = append(words, flagValues(s.starter.values.notDefault()).optionWords()...)
	for _, f := range s.featuresOfArguments() {
		words = append(words, flagValues(f.values.notDefault()).featureWords(f.name)...)
	}
	quoted := make([]string, 0, len(words))
	for _, word := range words {
		quoted = append(quoted, s.quoted(word))
	}
	return strings.Join(quoted, " ")
}

func (s Selection) RunAfterRender(ctx context.Context, output string) error {
	for _, folderName := range slices.Sorted(maps.Keys(s.folders)) {
		if err := s.folders[folderName].runAfterRender(ctx, output); err != nil {
			return err
		}
	}
	return nil
}

func (s Selection) RunTests(ctx context.Context, output string) error {
	for _, folderName := range slices.Sorted(maps.Keys(s.folders)) {
		if err := s.folders[folderName].runTests(ctx, output); err != nil {
			return err
		}
	}
	return nil
}

func (s Selection) writeSection(preview *strings.Builder, heading string, lines []string) {
	if len(lines) == 0 {
		fmt.Fprintf(preview, "\n%s: none\n", heading)
		return
	}
	fmt.Fprintf(preview, "\n%s:\n", heading)
	for _, line := range lines {
		fmt.Fprintf(preview, "  %s\n", line)
	}
}

func (s Selection) featureLines() []string {
	lines := make([]string, 0, len(s.features))
	for _, f := range s.features {
		lines = append(lines, f.line())
	}
	return lines
}

func (s Selection) optionLines() []string {
	lines := s.starter.optionLines()
	for _, f := range s.features {
		lines = append(lines, f.optionLines()...)
	}
	return lines
}

func (s Selection) featuresOfArguments() []selectedFeature {
	var requested, others []selectedFeature
	for _, f := range s.features {
		switch {
		case f.requested():
			requested = append(requested, f)
		case len(f.values.notDefault()) > 0:
			others = append(others, f)
		}
	}
	slices.SortFunc(requested, func(a, b selectedFeature) int { return strings.Compare(a.name, b.name) })
	return append(requested, others...)
}

func (s Selection) quoted(word string) string {
	if shellWord.MatchString(word) {
		return word
	}
	return "'" + strings.ReplaceAll(word, "'", `'\''`) + "'"
}

func (s Selection) feature(featureName string) selectedFeature {
	for _, f := range s.features {
		if f.name == featureName {
			return f
		}
	}
	return selectedFeature{}
}

func (s Selection) fileLines(files []file) []string {
	lines := make([]string, 0, len(files))
	for _, f := range slices.SortedFunc(slices.Values(files), func(a, b file) int { return strings.Compare(a.path, b.path) }) {
		lines = append(lines, f.path+": "+f.owner)
	}
	return lines
}

func (s Selection) files() ([]file, error) {
	if err := CheckRepoName(s.name); err != nil {
		return nil, err
	}
	if err := s.checkValues(); err != nil {
		return nil, err
	}
	files := make([]file, 0, len(s.templates))
	writers := make(map[string]template, len(s.templates))
	for _, t := range s.templates {
		f, err := t.render(s)
		if err != nil {
			return nil, err
		}
		if other, ok := writers[f.path]; ok {
			return nil, fmt.Errorf("the template %s of %s and the template %s of %s both write %s",
				other.catalogPath, other.owner, t.catalogPath, t.owner, f.path)
		}
		writers[f.path] = t
		files = append(files, f)
	}
	return files, nil
}

func (s Selection) checkValues() error {
	var errs []error
	for _, optionName := range s.starter.values.withoutValue() {
		errs = append(errs, fmt.Errorf("the option %s of the starter %s has no default and no value: give it with --option %s=<value>", optionName, s.starter.name, optionName))
	}
	for _, f := range s.features {
		for _, optionName := range f.values.withoutValue() {
			errs = append(errs, fmt.Errorf("the option %s of the feature %s has no default and no value: give it with --feature %s:%s=<value>", optionName, f.name, f.name, optionName))
		}
	}
	return errors.Join(errs...)
}

func (s Selection) checkEmpty(output string) error {
	entries, err := os.ReadDir(filepath.Clean(output))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the output folder %s: %w", output, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("the output folder %s is not empty", output)
	}
	return nil
}

func (s Selection) render(label, text string, d delimiters, featureName string) (string, error) {
	option := func(reference string) (string, error) { return s.option(featureName, reference) }
	funcs := texttemplate.FuncMap{"option": option, "items": s.items, "unique": s.unique, "githubRepository": github.RepositoryOf, "kebab": textcase.Kebab, "snake": textcase.Snake, "camel": textcase.Camel, "pascal": textcase.Pascal, "upper": strings.ToUpper}
	parsed, err := texttemplate.New(label).Delims(d.left, d.right).Funcs(funcs).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse %w", err)
	}
	var rendered strings.Builder
	if err := parsed.Execute(&rendered, map[string]any{"Name": s.name}); err != nil {
		return "", fmt.Errorf("render %w", err)
	}
	return rendered.String(), nil
}

func (s Selection) option(featureName, reference string) (string, error) {
	readable := s.readable(featureName)
	if value, ok := readable[reference]; ok {
		return value, nil
	}
	if len(readable) == 0 {
		return "", fmt.Errorf("unknown option %q: the template can read no option", reference)
	}
	return "", fmt.Errorf("unknown option %q: the template can read %s", reference, strings.Join(slices.Sorted(maps.Keys(readable)), ", "))
}

func (s Selection) readable(featureName string) map[string]string {
	reader := s.feature(featureName)
	readable := reader.values.byReference("")
	for _, needName := range reader.needs {
		maps.Copy(readable, s.feature(needName).readableAsNeed())
	}
	maps.Copy(readable, s.starter.readable())
	return readable
}

func (s Selection) items(name string) ([]map[string]string, error) {
	declared, err := s.extensionPoint(name)
	if err != nil {
		return nil, err
	}
	return declared.items, nil
}

func (s Selection) unique(field string, names ...string) ([]string, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("unique %q names no extension point", field)
	}
	var unique []string
	for _, name := range names {
		declared, err := s.extensionPoint(name)
		if err != nil {
			return nil, err
		}
		values, err := declared.values(field)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			if !slices.Contains(unique, value) {
				unique = append(unique, value)
			}
		}
	}
	return unique, nil
}

func (s Selection) extensionPoint(name string) (extensionPoint, error) {
	declared, ok := s.extensionPoints[name]
	if !ok {
		if len(s.extensionPoints) == 0 {
			return extensionPoint{}, fmt.Errorf("unknown extension point %q: neither the starter nor a selected feature declares an extension point", name)
		}
		return extensionPoint{}, fmt.Errorf("unknown extension point %q: the starter and the selected features declare %s",
			name, strings.Join(slices.Sorted(maps.Keys(s.extensionPoints)), ", "))
	}
	return declared, nil
}

const starterPrefix = "starter."

type selectedStarter struct {
	name    string
	options options
	values  optionValues
}

func (s selectedStarter) optionLines() []string {
	return s.values.lines(starterPrefix)
}

func (s selectedStarter) readable() map[string]string {
	return s.values.byReference(starterPrefix)
}

func (s selectedStarter) questions() []Question {
	return s.options.questions("", s.values)
}

type selectedFeature struct {
	name     string
	given    bool
	answered bool
	options  options
	values   optionValues
	needs    []string
	neededBy []string
}

func (f selectedFeature) line() string {
	return f.name + ": " + f.source()
}

func (f selectedFeature) source() string {
	switch {
	case !f.requested():
		return f.asNeed().source()
	case len(f.neededBy) > 0:
		return f.requestedBy() + ", " + f.asNeed().source()
	}
	return f.requestedBy()
}

func (f selectedFeature) requested() bool {
	return f.given || f.answered
}

func (f selectedFeature) requestedBy() string {
	if f.given {
		return "command line"
	}
	return "answered"
}

func (f selectedFeature) asNeed() Need {
	return Need{Feature: f.name, NeededBy: slices.Clone(f.neededBy)}
}

func (f selectedFeature) optionLines() []string {
	return f.values.lines(f.prefix())
}

func (f selectedFeature) questions() []Question {
	return f.options.questions(f.name, f.values)
}

func (f selectedFeature) readableAsNeed() map[string]string {
	return f.values.byReference(f.prefix())
}

func (f selectedFeature) prefix() string {
	return f.name + "."
}
