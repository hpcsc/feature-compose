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
)

type Selection struct {
	folders         map[string]folder
	name            string
	values          map[string]string
	templates       []template
	extensionPoints map[string]extensionPoint
}

func (s Selection) Write(output string) error {
	if err := s.checkEmpty(output); err != nil {
		return err
	}
	rendered := make([]file, 0, len(s.templates))
	writers := make(map[string]template, len(s.templates))
	for _, t := range s.templates {
		f, err := t.render(s)
		if err != nil {
			return err
		}
		if other, ok := writers[f.path]; ok {
			return fmt.Errorf("the template %s of %s and the template %s of %s both write %s",
				other.catalogPath, other.owner, t.catalogPath, t.owner, f.path)
		}
		writers[f.path] = t
		rendered = append(rendered, f)
	}
	for _, f := range rendered {
		if err := f.write(output); err != nil {
			return err
		}
	}
	return nil
}

func (s Selection) RunAfterRender(ctx context.Context, output string) error {
	for _, folderName := range slices.Sorted(maps.Keys(s.folders)) {
		if err := s.folders[folderName].runAfterRender(ctx, output); err != nil {
			return err
		}
	}
	return nil
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

func (s Selection) render(label, text string, d delimiters) (string, error) {
	funcs := texttemplate.FuncMap{"option": s.option, "items": s.items, "unique": s.unique, "githubRepository": parseGitHubRepository, "kebab": kebab, "snake": snake, "camel": camel, "pascal": pascal}
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

func (s Selection) option(reference string) (string, error) {
	optionName, ok := strings.CutPrefix(reference, "starter.")
	value, found := s.values[optionName]
	if !ok || !found {
		return "", fmt.Errorf("unknown option %q: the options of the starter are starter.%s",
			reference, strings.Join(slices.Sorted(maps.Keys(s.values)), ", starter."))
	}
	return value, nil
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
