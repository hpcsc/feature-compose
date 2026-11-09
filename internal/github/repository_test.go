//go:build unit

package github_test

import (
	"strconv"
	"testing"

	"github.com/hpcsc/feature-compose/internal/github"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	t.Run("of", func(t *testing.T) {
		for _, module := range []string{"github.com/acme/ledger", "github.com/acme/ledger/tools"} {
			t.Run("the module "+module+" gives the account and the repository", func(t *testing.T) {
				repository, err := github.RepositoryOf(module)

				require.NoError(t, err)
				require.Equal(t, github.Repository{Account: "acme", Name: "ledger"}, repository)
				require.Equal(t, "acme/ledger", repository.String())
			})
		}

		for _, module := range []string{"example.com/acme/ledger", "github.com/acme", "github.com//ledger", "github.com/acme/", "github.com/acme/..", "github.com/./ledger"} {
			t.Run("the module "+module+" has no repository, and the message names the module", func(t *testing.T) {
				_, err := github.RepositoryOf(module)

				require.ErrorContains(t, err, strconv.Quote(module))
				require.ErrorContains(t, err, "GitHub")
			})
		}
	})
}
