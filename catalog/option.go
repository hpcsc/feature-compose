package catalog

import (
	"fmt"
	"regexp"
)

type option struct {
	name         string
	defaultValue string
	pattern      *regexp.Regexp
}

func (o option) check(value string) error {
	if o.pattern != nil && !o.pattern.MatchString(value) {
		return fmt.Errorf("the value %q of the option %s does not match its pattern %s", value, o.name, o.pattern)
	}
	return nil
}
