package catalog

import (
	"fmt"
	"slices"
	"strings"
)

type githubRepository struct {
	Account string
	Name    string
}

func parseGitHubRepository(module string) (githubRepository, error) {
	rest, onGitHub := strings.CutPrefix(module, "github.com/")
	segments := strings.Split(rest, "/")
	if !onGitHub || len(segments) < 2 || !githubRepositoryPart(segments[0]) || !githubRepositoryPart(segments[1]) {
		return githubRepository{}, fmt.Errorf("the module %q is not github.com/<account>/<repo>, so it has no GitHub repository", module)
	}
	return githubRepository{Account: segments[0], Name: segments[1]}, nil
}

func (r githubRepository) String() string {
	return r.Account + "/" + r.Name
}

func githubRepositoryPart(segment string) bool {
	return !slices.Contains([]string{"", ".", ".."}, segment)
}
