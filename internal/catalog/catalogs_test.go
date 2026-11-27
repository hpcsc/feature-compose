//go:build unit

package catalog_test

import (
	"context"
	"maps"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestCatalogs(t *testing.T) {
	named := func(name string, files map[string]string) catalog.Catalog {
		fsys := fstest.MapFS{}
		for filePath, contents := range files {
			fsys[filePath] = &fstest.MapFile{Data: []byte(contents)}
		}
		return catalog.FromOrigin(fsys, catalog.Origin{Name: name, Source: "https://example.com/" + name + ".git", Ref: "main", Commit: "0123abc"})
	}
	twoCatalogs := func() catalog.Catalogs {
		return catalog.NewCatalogs(
			named("hpcsc", map[string]string{
				"starters/service/starter.yml":        "summary: A service.\nfolders:\n  app:\n    path: .\n",
				"starters/service/files/app/go.mod":   "module {{ .Name }}\n",
				"starters/worker/starter.yml":         "summary: A worker.\nfolders:\n  app:\n    path: .\n",
				"starters/worker/files/app/README.md": "# {{ .Name }}\n",
				"features/db/feature.yml":             "summary: A database.\ntargets: [service, worker]\n",
				"features/cli/feature.yml":            "summary: A CLI.\ntargets: [service]\nneeds: [db]\n",
			}),
			named("team", map[string]string{
				"starters/worker/starter.yml":         "summary: The worker of the team.\nfolders:\n  app:\n    path: .\n",
				"starters/worker/files/app/README.md": "# {{ .Name }} of the team\n",
				"features/cli/feature.yml":            "summary: The CLI of the team.\ntargets: [worker]\n",
			}),
		)
	}

	t.Run("starter", func(t *testing.T) {
		t.Run("a full name finds the starter of that catalog", func(t *testing.T) {
			starter, err := twoCatalogs().Starter("team/worker")

			require.NoError(t, err)
			require.Equal(t, "team/worker", starter.FullName())
		})

		t.Run("a short name that one catalog has finds the starter of that catalog", func(t *testing.T) {
			starter, err := twoCatalogs().Starter("service")

			require.NoError(t, err)
			require.Equal(t, "hpcsc/service", starter.FullName())
		})

		t.Run("a short name that two catalogs have stops, and the message lists the full names", func(t *testing.T) {
			_, err := twoCatalogs().Starter("worker")

			require.EqualError(t, err, `more than one catalog has the starter "worker": give its full name, one of hpcsc/worker, team/worker`)
		})

		t.Run("an unknown name stops, and the message lists the full name of each starter", func(t *testing.T) {
			_, err := twoCatalogs().Starter("nope")

			require.EqualError(t, err, `unknown starter "nope": the catalogs have hpcsc/service, hpcsc/worker, team/worker`)
		})

		t.Run("the full name of a starter of a catalog with no name is its short name", func(t *testing.T) {
			catalogs := catalog.NewCatalogs(catalog.New(fstest.MapFS{
				"starters/service/starter.yml":         &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  app:\n    path: .\n")},
				"starters/service/files/app/README.md": &fstest.MapFile{Data: []byte("# {{ .Name }}\n")},
			}))

			starter, err := catalogs.Starter("service")

			require.NoError(t, err)
			require.Equal(t, "service", starter.FullName())
		})

		t.Run("a named catalog with no folder starters stops the search, and the message names the catalog", func(t *testing.T) {
			catalogs := catalog.NewCatalogs(
				named("hpcsc", map[string]string{"starters/service/starter.yml": "summary: A service.\n"}),
				named("team", map[string]string{"features/cli/feature.yml": "targets: [worker]\n"}),
			)

			_, err := catalogs.Starter("service")

			require.ErrorContains(t, err, "the catalog team: read the starters of the catalog")
		})

		t.Run("an error of a named catalog starts with the name of the catalog", func(t *testing.T) {
			catalogs := catalog.NewCatalogs(named("team", map[string]string{"starters/worker/starter.yml": "folders: [app\n"}))

			_, err := catalogs.Starter("worker")

			require.ErrorContains(t, err, "the catalog team: read starters/worker/starter.yml")
		})

		t.Run("the features of a starter come only from its catalog, also when a feature of another catalog targets a starter with the same name", func(t *testing.T) {
			starter, err := twoCatalogs().Starter("team/worker")
			require.NoError(t, err)

			featureNames, err := starter.FeatureNames()

			require.NoError(t, err)
			require.Equal(t, []string{"cli"}, featureNames)
			require.Equal(t, "team/cli", starter.FullFeatureName("cli"))
		})
	})

	t.Run("definitions", func(t *testing.T) {
		t.Run("give the full name of each starter, feature, target and need", func(t *testing.T) {
			definitions, err := twoCatalogs().Definitions()

			require.NoError(t, err)
			require.Equal(t, []string{"hpcsc/service", "hpcsc/worker", "team/worker"}, slices.Sorted(maps.Keys(definitions.Starters)))
			require.Equal(t, []string{"hpcsc/cli", "hpcsc/db", "team/cli"}, slices.Sorted(maps.Keys(definitions.Features)))
			require.Equal(t, []string{"hpcsc/service", "hpcsc/worker"}, definitions.Features["hpcsc/db"].Targets)
			require.Equal(t, []string{"hpcsc/db"}, slices.Sorted(maps.Keys(definitions.Features["hpcsc/cli"].Needs)))
		})

		t.Run("an error of a named catalog starts with the name of the catalog", func(t *testing.T) {
			_, err := catalog.NewCatalogs(named("team", map[string]string{"starters/worker/starter.yml": "folders: [app\n"})).Definitions()

			require.ErrorContains(t, err, "the catalog team: read starters/worker/starter.yml")
		})
	})

	t.Run("check", func(t *testing.T) {
		t.Run("puts the name of the catalog before each mistake of that catalog", func(t *testing.T) {
			catalogs := catalog.NewCatalogs(
				named("hpcsc", map[string]string{
					"starters/service/starter.yml":         "summary: A service.\nfolders:\n  app:\n    path: .\n",
					"starters/service/files/app/README.md": "# {{ .Name }}\n",
				}),
				named("team", map[string]string{
					"starters/worker/starter.yml":         "summary: A worker.\nfolders:\n  app:\n    path: .\n",
					"starters/worker/files/app/README.md": "# {{ .Name }}\n",
					"features/cli/feature.yml":            "summary: A CLI.\ntargets: [worker]\nneeds: [vault]\n",
				}),
			)

			mistakes, err := catalogs.Check(context.Background())

			require.NoError(t, err)
			require.Equal(t, []string{"the catalog team: the starter worker: the feature cli needs vault, which is not a feature of the catalog"}, mistakes)
		})

		t.Run("an error of a named catalog starts with the name of the catalog", func(t *testing.T) {
			_, err := catalog.NewCatalogs(named("team", map[string]string{"starters/worker/starter.yml": "folders: [app\n"})).Check(context.Background())

			require.ErrorContains(t, err, "the catalog team: read starters/worker/starter.yml")
		})
	})

	t.Run("combinations", func(t *testing.T) {
		t.Run("the text of each combination starts with the full name of its starter", func(t *testing.T) {
			combinations, err := twoCatalogs().Combinations()
			require.NoError(t, err)

			var texts []string
			for _, combination := range combinations {
				texts = append(texts, combination.String())
			}

			require.Equal(t, []string{"hpcsc/service", "hpcsc/service --feature cli", "hpcsc/service --feature db", "hpcsc/worker", "hpcsc/worker --feature db", "team/worker", "team/worker --feature cli"}, texts)
		})

		t.Run("test makes the repo of a combination from the catalog that it names", func(t *testing.T) {
			catalogs := twoCatalogs()
			combinations, err := catalogs.Combinations()
			require.NoError(t, err)
			teamWorker := combinations[slices.IndexFunc(combinations, func(c catalog.Combination) bool { return c.String() == "team/worker" })]
			output := filepath.Join(t.TempDir(), "my-repo")

			err = catalogs.TestCombination(context.Background(), teamWorker, "my-repo", output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"README.md": "# my-repo of the team\n"}, filesIn(t, output))
		})

		t.Run("an error of a named catalog starts with the name of the catalog", func(t *testing.T) {
			_, err := catalog.NewCatalogs(named("team", map[string]string{"starters/worker/starter.yml": "folders: [app\n"})).Combinations()

			require.ErrorContains(t, err, "the catalog team: read starters/worker/starter.yml")
		})
	})
}
