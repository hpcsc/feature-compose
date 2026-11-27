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
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestShowCommand(t *testing.T) {
	runShow := func(t *testing.T, files fstest.MapFS, arguments ...string) (string, error) {
		t.Helper()
		var printed, stderr bytes.Buffer
		command := newShowCommand(func(context.Context, string) (catalog.Catalogs, error) {
			return catalog.NewCatalogs(catalog.New(files)), nil
		})
		command.Writer = &printed
		command.ErrWriter = &stderr
		err := command.Run(context.Background(), append([]string{"show"}, arguments...))
		return printed.String(), err
	}
	withEachField := func() fstest.MapFS {
		return fstest.MapFS{
			"starters/api/starter.yml": &fstest.MapFile{Data: []byte(`summary: An API.
options:
  region:
    values: [us, eu]
    default: us
  module:
    default: example.com/{{ .Name }}
    pattern: '^[a-z./]+$'
folders:
  go:
    path: .
    after_render: [go mod tidy, gofmt -w .]
    test: [task test:unit]
extension_points:
  api.middleware:
    summary: A middleware.
    file: go/server.go
    item: [code, import]
delimiters:
  - glob: "go/**/Taskfile*.yml"
    left: "[["
    right: "]]"
`)},
			"starters/api/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
			"starters/worker/starter.yml":    &fstest.MapFile{Data: []byte("summary: A worker.\nfolders:\n  go:\n    path: .\n")},
			"starters/worker/files/go/x.txt": &fstest.MapFile{Data: []byte("x\n")},
			"features/auth/feature.yml": &fstest.MapFile{Data: []byte(`summary: Auth.
targets: [api]
needs:
  cli:
  log:
    level: debug
files:
  - path: files/
options:
  method:
    values: [oauth, api-key]
    default: api-key
extension_points:
  auth.checks:
    summary: A check.
    file: go/auth.go
    item: [code]
add_to:
  api.middleware:
    - code: auth.Middleware
      import: internal/auth
`)},
			"features/cli/feature.yml": &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [worker, api]\n")},
			"features/log/feature.yml": &fstest.MapFile{Data: []byte("summary: Logs.\ntargets: [api]\noptions:\n  level:\n    default: info\n    when: 'starter.region == \"eu\"'\n")},
		}
	}
	const workerAndCLI = `
		"worker": {"summary": "A worker.", "folders": {"go": {"path": ".", "after_render": [], "test": []}}, "options": {}, "extension_points": {}}`
	const cliJSON = `
		"cli": {"summary": "A CLI.", "targets": ["worker", "api"], "needs": {}, "options": {}, "extension_points": {}, "add_to": {}}`

	t.Run("json", func(t *testing.T) {
		t.Run("--catalog prints the catalog in that folder, also with no config file", func(t *testing.T) {
			catalogDir := writtenCatalog(t, map[string]string{"starters/other/starter.yml": "summary: Other.\nfolders:\n  docs:\n    path: .\n"})
			command := newShowCommand(func(context.Context, string) (catalog.Catalogs, error) {
				return catalog.Catalogs{}, errors.New("no config file")
			})
			var printed bytes.Buffer
			command.Writer, command.ErrWriter = &printed, io.Discard

			err := command.Run(context.Background(), []string{"show", "--catalog", catalogDir, "--output", "json"})

			require.NoError(t, err)
			require.Equal(t, []string{"other"}, slices.Sorted(maps.Keys(showOutput(t, printed.String()).Starters)))
		})

		t.Run("--output json prints each starter and each feature with each field, a field with no value as [], {} or an empty text, and the lists in the order of the files", func(t *testing.T) {
			printed, err := runShow(t, withEachField(), "--output", "json")

			require.NoError(t, err)
			require.JSONEq(t, `{
				"starters": {
					"api": {
						"summary": "An API.",
						"folders": {"go": {"path": ".", "after_render": ["go mod tidy", "gofmt -w ."], "test": ["task test:unit"]}},
						"options": {
							"region": {"default": "us", "pattern": "", "values": ["us", "eu"], "when": ""},
							"module": {"default": "example.com/{{ .Name }}", "pattern": "^[a-z./]+$", "values": [], "when": ""}
						},
						"extension_points": {"api.middleware": {"summary": "A middleware.", "file": "go/server.go", "item": ["code", "import"]}}
					},`+workerAndCLI+`
				},
				"features": {
					"auth": {
						"summary": "Auth.",
						"targets": ["api"],
						"needs": {"cli": {}, "log": {"level": "debug"}},
						"options": {"method": {"default": "api-key", "pattern": "", "values": ["oauth", "api-key"], "when": ""}},
						"extension_points": {"auth.checks": {"summary": "A check.", "file": "go/auth.go", "item": ["code"]}},
						"add_to": {"api.middleware": [{"code": "auth.Middleware", "import": "internal/auth"}]}
					},`+cliJSON+`,
					"log": {"summary": "Logs.", "targets": ["api"], "needs": {}, "options": {"level": {"default": "info", "pattern": "", "values": [], "when": "starter.region == \"eu\""}}, "extension_points": {}, "add_to": {}}
				}
			}`, printed)
		})

		t.Run("show <starter> prints only that starter and the features whose targets name it", func(t *testing.T) {
			printed, err := runShow(t, withEachField(), "worker", "--output", "json")

			require.NoError(t, err)
			require.JSONEq(t, `{"starters": {`+workerAndCLI+`}, "features": {`+cliJSON+`}}`, printed)
		})

		for _, arguments := range [][]string{{}, {"worker"}} {
			t.Run("with no flag and a stdout that is not a terminal, show "+strings.Join(arguments, " ")+" prints the same JSON as with --output json", func(t *testing.T) {
				withFlag, err := runShow(t, withEachField(), append(slices.Clone(arguments), "--output", "json")...)
				require.NoError(t, err)

				withoutFlag, err := runShow(t, withEachField(), arguments...)

				require.NoError(t, err)
				require.Equal(t, withFlag, withoutFlag)
			})
		}

		t.Run("features holds a feature whose targets name no starter of the catalog, and a file directly in features is no feature", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/starter.yml":    &fstest.MapFile{Data: []byte("summary: An API.\nfolders:\n  go:\n    path: .\n")},
				"features/orphan/feature.yml": &fstest.MapFile{Data: []byte("summary: Orphan.\ntargets: [nowhere]\n")},
				"features/README.md":          &fstest.MapFile{Data: []byte("the features\n")},
			}

			printed, err := runShow(t, files, "--output", "json")

			require.NoError(t, err)
			require.Equal(t, []string{"orphan"}, slices.Sorted(maps.Keys(showOutput(t, printed).Features)))
		})

		t.Run("the JSON is indented by two spaces", func(t *testing.T) {
			files := fstest.MapFS{"starters/api/starter.yml": &fstest.MapFile{Data: []byte("summary: An API.\n")}}

			printed, err := runShow(t, files, "--output", "json")

			require.NoError(t, err)
			require.Equal(t, `{
  "starters": {
    "api": {
      "summary": "An API.",
      "folders": {},
      "options": {},
      "extension_points": {}
    }
  },
  "features": {}
}
`, printed)
		})

		t.Run("a catalog with no features folder gives no feature", func(t *testing.T) {
			files := fstest.MapFS{"starters/api/starter.yml": &fstest.MapFile{Data: []byte("summary: An API.\n")}}

			printed, err := runShow(t, files, "--output", "json")

			require.NoError(t, err)
			require.JSONEq(t, `{"starters": {"api": {"summary": "An API.", "folders": {}, "options": {}, "extension_points": {}}}, "features": {}}`, printed)
		})

		t.Run("show reads no template, so a template outside the folders of its starter does not stop it", func(t *testing.T) {
			files := withEachField()
			files["starters/api/files/stray.txt"] = &fstest.MapFile{Data: []byte("stray\n")}

			printed, err := runShow(t, files, "--output", "json")

			require.NoError(t, err)
			require.Contains(t, showOutput(t, printed).Starters, "api")
		})

		t.Run("the doc of the commands names the path of each field of the JSON, and no other path", func(t *testing.T) {
			printed, err := runShow(t, withEachField(), "--output", "json")
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal([]byte(printed), &document))
			commands, err := os.ReadFile("../../docs/commands.md")
			require.NoError(t, err)
			section := string(commands)[strings.Index(string(commands), "## Show the Catalog"):]
			section = section[:strings.Index(section[1:], "\n## ")+1]

			var documented []string
			for _, row := range regexp.MustCompile("(?m)^\\| `((?:starters|features)\\.[^`]+)`").FindAllStringSubmatch(section, -1) {
				documented = append(documented, row[1])
			}

			require.ElementsMatch(t, fieldPaths(document, ""), documented)
		})
	})

	t.Run("help", func(t *testing.T) {
		t.Run("--help shows --catalog and the catalog that show reads with it", func(t *testing.T) {
			printed, err := runShow(t, withEachField(), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "--catalog")
			require.Contains(t, printed, "show the catalog in this folder, or only this catalog of the config file")
		})

		t.Run("--help shows what show does, its optional starter and the json output", func(t *testing.T) {
			printed, err := runShow(t, withEachField(), "--help")

			require.NoError(t, err)
			for _, part := range []string{"show - show the starters, the features and the extension points of the catalog\n", "options] [<starter>]\n", "--output string   print the catalog in this format: json\n"} {
				require.Contains(t, printed, part)
			}
		})
	})

	t.Run("tui", func(t *testing.T) {
		t.Run("a TUI that stops with an error gives that error to show", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			command := &cli.Command{Reader: strings.NewReader(""), Writer: io.Discard}

			err := browse(ctx, command, catalog.Definitions{}, "")

			require.ErrorIs(t, err, tea.ErrProgramKilled)
		})

		t.Run("an unknown starter stops the TUI before it starts", func(t *testing.T) {
			command := &cli.Command{Reader: strings.NewReader(""), Writer: io.Discard}

			err := browse(context.Background(), command, catalog.Definitions{Starters: map[string]catalog.StarterDefinition{"api": {}}}, "nope")

			require.EqualError(t, err, `unknown starter "nope": the catalog has api`)
		})
	})

	t.Run("stop", func(t *testing.T) {
		for name, test := range map[string]struct {
			files     fstest.MapFS
			arguments []string
			message   string
		}{
			"an unknown starter":                    {withEachField(), []string{"nope"}, `unknown starter "nope": the catalog has api, worker`},
			"an unknown starter with --output json": {withEachField(), []string{"nope", "--output", "json"}, `unknown starter "nope": the catalog has api, worker`},
			"a second argument":                     {withEachField(), []string{"api", "worker"}, "show takes at most one argument: show [<starter>]"},
			"an output other than json":             {withEachField(), []string{"--output", "yaml"}, `unknown output "yaml": show prints only json`},
			"a starter.yml that is not valid YAML":  {fstest.MapFS{"starters/api/starter.yml": &fstest.MapFile{Data: []byte("summary: [\n")}}, nil, "read starters/api/starter.yml"},
			"a feature.yml that is not valid YAML":  {fstest.MapFS{"starters/api/starter.yml": &fstest.MapFile{Data: []byte("summary: An API.\n")}, "features/cli/feature.yml": &fstest.MapFile{Data: []byte("targets: [\n")}}, nil, "read features/cli/feature.yml"},
		} {
			t.Run(name+" stops show before it prints anything, and the message says why", func(t *testing.T) {
				printed, err := runShow(t, test.files, test.arguments...)

				require.ErrorContains(t, err, test.message)
				require.Empty(t, printed)
			})
		}

		t.Run("a catalog that cannot load stops show with its error before it prints anything", func(t *testing.T) {
			var printed bytes.Buffer
			command := newShowCommand(func(context.Context, string) (catalog.Catalogs, error) {
				return catalog.Catalogs{}, errors.New("no config file at /home/config.yml")
			})
			command.Writer, command.ErrWriter = &printed, &printed

			err := command.Run(context.Background(), []string{"show"})

			require.ErrorContains(t, err, "no config file at /home/config.yml")
			require.Empty(t, printed.String())
		})
	})
}

func showOutput(t *testing.T, printed string) catalog.Definitions {
	t.Helper()
	var definitions catalog.Definitions
	require.NoError(t, json.Unmarshal([]byte(printed), &definitions))
	return definitions
}

func fieldPaths(value any, parent string) []string {
	object, ok := value.(map[string]any)
	if !ok {
		return []string{parent}
	}
	var paths []string
	for key, field := range object {
		if slices.Contains([]string{"starters", "features", "folders", "options", "extension_points", "add_to", "needs"}, lastSegment(parent)) || strings.HasSuffix(parent, ".needs.<name>") {
			key = "<name>"
		}
		child := key
		if parent != "" {
			child = parent + "." + key
		}
		for _, path := range fieldPaths(field, child) {
			if !slices.Contains(paths, path) {
				paths = append(paths, path)
			}
		}
	}
	return paths
}

func lastSegment(path string) string {
	return path[strings.LastIndex(path, ".")+1:]
}
