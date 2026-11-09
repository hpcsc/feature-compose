package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	texttemplate "text/template"

	"github.com/hpcsc/feature-compose/internal/github"
	"github.com/hpcsc/feature-compose/internal/textcase"
)

type Selection struct {
	folders         map[string]folder
	name            string
	starterName     string
	featureNames    []string
	starterValues   map[string]string
	featureValues   map[string]map[string]string
	templates       []template
	extensionPoints map[string]extensionPoint
	addedByNeeds    []Need
	needs           map[string][]string
}

func (s Selection) AddedByNeeds() []Need {
	added := make([]Need, 0, len(s.addedByNeeds))
	for _, need := range s.addedByNeeds {
		added = append(added, Need{Feature: need.Feature, NeededBy: slices.Clone(need.NeededBy)})
	}
	return added
}

func (s Selection) Write(output string) error {
	if err := s.checkEmpty(output); err != nil {
		return err
	}
	files, err := s.files()
	if err != nil {
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
	slices.SortFunc(files, func(a, b file) int { return strings.Compare(a.path, b.path) })
	var preview strings.Builder
	fmt.Fprintf(&preview, "Repo: %s\nStarter: %s\n", s.name, s.starterName)
	if len(s.featureNames) == 0 {
		preview.WriteString("\nFeatures: none\n")
	} else {
		preview.WriteString("\nFeatures:\n")
	}
	for _, featureName := range s.featureNames {
		fmt.Fprintf(&preview, "  %s: %s\n", featureName, s.featureSource(featureName))
	}
	if len(files) == 0 {
		preview.WriteString("\nFiles: none\n")
	} else {
		preview.WriteString("\nFiles:\n")
	}
	for _, f := range files {
		fmt.Fprintf(&preview, "  %s: %s\n", f.path, f.owner)
	}
	return preview.String(), nil
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

func (s Selection) featureSource(featureName string) string {
	for _, need := range s.addedByNeeds {
		if need.Feature == featureName {
			return "needed by " + strings.Join(need.NeededBy, ", ")
		}
	}
	return "command line"
}

func (s Selection) files() ([]file, error) {
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
	if optionName, ok := strings.CutPrefix(reference, "starter."); ok {
		if value, ok := s.starterValues[optionName]; ok {
			return value, nil
		}
	} else if needName, optionName, ok := strings.Cut(reference, "."); ok {
		if value, ok := s.featureValues[needName][optionName]; ok && slices.Contains(s.needs[featureName], needName) {
			return value, nil
		}
	} else if value, ok := s.featureValues[featureName][reference]; ok {
		return value, nil
	}
	readable := slices.Collect(maps.Keys(s.featureValues[featureName]))
	for _, needName := range s.needs[featureName] {
		for optionName := range s.featureValues[needName] {
			readable = append(readable, needName+"."+optionName)
		}
	}
	for optionName := range s.starterValues {
		readable = append(readable, "starter."+optionName)
	}
	if len(readable) == 0 {
		return "", fmt.Errorf("unknown option %q: the template can read no option", reference)
	}
	slices.Sort(readable)
	return "", fmt.Errorf("unknown option %q: the template can read %s", reference, strings.Join(readable, ", "))
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
