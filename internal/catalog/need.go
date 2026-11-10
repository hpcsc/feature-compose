package catalog

import "strings"

type Need struct {
	Feature  string
	NeededBy []string
}

func (n Need) String() string {
	return n.Feature + ": " + n.source()
}

func (n Need) source() string {
	return "needed by " + strings.Join(n.NeededBy, ", ")
}
