package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type File struct {
	CatalogDir string
}

func Load() (File, error) {
	path, err := FilePath()
	if err != nil {
		return File{}, err
	}
	contents, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, fmt.Errorf("no config file at %s: write the folder of the catalog in it as \"catalog: <folder>\"", path)
	}
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	var settings struct {
		Catalog string `yaml:"catalog"`
	}
	if err := yaml.Unmarshal(contents, &settings); err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	if settings.Catalog == "" {
		return File{}, fmt.Errorf("the config file %s names no catalog: write the folder of the catalog in it as \"catalog: <folder>\"", path)
	}
	catalogDir, err := absolute(settings.Catalog, filepath.Dir(path))
	if err != nil {
		return File{}, err
	}
	info, err := os.Stat(catalogDir)
	if err != nil || !info.IsDir() {
		return File{}, fmt.Errorf("the catalog folder %s that %s names is not a folder", catalogDir, path)
	}
	return File{CatalogDir: catalogDir}, nil
}

func FilePath() (string, error) {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "feature-compose", "config.yml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find the config file: %w", err)
	}
	return filepath.Join(home, ".config", "feature-compose", "config.yml"), nil
}

func absolute(folder, configDir string) (string, error) {
	if folder == "~" || strings.HasPrefix(folder, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find the home folder for the catalog %s: %w", folder, err)
		}
		return filepath.Join(home, strings.TrimPrefix(folder, "~")), nil
	}
	if filepath.IsAbs(folder) {
		return folder, nil
	}
	return filepath.Join(configDir, folder), nil
}
