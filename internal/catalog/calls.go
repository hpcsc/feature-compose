package catalog

import (
	"slices"
	"strings"
	texttemplate "text/template"
	"text/template/parse"
)

type calls struct {
	trees []*parse.Tree
}

func (c calls) with(label, text string, d delimiters) (calls, error) {
	parsed, err := texttemplate.New(label).Delims(d.left, d.right).Funcs(Selection{}.funcs("")).Parse(text)
	if err != nil {
		return calls{}, err
	}
	templates := parsed.Templates()
	slices.SortFunc(templates, func(a, b *texttemplate.Template) int { return strings.Compare(a.Name(), b.Name()) })
	trees := slices.Clone(c.trees)
	for _, t := range templates {
		trees = append(trees, t.Tree)
	}
	return calls{trees: trees}, nil
}

func (c calls) argumentsOf(function string) []string {
	var arguments []string
	for _, tree := range c.trees {
		arguments = c.argumentsIn(tree.Root, function, arguments)
	}
	return arguments
}

func (c calls) argumentsIn(node parse.Node, function string, arguments []string) []string {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return arguments
		}
		for _, child := range n.Nodes {
			arguments = c.argumentsIn(child, function, arguments)
		}
	case *parse.ActionNode:
		return c.argumentsIn(n.Pipe, function, arguments)
	case *parse.IfNode:
		return c.argumentsInBranch(n.BranchNode, function, arguments)
	case *parse.RangeNode:
		return c.argumentsInBranch(n.BranchNode, function, arguments)
	case *parse.WithNode:
		return c.argumentsInBranch(n.BranchNode, function, arguments)
	case *parse.TemplateNode:
		return c.argumentsIn(n.Pipe, function, arguments)
	case *parse.PipeNode:
		if n == nil {
			return arguments
		}
		for _, command := range n.Cmds {
			arguments = c.argumentsIn(command, function, arguments)
		}
	case *parse.CommandNode:
		arguments = c.argumentOfCall(n, function, arguments)
		for _, argument := range n.Args {
			arguments = c.argumentsIn(argument, function, arguments)
		}
	case *parse.ChainNode:
		return c.argumentsIn(n.Node, function, arguments)
	}
	return arguments
}

func (c calls) argumentsInBranch(branch parse.BranchNode, function string, arguments []string) []string {
	arguments = c.argumentsIn(branch.Pipe, function, arguments)
	arguments = c.argumentsIn(branch.List, function, arguments)
	return c.argumentsIn(branch.ElseList, function, arguments)
}

func (c calls) argumentOfCall(command *parse.CommandNode, function string, arguments []string) []string {
	if len(command.Args) < 2 {
		return arguments
	}
	called, isIdentifier := command.Args[0].(*parse.IdentifierNode)
	argument, isText := command.Args[1].(*parse.StringNode)
	if isIdentifier && isText && called.Ident == function && !slices.Contains(arguments, argument.Text) {
		arguments = append(arguments, argument.Text)
	}
	return arguments
}
