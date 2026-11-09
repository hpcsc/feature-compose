//go:build unit

package shell_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hpcsc/feature-compose/internal/shell"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("runs the command with sh in the folder, and gives its output and its errors", func(t *testing.T) {
		t.Chdir(t.TempDir())
		folder := t.TempDir()

		printed, err := shell.Run(context.Background(), folder, "echo out; echo err >&2; touch made-here")

		require.NoError(t, err)
		require.Equal(t, "out\nerr\n", string(printed))
		require.FileExists(t, filepath.Join(folder, "made-here"))
	})

	t.Run("a command that fails gives its output and the exit status", func(t *testing.T) {
		t.Chdir(t.TempDir())

		printed, err := shell.Run(context.Background(), t.TempDir(), "echo broken module >&2; exit 3")

		require.ErrorContains(t, err, "exit status 3")
		require.Equal(t, "broken module\n", string(printed))
	})

	t.Run("a context that ends stops the command and its children", func(t *testing.T) {
		t.Chdir(t.TempDir())
		folder := t.TempDir()
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, err := shell.Run(ctx, folder, "sh -c 'sleep 1; touch finished'; true")

		require.Error(t, err)
		time.Sleep(2 * time.Second)
		require.NoFileExists(t, filepath.Join(folder, "finished"))
	})

	t.Run("a context that ends does not wait for a child that left the process group", func(t *testing.T) {
		t.Chdir(t.TempDir())
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		started := time.Now()

		_, err := shell.Run(ctx, t.TempDir(), "set -m; sleep 8 & wait")

		require.Error(t, err)
		require.Less(t, time.Since(started), 4*time.Second)
	})

	t.Run("a folder that does not exist gives an error", func(t *testing.T) {
		t.Chdir(t.TempDir())

		_, err := shell.Run(context.Background(), filepath.Join(t.TempDir(), "missing"), "true")

		require.ErrorIs(t, err, os.ErrNotExist)
	})
}
