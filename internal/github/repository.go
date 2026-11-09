package github

import (
	"fmt"
	"slices"
	"strings"
)

type Repository struct {
	Account string
	Name    string
}

func RepositoryOf(module string) (Repository, error) {
	rest, onGitHub := strings.CutPrefix(module, "github.com/")
	segments := strings.Split(rest, "/")
	if !onGitHub || len(segments) < 2 || !isPart(segments[0]) || !isPart(segments[1]) {
		return Repository{}, fmt.Errorf("the module %q is not github.com/<account>/<repo>, so it has no GitHub repository", module)
	}
	return Repository{Account: segments[0], Name: segments[1]}, nil
}

func (r Repository) String() string {
	return r.Account + "/" + r.Name
}

func isPart(segment string) bool {
	return !slices.Contains([]string{"", ".", ".."}, segment)
}
