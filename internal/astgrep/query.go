package astgrep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

type Node struct {
	Text  string
	Start int
	End   int
}

type Match struct {
	Node
	Single map[string]Node
	Multi  map[string][]Node
}

type Query struct {
	Language string
	Pattern  string
	Rule     map[string]any
}

func (q Query) Empty() bool {
	return q.Pattern == "" && q.Rule == nil
}

func (q Query) Mistakes() []string {
	var mistakes []string
	if !q.Empty() && q.Language == "" {
		mistakes = append(mistakes, "has a pattern or a rule, and no language")
	}
	if q.Pattern != "" && q.Rule != nil {
		mistakes = append(mistakes, "has both a pattern and a rule, and it must have one of them")
	}
	return mistakes
}

func (q Query) Metavariables() []string {
	var names []string
	for _, pattern := range q.patterns() {
		for _, found := range metavariable.FindAllStringSubmatch(pattern, -1) {
			if !slices.Contains(names, found[1]) {
				names = append(names, found[1])
			}
		}
	}
	return names
}

func (q Query) rule() map[string]any {
	if q.Rule != nil {
		return q.Rule
	}
	return map[string]any{"pattern": q.Pattern}
}

func (q Query) inlineRule() (string, error) {
	inline, err := json.Marshal(map[string]any{"id": "feature-compose", "language": q.Language, "rule": q.rule()})
	if err != nil {
		return "", fmt.Errorf("write the ast-grep rule: %w", err)
	}
	return string(inline), nil
}

func (q Query) scan(ctx context.Context, inline string, input []byte) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ast-grep", "scan", "--stdin", "--json=compact", "--inline-rules", inline)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(input), &stdout, &stderr
	err := cmd.Run()
	if strings.Contains(stderr.String(), "Error:") {
		return nil, fmt.Errorf("ast-grep refuses the rule, or the language %q: %s", q.Language, firstLine(stderr.String()))
	}
	if err != nil {
		return nil, fmt.Errorf("ast-grep failed: %w\n%s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func (q Query) patterns() []string {
	var patterns []string
	var collect func(value any, key string)
	collect = func(value any, key string) {
		switch v := value.(type) {
		case string:
			if key == "pattern" || key == "context" {
				patterns = append(patterns, v)
			}
		case map[string]any:
			for _, inner := range slices.Sorted(maps.Keys(v)) {
				collect(v[inner], inner)
			}
		case []any:
			for _, inner := range v {
				collect(inner, key)
			}
		}
	}
	collect(q.rule(), "")
	return patterns
}

type foundNode struct {
	Text  string `json:"text"`
	Range struct {
		ByteOffset struct {
			Start int `json:"start"`
			End   int `json:"end"`
		} `json:"byteOffset"`
	} `json:"range"`
}

func (n foundNode) node() Node {
	return Node{Text: n.Text, Start: n.Range.ByteOffset.Start, End: n.Range.ByteOffset.End}
}

type foundMatch struct {
	foundNode
	MetaVariables struct {
		Single map[string]foundNode   `json:"single"`
		Multi  map[string][]foundNode `json:"multi"`
	} `json:"metaVariables"`
}

func (f foundMatch) match() Match {
	m := Match{Node: f.node(), Single: map[string]Node{}, Multi: map[string][]Node{}}
	for name, n := range f.MetaVariables.Single {
		m.Single[name] = n.node()
	}
	for name, nodes := range f.MetaVariables.Multi {
		for _, n := range nodes {
			m.Multi[name] = append(m.Multi[name], n.node())
		}
	}
	return m
}

var metavariable = regexp.MustCompile(`\$(?:\$\$)?([A-Z_][A-Z0-9_]*)`)

var commentKinds = []string{"comment", "line_comment", "block_comment"}

func Find(ctx context.Context, file string, q Query) ([]Match, error) {
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	inline, err := q.inlineRule()
	if err != nil {
		return nil, err
	}
	stdout, err := q.scan(ctx, inline, contents)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	var found []foundMatch
	if err := json.Unmarshal(stdout, &found); err != nil {
		return nil, fmt.Errorf("read the output of ast-grep for %s: %w", file, err)
	}
	matches := make([]Match, 0, len(found))
	for _, f := range found {
		matches = append(matches, f.match())
	}
	return matches, nil
}

func Check(ctx context.Context, q Query) error {
	inline, err := q.inlineRule()
	if err != nil {
		return err
	}
	if _, err := q.scan(ctx, inline, nil); err != nil {
		return err
	}
	for _, pattern := range q.patterns() {
		cmd := exec.CommandContext(ctx, "ast-grep", "run", "--stdin", "--lang", q.Language, "--pattern", pattern, "--debug-query=pattern")
		cmd.Stdin = strings.NewReader("")
		out, _ := cmd.CombinedOutput()
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "ERROR") {
				return fmt.Errorf("the pattern %q is not valid %s: ast-grep reads a part of it as an ERROR node, so it matches no place", pattern, q.Language)
			}
		}
	}
	return nil
}

func Comments(ctx context.Context, file, language string) []Node {
	var comments []Node
	for _, kind := range commentKinds {
		matches, err := Find(ctx, file, Query{Language: language, Rule: map[string]any{"kind": kind}})
		if err != nil {
			continue
		}
		for _, m := range matches {
			comments = append(comments, m.Node)
		}
	}
	return comments
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return line
}
