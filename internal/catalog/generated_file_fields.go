package catalog

import (
	"fmt"
	"slices"

	"github.com/hpcsc/feature-compose/internal/astgrep"
)

type generatedFileFields struct {
	query     astgrep.Query
	after     string
	before    string
	separator string
	insert    string
}

func (g generatedFileFields) isSet() bool {
	return !g.query.Empty()
}

func (g generatedFileFields) needsComments() bool {
	return g.before != ""
}

func (g generatedFileFields) mistakes(starterFolders folders, file string) []string {
	var mistakes []string
	if !starterFolders.generates(file) {
		mistakes = append(mistakes, fmt.Sprintf("is in the file %q, which is not in a folder with a generate command", file))
	}
	mistakes = append(mistakes, g.query.Mistakes()...)
	if g.insert == "" {
		mistakes = append(mistakes, "has no insert")
	}
	p, err := newPosition(g.after, g.before)
	if err != nil {
		return append(mistakes, err.Error())
	}
	if p.name != wholeMatch && !slices.Contains(g.query.Metavariables(), p.name) {
		mistakes = append(mistakes, fmt.Sprintf("puts its items %s %s, which is not a metavariable of its pattern or its rule, and is not %s", p.side(), p.name, wholeMatch))
	}
	return mistakes
}

func (g generatedFileFields) unusedMistakes() []string {
	if g.isSet() || (g.after == "" && g.before == "" && g.separator == "" && g.insert == "") {
		return nil
	}
	return []string{"has after, before, separator or insert, which only an extension point with a pattern or a rule uses"}
}

func (g generatedFileFields) inserted(contents string, m astgrep.Match, texts []string, comments []astgrep.Node) (string, error) {
	p, err := newPosition(g.after, g.before)
	if err != nil {
		return "", err
	}
	return p.inserted(contents, m, texts, g.separator, comments)
}
