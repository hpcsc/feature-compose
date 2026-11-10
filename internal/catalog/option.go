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

func (o option) hasValues() bool {
	return len(o.accepted) > 0
}

type options struct {
	declarer string
	declared map[string]option
}

func (o options) withDefaults(given map[string]string, name string) (optionValues, error) {
	for _, optionName := range slices.Sorted(maps.Keys(given)) {
		declared, ok := o.declared[optionName]
		if !ok {
			if len(o.declared) == 0 {
				return optionValues{}, fmt.Errorf("unknown option %q: %s has no option", optionName, o.declarer)
			}
			return optionValues{}, fmt.Errorf("unknown option %q: the options of %s are %s", optionName, o.declarer, strings.Join(slices.Sorted(maps.Keys(o.declared)), ", "))
		}
		if err := declared.check(given[optionName]); err != nil {
			return optionValues{}, fmt.Errorf("%s: %w", o.declarer, err)
		}
	}
	values := optionValues{byName: make(map[string]string, len(o.declared)), given: make(map[string]bool, len(given)), defaults: make(map[string]string, len(o.declared))}
	for _, optionName := range slices.Sorted(maps.Keys(o.declared)) {
		defaultValue, err := Selection{name: name}.render("the default of the option "+optionName+" of "+o.declarer, o.declared[optionName].defaultValue, defaultDelimiters, "")
		if err != nil {
			return optionValues{}, err
		}
		values.defaults[optionName] = defaultValue
		value, ok := given[optionName]
		if ok {
			values.given[optionName] = true
		} else {
			value = defaultValue
		}
		values.byName[optionName] = value
	}
	return values, nil
}

func (o options) defaultsOfOptionsWithValues() map[string]string {
	defaults := map[string]string{}
	for optionName, declared := range o.declared {
		if declared.hasValues() {
			defaults[optionName] = declared.defaultValue
		}
	}
	return defaults
}

func (o options) otherValuesOfOptionsWithValues() []map[string]string {
	var others []map[string]string
	for _, optionName := range slices.Sorted(maps.Keys(o.declared)) {
		declared := o.declared[optionName]
		for _, value := range declared.accepted {
			if value == declared.defaultValue {
				continue
			}
			values := o.defaultsOfOptionsWithValues()
			values[optionName] = value
			others = append(others, values)
		}
	}
	return others
}
