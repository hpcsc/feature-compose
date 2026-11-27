//go:build unit

package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcsc/feature-compose/internal/registry"
	"github.com/stretchr/testify/require"
)

func TestCatalogCommand(t *testing.T) {
	isolated := func(t *testing.T) {
		t.Helper()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
		t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	}
	git := func(t *testing.T, args ...string) string {
		t.Helper()
		printed, err := exec.Command("git", args...).CombinedOutput()
		require.NoError(t, err, string(printed))
		return strings.TrimSpace(string(printed))
	}
	commit := func(t *testing.T, dir, starterName string) string {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "starters", starterName), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "starters", starterName, "starter.yml"), []byte("summary: A starter.\n"), 0o644))
		git(t, "-C", dir, "add", "--all")
		git(t, "-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "--quiet", "--message", "change")
		return git(t, "-C", dir, "rev-parse", "HEAD")
	}
	gitSource := func(t *testing.T) (string, string) {
		t.Helper()
		dir := filepath.Join(t.TempDir(), "source")
		git(t, "init", "--quiet", "--initial-branch=main", dir)
		commit(t, dir, "service")
		return dir, "file://" + dir
	}
	run := func(t *testing.T, arguments ...string) (string, error) {
		t.Helper()
		var printed bytes.Buffer
		command := newCatalogCommand(registry.Open)
		command.Writer, command.ErrWriter = &printed, io.Discard
		err := command.Run(context.Background(), append([]string{"catalog"}, arguments...))
		return printed.String(), err
	}
	ran := func(t *testing.T, arguments ...string) string {
		t.Helper()
		printed, err := run(t, arguments...)
		require.NoError(t, err)
		return printed
	}

	t.Run("add", func(t *testing.T) {
		t.Run("a git URL prints the source, the commit, and that the starters can run commands", func(t *testing.T) {
			isolated(t)
			dir, url := gitSource(t)
			head := git(t, "-C", dir, "rev-parse", "HEAD")

			printed := ran(t, "add", "hpcsc", url)

			require.Equal(t, "Added the catalog hpcsc from "+url+" at the commit "+head[:7]+`.
The starters of a catalog run their commands on this machine. new --dry-run lists them before new runs them.
See the starters and the features of the catalog with:
  feature-compose show
`, printed)
		})

		t.Run("a local folder prints its full path", func(t *testing.T) {
			isolated(t)
			dir, _ := gitSource(t)

			printed := ran(t, "add", "team", dir)

			require.True(t, strings.HasPrefix(printed, "Added the catalog team from the folder "+dir+".\n"), printed)
		})

		t.Run("one argument stops, and the message gives the form of add", func(t *testing.T) {
			isolated(t)

			_, err := run(t, "add", "hpcsc")

			require.EqualError(t, err, "catalog add needs two arguments: catalog add <name> <source>")
		})
	})

	t.Run("list", func(t *testing.T) {
		t.Run("prints a table of the catalogs, and names catalog sync for a catalog with no clone", func(t *testing.T) {
			isolated(t)
			dir, url := gitSource(t)
			head := git(t, "-C", dir, "rev-parse", "HEAD")
			ran(t, "add", "hpcsc", url, "--ref", "main")
			ran(t, "add", "gone", url)
			ran(t, "add", "team", dir)
			require.NoError(t, os.RemoveAll(filepath.Join(os.Getenv("XDG_DATA_HOME"), "feature-compose", "catalogs", "gone")))

			printed := ran(t, "list")

			width := len(url)
			require.Equal(t, strings.Join([]string{
				"NAME   SOURCE" + strings.Repeat(" ", width-4) + "PATH  REF   COMMIT",
				"gone   " + url + "  -     -     no clone",
				"hpcsc  " + url + "  -     main  " + head[:7],
				"team   " + dir + strings.Repeat(" ", width-len(dir)) + "  -     -     -",
				"Get the clone of 1 catalog with: feature-compose catalog sync gone",
				"",
			}, "\n"), printed)
		})

		t.Run("with no catalog, names catalog add", func(t *testing.T) {
			isolated(t)

			printed := ran(t, "list")

			require.Equal(t, "The config file has no catalog. Add one with: feature-compose catalog add <name> <source>\n", printed)
		})
	})

	t.Run("sync", func(t *testing.T) {
		t.Run("prints the old and the new commit of each catalog", func(t *testing.T) {
			isolated(t)
			dir, url := gitSource(t)
			old := git(t, "-C", dir, "rev-parse", "HEAD")
			ran(t, "add", "hpcsc", url)
			ran(t, "add", "same", url, "--ref", old)
			ran(t, "add", "team", dir)
			newest := commit(t, dir, "worker")

			printed := ran(t, "sync")

			require.Equal(t, "hpcsc: from "+old[:7]+" to "+newest[:7]+"\nsame: no new commit, at "+old[:7]+"\nteam: a local folder, so sync gets nothing\n", printed)
		})

		t.Run("a catalog whose clone is gone prints that sync cloned it again", func(t *testing.T) {
			isolated(t)
			dir, url := gitSource(t)
			head := git(t, "-C", dir, "rev-parse", "HEAD")
			ran(t, "add", "hpcsc", url)
			require.NoError(t, os.RemoveAll(filepath.Join(os.Getenv("XDG_DATA_HOME"), "feature-compose", "catalogs", "hpcsc")))

			printed := ran(t, "sync")

			require.Equal(t, "hpcsc: cloned again, at "+head[:7]+"\n", printed)
		})

		t.Run("an unknown name stops with the message of the registry, and prints nothing", func(t *testing.T) {
			isolated(t)
			_, url := gitSource(t)
			ran(t, "add", "hpcsc", url)

			printed, err := run(t, "sync", "nope")

			require.ErrorContains(t, err, "has no catalog nope: it has hpcsc")
			require.Empty(t, printed)
		})

		t.Run("a catalog that fails prints its error, and sync stops with the names of the catalogs that failed", func(t *testing.T) {
			isolated(t)
			dir, url := gitSource(t)
			ran(t, "add", "gone", url)
			require.NoError(t, os.RemoveAll(dir))

			printed, err := run(t, "sync", "gone")

			require.EqualError(t, err, "the sync of 1 of 1 catalog failed: gone")
			require.True(t, strings.HasPrefix(printed, "gone: FAIL\n      git cannot read "+url+": check that git clone "+url+" works\n"), printed)
		})
	})

	t.Run("set", func(t *testing.T) {
		t.Run("prints the catalog with its new ref and commit", func(t *testing.T) {
			isolated(t)
			dir, url := gitSource(t)
			old := git(t, "-C", dir, "rev-parse", "HEAD")
			git(t, "-C", dir, "tag", "v1")
			commit(t, dir, "worker")
			ran(t, "add", "hpcsc", url)

			printed := ran(t, "set", "hpcsc", "--ref", "v1")

			require.Equal(t, "Changed the catalog hpcsc:\nNAME   SOURCE"+strings.Repeat(" ", len(url)-4)+"PATH  REF  COMMIT\nhpcsc  "+url+"  -     v1   "+old[:7]+"\n", printed)
		})

		t.Run("--path of a local folder prints the catalog with its new path", func(t *testing.T) {
			isolated(t)
			dir, _ := gitSource(t)
			commit(t, filepath.Join(dir, "other"), "worker")
			ran(t, "add", "team", dir)

			printed := ran(t, "set", "team", "--path", "other")

			require.Equal(t, "Changed the catalog team:\nNAME  SOURCE"+strings.Repeat(" ", len(dir)-4)+"PATH   REF  COMMIT\nteam  "+dir+"  other  -    -\n", printed)
		})

		t.Run("a ref that the source does not have stops with the message of the registry, and prints nothing", func(t *testing.T) {
			isolated(t)
			_, url := gitSource(t)
			ran(t, "add", "hpcsc", url)

			printed, err := run(t, "set", "hpcsc", "--ref", "v9")

			require.EqualError(t, err, "the source "+url+" has no ref v9")
			require.Empty(t, printed)
		})

		t.Run("with no flag stops, and the message names the flags", func(t *testing.T) {
			isolated(t)

			_, err := run(t, "set", "hpcsc")

			require.EqualError(t, err, "catalog set needs --path, --ref or both")
		})
	})

	t.Run("remove", func(t *testing.T) {
		t.Run("an unknown name stops with the message of the registry, and prints nothing", func(t *testing.T) {
			isolated(t)
			_, url := gitSource(t)
			ran(t, "add", "hpcsc", url)

			printed, err := run(t, "remove", "nope")

			require.ErrorContains(t, err, "has no catalog nope: it has hpcsc")
			require.Empty(t, printed)
		})

		t.Run("prints the name of the catalog, and list then has no catalog", func(t *testing.T) {
			isolated(t)
			_, url := gitSource(t)
			ran(t, "add", "hpcsc", url)

			printed := ran(t, "remove", "hpcsc")

			require.Equal(t, []string{"Removed the catalog hpcsc.\n", "The config file has no catalog. Add one with: feature-compose catalog add <name> <source>\n"}, []string{printed, ran(t, "list")})
		})
	})
}
