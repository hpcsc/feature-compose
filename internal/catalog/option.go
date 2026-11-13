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
	when         expression
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

func (o option) patternText() string {
	if o.pattern == nil {
		return ""
	}
	return o.pattern.String()
}

func (o option) hasValues() bool {
	return len(o.accepted) > 0
}

func (o option) hasWhen() bool {
	return o.when.text != ""
}

func (o option) applies(declarer string, whenReads map[string]string) (bool, error) {
	applies, err := o.when.trueFor(whenReads)
	if err != nil {
		return false, fmt.Errorf("the when %q of the option %s of %s: %w", o.when.text, o.name, declarer, err)
	}
	return applies, nil
}

func (o option) refusal(declarer, value, source string, whenReads map[string]string) error {
	var read []string
	for _, reference := range o.when.references() {
		read = append(read, fmt.Sprintf("%s is %q", reference, whenReads[reference]))
	}
	return fmt.Errorf("the option %s of %s applies only when %s, and %s, but %s gives it %q", o.name, declarer, o.when.text, strings.Join(read, " and "), source, value)
}

type options struct {
	declarer string
	declared map[string]option
}

func (o options) references(prefix string) map[string]bool {
	references := make(map[string]bool, len(o.declared))
	for optionName := range o.declared {
		references[prefix+optionName] = true
	}
	return references
}

func (o options) withDefaults(given map[string]string, setByNeeds valuesSetByNeeds, answered map[string]string, name string, ofNeedsAndStarter map[string]string) (optionValues, error) {
	if err := o.check(given); err != nil {
		return optionValues{}, err
	}
	if err := o.check(answered); err != nil {
		return optionValues{}, err
	}
	values := optionValues{
		byName:      make(map[string]string, len(o.declared)),
		given:       make(map[string]bool, len(given)),
		needOf:      map[string][]string{},
		answered:    make(map[string]bool, len(answered)),
		defaults:    make(map[string]string, len(o.declared)),
		notApplying: map[string]bool{},
	}
	choose := func(optionName string, whenReads map[string]string) error {
		declared := o.declared[optionName]
		defaultValue, err := Selection{name: name}.render("the default of the option "+optionName+" of "+o.declarer, declared.defaultValue, defaultDelimiters, "")
		if err != nil {
			return err
		}
		values.defaults[optionName] = defaultValue
		givenValue, isGiven := given[optionName]
		valueOfNeeds, needOf, needsSet := setByNeeds.value(optionName)
		answeredValue, isAnswered := answered[optionName]
		if isGiven {
			if err := setByNeeds.checkGiven(optionName, givenValue); err != nil {
				return err
			}
		}
		if needsSet {
			values.needOf[optionName] = needOf
		}
		applies, err := declared.applies(o.declarer, whenReads)
		if err != nil {
			return err
		}
		if !applies {
			switch {
			case isGiven && givenValue != defaultValue:
				return declared.refusal(o.declarer, givenValue, "the command line", whenReads)
			case needsSet && valueOfNeeds != defaultValue:
				return declared.refusal(o.declarer, valueOfNeeds, "the need of "+strings.Join(needOf, ", "), whenReads)
			}
			values.notApplying[optionName] = true
			isAnswered = false
		}
		switch {
		case isGiven:
			values.byName[optionName] = givenValue
			values.given[optionName] = true
		case needsSet:
			values.byName[optionName] = valueOfNeeds
		case isAnswered:
			values.byName[optionName] = answeredValue
			values.answered[optionName] = true
		default:
			values.byName[optionName] = defaultValue
		}
		return nil
	}
	withoutWhen, withWhen := o.splitByWhen()
	for _, optionName := range withoutWhen {
		if err := choose(optionName, nil); err != nil {
			return optionValues{}, err
		}
	}
	whenReads := maps.Clone(ofNeedsAndStarter)
	maps.Copy(whenReads, values.byName)
	for _, optionName := range withWhen {
		if err := choose(optionName, whenReads); err != nil {
			return optionValues{}, err
		}
	}
	return values, nil
}

func (o options) check(values map[string]string) error {
	for _, optionName := range slices.Sorted(maps.Keys(values)) {
		declared, ok := o.declared[optionName]
		if !ok {
			if len(o.declared) == 0 {
				return fmt.Errorf("unknown option %q: %s has no option", optionName, o.declarer)
			}
			return fmt.Errorf("unknown option %q: the options of %s are %s", optionName, o.declarer, strings.Join(slices.Sorted(maps.Keys(o.declared)), ", "))
		}
		if err := declared.check(values[optionName]); err != nil {
			return fmt.Errorf("%s: %w", o.declarer, err)
		}
	}
	return nil
}

func (o options) splitByWhen() (withoutWhen, withWhen []string) {
	for _, optionName := range slices.Sorted(maps.Keys(o.declared)) {
		if o.declared[optionName].hasWhen() {
			withWhen = append(withWhen, optionName)
		} else {
			withoutWhen = append(withoutWhen, optionName)
		}
	}
	return withoutWhen, withWhen
}

func (o options) questions(featureName string, values optionValues) []Question {
	var questions []Question
	for _, optionName := range slices.Sorted(maps.Keys(o.declared)) {
		if values.notApplying[optionName] {
			continue
		}
		questions = append(questions, Question{
			Feature:  featureName,
			Option:   optionName,
			Value:    values.byName[optionName],
			Source:   values.source(optionName),
			Given:    values.given[optionName],
			NeedOf:   slices.Clone(values.needOf[optionName]),
			Missing:  values.missing(optionName),
			Default:  values.defaults[optionName],
			Accepted: slices.Clone(o.declared[optionName].accepted),
			Pattern:  o.declared[optionName].patternText(),
			options:  o,
		})
	}
	return questions
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

func (o options) otherValuesOfOptionsWithValues(fixed map[string]string) []map[string]string {
	var others []map[string]string
	for _, optionName := range slices.Sorted(maps.Keys(o.declared)) {
		if _, ok := fixed[optionName]; ok {
			continue
		}
		declared := o.declared[optionName]
		for _, value := range declared.accepted {
			if value == declared.defaultValue {
				continue
			}
			values := o.defaultsOfOptionsWithValues()
			maps.Copy(values, fixed)
			values[optionName] = value
			others = append(others, values)
		}
	}
	return others
}
