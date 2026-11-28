package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	texttemplate "text/template"

	"github.com/hpcsc/feature-compose/internal/astgrep"
	"github.com/hpcsc/feature-compose/internal/github"
	"github.com/hpcsc/feature-compose/internal/indent"
	"github.com/hpcsc/feature-compose/internal/textcase"
)

const writeFilesStep = "Write the files of the starter and the features"

var shellWord = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

type Selection struct {
	folders         folders
	removals        []removal
	name            string
	starter         selectedStarter
	features        []selectedFeature
	templates       []template
	extensionPoints map[string]extensionPoint
}

func (s Selection) Name() string {
	return s.name
}

func (s Selection) StarterFullName() string {
	return s.starter.origin.fullName(s.starter.name)
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
	if s.folders.hasGenerator() {
		return fmt.Errorf("the starter %s runs a generate command, so only MakeRepo writes its repo", s.starter.name)
	}
	return s.writeFiles(output)
}

func (s Selection) writeFiles(output string) error {
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

func (s Selection) MakeRepo(ctx context.Context, output, version string, steps Steps) error {
	s, err := s.withToolchains()
	if err != nil {
		return err
	}
	if err := s.writeRepo(ctx, output, steps); err != nil {
		return err
	}
	if err := s.runAfterRender(ctx, output, steps); err != nil {
		return err
	}
	return s.writeSelectionFile(output, version)
}

func (s Selection) writeSelectionFile(output, version string) error {
	featureValues := make(map[string]map[string]string, len(s.features))
	var requested []string
	for _, f := range s.features {
		featureValues[f.name] = f.values.byName
		if f.requested() {
			requested = append(requested, f.name)
		}
	}
	slices.Sort(requested)
	var origin *Origin
	if s.starter.origin.Name != "" {
		origin = &s.starter.origin
	}
	contents, err := selectionFile{
		Version:       version,
		Catalog:       origin,
		StarterName:   s.starter.name,
		Name:          s.name,
		StarterValues: s.starter.values.byName,
		FeatureValues: featureValues,
		Requested:     requested,
	}.contents()
	if err != nil {
		return err
	}
	return file{path: selectionFilePath, contents: contents}.write(output)
}

func (s Selection) Preview() (string, error) {
	files, err := s.files()
	if err != nil {
		return "", err
	}
	if s, err = s.withToolchains(); err != nil {
		return "", err
	}
	if err := s.checkGenerateValues(); err != nil {
		return "", err
	}
	var preview strings.Builder
	fmt.Fprintf(&preview, "Repo: %s\nStarter: %s\n", s.name, s.StarterFullName())
	s.writeSection(&preview, "Features", s.featureLines())
	s.writeSection(&preview, "Options", s.optionLines())
	s.writeSection(&preview, "Commands", s.commandLines())
	s.writeSection(&preview, "Files", s.fileLines(append(files, file{path: selectionFilePath, owner: selectionFileOwner})))
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
	words := []string{s.starter.origin.fullName(s.starter.name), s.name}
	if output != s.name {
		words = append(words, "--output-dir", output)
	}
	words = append(words, flagValues(s.starter.values.valuesOfArguments()).optionWords()...)
	for _, f := range s.featuresOfArguments() {
		words = append(words, flagValues(f.values.valuesOfArguments()).featureWords(f.name)...)
	}
	quoted := make([]string, 0, len(words))
	for _, word := range words {
		quoted = append(quoted, s.quoted(word))
	}
	return strings.Join(quoted, " ")
}

func (s Selection) RunAfterRender(ctx context.Context, output string) error {
	s, err := s.withToolchains()
	if err != nil {
		return err
	}
	return s.runAfterRender(ctx, output, NoSteps{})
}

func (s Selection) runAfterRender(ctx context.Context, output string, steps Steps) error {
	for _, folderName := range slices.Sorted(maps.Keys(s.folders)) {
		if err := s.folders[folderName].runAfterRender(ctx, output, steps); err != nil {
			return err
		}
	}
	return nil
}

func (s Selection) RunTests(ctx context.Context, output string) error {
	s, err := s.withToolchains()
	if err != nil {
		return err
	}
	return s.runTests(ctx, output)
}

func (s Selection) runTests(ctx context.Context, output string) error {
	for _, folderName := range slices.Sorted(maps.Keys(s.folders)) {
		if err := s.folders[folderName].runTests(ctx, output); err != nil {
			return err
		}
	}
	return nil
}

func (s Selection) withToolchains() (Selection, error) {
	files, err := s.files()
	if err != nil {
		return Selection{}, err
	}
	withTools := make(folders, len(s.folders))
	for name, f := range s.folders {
		withTools[name] = f
		if !f.pinnedTools {
			continue
		}
		target := path.Join(f.path, toolchainFile)
		i := slices.IndexFunc(files, func(written file) bool { return written.path == target })
		if i < 0 {
			return Selection{}, fmt.Errorf("the folder %s has pinned_tools, but the selection writes no %s in it", name, target)
		}
		if f.toolchain, err = newToolchain(fmt.Sprintf("%s of the folder %s, which %s writes", target, name, files[i].owner), files[i].contents); err != nil {
			return Selection{}, err
		}
		withTools[name] = f
	}
	s.folders = withTools
	return s, nil
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
		case len(f.values.valuesOfArguments()) > 0:
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
		written, err := t.when.trueFor(s.readable(t.featureName))
		if err != nil {
			return nil, fmt.Errorf("the when %q of %s: %w", t.when.text, t.owner, err)
		}
		if !written {
			continue
		}
		f, err := t.render(s)
		if err != nil {
			return nil, err
		}
		if f.path == selectionFilePath {
			return nil, fmt.Errorf("%s and %s both write %s", t.description(), selectionFileOwner, f.path)
		}
		if other, ok := writers[f.path]; ok {
			return nil, fmt.Errorf("%s and %s both write %s", other.description(), t.description(), f.path)
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
	return s.renderData(label, text, d, featureName, map[string]any{"Name": s.name})
}

func (s Selection) renderData(label, text string, d delimiters, featureName string, data map[string]any) (string, error) {
	parsed, err := texttemplate.New(label).Delims(d.left, d.right).Funcs(s.funcs(featureName)).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse %w", err)
	}
	var rendered strings.Builder
	if err := parsed.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("render %w", err)
	}
	return rendered.String(), nil
}

