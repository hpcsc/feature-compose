package catalog

import (
	"fmt"
	"strings"
)

type Question struct {
	Feature  string
	Option   string
	Value    string
	Source   string
	Given    bool
	Missing  bool
	Default  string
	Accepted []string
	Pattern  string
	options  options
}

func (q Question) Check(value string) error {
	if err := q.options.check(map[string]string{q.Option: value}); err != nil {
		return err
	}
	if value == "" && q.Default == "" {
		return fmt.Errorf("the option %s of %s has no default, so it needs a value", q.Option, q.options.declarer)
	}
	if q.Feature != "" && strings.Contains(value, ",") {
		return fmt.Errorf("the value %q of the option %s of %s holds a comma, which --feature cannot give", value, q.Option, q.options.declarer)
	}
	return nil
}
