package catalog

import (
	"fmt"
	"strings"
)

type Question struct {
	Option   string
	Default  string
	Accepted []string
	feature  string
	options  options
}

func (q Question) Declarer() string {
	return q.options.declarer
}

func (q Question) Check(value string) error {
	if err := q.options.check(map[string]string{q.Option: value}); err != nil {
		return err
	}
	if value == "" && q.Default == "" {
		return fmt.Errorf("the option %s of %s has no default, so it needs a value", q.Option, q.options.declarer)
	}
	if q.feature != "" && strings.Contains(value, ",") {
		return fmt.Errorf("the value %q of the option %s of %s holds a comma, which --feature cannot give", value, q.Option, q.options.declarer)
	}
	return nil
}