func (s Selection) funcs(featureName string) texttemplate.FuncMap {
	option := func(reference string) (string, error) { return s.option(featureName, reference) }
	return texttemplate.FuncMap{"option": option, "items": s.items, "unique": s.unique, "indent": s.indent, "githubRepository": github.RepositoryOf, "kebab": textcase.Kebab, "snake": textcase.Snake, "camel": textcase.Camel, "pascal": textcase.Pascal, "upper": strings.ToUpper}
}

func (s Selection) indent(spaces int, text string) string {
	return indent.Lines(text, strings.Repeat(" ", spaces))
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
	readable := s.readableOfNeedsAndStarter(reader.needs)
	maps.Copy(readable, reader.values.byReference(""))
	return readable
}

func (s Selection) readableOfNeedsAndStarter(needNames []string) map[string]string {
	readable := s.starter.readable()
	for _, needName := range needNames {
		maps.Copy(readable, s.feature(needName).readableAsNeed())
	}
	return readable
}

func (s Selection) items(name string) ([]map[string]string, error) {
	declared, err := s.extensionPoint(name)
	if err != nil {
		return nil, err
	}
	return declared.written(s.readable)
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
		values, err := declared.values(field, s.readable)
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

func (s Selection) writeRepo(ctx context.Context, output string, steps Steps) error {
	if !s.folders.hasGenerator() {
		return s.step(steps, writeFilesStep, func() error { return s.writeFiles(output) })
	}
	files, err := s.files()
	if err != nil {
		return err
	}
	if err := s.checkEmpty(output); err != nil {
		return err
	}
	if s.changesGeneratedFiles() {
		if err := astgrep.Installed(); err != nil {
			return fmt.Errorf("the starter changes the files of its generator with ast-grep: %w", err)
		}
	}
	if err := s.checkGenerateValues(); err != nil {
		return err
	}
	_, statErr := os.Stat(output)
	existed := statErr == nil
	written := map[string]bool{}
	if err := s.writeGeneratedRepo(ctx, output, files, written, steps); err != nil {
		if existed {
			s.removeEntries(output, written)
		} else {
			_ = os.RemoveAll(output)
		}
		return err
	}
	return nil
}

func (s Selection) checkGenerateValues() error {
	readable := s.readable("")
	for _, folderName := range slices.Sorted(maps.Keys(s.folders)) {
		f := s.folders[folderName]
		if !f.generates() {
			continue
		}
		parsed, err := calls{}.with("the generate command of the folder "+f.name, f.generate, defaultDelimiters)
		if err != nil {
			return fmt.Errorf("the generate command of the folder %s does not parse: %w", f.name, err)
		}
		for _, reference := range parsed.argumentsOf("option") {
			if value, ok := readable[reference]; ok && !shellWord.MatchString(value) {
				return fmt.Errorf("the generate command of the folder %s reads the option %s, whose value %q is not one shell word: the value can hold only letters, digits and the characters _@%%+=:,./-", f.name, reference, value)
			}
		}
	}
	return nil
}

func (s Selection) removeEntries(output string, names map[string]bool) {
	for name := range names {
		_ = os.RemoveAll(filepath.Join(output, name))
	}
}

func (s Selection) writeGeneratedRepo(ctx context.Context, output string, files []file, written map[string]bool, steps Steps) error {
	generatedPaths, err := s.generate(ctx, output, steps)
	for repoPath := range generatedPaths {
		written[s.topEntry(repoPath)] = true
	}
	if err != nil {
		return err
	}
	if len(s.removals) > 0 {
		if err := s.step(steps, "Remove the generated files and code that the starter names", func() error { return s.remove(ctx, output, generatedPaths) }); err != nil {
			return err
		}
	}
	err = s.step(steps, writeFilesStep, func() error {
		for _, f := range files {
			if generatedPaths[f.path] && f.featureName != "" {
				return fmt.Errorf("%s writes %s, which the generate command also writes: only a file of the starter can replace a generated file", f.owner, f.path)
			}
			written[s.topEntry(f.path)] = true
			if err := f.write(output); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || !s.hasGeneratedExtensionPoint() {
		return err
	}
	return s.step(steps, "Add the items to the generated files", func() error { return s.insertItems(ctx, output, generatedPaths) })
}

func (s Selection) step(steps Steps, label string, run func() error) error {
	end := steps.Start(label)
	err := run()
	end(err)
	return err
}

func (s Selection) topEntry(repoPath string) string {
	entry, _, _ := strings.Cut(repoPath, "/")
	return entry
}

func (s Selection) changesGeneratedFiles() bool {
	return slices.ContainsFunc(s.removals, func(r removal) bool { return !r.query.Empty() }) || s.hasGeneratedExtensionPoint()
}

func (s Selection) hasGeneratedExtensionPoint() bool {
	return slices.ContainsFunc(slices.Collect(maps.Values(s.extensionPoints)), extensionPoint.inGeneratedFile)
}

func (s Selection) generate(ctx context.Context, output string, steps Steps) (map[string]bool, error) {
	generatedPaths := map[string]bool{}
	for _, folderName := range slices.Sorted(maps.Keys(s.folders)) {
		f := s.folders[folderName]
		if !f.generates() {
			continue
		}
		end := steps.Start("Generate the folder " + f.name)
		paths, err := f.runGenerate(ctx, output, s.name, func(generatedFolder string) (string, error) {
			return s.renderData("the generate command of the folder "+f.name, f.generate, defaultDelimiters, "", map[string]any{"Name": s.name, "Folder": s.quoted(generatedFolder)})
		})
		end(err)
		for _, p := range paths {
			generatedPaths[p] = true
		}
		if err != nil {
			return generatedPaths, err
		}
	}
	return generatedPaths, nil
}

func (s Selection) commandLines() []string {
	var lines []string
	folderNames := slices.Sorted(maps.Keys(s.folders))
	for _, folderName := range folderNames {
		f := s.folders[folderName]
		if !f.generates() {
			continue
		}
		command, err := s.renderData("the generate command of the folder "+f.name, f.generate, defaultDelimiters, "", map[string]any{"Name": s.name, "Folder": "<new folder>"})
		if err != nil {
			command = f.generate
		}
		lines = append(lines, folderName+": "+f.toolchain.shown(command))
	}
	for _, folderName := range folderNames {
		f := s.folders[folderName]
		for _, command := range f.afterRender {
			lines = append(lines, folderName+": "+f.toolchain.shown(command))
		}
	}
	return lines
}

func (s Selection) remove(ctx context.Context, output string, generatedPaths map[string]bool) error {
	for i, r := range s.removals {
		label := fmt.Sprintf("the removal %d of the starter", i+1)
		repoPath, err := s.repoPathOf(r.file, label)
		if err != nil {
			return err
		}
		if !generatedPaths[repoPath] {
			return fmt.Errorf("%s removes %s, which the generate command does not write", label, repoPath)
		}
		onDisk := filepath.Join(output, filepath.FromSlash(repoPath))
		if r.query.Empty() {
			if err := os.Remove(onDisk); err != nil {
				return fmt.Errorf("%s: %w", label, err)
			}
			s.removeEmptyFolders(output, path.Dir(repoPath))
			delete(generatedPaths, repoPath)
			continue
		}
		query, err := s.renderedQuery(r.query, label)
		if err != nil {
			return err
		}
		match, contents, err := s.onlyMatch(ctx, onDisk, repoPath, query, label)
		if err != nil {
			return err
		}
		if err := os.WriteFile(onDisk, []byte(r.removedFrom(contents, match)), 0o644); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}
	return nil
}

func (s Selection) insertItems(ctx context.Context, output string, generatedPaths map[string]bool) error {
	for _, name := range slices.Sorted(maps.Keys(s.extensionPoints)) {
		e := s.extensionPoints[name]
		if !e.inGeneratedFile() {
			continue
		}
		items, err := e.writtenItems(s.readable)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			continue
		}
		label := fmt.Sprintf("the extension point %q", name)
		repoPath, err := s.repoPathOf(e.file, label)
		if err != nil {
			return err
		}
		if !generatedPaths[repoPath] {
			return fmt.Errorf("%s is in %s, which the generate command does not write", label, repoPath)
		}
		texts, err := s.insertTexts(e, items, label)
		if err != nil {
			return err
		}
		query, err := s.renderedQuery(e.generatedFile.query, label)
		if err != nil {
			return err
		}
		onDisk := filepath.Join(output, filepath.FromSlash(repoPath))
		match, contents, err := s.onlyMatch(ctx, onDisk, repoPath, query, label)
		if err != nil {
			return err
		}
		var comments []astgrep.Node
		if e.generatedFile.needsComments() {
			comments = astgrep.Comments(ctx, onDisk, query.Language)
		}
		changed, err := e.generatedFile.inserted(contents, match, texts, comments)
		if err != nil {
			return fmt.Errorf("%s in %s %w", label, repoPath, err)
		}
		if err := os.WriteFile(onDisk, []byte(changed), 0o644); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}
	return nil
}

func (s Selection) insertTexts(e extensionPoint, items []item, label string) ([]string, error) {
	texts := make([]string, 0, len(items))
	for i, added := range items {
		data := map[string]any{"Name": s.name}
		for _, field := range slices.Sorted(maps.Keys(added.fields)) {
			rendered, err := s.render(fmt.Sprintf("the field %s of the item %d of %s", field, i+1, label), added.fields[field], defaultDelimiters, added.featureName)
			if err != nil {
				return nil, err
			}
			data[field] = rendered
		}
		text, err := s.renderData("the insert of "+label, e.generatedFile.insert, defaultDelimiters, "", data)
		if err != nil {
			return nil, err
		}
		texts = append(texts, text)
	}
	return texts, nil
}

func (s Selection) repoPathOf(catalogPath, label string) (string, error) {
	folderName, pathInFolder, _ := strings.Cut(catalogPath, "/")
	f, ok := s.folders[folderName]
	if !ok || pathInFolder == "" {
		return "", fmt.Errorf("%s names the file %q, which is not in a folder of the starter", label, catalogPath)
	}
	rendered, err := s.render(label, pathInFolder, defaultDelimiters, "")
	if err != nil {
		return "", err
	}
	return path.Join(f.path, rendered), nil
}

func (s Selection) renderedQuery(q astgrep.Query, label string) (astgrep.Query, error) {
	pattern, err := s.render("the pattern of "+label, q.Pattern, defaultDelimiters, "")
	if err != nil {
		return astgrep.Query{}, err
	}
	rendered := astgrep.Query{Language: q.Language, Pattern: pattern}
	if q.Rule != nil {
		rule, err := s.renderedValue(q.Rule, "the rule of "+label)
		if err != nil {
			return astgrep.Query{}, err
		}
		rendered.Rule = rule.(map[string]any)
	}
	return rendered, nil
}

func (s Selection) renderedValue(value any, label string) (any, error) {
	switch v := value.(type) {
	case string:
		return s.render(label, v, defaultDelimiters, "")
	case map[string]any:
		rendered := make(map[string]any, len(v))
		for key, inner := range v {
			r, err := s.renderedValue(inner, label)
			if err != nil {
				return nil, err
			}
			rendered[key] = r
		}
		return rendered, nil
	case []any:
		rendered := make([]any, 0, len(v))
		for _, inner := range v {
			r, err := s.renderedValue(inner, label)
			if err != nil {
				return nil, err
			}
			rendered = append(rendered, r)
		}
		return rendered, nil
	}
	return value, nil
}

func (s Selection) removeEmptyFolders(output, repoFolder string) {
	for ; repoFolder != "." && repoFolder != "/"; repoFolder = path.Dir(repoFolder) {
		if os.Remove(filepath.Join(output, filepath.FromSlash(repoFolder))) != nil {
			return
		}
	}
}

func (s Selection) onlyMatch(ctx context.Context, onDisk, repoPath string, q astgrep.Query, label string) (astgrep.Match, string, error) {
	matches, err := astgrep.Find(ctx, onDisk, q)
	if err != nil {
		return astgrep.Match{}, "", fmt.Errorf("%s: %w", label, err)
	}
	if len(matches) != 1 {
		return astgrep.Match{}, "", fmt.Errorf("%s matches %d places in %s, and it must match exactly one: the generator writes this file, so a new version of the generator can change it", label, len(matches), repoPath)
	}
	contents, err := os.ReadFile(onDisk)
	if err != nil {
		return astgrep.Match{}, "", fmt.Errorf("%s: %w", label, err)
	}
	return matches[0], string(contents), nil
}

const starterPrefix = "starter."

type selectedStarter struct {
	name    string
	origin  Origin
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
