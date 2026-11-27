package catalog

import (
	"fmt"
	"strings"

	"github.com/hpcsc/feature-compose/internal/astgrep"
)

type removal struct {
	file  string
	query astgrep.Query
}

func newRemoval(declared RemovalDefinition) removal {
	return removal{file: declared.File, query: declared.query()}
}

func (r removal) mistakes(starterFolders folders) []string {
	var mistakes []string
	if !starterFolders.generates(r.file) {
		mistakes = append(mistakes, fmt.Sprintf("names the file %q, which is not in a folder with a generate command", r.file))
	}
	return append(mistakes, r.query.Mistakes()...)
}

func (r removal) removedFrom(contents string, m astgrep.Match) string {
	lineStart := strings.LastIndexByte(contents[:m.Start], '\n') + 1
	lineEnd := len(contents)
	if i := strings.IndexByte(contents[m.End:], '\n'); i >= 0 {
		lineEnd = m.End + i
	}
	if strings.TrimSpace(contents[lineStart:m.Start]+contents[m.End:lineEnd]) != "" {
		return contents[:m.Start] + contents[m.End:]
	}
	if lineEnd < len(contents) {
		lineEnd++
	}
	return contents[:lineStart] + contents[lineEnd:]
}
