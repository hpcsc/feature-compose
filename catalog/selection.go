package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
	texttemplate "text/template"
)

type Selection struct {
	starter Starter
	name    string
	values  map[string]string
}

func (s Selection) Write(output string) error {
	if err := s.checkEmpty(output); err != nil {
		return err
	}
	rendered := make([]file, 0, len(s.starter.templates))
	writers := make(map[string]string, len(s.starter.templates))
	for _, t := range s.starter.templates {
		f, err := t.render(s)
		if err != nil {
			return err
		}
		if other, ok := writers[f.path]; ok {
			return fmt.Errorf("the templates %s and %s both write %s", other, t.path, f.path)
		}
		writers[f.path] = t.path
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
	for _, f := range s.starter.folders {
		if err := f.runAfterRender(ctx, output); err != nil {
			return err
		}
	}
	return nil
}

func (s Selection) checkEmpty(output string) error {
	entries, err := os.ReadDir(output)
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

func (s Selection) render(name, text string, d delimiters) (string, error) {
	funcs := texttemplate.FuncMap{"option": s.option, "kebab": kebab, "snake": snake, "camel": camel, "pascal": pascal}
	parsed, err := texttemplate.New(name).Delims(d.left, d.right).Funcs(funcs).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse %w", err)
	}
	var rendered strings.Builder
	if err := parsed.Execute(&rendered, map[string]any{"Name": s.name}); err != nil {
		return "", fmt.Errorf("render %w", err)
	}
	return rendered.String(), nil
}

func (s Selection) option(name string) (string, error) {
	optionName, ok := strings.CutPrefix(name, "starter.")
	value, found := s.values[optionName]
	if !ok || !found {
		return "", fmt.Errorf("unknown option %q: the options of the starter are starter.%s",
			name, strings.Join(slices.Sorted(maps.Keys(s.values)), ", starter."))
	}
	return value, nil
}
