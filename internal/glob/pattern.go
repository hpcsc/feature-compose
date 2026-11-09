package glob

import (
	"fmt"
	"path"
	"strings"
)

type Pattern struct {
	segments []string
}

func Compile(pattern string) (Pattern, error) {
	segments := strings.Split(pattern, "/")
	for _, segment := range segments {
		if segment == "" {
			return Pattern{}, fmt.Errorf("the glob %q has an empty segment", pattern)
		}
		if _, err := path.Match(segment, ""); err != nil {
			return Pattern{}, fmt.Errorf("the glob %q is not valid: %w", pattern, err)
		}
	}
	return Pattern{segments: segments}, nil
}

func (p Pattern) Match(slashPath string) bool {
	return p.matchFrom(p.segments, strings.Split(slashPath, "/"))
}

func (p Pattern) matchFrom(unmatchedPattern, unmatchedPath []string) bool {
	if len(unmatchedPattern) == 0 {
		return len(unmatchedPath) == 0
	}
	if unmatchedPattern[0] == "**" {
		for skipped := 0; skipped <= len(unmatchedPath); skipped++ {
			if p.matchFrom(unmatchedPattern[1:], unmatchedPath[skipped:]) {
				return true
			}
		}
		return false
	}
	if len(unmatchedPath) == 0 {
		return false
	}
	matched, _ := path.Match(unmatchedPattern[0], unmatchedPath[0])
	return matched && p.matchFrom(unmatchedPattern[1:], unmatchedPath[1:])
}
