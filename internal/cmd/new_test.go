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

	"github.com/creack/pty"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestNewCommand(t *testing.T) {
	const version = "v1.2.3"
	defaultSelectionFile := "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: github.com/hpcsc/my-service\nfeatures: {}\nrequested: []\n"
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
		command := newNewCommand(loaded(files), version)
		command.Reader = strings.NewReader("")
		command.Writer = &printed
		command.ErrWriter = &printed
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := command.Run(ctx, append([]string{"new"}, arguments...))
		return printed.String(), err
	}
	runInTerminal := func(t *testing.T, files fstest.MapFS, after, keys string, arguments ...string) (string, error) {
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
			for !strings.Contains(screen.String(), after) && ctx.Err() == nil {
				time.Sleep(10 * time.Millisecond)
			}
			for _, key := range keys {
				_, _ = master.WriteString(string(key))
				time.Sleep(50 * time.Millisecond)
			}
		}()
		command := newNewCommand(loaded(files), version)
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
		files["features/elsewhere/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Elsewhere.\ntargets: [worker]\n")}
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
			require.Equal(t, map[string]string{
				"my-service/go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"my-service/.feature-compose.yml": defaultSelectionFile,
			}, filesIn(t, workingFolder))
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
				".feature-compose.yml":   "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: github.com/hpcsc/my-service\nfeatures:\n  cli: {}\n  docs: {}\nrequested: [cli, docs]\n",
			}, filesIn(t, output))
		})

		t.Run("a feature given in two --feature flags is selected one time", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withFeatures(), "service", "my-service", "--output-dir", output, "--feature", "docs", "--feature", "docs")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"docs/index.md":        "docs\n",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: github.com/hpcsc/my-service\nfeatures:\n  docs: {}\nrequested: [docs]\n",
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
					"go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
					"auth.txt":             "oauth accounts",
					".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: github.com/hpcsc/my-service\nfeatures:\n  auth:\n    method: oauth\n    realm: accounts\nrequested: [auth]\n",
				}, filesIn(t, output))
			})
		}

		t.Run("--feature with only the name gives each option of the feature its default", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withFeatures(), "service", "my-service", "--output-dir", output, "--feature", "auth")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"auth.txt":             "api-key api",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: github.com/hpcsc/my-service\nfeatures:\n  auth:\n    method: api-key\n    realm: api\nrequested: [auth]\n",
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
				"go.mod":               "module github.com/acme/ledger\ngo 1.27.1\n",
				"tool.txt":             "my-tool acme/ledger {{ .Tag }}",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: My.Tool\noptions:\n  go-version: 1.27.1\n  module: github.com/acme/ledger\nfeatures:\n  tool: {}\nrequested: [tool]\n",
			}, filesIn(t, output))
		})

		t.Run("a value of --feature keeps its commas", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withFeatures(), "service", "my-service", "--output-dir", output, "--feature", "a,b")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"comma.md":             "comma\n",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: github.com/hpcsc/my-service\nfeatures:\n  a,b: {}\nrequested: ['a,b']\n",
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

		for name, test := range map[string]struct {
			flags         [2][]string
			selectionFile string
		}{
			"with the defaults": {
				flags:         [2][]string{{"--feature", "a", "--feature", "b"}, {"--feature", "b", "--feature", "a"}},
				selectionFile: "version: v1.2.3\nstarter: service\nname: my-service\noptions: {}\nfeatures:\n  a: {}\n  b:\n    mode: fast\nrequested: [a, b]\n",
			},
			"with a value of an option": {
				flags:         [2][]string{{"--feature", "a", "--feature", "b:mode=slow"}, {"--feature", "b:mode=slow", "--feature", "a"}},
				selectionFile: "version: v1.2.3\nstarter: service\nname: my-service\noptions: {}\nfeatures:\n  a: {}\n  b:\n    mode: slow\nrequested: [a, b]\n",
			},
		} {
			t.Run("the two orders of the flags of a feature and its need give the same files and the same output, "+name, func(t *testing.T) {
				var printed [2]string
				var written [2]map[string]string
				for i, order := range test.flags {
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
				require.Equal(t, test.selectionFile, written[0]["my-service/.feature-compose.yml"])
			})
		}

		t.Run("a closing message that cannot be written stops new with an error", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			command := newNewCommand(loaded(starterFiles("")), version)
			command.Writer, command.ErrWriter = failingWriter{}, io.Discard

			err := command.Run(context.Background(), []string{"new", "service", "my-service", "--output-dir", output})

			require.ErrorContains(t, err, "closed output")
		})

		t.Run("--output-dir writes the repo to that path", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "elsewhere")

			_, err := runNew(t, starterFiles(""), "service", "my-service", "--output-dir", output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				".feature-compose.yml": defaultSelectionFile,
			}, filesIn(t, output))
		})

		t.Run("each --option sets one option of the starter, and a value keeps its commas", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, starterFiles(""), "service", "my-service", "--output-dir", output,
				"--option", "module=example.com/a,b", "--option", "go-version=1.28.0")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module example.com/a,b\ngo 1.28.0\n",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.28.0\n  module: example.com/a,b\nfeatures: {}\nrequested: []\n",
			}, filesIn(t, output))
		})

		t.Run("the two orders of two --option flags give the same .feature-compose.yml", func(t *testing.T) {
			t.Chdir(t.TempDir())
			var written [2]string
			for i, order := range [2][]string{
				{"--option", "module=example.com/ledger", "--option", "go-version=1.28.0"},
				{"--option", "go-version=1.28.0", "--option", "module=example.com/ledger"},
			} {
				output := filepath.Join(t.TempDir(), "my-service")
				_, err := runNew(t, starterFiles(""), append([]string{"service", "my-service", "--output-dir", output}, order...)...)
				require.NoError(t, err)
				written[i] = filesIn(t, output)[".feature-compose.yml"]
			}

			require.Equal(t, written[0], written[1])
			require.Equal(t, "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.28.0\n  module: example.com/ledger\nfeatures: {}\nrequested: []\n", written[0])
		})

		t.Run("--option gives a value to an option with no default", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, withoutDefaults(), "service", "my-service", "--output-dir", output, "--option", "region=eu")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"region.txt":           "eu",
				"ran":                  "",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  region: eu\nfeatures: {}\nrequested: []\n",
			}, filesIn(t, output))
		})

		t.Run("with or without --no-prompt, new gives each option its default and reads no input", func(t *testing.T) {
			for _, arguments := range [][]string{{"--no-prompt"}, {}} {
				t.Chdir(t.TempDir())
				output := filepath.Join(t.TempDir(), "my-service")
				command := newNewCommand(loaded(starterFiles("")), version)
				command.Writer, command.ErrWriter, command.Reader = io.Discard, io.Discard, failingReader{t: t}

				err := command.Run(context.Background(), append([]string{"new", "service", "my-service", "--output-dir", output}, arguments...))

				require.NoError(t, err)
				require.Equal(t, map[string]string{
					"go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
					".feature-compose.yml": defaultSelectionFile,
				}, filesIn(t, output))
			}
		})

		for name, test := range map[string]struct {
			files          func() fstest.MapFS
			given          map[string]string
			givenByFeature map[string]map[string]string
			answers        catalog.Answers
			selectionFile  string
		}{
			"with features that answers select, and a need with an answered value, which the arguments name with --feature": {
				files:         withNeeds,
				answers:       catalog.Answers{Features: []string{"a", "w"}, FeatureValues: map[string]map[string]string{"b": {"mode": "slow"}}},
				selectionFile: "version: v1.2.3\nstarter: service\nname: my-service\noptions: {}\nfeatures:\n  a: {}\n  b:\n    mode: slow\n  c: {}\n  w: {}\nrequested: [a, b, w]\n",
			},
			"with a value of the starter that has a space and a quote": {
				files:         func() fstest.MapFS { return starterFiles("") },
				answers:       catalog.Answers{StarterValues: map[string]string{"module": "example.com/it's a b"}},
				selectionFile: "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: example.com/it's a b\nfeatures: {}\nrequested: []\n",
			},
			"with flags, and an answer with $HOME and * in a value": {
				files:          withFeatures,
				given:          map[string]string{"go-version": "1.28.0"},
				givenByFeature: map[string]map[string]string{"auth": {"method": "oauth"}},
				answers:        catalog.Answers{FeatureValues: map[string]map[string]string{"auth": {"realm": "a $HOME *"}}},
				selectionFile:  "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.28.0\n  module: github.com/hpcsc/my-service\nfeatures:\n  auth:\n    method: oauth\n    realm: a $HOME *\nrequested: [auth]\n",
			},
		} {
			t.Run("new with the arguments of a selection "+name+", as sh splits them, and --no-prompt writes the files of the selection, and a selection file whose requested lists each feature that the arguments name", func(t *testing.T) {
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
				written := filesIn(t, output)
				require.Equal(t, test.selectionFile, written[".feature-compose.yml"])
				delete(written, ".feature-compose.yml")
				require.Equal(t, filesIn(t, expected), written)
			})
		}

		t.Run("after the last file, new runs the after_render commands in the repo", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, starterFiles(`"cat go.mod > copy"`), "service", "my-service", "--output-dir", output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"copy":                 "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				".feature-compose.yml": defaultSelectionFile,
			}, filesIn(t, output))
		})

		t.Run("new writes .feature-compose.yml after the last after_render command, so no command finds it", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, starterFiles(`"ls -A > listed"`), "service", "my-service", "--output-dir", output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"listed":               "go.mod\nlisted\n",
				".feature-compose.yml": defaultSelectionFile,
			}, filesIn(t, output))
		})

		t.Run("an after_render command that fails stops new with the folder, the command and its output", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runNew(t, starterFiles(`"echo tidy failed >&2; exit 1"`), "service", "my-service", "--output-dir", output)

			require.ErrorContains(t, err, "folder go")
			require.ErrorContains(t, err, "echo tidy failed >&2; exit 1")
			require.ErrorContains(t, err, "tidy failed")
			require.Equal(t, map[string]string{"go.mod": "module github.com/hpcsc/my-service\ngo 1.27.1\n"}, filesIn(t, output))
		})

		t.Run("an after_render command that makes a folder at .feature-compose.yml stops new with an error that names the file, and new prints nothing", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			printed, err := runNew(t, starterFiles(`"mkdir .feature-compose.yml"`), "service", "my-service", "--output-dir", output)

			require.ErrorContains(t, err, "write .feature-compose.yml")
			require.Empty(t, printed)
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
				require.NoError(t, selection.MakeRepo(context.Background(), output, version))
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
			require.Equal(t, "Repo: my-service\nStarter: service\n\nFeatures:\n  cli: command line\n\nOptions:\n  starter.go-version: 1.27.1 (default)\n  starter.module: github.com/hpcsc/my-service (default)\n\nFiles:\n  .feature-compose.yml: feature-compose\n  cmd/my-service/main.go: the feature cli\n  go.mod: the starter\n", printed)
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

		t.Run("--dry-run shows the source need of <feature> for a value that a need sets", func(t *testing.T) {
			t.Chdir(t.TempDir())
			files := withNeeds()
			files["features/v/feature.yml"] = &fstest.MapFile{Data: []byte("summary: V.\ntargets: [service]\nneeds:\n  b:\n    mode: slow\n")}

			printed, err := runNew(t, files, "service", "my-service", "--feature", "v", "--dry-run")

			require.NoError(t, err)
			require.Contains(t, printed, "Options:\n  b.mode: slow (need of v)\n\n")
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
			"an empty name of the repo":                      {starterFiles(""), []string{"service", ""}},
			"an --option value without =":                    {starterFiles(""), []string{"service", "my-service", "--option", "module"}},
			"an unknown starter":                             {starterFiles(""), []string{"no-such-starter", "my-service"}},
			"an unknown option":                              {starterFiles(""), []string{"service", "my-service", "--option", "modul=x"}},
			"a value that does not match its pattern":        {starterFiles(""), []string{"service", "my-service", "--option", "go-version=1.20"}},
			"an unknown feature":                             {withFeatures(), []string{"service", "my-service", "--feature", "no-such-feature"}},
			"a feature that does not target the starter":     {withFeatures(), []string{"service", "my-service", "--feature", "elsewhere"}},
			"a --feature value that is not <option>=<value>": {withFeatures(), []string{"service", "my-service", "--feature", "auth:method"}},
			"one option of a feature with two values":        {withFeatures(), []string{"service", "my-service", "--feature", "auth:method=oauth,method=api-key"}},
			"a value that is not one of its values":          {withFeatures(), []string{"service", "my-service", "--feature", "auth:method=basic"}},
			"a template that fails to render":                {withBadTemplate("starters/service/files/go/bad.go", "package {{ .Nmae }}\n"), []string{"service", "my-service"}},
			"two templates that write one path":              {withBadTemplate("starters/service/files/go/go.mod", "module mine\n"), []string{"service", "my-service"}},
			"a template that writes .feature-compose.yml":    {withBadTemplate("starters/service/files/go/.feature-compose.yml", "version: mine\n"), []string{"service", "my-service"}},
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
			}, version)
			command.Writer, command.ErrWriter = &printed, &printed

			err := command.Run(context.Background(), []string{"new", "service", "my-service", "--dry-run"})

			require.EqualError(t, err, "no config file at /home/config.yml")
			require.Empty(t, printed.String())
		})

		t.Run("a preview that cannot be written stops --dry-run with an error", func(t *testing.T) {
			t.Chdir(t.TempDir())
			command := newNewCommand(loaded(starterFiles("")), version)
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

		t.Run("--help shows the arguments of new as optional", func(t *testing.T) {
			printed, err := runNew(t, withFeatures(), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "[<starter> [<name>]]")
		})

		t.Run("--help shows that --no-prompt starts no TUI, and what new does then with an option without a value", func(t *testing.T) {
			printed, err := runNew(t, withFeatures(), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "--no-prompt")
			require.Contains(t, printed, "start no TUI: give each option without a value its default, and stop on an option with no default and no value")
		})
	})

	t.Run("tui", func(t *testing.T) {
		t.Run("in a terminal, w and y write the repo, and new prints the command that makes the same repo", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			screen, err := runInTerminal(t, withFeatures(), "Repo", "wy", "service", "my-service", "--output-dir", output, "--feature", "cli", "--option", "go-version=1.27.1", "--option", "module=github.com/hpcsc/my-service")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":                 "module github.com/hpcsc/my-service\ngo 1.27.1\n",
				"cmd/my-service/main.go": "package main // my-service\n",
				".feature-compose.yml":   "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  go-version: 1.27.1\n  module: github.com/hpcsc/my-service\nfeatures:\n  cli: {}\nrequested: [cli]\n",
			}, filesIn(t, output))
			require.Contains(t, screen, "To make the same repo again, run:\r\n  feature-compose new service my-service --output-dir "+output+" --feature cli\r\n")
		})

		t.Run("in a terminal, a feature that space selects is in requested of .feature-compose.yml, and a feature that only its need adds is in features and not in requested", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")

			_, err := runInTerminal(t, withNeeds(), "Repo", "jjj wy", "service", "my-service", "--output-dir", output)

			require.NoError(t, err)
			require.Equal(t, "version: v1.2.3\nstarter: service\nname: my-service\noptions: {}\nfeatures:\n  a: {}\n  b:\n    mode: fast\nrequested: [a]\n", filesIn(t, output)[".feature-compose.yml"])
		})

		for name, keys := range map[string]string{
			"q":                        "q",
			"w, n back to the form, q": "wnq",
			"ctrl+c":                   "\x03",
		} {
			t.Run("in a terminal, "+name+" stops new with the cancel, and new writes nothing", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)
				output := filepath.Join(t.TempDir(), "my-service")

				_, err := runInTerminal(t, withFeatures(), "Repo", keys, "service", "my-service", "--output-dir", output, "--feature", "cli")

				require.EqualError(t, err, "you canceled new, so it wrote no file")
				require.NoDirExists(t, output)
				require.Empty(t, filesIn(t, workingFolder))
			})
		}

		for name, arguments := range map[string][]string{
			"with no argument":  nil,
			"with one argument": {"service"},
		} {
			t.Run("in a terminal, "+name+", new opens the form with the missing answers, and q cancels", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				screen, err := runInTerminal(t, withFeatures(), "Repo", "q", arguments...)

				require.EqualError(t, err, "you canceled new, so it wrote no file")
				require.Contains(t, screen, "missing")
				require.Empty(t, filesIn(t, workingFolder))
			})
		}

		for name, flag := range map[string]string{"--no-prompt": "--no-prompt", "--dry-run": "--dry-run"} {
			t.Run("in a terminal, "+name+" starts no TUI", func(t *testing.T) {
				t.Chdir(t.TempDir())
				output := filepath.Join(t.TempDir(), "my-service")

				screen, err := runInTerminal(t, withFeatures(), "a screen that new never shows", "", "service", "my-service", "--output-dir", output, flag)

				require.NoError(t, err)
				require.Contains(t, screen, "my-service")
				require.NotContains(t, screen, "feature-compose new")
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
				command := newNewCommand(loaded(withFeatures()), version)
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

		t.Run("in a terminal, a flag that gives an option another value than a need stops new before the TUI starts, and new writes nothing", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			files := withNeeds()
			files["features/v/feature.yml"] = &fstest.MapFile{Data: []byte("summary: V.\ntargets: [service]\nneeds:\n  b:\n    mode: slow\n")}

			screen, err := runInTerminal(t, files, "", "", "service", "my-service", "--feature", "v", "--feature", "b:mode=fast")

			require.ErrorContains(t, err, `"fast" from the command line and "slow" from the need of v`)
			require.NotContains(t, screen, "feature-compose new")
			require.NoDirExists(t, filepath.Join(workingFolder, "my-service"))
		})

		t.Run("in a terminal, a feature that does not target the starter stops new before the TUI starts", func(t *testing.T) {
			screen, err := runInTerminal(t, withFeatures(), "", "", "service", "my-service", "--feature", "elsewhere")

			require.EqualError(t, err, "the feature elsewhere does not target the starter service: the features that target the starter service are a,b, auth, cli, docs")
			require.NotContains(t, screen, "feature-compose new")
		})

		t.Run("in a terminal, an unknown starter stops new before the TUI starts", func(t *testing.T) {
			screen, err := runInTerminal(t, withFeatures(), "", "", "nope", "my-service")

			require.EqualError(t, err, `unknown starter "nope": the catalog has service`)
			require.NotContains(t, screen, "feature-compose new")
		})

		for name, arguments := range map[string][]string{
			"no argument and --no-prompt": {"--no-prompt"},
			"one argument and --dry-run":  {"service", "--dry-run"},
		} {
			t.Run("in a terminal, "+name+" stop new, name <starter> and <name>, and start no TUI", func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)

				screen, err := runInTerminal(t, withFeatures(), "", "", arguments...)

				require.ErrorContains(t, err, "<starter>")
				require.ErrorContains(t, err, "<name>")
				require.NotContains(t, screen, "feature-compose new")
				require.Empty(t, filesIn(t, workingFolder))
			})
		}
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

		t.Run("a feature of the catalog that does not target the starter stops new, the message says so and lists the features that target the starter, and new makes no output folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)

			_, err := runNew(t, withFeatures(), "service", "my-service", "--feature", "elsewhere", "--no-prompt")

			require.EqualError(t, err, "the feature elsewhere does not target the starter service: the features that target the starter service are a,b, auth, cli, docs")
			require.NoDirExists(t, filepath.Join(workingFolder, "my-service"))
		})

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

		for name, arguments := range map[string][]string{
			"the need first":          {"--feature", "v", "--feature", "b:mode=fast"},
			"the flag first":          {"--feature", "b:mode=fast", "--feature", "v"},
			"the need first, dry run": {"--feature", "v", "--feature", "b:mode=fast", "--dry-run"},
			"the flag first, dry run": {"--feature", "b:mode=fast", "--feature", "v", "--dry-run"},
		} {
			t.Run("a flag that gives an option another value than a need stops new, names both sources, prints nothing and makes no output folder: "+name, func(t *testing.T) {
				workingFolder := t.TempDir()
				t.Chdir(workingFolder)
				files := withNeeds()
				files["features/v/feature.yml"] = &fstest.MapFile{Data: []byte("summary: V.\ntargets: [service]\nneeds:\n  b:\n    mode: slow\n")}

				printed, err := runNew(t, files, append([]string{"service", "my-service"}, arguments...)...)

				require.ErrorContains(t, err, `the option mode of the feature b has two values: "fast" from the command line and "slow" from the need of v`)
				require.Equal(t, "", printed)
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
			}, version)
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
