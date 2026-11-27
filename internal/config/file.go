package config

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

var catalogName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type File struct {
	path     string
	catalogs map[string]Entry
}

func Load() (File, error) {
	path, err := filePath()
	if err != nil {
		return File{}, err
	}
	file := File{path: path, catalogs: map[string]Entry{}}
	contents, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	var settings struct {
		Catalog  any              `yaml:"catalog"`
		Catalogs map[string]Entry `yaml:"catalogs"`
	}
	if err := yaml.Unmarshal(contents, &settings); err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	if settings.Catalog != nil {
		return File{}, fmt.Errorf("%s has the key catalog, which feature-compose does not read: remove the key, and then add the catalog with feature-compose catalog add <name> <folder>", path)
	}
	for _, name := range slices.Sorted(maps.Keys(settings.Catalogs)) {
		if file, err = file.WithCatalog(name, settings.Catalogs[name]); err != nil {
			return File{}, fmt.Errorf("read %s: %w", path, err)
		}
	}
	return file, nil
}

func (f File) Path() string {
	return f.path
}

func (f File) Names() []string {
	return slices.Sorted(maps.Keys(f.catalogs))
}

func (f File) Entry(name string) (Entry, error) {
	entry, ok := f.catalogs[name]
	if ok {
		return entry, nil
	}
	if len(f.catalogs) == 0 {
		return Entry{}, fmt.Errorf("the config file %s has no catalog %s, and no other catalog: add one with feature-compose catalog add <name> <source>", f.path, name)
	}
	return Entry{}, fmt.Errorf("the config file %s has no catalog %s: it has %s", f.path, name, strings.Join(f.Names(), ", "))
}

func (f File) WithCatalog(name string, e Entry) (File, error) {
	if err := CheckCatalogName(name); err != nil {
		return File{}, err
	}
	if other, ok := f.nameInOtherCase(name); ok {
		return File{}, fmt.Errorf("the config already has the catalog %s, whose name differs from %s only in case: two such names share one clone folder on a file system that ignores case", other, name)
	}
	if e.Source == "" {
		return File{}, fmt.Errorf("the catalog %s has no source", name)
	}
	if strings.HasPrefix(e.Source, "-") {
		return File{}, fmt.Errorf("the source %s of the catalog %s starts with '-', so git reads it as an option", e.Source, name)
	}
	cleaned, err := e.cleaned()
	if err != nil {
		return File{}, err
	}
	f.catalogs = maps.Clone(f.catalogs)
	f.catalogs[name] = cleaned
	return f, nil
}

func (f File) WithoutCatalog(name string) File {
	f.catalogs = maps.Clone(f.catalogs)
	delete(f.catalogs, name)
	return f
}

func (f File) nameInOtherCase(name string) (string, bool) {
	for other := range f.catalogs {
		if other != name && strings.EqualFold(other, name) {
			return other, true
		}
	}
	return "", false
}

func (f File) Save() error {
	var contents strings.Builder
	encoder := yaml.NewEncoder(&contents)
	encoder.SetIndent(2)
	settings := struct {
		Catalogs map[string]Entry `yaml:"catalogs"`
	}{Catalogs: f.catalogs}
	if err := encoder.Encode(settings); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	target := f.path
	if resolved, err := filepath.EvalSymlinks(f.path); err == nil {
		target = resolved
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".config-*.yml")
	if err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.WriteString(contents.String()); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	if err := os.Rename(temporary.Name(), target); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	return nil
}

func (f File) SourceFolder(e Entry) (string, error) {
	if e.IsGitURL() {
		return "", fmt.Errorf("the source %s is a git URL, not a folder", e.Source)
	}
	if e.Source == "~" || strings.HasPrefix(e.Source, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find the home folder for the source %s: %w", e.Source, err)
		}
		return filepath.Join(home, strings.TrimPrefix(e.Source, "~")), nil
	}
	if filepath.IsAbs(e.Source) {
		return e.Source, nil
	}
	return filepath.Join(filepath.Dir(f.path), e.Source), nil
}

func filePath() (string, error) {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "feature-compose", "config.yml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find the config file: %w", err)
	}
	return filepath.Join(home, ".config", "feature-compose", "config.yml"), nil
}

func CheckCatalogName(name string) error {
	if !catalogName.MatchString(name) {
		return fmt.Errorf("the name %q of the catalog must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'", name)
	}
	return nil
}
