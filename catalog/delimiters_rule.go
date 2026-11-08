package catalog

import (
	"fmt"
	"path"
	"strings"
)

type delimitersRule struct {
	globSegments []string
	delimiters   delimiters
}

func newDelimitersRule(glob string, d delimiters) (delimitersRule, error) {
	if d.left == "" || d.right == "" {
		return delimitersRule{}, fmt.Errorf("the delimiters of the glob %q need both left and right", glob)
	}
	segments := strings.Split(glob, "/")
	for _, segment := range segments {
		if segment == "" {
			return delimitersRule{}, fmt.Errorf("the glob %q has an empty segment", glob)
		}
		if _, err := path.Match(segment, ""); err != nil {
			return delimitersRule{}, fmt.Errorf("the glob %q is not valid: %w", glob, err)
		}
	}
	return delimitersRule{globSegments: segments, delimiters: d}, nil
}

func (r delimitersRule) matches(catalogPath string) bool {
	return r.matchFrom(r.globSegments, strings.Split(catalogPath, "/"))
}

func (r delimitersRule) matchFrom(unmatchedGlob, unmatchedPath []string) bool {
	if len(unmatchedGlob) == 0 {
		return len(unmatchedPath) == 0
	}
	if unmatchedGlob[0] == "**" {
		for skipped := 0; skipped <= len(unmatchedPath); skipped++ {
			if r.matchFrom(unmatchedGlob[1:], unmatchedPath[skipped:]) {
				return true
			}
		}
		return false
	}
	if len(unmatchedPath) == 0 {
		return false
	}
	matched, _ := path.Match(unmatchedGlob[0], unmatchedPath[0])
	return matched && r.matchFrom(unmatchedGlob[1:], unmatchedPath[1:])
}
