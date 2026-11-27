package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/astgrep"
	"github.com/hpcsc/feature-compose/internal/indent"
)

const wholeMatch = "match"

type position struct {
	before bool
	name   string
}

func newPosition(after, before string) (position, error) {
	switch {
	case after != "" && before != "":
		return position{}, errors.New("has both after and before, and it must have one of them")
	case after != "":
		return position{name: after}, nil
	case before != "":
		return position{before: true, name: before}, nil
	}
	return position{}, errors.New("has neither after nor before, and it must have one of them")
}

func (p position) side() string {
	if p.before {
		return "before"
	}
	return "after"
}

func (p position) nodes(m astgrep.Match) []astgrep.Node {
	if p.name == wholeMatch {
		return []astgrep.Node{m.Node}
	}
	if n, ok := m.Single[p.name]; ok {
		return []astgrep.Node{n}
	}
	return m.Multi[p.name]
}

func (p position) inserted(contents string, m astgrep.Match, texts []string, separator string, comments []astgrep.Node) (string, error) {
	nodes := p.nodes(m)
	content := slices.DeleteFunc(slices.Clone(nodes), func(n astgrep.Node) bool { return separator != "" && n.Text == separator })
	if len(content) == 0 {
		return "", fmt.Errorf("%s matches no node, so the items cannot go next to it", p.name)
	}
	var inserted strings.Builder
	if p.before {
		next := content[0]
		prefix := p.indentOf(contents, next.Start)
		lineStart := strings.LastIndexByte(contents[:next.Start], '\n') + 1
		if strings.TrimSpace(contents[lineStart:next.Start]) != "" {
			for _, text := range texts {
				inserted.WriteString(strings.TrimPrefix(indent.Lines(text, prefix), prefix) + separator + "\n" + prefix)
			}
			return contents[:next.Start] + inserted.String() + contents[next.Start:], nil
		}
		for lineStart > 0 {
			above := strings.LastIndexByte(contents[:lineStart-1], '\n') + 1
			if !p.isCommentLine(contents, above, lineStart-1, comments) {
				break
			}
			lineStart = above
		}
		for _, text := range texts {
			inserted.WriteString(indent.Lines(text, prefix) + separator + "\n")
		}
		return contents[:lineStart] + inserted.String() + contents[lineStart:], nil
	}
	previous := content[len(content)-1]
	prefix := p.indentOf(contents, previous.Start)
	end := previous.Start + len(strings.TrimRight(contents[previous.Start:previous.End], " \t\n"))
	endsWithSeparator := separator != "" && nodes[len(nodes)-1].Text == separator
	if endsWithSeparator {
		end = nodes[len(nodes)-1].End
	}
	for _, text := range texts {
		if endsWithSeparator {
			inserted.WriteString("\n" + indent.Lines(text, prefix) + separator)
			continue
		}
		inserted.WriteString(separator + "\n" + indent.Lines(text, prefix))
	}
	return contents[:end] + inserted.String() + contents[end:], nil
}

func (p position) indentOf(contents string, offset int) string {
	lineStart := strings.LastIndexByte(contents[:offset], '\n') + 1
	line := contents[lineStart:offset]
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func (p position) isCommentLine(contents string, lineStart, lineEnd int, comments []astgrep.Node) bool {
	line := contents[lineStart:lineEnd]
	textStart := lineStart + len(line) - len(strings.TrimLeft(line, " \t"))
	textEnd := lineStart + len(strings.TrimRight(line, " \t\r"))
	if textStart >= textEnd {
		return false
	}
	return slices.ContainsFunc(comments, func(c astgrep.Node) bool { return c.Start <= textStart && c.End >= textEnd })
}
