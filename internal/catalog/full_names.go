package catalog

import (
	"fmt"
	"slices"
	"strings"
)

type fullNames []string

func (names fullNames) find(name string) (string, error) {
	if slices.Contains(names, name) {
		return name, nil
	}
	var matches []string
	for _, fullName := range names {
		if catalogName, starterName := SplitFullName(fullName); catalogName != "" && starterName == name {
			matches = append(matches, fullName)
		}
	}
	switch len(matches) {
	case 0:
		return "", names.unknown(name)
	case 1:
		return matches[0], nil
	}
	return "", fmt.Errorf("more than one catalog has the starter %q: give its full name, one of %s", name, strings.Join(matches, ", "))
}

func (names fullNames) unknown(name string) error {
	catalogNames := map[string]bool{}
	for _, fullName := range names {
		catalogName, _ := SplitFullName(fullName)
		catalogNames[catalogName] = true
	}
	if len(catalogNames) > 1 {
		return fmt.Errorf("unknown starter %q: the catalogs have %s", name, strings.Join(names, ", "))
	}
	return fmt.Errorf("unknown starter %q: the catalog has %s", name, strings.Join(names, ", "))
}
