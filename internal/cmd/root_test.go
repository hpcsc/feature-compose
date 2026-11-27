//go:build unit

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hpcsc/feature-compose/internal/version"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	writeFile := func(t *testing.T, path, contents string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	}
	catalogWith := func(t *testing.T, starterNames ...string) string {
		t.Helper()
		catalogDir := t.TempDir()
		for _, starterName := range starterNames {
			writeFile(t, filepath.Join(catalogDir, "starters", starterName, "starter.yml"), "summary: The starter "+starterName+".\nfolders:\n  go:\n    path: .\n")
			writeFile(t, filepath.Join(catalogDir, "starters", starterName, "files", "go", "README.md"), "# {{ .Name }}\n")
		}
		return catalogDir
	}
	configWith := func(t *testing.T, catalogDirs map[string]string) string {
		t.Helper()
		configHome := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", configHome)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		contents := "catalogs:\n"
		for _, name := range slices.Sorted(maps.Keys(catalogDirs)) {
			contents += "  " + name + ":\n    source: " + catalogDirs[name] + "\n"
		}
		writeFile(t, filepath.Join(configHome, "feature-compose", "config.yml"), contents)
		return configHome
	}
	run := func(t *testing.T, arguments ...string) (string, error) {
		t.Helper()
		var printed bytes.Buffer
		command := newCommand()
		command.Writer, command.ErrWriter = &printed, io.Discard
		err := command.Run(context.Background(), append([]string{"feature-compose"}, arguments...))
		return printed.String(), err
	}

	t.Run("run", func(t *testing.T) {
		t.Run("new reads the starters of the catalogs of the config file, by their full names", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			configWith(t, map[string]string{"team": catalogWith(t, "alpha")})

			_, err := run(t, "new", "no-such-starter", "x")

			require.EqualError(t, err, `unknown starter "no-such-starter": the catalog has team/alpha`)
			require.Empty(t, filesIn(t, workingFolder))
		})

		t.Run("new writes the version of feature-compose and the catalog of the starter to .feature-compose.yml of the repo", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			catalogDir := catalogWith(t, "alpha")
			configWith(t, map[string]string{"team": catalogDir})

			printed, err := run(t, "new", "alpha", "my-repo")

			require.NoError(t, err)
			require.Equal(t, "Made my-repo from the starter team/alpha.\n", printed)
			require.Equal(t, "version: "+version.Current()+"\ncatalog:\n  name: team\n  source: "+catalogDir+"\nstarter: alpha\nname: my-repo\noptions: {}\nfeatures: {}\nrequested: []\n", filesIn(t, workingFolder)["my-repo/.feature-compose.yml"])
		})

		t.Run("with two catalogs, new makes a repo from a starter of each", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			configWith(t, map[string]string{"team": catalogWith(t, "alpha"), "work": catalogWith(t, "beta")})

			_, alphaErr := run(t, "new", "team/alpha", "first")
			_, betaErr := run(t, "new", "beta", "second")

			require.NoError(t, errors.Join(alphaErr, betaErr))
			require.Equal(t, map[string]string{"first/README.md": "# first\n", "second/README.md": "# second\n"}, withoutSelectionFiles(filesIn(t, workingFolder)))
		})

		t.Run("a short name that two catalogs have stops new, and the message lists the full names", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			configWith(t, map[string]string{"team": catalogWith(t, "alpha"), "work": catalogWith(t, "alpha")})

			_, err := run(t, "new", "alpha", "x")

			require.EqualError(t, err, `more than one catalog has the starter "alpha": give its full name, one of team/alpha, work/alpha`)
			require.Empty(t, filesIn(t, workingFolder))
		})

		t.Run("--catalog with the name of a catalog of the config reads only that catalog", func(t *testing.T) {
			t.Chdir(t.TempDir())
			configWith(t, map[string]string{"team": catalogWith(t, "alpha"), "work": catalogWith(t, "alpha", "beta")})

			printed, err := run(t, "show", "--catalog", "work")

			require.NoError(t, err)
			require.Equal(t, []string{"work/alpha", "work/beta"}, starterNamesOf(t, printed))
		})

		t.Run("--catalog with a name that is not a folder and not a catalog of the config stops, and the message lists the catalogs", func(t *testing.T) {
			t.Chdir(t.TempDir())
			configHome := configWith(t, map[string]string{"team": catalogWith(t, "alpha")})

			_, err := run(t, "show", "--catalog", "nope")

			require.EqualError(t, err, "the config file "+filepath.Join(configHome, "feature-compose", "config.yml")+" has no catalog nope: it has team")
		})

		t.Run("show prints the starters and the features of each catalog of the config file, by their full names", func(t *testing.T) {
			t.Chdir(t.TempDir())
			catalogDir := catalogWith(t, "alpha")
			writeFile(t, filepath.Join(catalogDir, "features", "extra", "feature.yml"), "summary: Extra.\ntargets: [alpha]\n")
			configWith(t, map[string]string{"team": catalogDir, "work": catalogWith(t, "beta")})

			printed, err := run(t, "show")

			require.NoError(t, err)
			require.Equal(t, []string{"team/alpha", "work/beta"}, starterNamesOf(t, printed))
			require.Contains(t, printed, `"team/extra": {`)
		})

		t.Run("check finds the mistakes of each catalog of the config file, with the name of the catalog before each", func(t *testing.T) {
			t.Chdir(t.TempDir())
			catalogDir := catalogWith(t, "alpha")
			writeFile(t, filepath.Join(catalogDir, "features", "extra", "feature.yml"), "targets: [alpha]\nadd_to:\n  cli.commands:\n    - code: newExtraCommand()\n")
			configWith(t, map[string]string{"team": catalogDir})

			printed, err := run(t, "check")

			require.EqualError(t, err, "the catalog has 1 mistake")
			require.Equal(t, "the catalog team: the starter alpha: the feature extra adds to the extension point \"cli.commands\", which the starter, the feature and its needs do not declare\n", printed)
		})

		t.Run("with no config file, check stops and names catalog add", func(t *testing.T) {
			t.Chdir(t.TempDir())
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)

			_, err := run(t, "check")

			require.EqualError(t, err, "the config file "+filepath.Join(configHome, "feature-compose", "config.yml")+" has no catalog: add one with feature-compose catalog add <name> <source>")
		})

		t.Run("with no config file, new stops, names catalog add, and makes no output folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			_, err := run(t, "new", "go-event-sourcing-service", "x")

			require.ErrorContains(t, err, "add one with feature-compose catalog add <name> <source>")
			require.Empty(t, filesIn(t, workingFolder))
		})

		t.Run("add starter into a catalog from a git URL stops, and writes no file into the clone", func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			dataHome := t.TempDir()
			t.Setenv("XDG_DATA_HOME", dataHome)
			t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			source := catalogWith(t, "alpha")
			for _, args := range [][]string{{"init", "--quiet", "--initial-branch=main"}, {"add", "--all"}, {"-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "--quiet", "--message", "start"}} {
				printed, err := exec.Command("git", append([]string{"-C", source}, args...)...).CombinedOutput()
				require.NoError(t, err, string(printed))
			}
			_, err := run(t, "catalog", "add", "hpcsc", "file://"+source)
			require.NoError(t, err)
			clone := filepath.Join(dataHome, "feature-compose", "catalogs", "hpcsc")
			before := filesIn(t, clone)

			_, err = run(t, "add", "starter", "beta", "--catalog", "hpcsc")

			require.EqualError(t, err, "the catalog hpcsc is a clone of file://"+source+", and the next catalog sync removes each change in the clone: give --catalog <folder> with a clone of your own")
			require.Equal(t, before, filesIn(t, clone))
		})

		t.Run("add starter with no --catalog writes to the one catalog of the config file", func(t *testing.T) {
			t.Chdir(t.TempDir())
			catalogDir := catalogWith(t, "alpha")
			configWith(t, map[string]string{"team": catalogDir})

			_, err := run(t, "add", "starter", "beta")

			require.NoError(t, err)
			require.FileExists(t, filepath.Join(catalogDir, "starters", "beta", "starter.yml"))
		})

		t.Run("an error to a stream that is not a terminal has no colour codes, also when stdout is a terminal", func(t *testing.T) {
			var written bytes.Buffer

			writeError(&written, errors.New("unknown starter \"nope\""))

			require.Equal(t, "unknown starter \"nope\"\n", written.String())
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
			require.Contains(t, printed.String(), "show the starters, the features and the extension points of the catalog")
			require.Contains(t, printed.String(), "find the mistakes of the starters, the features and the templates of the catalog")
			require.Contains(t, printed.String(), "replace feature-compose with the latest release")
		})
	})
}

func withoutSelectionFiles(files map[string]string) map[string]string {
	maps.DeleteFunc(files, func(path string, _ string) bool { return strings.HasSuffix(path, "/.feature-compose.yml") })
	return files
}

func starterNamesOf(t *testing.T, printed string) []string {
	t.Helper()
	var shown struct {
		Starters map[string]any `json:"starters"`
	}
	require.NoError(t, json.Unmarshal([]byte(printed), &shown))
	return slices.Sorted(maps.Keys(shown.Starters))
}
