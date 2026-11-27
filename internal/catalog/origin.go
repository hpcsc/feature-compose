package catalog

import "strings"

type Origin struct {
	Name   string `yaml:"name"`
	Source string `yaml:"source"`
	Path   string `yaml:"path,omitempty"`
	Ref    string `yaml:"ref,omitempty"`
	Commit string `yaml:"commit,omitempty"`
}

func (o Origin) fullName(name string) string {
	if o.Name == "" {
		return name
	}
	return o.Name + "/" + name
}

func SplitFullName(fullName string) (catalogName, name string) {
	catalogName, name, found := strings.Cut(fullName, "/")
	if !found {
		return "", fullName
	}
	return catalogName, name
}
