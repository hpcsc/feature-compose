//go:build unit

package index_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/hpcsc/feature-compose/internal/index"
	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	served := func(t *testing.T, status int, contents string) string {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(contents))
		}))
		t.Cleanup(server.Close)
		return server.URL + "/catalogs.yml"
	}

	t.Run("fetch", func(t *testing.T) {
		t.Run("gives each catalog of the index with its summary, source, path and ref", func(t *testing.T) {
			url := served(t, http.StatusOK, "catalogs:\n  hpcsc:\n    summary: The catalog of feature-compose.\n    source: https://github.com/hpcsc/feature-compose\n    path: catalog\n  team:\n    summary: The catalog of the team.\n    source: git@github.com:team/catalog.git\n    ref: v1.0.0\n")

			known, err := index.Fetch(context.Background(), http.DefaultClient, url)
			require.NoError(t, err)
			hpcsc, err := known.Entry("hpcsc")
			require.NoError(t, err)
			team, err := known.Entry("team")
			require.NoError(t, err)

			require.Equal(t, []string{"hpcsc", "team"}, known.Names())
			require.Equal(t, []index.Entry{
				{Summary: "The catalog of feature-compose.", Source: "https://github.com/hpcsc/feature-compose", Path: "catalog"},
				{Summary: "The catalog of the team.", Source: "git@github.com:team/catalog.git", Ref: "v1.0.0"},
			}, []index.Entry{hpcsc, team})
		})

		t.Run("a status that is not 200 stops, and the message names the URL and the status", func(t *testing.T) {
			url := served(t, http.StatusNotFound, "")

			_, err := index.Fetch(context.Background(), http.DefaultClient, url)

			require.EqualError(t, err, "get the index of known catalogs from "+url+": 404 Not Found")
		})

		t.Run("a catalog with no source stops, and the message names the URL and the catalog", func(t *testing.T) {
			url := served(t, http.StatusOK, "catalogs:\n  team:\n    summary: The catalog of the team.\n")

			_, err := index.Fetch(context.Background(), http.DefaultClient, url)

			require.EqualError(t, err, "read the index of known catalogs from "+url+": the catalog team has no source")
		})

		t.Run("a control character in a value stops, so an entry cannot move the cursor or change the colours of the terminal", func(t *testing.T) {
			url := served(t, http.StatusOK, "catalogs:\n  team:\n    summary: The catalog of the team.\n    source: \"https://example.com/\\e[31mteam.git\"\n")

			_, err := index.Fetch(context.Background(), http.DefaultClient, url)

			require.EqualError(t, err, "read the index of known catalogs from "+url+": the source of the catalog team holds a control character")
		})

		t.Run("a server that does not answer stops, and the message names the URL", func(t *testing.T) {
			server := httptest.NewServer(http.NotFoundHandler())
			url := server.URL + "/catalogs.yml"
			server.Close()

			_, err := index.Fetch(context.Background(), http.DefaultClient, url)

			require.ErrorContains(t, err, "get the index of known catalogs from "+url+": ")
		})

		t.Run("a name that is not a catalog name stops", func(t *testing.T) {
			url := served(t, http.StatusOK, "catalogs:\n  team/a:\n    source: https://example.com/a.git\n")

			_, err := index.Fetch(context.Background(), http.DefaultClient, url)

			require.ErrorContains(t, err, `read the index of known catalogs from `+url+`: the name "team/a" of the catalog`)
		})
	})

	t.Run("entry", func(t *testing.T) {
		t.Run("an unknown name stops, and the message lists the names and the command with a source", func(t *testing.T) {
			url := served(t, http.StatusOK, "catalogs:\n  hpcsc:\n    source: https://github.com/hpcsc/feature-compose\n")
			known, err := index.Fetch(context.Background(), http.DefaultClient, url)
			require.NoError(t, err)

			_, err = known.Entry("nope")

			require.EqualError(t, err, "the index of known catalogs at "+url+" has no catalog nope: it has hpcsc. Give the source with catalog add nope <source>")
		})
	})

	t.Run("the index of this repo", func(t *testing.T) {
		const repoFiles = "https://raw.githubusercontent.com/hpcsc/feature-compose/main/"
		repoIndex := func(t *testing.T) index.File {
			t.Helper()
			require.True(t, strings.HasPrefix(index.DefaultURL, repoFiles), index.DefaultURL)
			contents, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(strings.TrimPrefix(index.DefaultURL, repoFiles))))
			require.NoError(t, err)
			known, err := index.Fetch(context.Background(), http.DefaultClient, served(t, http.StatusOK, string(contents)))
			require.NoError(t, err)
			return known
		}

		t.Run("is the file of this repo that the default URL names, and it names the catalog of this repo at a path that holds the folder starters", func(t *testing.T) {
			hpcsc, err := repoIndex(t).Entry("hpcsc")

			require.NoError(t, err)
			require.Equal(t, config.Entry{Source: "https://github.com/hpcsc/feature-compose", Path: "catalog"}, hpcsc.ConfigEntry())
			require.DirExists(t, filepath.Join("..", "..", hpcsc.Path, "starters"))
		})

		t.Run("has each catalog that the README adds with no source", func(t *testing.T) {
			readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
			require.NoError(t, err)
			added := regexp.MustCompile(`(?m)^\s*feature-compose catalog add ([A-Za-z0-9._-]+)\s*$`).FindAllStringSubmatch(string(readme), -1)
			require.NotEmpty(t, added)

			for _, match := range added {
				_, err := repoIndex(t).Entry(match[1])
				require.NoError(t, err)
			}
		})
	})
}
