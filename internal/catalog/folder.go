package catalog

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/shell"
)

type folder struct {
	name        string
	path        string
	generate    string
	afterRender []string
	test        []string
}

func (f folder) generates() bool {
	return f.generate != ""
}

func (f folder) runGenerate(ctx context.Context, output, repoName string, command func(generated string) (string, error)) ([]string, error) {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return nil, fmt.Errorf("find the path of the output folder %s: %w", output, err)
	}
	parent := filepath.Dir(absolute)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, fmt.Errorf("make the parent folder of %s: %w", output, err)
	}
	temporary, err := os.MkdirTemp(parent, ".feature-compose-generate-")
	if err != nil {
		return nil, fmt.Errorf("make a temporary folder for the generate command of the folder %s: %w", f.name, err)
	}
	defer os.RemoveAll(temporary)
	generated := filepath.Join(temporary, repoName)
	rendered, err := command(generated)
	if err != nil {
		return nil, fmt.Errorf("the generate command of the folder %s: %w", f.name, err)
	}
	printed, err := shell.Run(ctx, temporary, rendered)
	if err != nil {
		return nil, fmt.Errorf("the generate command %q of the folder %s failed: %w\n%s", rendered, f.name, err, printed)
	}
	if info, err := os.Stat(generated); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("the generate command %q of the folder %s wrote no folder at the path that {{ .Folder }} gives\n%s", rendered, f.name, printed)
	}
	var paths []string
	err = filepath.WalkDir(generated, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(generated, name)
		if err != nil {
			return err
		}
		paths = append(paths, path.Join(f.path, filepath.ToSlash(relative)))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read the files of the generate command of the folder %s: %w", f.name, err)
	}
	destination := filepath.Join(output, filepath.FromSlash(f.path))
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return nil, fmt.Errorf("make the folder %s: %w", f.name, err)
	}
	entries, err := os.ReadDir(generated)
	if err != nil {
		return nil, fmt.Errorf("read the files of the generate command of the folder %s: %w", f.name, err)
	}
	for _, entry := range entries {
		if err := os.Rename(filepath.Join(generated, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			return paths, fmt.Errorf("move the files of the generate command of the folder %s: %w", f.name, err)
		}
	}
	return paths, nil
}

func (f folder) runAfterRender(ctx context.Context, output string) error {
	return f.runAll(ctx, output, "after_render", f.afterRender)
}

func (f folder) runTests(ctx context.Context, output string) error {
	return f.runAll(ctx, output, "test", f.test)
}

func (f folder) runAll(ctx context.Context, output, list string, commands []string) error {
	dir := filepath.Join(output, filepath.FromSlash(f.path))
	for _, command := range commands {
		printed, err := shell.Run(ctx, dir, command)
		if err != nil {
			return fmt.Errorf("the %s command %q of the folder %s failed: %w\n%s", list, command, f.name, err, printed)
		}
	}
	return nil
}

type folders map[string]folder

func (all folders) hasGenerator() bool {
	return slices.ContainsFunc(slices.Collect(maps.Values(all)), folder.generates)
}

func (all folders) generates(file string) bool {
	folderName, pathInFolder, _ := strings.Cut(file, "/")
	f, ok := all[folderName]
	return ok && f.generates() && pathInFolder != ""
}
