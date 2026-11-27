package catalog

import "strings"

type Origin struct {
	Name   string `json:"name" yaml:"name"`
	Source string `json:"source" yaml:"source"`
	Path   string `json:"path" yaml:"path,omitempty"`
	Ref    string `json:"ref" yaml:"ref,omitempty"`
	Commit string `json:"commit" yaml:"commit,omitempty"`
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
