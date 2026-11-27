//go:build unit

package astgrep_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/feature-compose/internal/astgrep"
	"github.com/stretchr/testify/require"
)

func TestASTGrep(t *testing.T) {
	withVersion := func(t *testing.T, printed string) {
		t.Helper()
		bin := t.TempDir()
		script := "#!/bin/sh\necho '" + printed + "'\n"
		require.NoError(t, os.WriteFile(filepath.Join(bin, "ast-grep"), []byte(script), 0o755))
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	t.Run("installed", func(t *testing.T) {
		t.Run("refuses a version older than the minimum, and the message names both versions", func(t *testing.T) {
			withVersion(t, "ast-grep 0.45.2")

			err := astgrep.Installed()

			require.EqualError(t, err, "ast-grep is version 0.45.2, and feature-compose needs version 0.45.3 or later")
		})

		t.Run("accepts the minimum version, and a later minor version whose patch number is lower than the patch number of the minimum", func(t *testing.T) {
			withVersion(t, "ast-grep 0.45.3")
			minimum := astgrep.Installed()
			withVersion(t, "ast-grep 0.51.0")
			later := astgrep.Installed()

			require.Equal(t, []error{nil, nil}, []error{minimum, later})
		})

		t.Run("refuses a version that it cannot read, and the message names the version", func(t *testing.T) {
			withVersion(t, "ast-grep nightly")

			err := astgrep.Installed()

			require.EqualError(t, err, `read the version "nightly" of ast-grep: it is not three numbers with dots between them`)
		})

		t.Run("refuses a PATH without ast-grep, and the message tells where to get it", func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())

			err := astgrep.Installed()

			require.ErrorContains(t, err, "install it from https://ast-grep.github.io")
		})
	})
}
