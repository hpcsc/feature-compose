package catalog

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

type option struct {
	name         string
	defaultValue string
	pattern      *regexp.Regexp
	accepted     []string
}

func newOption(name, defaultValue, pattern string, accepted []string) (option, error) {
	declared := option{name: name, defaultValue: defaultValue, accepted: accepted}
	if pattern != "" {
		var err error
		declared.pattern, err = regexp.Compile(pattern)
		if err != nil {
			return option{}, fmt.Errorf("the pattern of the option %s: %w", name, err)
		}
	}
	return declared, nil
}

func (o option) check(value string) error {
	if len(o.accepted) > 0 && !slices.Contains(o.accepted, value) {
		return fmt.Errorf("the value %q of the option %s is not one of its values: %s", value, o.name, strings.Join(o.accepted, ", "))
	}
	if o.pattern != nil && !o.pattern.MatchString(value) {
		return fmt.Errorf("the value %q of the option %s does not match its pattern %s", value, o.name, o.pattern)
	}
	return nil
}

type options struct {
	declarer string
	declared map[string]option
}

func (o options) withDefaults(given map[string]string, name string) (map[string]string, error) {
	for _, optionName := range slices.Sorted(maps.Keys(given)) {
		declared, ok := o.declared[optionName]
		if !ok {
			if len(o.declared) == 0 {
				return nil, fmt.Errorf("unknown option %q: %s has no option", optionName, o.declarer)
			}
			return nil, fmt.Errorf("unknown option %q: the options of %s are %s", optionName, o.declarer, strings.Join(slices.Sorted(maps.Keys(o.declared)), ", "))
		}
		if err := declared.check(given[optionName]); err != nil {
			return nil, fmt.Errorf("%s: %w", o.declarer, err)
		}
	}
	values := make(map[string]string, len(o.declared))
	for _, optionName := range slices.Sorted(maps.Keys(o.declared)) {
		value, ok := given[optionName]
		if !ok {
			var err error
			value, err = Selection{name: name}.render("the default of the option "+optionName+" of "+o.declarer, o.declared[optionName].defaultValue, defaultDelimiters, "")
			if err != nil {
				return nil, err
			}
		}
		values[optionName] = value
	}
	return values, nil
}
