//go:build unit

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestInitCommand(t *testing.T) {
	runInit := func(t *testing.T, arguments ...string) (string, error) {
		t.Helper()
		var printed bytes.Buffer
		command := newInitCommand()
		command.Writer, command.ErrWriter = &printed, io.Discard
		err := command.Run(context.Background(), append([]string{"init"}, arguments...))
		return printed.String(), err
	}

	t.Run("init", func(t *testing.T) {
		t.Run("writes a catalog with no mistake, whose starter and feature make repos that pass their commands", func(t *testing.T) {
			catalogDir := filepath.Join(t.TempDir(), "my-catalog")
			_, err := runInit(t, catalogDir)
			require.NoError(t, err)
			var printed bytes.Buffer
			check := newCheckCommand(func(context.Context, string) (catalog.Catalogs, error) {
				return catalog.Catalogs{}, errors.New("no config file")
			})
			check.Writer, check.ErrWriter = &printed, io.Discard

			err = check.Run(context.Background(), []string{"check", "--catalog", catalogDir, "--build"})

			require.NoError(t, err)
			require.Equal(t, `The catalog has no mistakes.
The catalog has 3 combinations. Make the repo of each, and run its after_render and test commands.
[1/3] ok    example (…)
[2/3] ok    example --feature greeting:language=en (…)
[3/3] ok    example --feature greeting:language=fr (…)
The repo of each combination passes its commands.
`, regexp.MustCompile(`\(\d+s\)`).ReplaceAllString(printed.String(), "(…)"))
		})

		t.Run("prints the folder of the catalog, the commands that try it, and the command that adds it to the config file with the name of its folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)

			printed, err := runInit(t, "team-catalog")

			require.NoError(t, err)
			require.Equal(t, `Made a catalog in team-catalog with the starter example and the feature greeting. Try it:
  feature-compose show --catalog team-catalog
  feature-compose new --catalog team-catalog example my-repo --feature greeting --dry-run
  feature-compose check --catalog team-catalog --build
To use it with no --catalog, add it to the config file:
  feature-compose catalog add team-catalog `+filepath.Join(workingFolder, "team-catalog")+`
`, printed)
		})

		t.Run("a folder whose name is not a catalog name gives the name my-catalog in the command that adds it", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)

			printed, err := runInit(t, "my catalog")

			require.NoError(t, err)
			require.Contains(t, printed, "  feature-compose catalog add my-catalog "+filepath.Join(workingFolder, "my catalog")+"\n")
		})

		t.Run("writes the catalog next to the other files of a folder", func(t *testing.T) {
			catalogDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(catalogDir, "README.md"), []byte("# my catalog\n"), 0o644))

			_, err := runInit(t, catalogDir)

			require.NoError(t, err)
			require.Equal(t, "# my catalog\n", filesIn(t, catalogDir)["README.md"])
			require.Contains(t, filesIn(t, catalogDir), "starters/example/starter.yml")
		})

		t.Run("a folder that holds a file of the catalog stops init, which writes no file", func(t *testing.T) {
			catalogDir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(catalogDir, "features", "greeting"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(catalogDir, "features", "greeting", "feature.yml"), []byte("targets: [mine]\n"), 0o644))

			_, err := runInit(t, catalogDir)

			require.EqualError(t, err, catalogDir+" already holds features/greeting/feature.yml, so init writes no file")
			require.Equal(t, map[string]string{"features/greeting/feature.yml": "targets: [mine]\n"}, filesIn(t, catalogDir))
		})

		for name, arguments := range map[string][]string{
			"no argument":   nil,
			"two arguments": {"one", "two"},
		} {
			t.Run("init with "+name+" stops with the form of init, and writes no file", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				_, err := runInit(t, arguments...)

				require.EqualError(t, err, "init needs one argument: init <folder>")
				require.Empty(t, filesIn(t, workingFolder))
			})
		}
	})

	t.Run("help", func(t *testing.T) {
		t.Run("--help shows what init does and its folder", func(t *testing.T) {
			printed, err := runInit(t, "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "init - write a small catalog with one starter and one feature to a folder, as the start of a catalog of your own\n")
			require.Contains(t, printed, "init [global options] <folder>\n")
		})
	})
}
