package catalog

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
)

var operators = []string{"==", "!=", "&&", "||", "!", "(", ")"}

type expression struct {
	text     string
	operator string
	name     string
	value    string
	operands []expression
}

func newExpression(text string) (expression, error) {
	tokens, err := expression{}.tokens(text)
	if err != nil {
		return expression{}, err
	}
	parsed, rest, err := expression{}.parsedOr(tokens)
	if err != nil {
		return expression{}, err
	}
	if len(rest) > 0 {
		return expression{}, fmt.Errorf("%s comes after the end of the expression", rest[0])
	}
	parsed.text = text
	return parsed, nil
}

func (e expression) trueFor(values map[string]string) (bool, error) {
	switch e.operator {
	case "":
		return true, nil
	case "!":
		held, err := e.operands[0].trueFor(values)
		return !held, err
	case "&&", "||":
		left, err := e.operands[0].trueFor(values)
		if err != nil {
			return false, err
		}
		right, err := e.operands[1].trueFor(values)
		if err != nil {
			return false, err
		}
		if e.operator == "&&" {
			return left && right, nil
		}
		return left || right, nil
	}
	value, ok := values[e.name]
	if !ok {
		return false, fmt.Errorf("unknown name %q: the names that it can read are %s", e.name, strings.Join(slices.Sorted(maps.Keys(values)), ", "))
	}
	switch e.operator {
	case "==":
		return value == e.value, nil
	case "!=":
		return value != e.value, nil
	}
	return value == "true", nil
}

func (e expression) references() []string {
	var references []string
	for _, operand := range e.operands {
		for _, reference := range operand.references() {
			if !slices.Contains(references, reference) {
				references = append(references, reference)
			}
		}
	}
	if e.name != "" {
		references = append(references, e.name)
	}
	return references
}

func (e expression) tokens(rest string) ([]string, error) {
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	if rest == "" {
		return nil, nil
	}
	var next string
	switch operator := slices.IndexFunc(operators, func(operator string) bool { return strings.HasPrefix(rest, operator) }); {
	case strings.HasPrefix(rest, `"`):
		end := strings.Index(rest[1:], `"`)
		if end < 0 {
			return nil, fmt.Errorf("the value %s has no closing quote", rest)
		}
		next = rest[:end+2]
	case operator >= 0:
		next = operators[operator]
	default:
		next = rest[:len(rest)-len(strings.TrimLeftFunc(rest, e.isNameRune))]
		if next == "" {
			return nil, fmt.Errorf("%q is not a part of an expression", rest[:1])
		}
	}
	others, err := e.tokens(rest[len(next):])
	if err != nil {
		return nil, err
	}
	return append([]string{next}, others...), nil
}

func (e expression) isNameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '.' || r == '_'
}

func (e expression) isName(token string) bool {
	return e.isNameRune([]rune(token)[0])
}

func (e expression) parsedOr(tokens []string) (expression, []string, error) {
	left, rest, err := e.parsedAnd(tokens)
	if len(rest) == 0 || rest[0] != "||" {
		return left, rest, err
	}
	right, rest, err := e.parsedOr(rest[1:])
	return expression{operator: "||", operands: []expression{left, right}}, rest, err
}

func (e expression) parsedAnd(tokens []string) (expression, []string, error) {
	left, rest, err := e.parsedUnary(tokens)
	if len(rest) == 0 || rest[0] != "&&" {
		return left, rest, err
	}
	right, rest, err := e.parsedAnd(rest[1:])
	return expression{operator: "&&", operands: []expression{left, right}}, rest, err
}

func (e expression) parsedUnary(tokens []string) (expression, []string, error) {
	if len(tokens) == 0 {
		return expression{}, nil, errors.New("the expression ends where a name is expected")
	}
	switch {
	case tokens[0] == "!":
		operand, rest, err := e.parsedUnary(tokens[1:])
		return expression{operator: "!", operands: []expression{operand}}, rest, err
	case tokens[0] == "(":
		inner, rest, err := e.parsedOr(tokens[1:])
		if err != nil {
			return expression{}, nil, err
		}
		if len(rest) == 0 || rest[0] != ")" {
			return expression{}, nil, errors.New("a ( has no closing )")
		}
		return inner, rest[1:], nil
	case !e.isName(tokens[0]):
		return expression{}, nil, fmt.Errorf("%s is where a name is expected", tokens[0])
	case len(tokens) > 1 && (tokens[1] == "==" || tokens[1] == "!="):
		if len(tokens) < 3 || tokens[2][0] != '"' {
			return expression{}, nil, fmt.Errorf("%s %s needs a quoted value after it", tokens[0], tokens[1])
		}
		return expression{operator: tokens[1], name: tokens[0], value: tokens[2][1 : len(tokens[2])-1]}, tokens[3:], nil
	}
	return expression{operator: "name", name: tokens[0]}, tokens[1:], nil
}
