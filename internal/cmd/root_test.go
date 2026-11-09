//go:build unit

package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/feature-compose/internal/version"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	t.Run("run", func(t *testing.T) {
		t.Run("new reads the starters of the catalog folder that the config file names", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			configHome := t.TempDir()
			catalogDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			require.NoError(t, os.MkdirAll(filepath.Join(catalogDir, "starters", "alpha", "files", "go"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(catalogDir, "starters", "alpha", "starter.yml"), []byte("folders:\n  go:\n    path: .\n"), 0o644))
			require.NoError(t, os.MkdirAll(filepath.Join(configHome, "feature-compose"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(configHome, "feature-compose", "config.yml"), []byte("catalog: "+catalogDir+"\n"), 0o644))
			command := newCommand()
			command.Writer, command.ErrWriter = io.Discard, io.Discard

			err := command.Run(context.Background(), []string{"feature-compose", "new", "no-such-starter", "x"})

			require.ErrorContains(t, err, `unknown starter "no-such-starter": the catalog has alpha`)
			require.Empty(t, filesIn(t, workingFolder))
		})

		t.Run("with no config file, new stops, names the path of the config file, and makes no output folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			command := newCommand()
			command.Writer, command.ErrWriter = io.Discard, io.Discard

			err := command.Run(context.Background(), []string{"feature-compose", "new", "go-event-sourcing-service", "x"})

			require.ErrorContains(t, err, "no config file at "+filepath.Join(configHome, "feature-compose", "config.yml"))
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
