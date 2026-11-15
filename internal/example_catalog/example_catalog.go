package example_catalog

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	StarterName = "example"
	FeatureName = "greeting"
)

//go:embed all:files
var embedded embed.FS

func Write(folder string) error {
	files, err := fs.Sub(embedded, "files")
	if err != nil {
		return err
	}
	var paths []string
	err = fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return err
	}
	for _, path := range paths {
		if _, err := os.Lstat(filepath.Join(folder, filepath.FromSlash(path))); !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%s already holds %s, so init writes no file", folder, path)
		}
	}
	for _, path := range paths {
		contents, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}
		target := filepath.Join(folder, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, contents, 0o644); err != nil {
			return err
		}
	}
	return nil
}
