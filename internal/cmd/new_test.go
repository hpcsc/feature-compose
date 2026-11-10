//go:build unit

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
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
		command.Reader = strings.NewReader("")
		command.Writer = &printed
		command.ErrWriter = &printed
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := command.Run(ctx, append([]string{"new"}, arguments...))
		return printed.String(), err
	}
	runInTerminal := func(t *testing.T, files fstest.MapFS, keys string, arguments ...string) (string, error) {
		t.Helper()
		master, tty, err := pty.Open()
		require.NoError(t, err)
		require.NoError(t, pty.Setsize(tty, &pty.Winsize{Rows: 40, Cols: 100}))
		var screen screenBuffer
		drained := make(chan struct{})
		go func() {
			_, _ = io.Copy(&screen, master)
			close(drained)
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		go func() {
			for !strings.Contains(screen.String(), "Write the repo?") && ctx.Err() == nil {
				time.Sleep(10 * time.Millisecond)
			}
			_, _ = master.WriteString(keys)
		}()
		command := newNewCommand(loaded(files))
		command.Reader, command.Writer, command.ErrWriter = tty, tty, tty

		runErr := command.Run(ctx, append([]string{"new"}, arguments...))

		require.NoError(t, tty.Close())
		<-drained
		require.NoError(t, master.Close())
		return screen.String(), runErr
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

	withNeeds := func() fstest.MapFS {
		files := fstest.MapFS{
			"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n")},
			"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }};{{ end }}`)},
			"features/a/feature.yml":              &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds: [b]\nadd_to:\n  api.middleware:\n    - code: a\n")},
			"features/b/feature.yml":              &fstest.MapFile{Data: []byte("summary: B.\ntargets: [service]\nfiles:\n  - path: files/\noptions:\n  mode:\n    values: [fast, slow]\n    default: fast\nadd_to:\n  api.middleware:\n    - code: b\n")},
			"features/b/files/go/mode.txt":        &fstest.MapFile{Data: []byte(`{{ option "mode" }}`)},
			"features/w/feature.yml":              &fstest.MapFile{Data: []byte("summary: W.\ntargets: [service]\nneeds: [b, c]\n")},
			"features/c/feature.yml":              &fstest.MapFile{Data: []byte("summary: C.\ntargets: [service]\n")},
		}
		return files
	}

	withoutDefaults := func() fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":         &fstest.MapFile{Data: []byte("options:\n  region:\n    pattern: '^[a-z]*$'\nfolders:\n  go:\n    path: .\n    after_render: [\"touch ran\"]\n")},
			"starters/service/files/go/region.txt": &fstest.MapFile{Data: []byte(`{{ option "starter.region" }}`)},
			"features/a/feature.yml":               &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds: [b]\n")},
			"features/b/feature.yml":               &fstest.MapFile{Data: []byte("summary: B.\ntargets: [service]\noptions:\n  token:\n    default: \"\"\n")},
		}
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

		t.Run("prints a line for each feature that a need added, with the features that need it, before the line that names the output folder", func(t *testing.T) {
			t.Chdir(t.TempDir())

			printed, err := runNew(t, withNeeds(), "service", "my-service", "--feature", "w", "--feature", "a")

			require.NoError(t, err)
			require.Equal(t, "b: needed by a, w\nc: needed by w\nMade my-service from the starter service.\n", printed)
		})

		t.Run("with no need in the selection, prints only the line that names the output folder", func(t *testing.T) {
			t.Chdir(t.TempDir())

			printed, err := runNew(t, withNeeds(), "service", "my-service", "--feature", "a", "--feature", "b")

			require.NoError(t, err)
			require.Equal(t, "Made my-service from the starter service.\n", printed)
		})

		for name, flags := range map[string][2][]string{
			"with the defaults":         {{"--feature", "a", "--feature", "b"}, {"--feature", "b", "--feature", "a"}},
			"with a value of an option": {{"--feature", "a", "--feature", "b:mode=slow"}, {"--feature", "b:mode=slow", "--feature", "a"}},
		} {
			t.Run("the two orders of the flags of a feature and its need give the same files and the same output, "+name, func(t *testing.T) {
				var printed [2]string
				var written [2]map[string]string
				for i, order := range flags {
					workingFolder := t.TempDir()
					t.Chdir(workingFolder)
					var err error
					printed[i], err = runNew(t, withNeeds(), append([]string{"service", "my-service"}, order...)...)
					require.NoError(t, err)
					written[i] = filesIn(t, workingFolder)
				}

				require.Equal(t, printed[0], printed[1])
				require.Equal(t, written[0], written[1])
				require.Equal(t, "b;a;", written[0]["my-service/server.go"])
			})
		}

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

		t.Run("--option gives a value to an option with no default", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withoutDefaults(), "service", "my-service", "--output-dir", output, "--option", "region=eu")

			require.NoError(t, err)
			require.Equal(t, map[string]string{"region.txt": "eu", "ran": ""}, filesIn(t, output))
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

		for name, test := range map[string]struct {
			files          func() fstest.MapFS
			given          map[string]string
			givenByFeature map[string]map[string]string
			answers        catalog.Answers
		}{
			"with features that answers select, and a need with an answered value": {
				files:   withNeeds,
				answers: catalog.Answers{Features: []string{"a", "w"}, FeatureValues: map[string]map[string]string{"b": {"mode": "slow"}}},
			},
			"with a value of the starter that has a space and a quote": {
				files:   func() fstest.MapFS { return starterFiles("") },
				answers: catalog.Answers{StarterValues: map[string]string{"module": "example.com/it's a b"}},
			},
			"with flags, and an answer with $HOME and * in a value": {
				files:          withFeatures,
				given:          map[string]string{"go-version": "1.28.0"},
				givenByFeature: map[string]map[string]string{"auth": {"method": "oauth"}},
				answers:        catalog.Answers{FeatureValues: map[string]map[string]string{"auth": {"realm": "a $HOME *"}}},
			},
		} {
			t.Run("new with the arguments of a selection "+name+", as sh splits them, and --no-prompt writes the files of the selection", func(t *testing.T) {
				t.Chdir(t.TempDir())
				starter, err := catalog.New(test.files()).Starter("service")
				require.NoError(t, err)
				selection, err := starter.Select("my-service", test.given, test.givenByFeature, test.answers)
				require.NoError(t, err)
				expected := filepath.Join(t.TempDir(), "my-service")
				require.NoError(t, selection.Write(expected))
				output := filepath.Join(t.TempDir(), "elsewhere")
				split, err := exec.Command("sh", "-c", `printf '%s\000' `+selection.Arguments(output)).Output()
				require.NoError(t, err)
				words := strings.Split(strings.TrimSuffix(string(split), "\x00"), "\x00")

				_, err = runNew(t, test.files(), append(words, "--no-prompt")...)

				require.NoError(t, err)
				require.Equal(t, filesIn(t, expected), filesIn(t, output))
			})
		}

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
				selection, err := starter.Select("my-service", combination.StarterValues, combination.FeatureValues, catalog.Answers{})
				require.NoError(t, err)
				output = filepath.Join(t.TempDir(), "my-service")
				require.NoError(t, selection.Write(output))
				fromSelection[combination.String()] = filesIn(t, output)
			}

			require.Len(t, combinations, 7)
			require.Equal(t, fromSelection, fromNew)
		})
	})

	t.Run("dry run", func(t *testing.T) {
		t.Run("--dry-run prints the preview of the repo, and writes no file and makes no output folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)

			printed, err := runNew(t, withFeatures(), "service", "my-service", "--feature", "cli", "--dry-run")

			require.NoError(t, err)
			require.Equal(t, "Repo: my-service\nStarter: service\n\nFeatures:\n  cli: command line\n\nOptions:\n  starter.go-version: 1.27.1 (default)\n  starter.module: github.com/hpcsc/my-service (default)\n\nFiles:\n  cmd/my-service/main.go: the feature cli\n  go.mod: the starter\n", printed)
			require.Empty(t, filesIn(t, workingFolder))
		})

		t.Run("with --output-dir set to an empty folder that exists, --dry-run runs no after_render command, so the folder and the working folder stay empty", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			output := t.TempDir()

			_, err := runNew(t, starterFiles(`"touch ran"`), "service", "my-service", "--output-dir", output, "--dry-run")

			require.NoError(t, err)
			require.Empty(t, filesIn(t, output))
			require.Empty(t, filesIn(t, workingFolder))
		})

		t.Run("an output folder that has entries does not stop --dry-run, and keeps its contents", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(output, "notes.txt"), []byte("my notes\n"), 0o644))

			printed, err := runNew(t, starterFiles(`"touch ran"`), "service", "my-service", "--output-dir", output, "--dry-run")

			require.NoError(t, err)
			require.Contains(t, printed, "  go.mod: the starter\n")
			require.Equal(t, map[string]string{"notes.txt": "my notes\n"}, filesIn(t, output))
		})

		t.Run("the two orders of the flags give the same preview", func(t *testing.T) {
			t.Chdir(t.TempDir())

			first, err := runNew(t, withFeatures(), "service", "my-service", "--feature", "cli", "--feature", "docs", "--dry-run")
			require.NoError(t, err)
			second, err := runNew(t, withFeatures(), "service", "my-service", "--dry-run", "--feature", "docs", "--feature", "cli")
			require.NoError(t, err)

			require.Equal(t, first, second)
		})

		t.Run("--dry-run lists each feature with its source, and a feature that a need added shows the line that new prints for it", func(t *testing.T) {
			t.Chdir(t.TempDir())
			made, err := runNew(t, withNeeds(), "service", "my-service", "--feature", "w", "--feature", "a")
			require.NoError(t, err)

			printed, err := runNew(t, withNeeds(), "service", "my-service", "--feature", "w", "--feature", "a", "--dry-run")

			require.NoError(t, err)
			require.Equal(t, "b: needed by a, w\nc: needed by w\nMade my-service from the starter service.\n", made)
			require.Contains(t, printed, "Features:\n  b: needed by a, w\n  a: command line\n  c: needed by w\n  w: command line\n\n")
		})

		t.Run("--dry-run shows each option with the value that the templates read and its source, for --option and for --feature", func(t *testing.T) {
			t.Chdir(t.TempDir())

			printed, err := runNew(t, withFeatures(), "service", "my-service", "--option", "go-version=1.28.0", "--feature", "auth:method=oauth", "--dry-run")

			require.NoError(t, err)
			require.Contains(t, printed, "Options:\n  starter.go-version: 1.28.0 (command line)\n  starter.module: github.com/hpcsc/my-service (default)\n  auth.method: oauth (command line)\n  auth.realm: api (default)\n\n")
		})

		t.Run("the two orders of the flags of a feature and its need give the same preview", func(t *testing.T) {
			t.Chdir(t.TempDir())

			first, err := runNew(t, withNeeds(), "service", "my-service", "--feature", "a", "--feature", "b:mode=slow", "--dry-run")
			require.NoError(t, err)
			second, err := runNew(t, withNeeds(), "service", "my-service", "--feature", "b:mode=slow", "--feature", "a", "--dry-run")
			require.NoError(t, err)

			require.Equal(t, first, second)
			require.Contains(t, first, "Features:\n  b: command line, needed by a\n  a: command line\n\n")
		})

		withBadTemplate := func(path, contents string) fstest.MapFS {
			files := starterFiles("")
			files[path] = &fstest.MapFile{Data: []byte(contents)}
			return files
		}
		for name, test := range map[string]struct {
			files     fstest.MapFS
			arguments []string
		}{
			"one argument":                                   {starterFiles(""), []string{"service"}},
			"a name of the repo that is not valid":           {starterFiles(""), []string{"service", "../escape"}},
			"an --option value without =":                    {starterFiles(""), []string{"service", "my-service", "--option", "module"}},
			"an unknown starter":                             {starterFiles(""), []string{"no-such-starter", "my-service"}},
			"an unknown option":                              {starterFiles(""), []string{"service", "my-service", "--option", "modul=x"}},
			"a value that does not match its pattern":        {starterFiles(""), []string{"service", "my-service", "--option", "go-version=1.20"}},
			"an unknown feature":                             {withFeatures(), []string{"service", "my-service", "--feature", "no-such-feature"}},
			"a --feature value that is not <option>=<value>": {withFeatures(), []string{"service", "my-service", "--feature", "auth:method"}},
			"one option of a feature with two values":        {withFeatures(), []string{"service", "my-service", "--feature", "auth:method=oauth,method=api-key"}},
			"a value that is not one of its values":          {withFeatures(), []string{"service", "my-service", "--feature", "auth:method=basic"}},
			"a template that fails to render":                {withBadTemplate("starters/service/files/go/bad.go", "package {{ .Nmae }}\n"), []string{"service", "my-service"}},
			"two templates that write one path":              {withBadTemplate("starters/service/files/go/go.mod", "module mine\n"), []string{"service", "my-service"}},
			"an option with no default and no value":         {withoutDefaults(), []string{"service", "my-service"}},
		} {
			t.Run(name+" stops --dry-run with the message that stops new, prints nothing, and makes no output folder", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)
				_, newErr := runNew(t, test.files, test.arguments...)
				require.Error(t, newErr)

				printed, err := runNew(t, test.files, append(test.arguments, "--dry-run")...)

				require.EqualError(t, err, newErr.Error())
				require.Empty(t, printed)
				require.Empty(t, filesIn(t, workingFolder))
			})
		}

		t.Run("a catalog that cannot load stops --dry-run with its error, and prints nothing", func(t *testing.T) {
			t.Chdir(t.TempDir())
			var printed bytes.Buffer
			command := newNewCommand(func() (catalog.Catalog, error) {
				return catalog.Catalog{}, errors.New("no config file at /home/config.yml")
			})
			command.Writer, command.ErrWriter = &printed, &printed

			err := command.Run(context.Background(), []string{"new", "service", "my-service", "--dry-run"})

			require.EqualError(t, err, "no config file at /home/config.yml")
			require.Empty(t, printed.String())
		})

		t.Run("a preview that cannot be written stops --dry-run with an error", func(t *testing.T) {
			t.Chdir(t.TempDir())
			command := newNewCommand(loaded(starterFiles("")))
			command.Writer, command.ErrWriter = failingWriter{}, io.Discard

			err := command.Run(context.Background(), []string{"new", "service", "my-service", "--dry-run"})

			require.ErrorContains(t, err, "closed output")
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

		t.Run("--help shows --dry-run and what it does", func(t *testing.T) {
			printed, err := runNew(t, withFeatures(), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "--dry-run")
			require.Contains(t, printed, "print a preview of the repo, and write no file and run no command")
		})

		t.Run("--help shows that --no-prompt starts no TUI, and what new does then with an option without a value", func(t *testing.T) {
			printed, err := runNew(t, withFeatures(), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "--no-prompt")
			require.Contains(t, printed, "start no TUI: give each option without a value its default, and stop on an option with no default and no value")
		})
	})

	t.Run("tui", func(t *testing.T) {
		t.Run("in a terminal, y at the confirm step writes the repo, and new prints the command that makes the same repo", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			screen, err := runInTerminal(t, withFeatures(), "y", "service", "my-service", "--output-dir", output, "--feature", "cli", "--option", "go-version=1.27.1", "--option", "module=github.com/hpcsc/my-service")

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module github.com/hpcsc/my-service\ngo 1.27.1\n", "cmd/my-service/main.go": "package main // my-service\n"}, filesIn(t, output))
			require.Contains(t, screen, "To make the same repo again, run:\r\n  feature-compose new service my-service --output-dir "+output+" --feature cli\r\n")
		})

		t.Run("in a terminal, n at the confirm step stops new with the cancel, and writes nothing", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			screen, err := runInTerminal(t, withFeatures(), "n", "service", "my-service", "--output-dir", output, "--feature", "cli", "--option", "go-version=1.27.1", "--option", "module=github.com/hpcsc/my-service")

			require.EqualError(t, err, "you canceled new, so it wrote no file")
			require.Contains(t, screen, "Write the repo?")
			require.NoDirExists(t, output)
		})

		for name, flag := range map[string]string{"--no-prompt": "--no-prompt", "--dry-run": "--dry-run"} {
			t.Run("in a terminal, "+name+" starts no TUI", func(t *testing.T) {
				t.Chdir(t.TempDir())
				output := filepath.Join(t.TempDir(), "my-service")

				screen, err := runInTerminal(t, withFeatures(), "", "service", "my-service", "--output-dir", output, flag)

				require.NoError(t, err)
				require.Contains(t, screen, "my-service")
				require.NotContains(t, screen, "Write the repo?")
			})
		}

		for name, test := range map[string]struct{ readerIsTerminal, writerIsTerminal bool }{
			"stdin is a terminal and stdout is not": {readerIsTerminal: true},
			"stdout is a terminal and stdin is not": {writerIsTerminal: true},
		} {
			t.Run("when "+name+", new starts no TUI and writes the repo", func(t *testing.T) {
				t.Chdir(t.TempDir())
				output := filepath.Join(t.TempDir(), "my-service")
				master, tty, err := pty.Open()
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = tty.Close()
					_ = master.Close()
				})
				command := newNewCommand(loaded(withFeatures()))
				command.Reader, command.Writer, command.ErrWriter = io.Reader(strings.NewReader("")), io.Writer(io.Discard), io.Discard
				if test.readerIsTerminal {
					command.Reader = tty
				}
				if test.writerIsTerminal {
					command.Writer = tty
					go func() { _, _ = io.Copy(io.Discard, master) }()
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				err = command.Run(ctx, []string{"new", "service", "my-service", "--output-dir", output})

				require.NoError(t, err)
				require.FileExists(t, filepath.Join(output, "go.mod"))
			})
		}

		t.Run("a TUI that stops with an error gives that error to new", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			command := &cli.Command{Reader: strings.NewReader(""), Writer: io.Discard}

			_, err := ask(ctx, command, catalog.New(starterFiles("")), "service", "my-service", nil, nil)

			require.ErrorIs(t, err, tea.ErrProgramKilled)
		})

		t.Run("an unknown starter stops new before the TUI starts", func(t *testing.T) {
			command := &cli.Command{Reader: strings.NewReader(""), Writer: io.Discard}

			_, err := ask(context.Background(), command, catalog.New(starterFiles("")), "nope", "my-service", nil, nil)

			require.EqualError(t, err, `unknown starter "nope": the catalog has service`)
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

		t.Run("a need that is not a feature of the starter stops new with the message of the catalog, and makes no output folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			files := starterFiles("")
			files["features/a/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds: [missing]\n")}

			_, err := runNew(t, files, "service", "my-service", "--feature", "a")

			require.ErrorContains(t, err, `the feature a needs "missing"`)
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

		for name, arguments := range map[string][]string{
			"with --no-prompt":    {"--no-prompt"},
			"without --no-prompt": {},
		} {
			t.Run("an option of the starter with no default and no value stops new "+name+", names the option and the flag that gives it, makes no output folder and runs no after_render command", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				_, err := runNew(t, withoutDefaults(), append([]string{"service", "my-service"}, arguments...)...)

				require.EqualError(t, err, "the option region of the starter service has no default and no value: give it with --option region=<value>")
				require.Empty(t, filesIn(t, workingFolder))
			})
		}

		t.Run("an option with no default of a feature that only a need selects stops new, and the message names the feature and the flag that gives it", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)

			_, err := runNew(t, withoutDefaults(), "service", "my-service", "--option", "region=eu", "--feature", "a")

			require.EqualError(t, err, "the option token of the feature b has no default and no value: give it with --feature b:token=<value>")
			require.Empty(t, filesIn(t, workingFolder))
		})

		for name, test := range map[string]struct {
			flags   []string
			message string
		}{
			"--option region=": {
				flags:   []string{"--option", "region=", "--feature", "b:token=t"},
				message: "the option region of the starter service has no default and no value: give it with --option region=<value>",
			},
			"--feature b:token=": {
				flags:   []string{"--option", "region=eu", "--feature", "b:token="},
				message: "the option token of the feature b has no default and no value: give it with --feature b:token=<value>",
			},
		} {
			t.Run(name+" with the empty text stops new for an option with no default", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				_, err := runNew(t, withoutDefaults(), append([]string{"service", "my-service", "--feature", "a"}, test.flags...)...)

				require.EqualError(t, err, test.message)
				require.Empty(t, filesIn(t, workingFolder))
			})
		}
	})
}

type screenBuffer struct {
	mu      sync.Mutex
	written bytes.Buffer
}

func (b *screenBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.written.Write(p)
}

func (b *screenBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.written.String()
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
