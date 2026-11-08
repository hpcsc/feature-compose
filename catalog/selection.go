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
	starter         Starter
	name            string
	values          map[string]string
	templates       []template
	extensionPoints []string
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
	for _, folderName := range slices.Sorted(maps.Keys(s.starter.folders)) {
		if err := s.starter.folders[folderName].runAfterRender(ctx, output); err != nil {
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
	funcs := texttemplate.FuncMap{"option": s.option, "items": s.items, "githubRepository": parseGitHubRepository, "kebab": kebab, "snake": snake, "camel": camel, "pascal": pascal}
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

func (s Selection) items(extensionPoint string) ([]map[string]string, error) {
	if !slices.Contains(s.extensionPoints, extensionPoint) {
		if len(s.extensionPoints) == 0 {
			return nil, fmt.Errorf("unknown extension point %q: no selected feature declares an extension point", extensionPoint)
		}
		return nil, fmt.Errorf("unknown extension point %q: the selected features declare %s",
			extensionPoint, strings.Join(s.extensionPoints, ", "))
	}
	return nil, nil
}
