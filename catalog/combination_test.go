//go:build combinations

package catalog_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/hpcsc/feature-compose/catalog"
	"github.com/stretchr/testify/require"
)

func TestCombination(t *testing.T) {
	t.Run("the repo of each combination of the catalog in the binary passes its after_render and test commands", func(t *testing.T) {
		combinations, err := catalog.Embedded().Combinations()
		require.NoError(t, err)

		for _, combination := range combinations {
			t.Run(combination.String(), func(t *testing.T) {
				folder, err := filepath.EvalSymlinks(t.TempDir())
				require.NoError(t, err)
				t.Chdir(folder)
				// mise refuses the mise.toml of the repo until the real path of its folder is trusted
				t.Setenv("MISE_TRUSTED_CONFIG_PATHS", folder)
				output := filepath.Join(folder, "my-service")
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
				defer cancel()
				starter, err := catalog.Embedded().Starter(combination.StarterName)
				require.NoError(t, err)

				selection, err := starter.Select("my-service", combination.StarterValues, combination.FeatureValues)
				require.NoError(t, err, combination.String())
				require.NoError(t, selection.Write(output), combination.String())
				require.NoError(t, selection.RunAfterRender(ctx, output), combination.String())
				require.NoError(t, selection.RunTests(ctx, output), combination.String())
			})
		}
	})
}
