//go:build unit

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/catalog"
	"github.com/stretchr/testify/require"
)

func TestNewCommand(t *testing.T) {
	starterFiles := func(afterRender string) fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml": &fstest.MapFile{Data: []byte(`
options:
  module:
    default: github.com/hpcsc/{{ .Name }}
  go-version:
    default: "1.27.1"
    pattern: '^1\.(2[7-9]|[3-9][0-9])(\.[0-9]+)?$'
folders:
  go:
    path: .
    after_render: [` + afterRender + `]
`)},
			"starters/service/files/go/go.mod.tmpl": &fstest.MapFile{Data: []byte(`module {{ option "starter.module" }}` + "\n" + `go {{ option "starter.go-version" }}` + "\n")},
		}
	}
	runNew := func(t *testing.T, files fstest.MapFS, arguments ...string) (string, error) {
		t.Helper()
		var printed bytes.Buffer
		command := newNewCommand(catalog.New(files))
		command.Writer = &printed
		command.ErrWriter = &printed
		err := command.Run(context.Background(), append([]string{"new"}, arguments...))
		return printed.String(), err
	}

	withFeatures := func() fstest.MapFS {
		files := starterFiles("")
		files["features/cli/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")}
		files["features/cli/files/go/cmd/{{ .Name }}/main.go.tmpl"] = &fstest.MapFile{Data: []byte("package main // {{ .Name }}\n")}
		files["features/docs/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service]\nfiles:\n  - path: files/\n")}
		files["features/docs/files/go/docs/index.md"] = &fstest.MapFile{Data: []byte("docs\n")}
		files["features/a,b/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A comma.\ntargets: [service]\nfiles:\n  - path: files/\n")}
		files["features/a,b/files/go/comma.md"] = &fstest.MapFile{Data: []byte("comma\n")}
		return files
	}

	t.Run("write", func(t *testing.T) {
		t.Run("writes the repo to a folder with the name of the repo in the working folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)

			_, err := runNew(t, starterFiles(""), "service", "my-service")

			require.NoError(t, err)
			require.Equal(t, map[string]string{"my-service/go.mod": "module github.com/hpcsc/my-service\ngo 1.27.1\n"}, filesIn(t, workingFolder))
		})

		t.Run("each --feature selects one feature, and new writes its files with the files of the starter", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withFeatures(), "service", "my-service", "--output-dir", output, "--feature", "cli", "--feature", "docs")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":                 "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"cmd/my-service/main.go": "package main // my-service\n",
				"docs/index.md":          "docs\n",
			}, filesIn(t, output))
		})

		t.Run("a value of --feature keeps its commas", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withFeatures(), "service", "my-service", "--output-dir", output, "--feature", "a,b")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":   "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"comma.md": "comma\n",
			}, filesIn(t, output))
		})

		t.Run("prints the folder that it made the repo in", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			printed, err := runNew(t, starterFiles(""), "service", "my-service", "--output-dir", output)

			require.NoError(t, err)
			require.Contains(t, printed, output)
		})

		t.Run("a closing message that cannot be written stops new with an error", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			command := newNewCommand(catalog.New(starterFiles("")))
			command.Writer, command.ErrWriter = failingWriter{}, io.Discard

			err := command.Run(context.Background(), []string{"new", "service", "my-service", "--output-dir", output})

			require.ErrorContains(t, err, "closed output")
		})

		t.Run("--output-dir writes the repo to that path", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "elsewhere")

			_, err := runNew(t, starterFiles(""), "service", "my-service", "--output-dir", output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module github.com/hpcsc/my-service\ngo 1.27.1\n"}, filesIn(t, output))
		})

		t.Run("each --option sets one option of the starter, and a value keeps its commas", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, starterFiles(""), "service", "my-service", "--output-dir", output,
				"--option", "module=example.com/a,b", "--option", "go-version=1.28.0")

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module example.com/a,b\ngo 1.28.0\n"}, filesIn(t, output))
		})

		t.Run("with or without --no-prompt, new gives each option its default and reads no input", func(t *testing.T) {
			for _, arguments := range [][]string{{"--no-prompt"}, {}} {
				t.Chdir(t.TempDir())
				output := filepath.Join(t.TempDir(), "my-service")
				command := newNewCommand(catalog.New(starterFiles("")))
				command.Writer, command.ErrWriter, command.Reader = io.Discard, io.Discard, failingReader{t: t}

				err := command.Run(context.Background(), append([]string{"new", "service", "my-service", "--output-dir", output}, arguments...))

				require.NoError(t, err)
				require.Equal(t, map[string]string{"go.mod": "module github.com/hpcsc/my-service\ngo 1.27.1\n"}, filesIn(t, output))
			}
		})

		t.Run("after the last file, new runs the after_render commands in the repo", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, starterFiles(`"cat go.mod > copy"`), "service", "my-service", "--output-dir", output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod": "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"copy":   "module github.com/hpcsc/my-service\ngo 1.27.1\n",
			}, filesIn(t, output))
		})

		t.Run("an after_render command that fails stops new with the folder, the command and its output", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, starterFiles(`"echo tidy failed >&2; exit 1"`), "service", "my-service", "--output-dir", output)

			require.ErrorContains(t, err, "folder go")
			require.ErrorContains(t, err, "echo tidy failed >&2; exit 1")
			require.ErrorContains(t, err, "tidy failed")
		})
	})

	t.Run("help", func(t *testing.T) {
		t.Run("--help shows --feature and its form", func(t *testing.T) {
			printed, err := runNew(t, withFeatures(), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "--feature")
			require.Contains(t, printed, "select a feature of the catalog")
		})
	})

	t.Run("stop", func(t *testing.T) {
		for name, arguments := range map[string][]string{
			"no argument":  {},
			"one argument": {"service"},
		} {
			t.Run(name+" stops new, names <starter> and <name>, and writes nothing", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				_, err := runNew(t, starterFiles(""), arguments...)

				require.ErrorContains(t, err, "<starter>")
				require.ErrorContains(t, err, "<name>")
				require.Empty(t, filesIn(t, workingFolder))
			})
		}

		for _, name := range []string{"", "../escape", "team/service"} {
			t.Run("the name "+strconv.Quote(name)+" stops new, and the working folder keeps its files", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)
				require.NoError(t, os.WriteFile("go.mod", []byte("module mine\n"), 0o644))

				_, err := runNew(t, starterFiles(`"touch ran"`), "service", name)

				require.ErrorContains(t, err, "the name "+strconv.Quote(name)+" of the repo")
				require.Equal(t, map[string]string{"go.mod": "module mine\n"}, filesIn(t, workingFolder))
			})
		}

		t.Run("an --option value without = stops new before the first file, and shows the form <name>=<value>", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)

			_, err := runNew(t, starterFiles(""), "service", "my-service", "--option", "module")

			require.ErrorContains(t, err, `"module"`)
			require.ErrorContains(t, err, "<name>=<value>")
			require.Empty(t, filesIn(t, workingFolder))
		})

		for name, arguments := range map[string][]string{
			"an unknown starter":                      {"no-such-starter", "my-service"},
			"an unknown option":                       {"service", "my-service", "--option", "modul=x"},
			"a value that does not match its pattern": {"service", "my-service", "--option", "go-version=1.20"},
		} {
			t.Run(name+" stops new, and makes no output folder", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				_, err := runNew(t, starterFiles(""), arguments...)

				require.Error(t, err)
				require.NoDirExists(t, filepath.Join(workingFolder, "my-service"))
			})
		}

		t.Run("an unknown feature stops new, also next to a known one, lists the features that target the starter, and makes no output folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)

			_, err := runNew(t, withFeatures(), "service", "my-service", "--feature", "cli", "--feature", "no-such-feature")

			require.ErrorContains(t, err, `unknown feature "no-such-feature"`)
			require.ErrorContains(t, err, "a,b, cli, docs")
			require.NoDirExists(t, filepath.Join(workingFolder, "my-service"))
		})

		t.Run("an output folder that has entries stops new, and keeps its contents", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(output, "notes.txt"), []byte("my notes\n"), 0o644))

			_, err := runNew(t, starterFiles(`"touch ran"`), "service", "my-service", "--output-dir", output)

			require.ErrorContains(t, err, output)
			require.Equal(t, map[string]string{"notes.txt": "my notes\n"}, filesIn(t, output))
		})
	})
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("closed output")
}

type failingReader struct {
	t *testing.T
}

func (r failingReader) Read([]byte) (int, error) {
	r.t.Error("the command read its input")
	return 0, io.EOF
}

func filesIn(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(data)
		return nil
	})
	require.NoError(t, err)
	return files
}
