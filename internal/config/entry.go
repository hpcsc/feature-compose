package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var scpLikeURL = regexp.MustCompile(`^[^/@:]+@[^/:]+:`)

type Entry struct {
	Source string `yaml:"source"`
	Path   string `yaml:"path,omitempty"`
	Ref    string `yaml:"ref,omitempty"`
}

func (e Entry) IsGitURL() bool {
	return IsGitURL(e.Source)
}

func (e Entry) SourceWithoutCredentials() string {
	scheme, rest, found := strings.Cut(e.Source, "://")
	if !found {
		return e.Source
	}
	authority, path := rest, ""
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		authority, path = rest[:i], rest[i:]
	}
	at := strings.LastIndex(authority, "@")
	if at < 0 {
		return e.Source
	}
	user := ""
	if scheme := strings.ToLower(scheme); scheme != "http" && scheme != "https" {
		name, _, _ := strings.Cut(authority[:at], ":")
		user = name + "@"
	}
	return scheme + "://" + user + authority[at+1:] + path
}

func (e Entry) WithAbsoluteSource() (Entry, error) {
	if e.Source == "" || e.IsGitURL() || filepath.IsAbs(e.Source) || e.Source == "~" || strings.HasPrefix(e.Source, "~/") {
		return e, nil
	}
	absolute, err := filepath.Abs(e.Source)
	if err != nil {
		return Entry{}, fmt.Errorf("find the path of the source %s: %w", e.Source, err)
	}
	e.Source = absolute
	return e, nil
}

func (e Entry) CheckCatalogIn(root string) error {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(e.Path), "starters"))
	if err == nil && info.IsDir() {
		return nil
	}
	if e.Path == "" {
		return fmt.Errorf("the source %s has no catalog at its root, because it has no folder starters: give the folder of the catalog in the source with --path", e.SourceWithoutCredentials())
	}
	return fmt.Errorf("the source %s has no catalog at the path %s, because that path has no folder starters: give the folder of the catalog in the source with --path", e.SourceWithoutCredentials(), e.Path)
}

func (e Entry) cleaned() (Entry, error) {
	if e.Path == "" {
		return e, nil
	}
	cleaned := filepath.Clean(e.Path)
	if cleaned == "." {
		e.Path = ""
		return e, nil
	}
	if !filepath.IsLocal(cleaned) {
		return Entry{}, fmt.Errorf("the path %s is outside the source: give a folder in the source", e.Path)
	}
	e.Path = filepath.ToSlash(cleaned)
	return e, nil
}

func IsGitURL(source string) bool {
	return strings.Contains(source, "://") || scpLikeURL.MatchString(source)
}
