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
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/internal/catalog"
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
	loaded := func(files fstest.MapFS) func() (catalog.Catalog, error) {
		return func() (catalog.Catalog, error) { return catalog.New(files), nil }
	}
	runNew := func(t *testing.T, files fstest.MapFS, arguments ...string) (string, error) {
		t.Helper()
		var printed bytes.Buffer
		command := newNewCommand(loaded(files))
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
		files["features/auth/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [service]\nfiles:\n  - path: files/\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  realm:\n    default: api\n")}
		files["features/auth/files/go/auth.txt"] = &fstest.MapFile{Data: []byte(`{{ option "method" }} {{ option "realm" }}`)}
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

		t.Run("a feature given in two --feature flags is selected one time", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withFeatures(), "service", "my-service", "--output-dir", output, "--feature", "docs", "--feature", "docs")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":        "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"docs/index.md": "docs\n",
			}, filesIn(t, output))
		})

		for name, flags := range map[string][]string{
			"one flag with two pairs":                           {"--feature", "auth:method=oauth,realm=accounts"},
			"two flags of one feature":                          {"--feature", "auth:method=oauth", "--feature", "auth:realm=accounts"},
			"a flag with a value and a flag with only the name": {"--feature", "auth", "--feature", "auth:method=oauth,realm=accounts"},
			"one value given two times":                         {"--feature", "auth:method=oauth,realm=accounts", "--feature", "auth:method=oauth"},
		} {
			t.Run("--feature <name>:<option>=<value> sets the options of the feature, with "+name, func(t *testing.T) {
				t.Chdir(t.TempDir())
				output := filepath.Join(t.TempDir(), "my-service")

				_, err := runNew(t, withFeatures(), append([]string{"service", "my-service", "--output-dir", output}, flags...)...)

				require.NoError(t, err)
				require.Equal(t, map[string]string{
					"go.mod":   "module github.com/hpcsc/my-service\ngo 1.27.1\n",
					"auth.txt": "oauth accounts",
				}, filesIn(t, output))
			})
		}

		t.Run("--feature with only the name gives each option of the feature its default", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withFeatures(), "service", "my-service", "--output-dir", output, "--feature", "auth")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":   "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"auth.txt": "api-key api",
			}, filesIn(t, output))
		})

		t.Run("a value of an option of a feature keeps its = signs", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withFeatures(), "service", "my-service", "--output-dir", output, "--feature", "auth:realm=a=b")

			require.NoError(t, err)
			require.Equal(t, "api-key a=b", filesIn(t, output)["auth.txt"])
		})

		t.Run("new renders the templates of a feature with its delimiters, its extension points, githubRepository and the case functions", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-tool")
			files := starterFiles("")
			files["features/tool/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A tool.\ntargets: [service]\nfiles:\n  - path: files/\nextension_points:\n  tool.commands:\n    file: go/tool.txt\n    item: [code]\ndelimiters:\n  - glob: go/tool.txt\n    left: \"[[\"\n    right: \"]]\"\n")}
			files["features/tool/files/go/tool.txt"] = &fstest.MapFile{Data: []byte(`[[ kebab .Name ]] [[ githubRepository (option "starter.module") ]] [[ range items "tool.commands" ]]x[[ end ]]{{ .Tag }}`)}

			_, err := runNew(t, files, "service", "My.Tool", "--output-dir", output, "--feature", "tool", "--option", "module=github.com/acme/ledger")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":   "module github.com/acme/ledger\ngo 1.27.1\n",
				"tool.txt": "my-tool acme/ledger {{ .Tag }}",
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
			command := newNewCommand(loaded(starterFiles("")))
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
				command := newNewCommand(loaded(starterFiles("")))
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

		t.Run("the name of each combination of the catalog, given to new with a name, makes the repo of that combination", func(t *testing.T) {
			t.Chdir(t.TempDir())
			files := withFeatures()
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("options:\n  region:\n    values: [eu, us]\n    default: us\n  module:\n    default: example.com/{{ .Name }}\nfolders:\n  go:\n    path: .\n")}
			files["starters/service/files/go/go.mod.tmpl"] = &fstest.MapFile{Data: []byte(`module {{ option "starter.module" }} // {{ option "starter.region" }}`)}
			combinations, err := catalog.New(files).Combinations()
			require.NoError(t, err)
			fromNew := map[string]map[string]string{}
			fromSelection := map[string]map[string]string{}

			for _, combination := range combinations {
				words := strings.Fields(combination.String())
				output := filepath.Join(t.TempDir(), "my-service")
				_, err := runNew(t, files, append([]string{words[0], "my-service", "--output-dir", output}, words[1:]...)...)
				require.NoError(t, err, combination.String())
				fromNew[combination.String()] = filesIn(t, output)
				starter, err := catalog.New(files).Starter(combination.StarterName)
				require.NoError(t, err)
				selection, err := starter.Select("my-service", combination.StarterValues, combination.FeatureValues)
				require.NoError(t, err)
				output = filepath.Join(t.TempDir(), "my-service")
				require.NoError(t, selection.Write(output))
				fromSelection[combination.String()] = filesIn(t, output)
			}

			require.Len(t, combinations, 7)
			require.Equal(t, fromSelection, fromNew)
		})
	})

	t.Run("help", func(t *testing.T) {
		t.Run("--help shows --feature and its form <name>[:<option>=<value>,...]", func(t *testing.T) {
			printed, err := runNew(t, withFeatures(), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "--feature")
			require.Contains(t, printed, "select a feature of the catalog")
			require.Contains(t, printed, "<name>[:<option>=<value>,...]")
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
			require.ErrorContains(t, err, "a,b, auth, cli, docs")
			require.NoDirExists(t, filepath.Join(workingFolder, "my-service"))
		})

		for flag, message := range map[string]string{
			"auth:method":             `the pair "method"`,
			"auth:":                   `the pair ""`,
			"auth:method=oauth,":      `the pair ""`,
			"auth:=oauth":             `the pair "=oauth"`,
			"auth:method=oauth,realm": `the pair "realm"`,
		} {
			t.Run("the --feature value "+strconv.Quote(flag)+" stops new, names the value and shows the form, and makes no output folder", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				_, err := runNew(t, withFeatures(), "service", "my-service", "--feature", flag)

				require.ErrorContains(t, err, strconv.Quote(flag))
				require.ErrorContains(t, err, message)
				require.ErrorContains(t, err, "<name>:<option>=<value>")
				require.NoDirExists(t, filepath.Join(workingFolder, "my-service"))
			})
		}

		for name, flags := range map[string][]string{
			"in one flag":  {"--feature", "auth:method=oauth,method=api-key"},
			"in two flags": {"--feature", "auth:method=oauth", "--feature", "auth:method=api-key"},
		} {
			t.Run("one option of a feature with two different values "+name+" stops new, names the feature, the option and both values, and makes no output folder", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				_, err := runNew(t, withFeatures(), append([]string{"service", "my-service"}, flags...)...)

				require.ErrorContains(t, err, `the option method of the feature auth has two values: "oauth" and "api-key"`)
				require.NoDirExists(t, filepath.Join(workingFolder, "my-service"))
			})
		}

		for name, flag := range map[string]string{
			"a value that is not one of the values of its option": "auth:method=basic",
			"an option that the feature does not declare":         "auth:metod=oauth",
		} {
			t.Run(name+" stops new with the message of the catalog, and makes no output folder", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				_, err := runNew(t, withFeatures(), "service", "my-service", "--feature", flag)

				require.ErrorContains(t, err, "the feature auth")
				require.NoDirExists(t, filepath.Join(workingFolder, "my-service"))
			})
		}

		t.Run("a catalog that cannot load stops new with its error, and makes no output folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			command := newNewCommand(func() (catalog.Catalog, error) {
				return catalog.Catalog{}, errors.New("no config file at /home/config.yml")
			})
			command.Writer, command.ErrWriter = io.Discard, io.Discard

			err := command.Run(context.Background(), []string{"new", "service", "my-service"})

			require.ErrorContains(t, err, "no config file at /home/config.yml")
			require.Empty(t, filesIn(t, workingFolder))
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
