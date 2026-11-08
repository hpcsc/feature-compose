//go:build unit

package cmd

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/hpcsc/feature-compose/internal/version"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	t.Run("run", func(t *testing.T) {
		t.Run("new reaches the starters of the catalog in the binary", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			command := newCommand()
			command.Writer, command.ErrWriter = io.Discard, io.Discard

			err := command.Run(context.Background(), []string{"feature-compose", "new", "no-such-starter", "x"})

			require.ErrorContains(t, err, "go-event-sourcing-service")
			require.Empty(t, filesIn(t, workingFolder))
		})

		t.Run("--version prints the name of the tool and its version", func(t *testing.T) {
			var printed bytes.Buffer
			command := newCommand()
			command.Writer, command.ErrWriter = &printed, &printed

			err := command.Run(context.Background(), []string{"feature-compose", "--version"})

			require.NoError(t, err)
			require.Equal(t, "feature-compose version "+version.Current()+"\n", printed.String())
		})

		t.Run("with no command, prints its commands and reads no input", func(t *testing.T) {
			var printed bytes.Buffer
			command := newCommand()
			command.Writer, command.ErrWriter, command.Reader = &printed, &printed, failingReader{t: t}

			err := command.Run(context.Background(), []string{"feature-compose"})

			require.NoError(t, err)
			require.Contains(t, printed.String(), "make a new repo from a starter")
			require.Contains(t, printed.String(), "replace feature-compose with the latest release")
		})
	})
}
