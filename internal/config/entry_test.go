//go:build unit

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/stretchr/testify/require"
)

func TestEntry(t *testing.T) {
	t.Run("is git URL", func(t *testing.T) {
		for source, isGitURL := range map[string]bool{
			"https://github.com/hpcsc/feature-compose": true,
			"ssh://git@github.com/team/catalog.git":    true,
			"file:///srv/git/catalog.git":              true,
			"git@github.com:team/catalog.git":          true,
			"/Users/me/code/team-catalog":              false,
			"~/code/team-catalog":                      false,
			"code/team:catalog":                        false,
			"catalog":                                  false,
		} {
			t.Run(source, func(t *testing.T) {
				require.Equal(t, isGitURL, config.Entry{Source: source}.IsGitURL())
			})
		}
	})

	t.Run("source without credentials", func(t *testing.T) {
		for source, shown := range map[string]string{
			"https://x-access-token:secret@github.com/team/catalog.git": "https://github.com/team/catalog.git",
			"https://secret@github.com/team/catalog.git":                "https://github.com/team/catalog.git",
			"https://user:se%zzcret@github.com/team/catalog.git":        "https://github.com/team/catalog.git",
			"ssh://git:secret@github.com/team/catalog.git":              "ssh://git@github.com/team/catalog.git",
			"git@github.com:team/catalog.git":                           "git@github.com:team/catalog.git",
			"https://github.com/team/catalog.git":                       "https://github.com/team/catalog.git",
			"/code/team-catalog":                                        "/code/team-catalog",
		} {
			t.Run(source+" shows as "+shown, func(t *testing.T) {
				require.Equal(t, shown, config.Entry{Source: source}.SourceWithoutCredentials())
			})
		}
	})

	t.Run("with absolute source", func(t *testing.T) {
		t.Run("a relative folder is relative to the working folder", func(t *testing.T) {
			work := t.TempDir()
			t.Chdir(work)

			entry, err := config.Entry{Source: "team-catalog"}.WithAbsoluteSource()

			require.NoError(t, err)
			require.Equal(t, filepath.Join(work, "team-catalog"), entry.Source)
		})

		for _, source := range []string{"", "~/code/team-catalog", "/code/team-catalog", "https://github.com/team/catalog.git"} {
			t.Run("the source "+source+" stays as it is", func(t *testing.T) {
				t.Chdir(t.TempDir())

				entry, err := config.Entry{Source: source}.WithAbsoluteSource()

				require.NoError(t, err)
				require.Equal(t, source, entry.Source)
			})
		}
	})

	t.Run("check catalog in", func(t *testing.T) {
		t.Run("a path with the folder starters has a catalog", func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(root, "catalog", "starters"), 0o755))

			require.NoError(t, config.Entry{Source: "/code/a", Path: "catalog"}.CheckCatalogIn(root))
		})

		t.Run("a path with no folder starters stops, and the message names the source with no credentials, the path and --path", func(t *testing.T) {
			err := config.Entry{Source: "https://user:secret@example.com/a.git", Path: "catalog"}.CheckCatalogIn(t.TempDir())

			require.EqualError(t, err, "the source https://example.com/a.git has no catalog at the path catalog, because that path has no folder starters: give the folder of the catalog in the source with --path")
		})
	})
}
