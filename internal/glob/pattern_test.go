//go:build unit

package glob_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/glob"
	"github.com/stretchr/testify/require"
)

func TestPattern(t *testing.T) {
	t.Run("compile", func(t *testing.T) {
		for pattern, message := range map[string]string{
			"go//Taskfile.yml": `the glob "go//Taskfile.yml" has an empty segment`,
			"go/[a-/x.yml":     `the glob "go/[a-/x.yml" is not valid`,
		} {
			t.Run("the pattern "+pattern+" stops, and the message names it", func(t *testing.T) {
				_, err := glob.Compile(pattern)

				require.ErrorContains(t, err, message)
			})
		}
	})

	t.Run("match", func(t *testing.T) {
		for slashPath, matches := range map[string]bool{
			"go/Taskfile.yml":           true,
			"go/a/b/Taskfile.test.yml":  true,
			"web/Taskfile.yml":          false,
			"go/a/b/Makefile":           false,
			"go/a/b/Taskfile.yml/other": false,
		} {
			t.Run("go/**/Taskfile*.yml matches the path "+slashPath+" only when ** stands for any number of whole segments", func(t *testing.T) {
				pattern, err := glob.Compile("go/**/Taskfile*.yml")
				require.NoError(t, err)

				require.Equal(t, matches, pattern.Match(slashPath))
			})
		}

		t.Run("** at the end matches the rest of the path", func(t *testing.T) {
			pattern, err := glob.Compile("go/.github/**")
			require.NoError(t, err)

			require.Equal(t, []bool{true, false}, []bool{pattern.Match("go/.github/workflows/ci.yml"), pattern.Match("web/.github/ci.yml")})
		})

		t.Run("* matches within one segment, and not across a /", func(t *testing.T) {
			pattern, err := glob.Compile("go/*.yml")
			require.NoError(t, err)

			require.Equal(t, []bool{true, false}, []bool{pattern.Match("go/ci.yml"), pattern.Match("go/a/ci.yml")})
		})
	})
}
