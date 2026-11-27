package index

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"github.com/hpcsc/feature-compose/internal/config"
	"go.yaml.in/yaml/v3"
)

const (
	DefaultURL = "https://raw.githubusercontent.com/hpcsc/feature-compose/main/catalogs.yml"
	maxSize    = 1 << 20
)

type Entry struct {
	Summary string `yaml:"summary"`
	Source  string `yaml:"source"`
	Path    string `yaml:"path"`
	Ref     string `yaml:"ref"`
}

func (e Entry) ConfigEntry() config.Entry {
	return config.Entry{Source: e.Source, Path: e.Path, Ref: e.Ref}
}

func (e Entry) controlCharacterField() (string, bool) {
	for _, field := range [][2]string{{"summary", e.Summary}, {"source", e.Source}, {"path", e.Path}, {"ref", e.Ref}} {
		if strings.IndexFunc(field[1], unicode.IsControl) >= 0 {
			return field[0], true
		}
	}
	return "", false
}

type File struct {
	url     string
	entries map[string]Entry
}

func Fetch(ctx context.Context, client *http.Client, url string) (File, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return File{}, fmt.Errorf("get the index of known catalogs from %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return File{}, fmt.Errorf("get the index of known catalogs from %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return File{}, fmt.Errorf("get the index of known catalogs from %s: %s", url, resp.Status)
	}
	contents, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return File{}, fmt.Errorf("get the index of known catalogs from %s: %w", url, err)
	}
	return parse(url, contents)
}

func parse(url string, contents []byte) (File, error) {
	var parsed struct {
		Catalogs map[string]Entry `yaml:"catalogs"`
	}
	if err := yaml.Unmarshal(contents, &parsed); err != nil {
		return File{}, fmt.Errorf("read the index of known catalogs from %s: %w", url, err)
	}
	for _, name := range slices.Sorted(maps.Keys(parsed.Catalogs)) {
		if err := config.CheckCatalogName(name); err != nil {
			return File{}, fmt.Errorf("read the index of known catalogs from %s: %w", url, err)
		}
		if parsed.Catalogs[name].Source == "" {
			return File{}, fmt.Errorf("read the index of known catalogs from %s: the catalog %s has no source", url, name)
		}
		if field, found := parsed.Catalogs[name].controlCharacterField(); found {
			return File{}, fmt.Errorf("read the index of known catalogs from %s: the %s of the catalog %s holds a control character", url, field, name)
		}
	}
	return File{url: url, entries: parsed.Catalogs}, nil
}

func (f File) Names() []string {
	return slices.Sorted(maps.Keys(f.entries))
}

func (f File) Entry(name string) (Entry, error) {
	entry, ok := f.entries[name]
	if ok {
		return entry, nil
	}
	if len(f.entries) == 0 {
		return Entry{}, fmt.Errorf("the index of known catalogs at %s has no catalog %s, and no other catalog: give the source with catalog add %s <source>", f.url, name, name)
	}
	return Entry{}, fmt.Errorf("the index of known catalogs at %s has no catalog %s: it has %s. Give the source with catalog add %s <source>", f.url, name, strings.Join(f.Names(), ", "), name)
}
