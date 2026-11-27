//go:build unit

package catalog_test

import (
	"maps"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestDefinitions(t *testing.T) {
	definitionsOf := func(t *testing.T) catalog.Definitions {
		t.Helper()
		files := fstest.MapFS{
			"starters/api/starter.yml":    &fstest.MapFile{Data: []byte("summary: An API.\n")},
			"starters/worker/starter.yml": &fstest.MapFile{Data: []byte("summary: A worker.\n")},
			"features/cli/feature.yml":    &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [worker, api]\n")},
			"features/auth/feature.yml":   &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\n")},
			"features/tools/feature.yml":  &fstest.MapFile{Data: []byte("summary: Tools.\ntargets: [worker]\n")},
		}
		definitions, err := catalog.New(files).Definitions()
		require.NoError(t, err)
		return definitions
	}

	t.Run("for starter", func(t *testing.T) {
		t.Run("holds only that starter and the features whose targets name it", func(t *testing.T) {
			worker, err := definitionsOf(t).ForStarter("worker")

			require.NoError(t, err)
			require.Equal(t, []string{"worker"}, slices.Sorted(maps.Keys(worker.Starters)))
			require.Equal(t, "A worker.", worker.Starters["worker"].Summary)
			require.Equal(t, []string{"cli", "tools"}, slices.Sorted(maps.Keys(worker.Features)))
		})

		t.Run("a short name that one catalog has holds the starter of that full name and the features that target it", func(t *testing.T) {
			definitions, err := catalog.NewCatalogs(
				catalog.FromOrigin(fstest.MapFS{"starters/api/starter.yml": &fstest.MapFile{Data: []byte("summary: An API.\n")}, "features/auth/feature.yml": &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\n")}}, catalog.Origin{Name: "hpcsc"}),
				catalog.FromOrigin(fstest.MapFS{"starters/worker/starter.yml": &fstest.MapFile{Data: []byte("summary: A worker.\n")}}, catalog.Origin{Name: "team"}),
			).Definitions()
			require.NoError(t, err)

			api, err := definitions.ForStarter("api")

			require.NoError(t, err)
			require.Equal(t, [][]string{{"hpcsc/api"}, {"hpcsc/auth"}}, [][]string{slices.Sorted(maps.Keys(api.Starters)), slices.Sorted(maps.Keys(api.Features))})
		})

		t.Run("an unknown starter is an error that lists the starters of the catalog", func(t *testing.T) {
			_, err := definitionsOf(t).ForStarter("nope")

			require.EqualError(t, err, `unknown starter "nope": the catalog has api, worker`)
		})
	})
}
