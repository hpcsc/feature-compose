package catalog

import (
	"fmt"

	"github.com/hpcsc/feature-compose/internal/glob"
)

type delimitersRule struct {
	pattern    glob.Pattern
	delimiters delimiters
}

func newDelimitersRule(pattern string, d delimiters) (delimitersRule, error) {
	if d.left == "" || d.right == "" {
		return delimitersRule{}, fmt.Errorf("the delimiters of the glob %q need both left and right", pattern)
	}
	compiled, err := glob.Compile(pattern)
	if err != nil {
		return delimitersRule{}, err
	}
	return delimitersRule{pattern: compiled, delimiters: d}, nil
}

func (r delimitersRule) matches(catalogPath string) bool {
	return r.pattern.Match(catalogPath)
}
