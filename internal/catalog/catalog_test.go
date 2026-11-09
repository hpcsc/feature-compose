//go:build unit

package catalog_test

import (
	"errors"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

type failingFS struct {
	files   fstest.MapFS
	failing string
}

func (f failingFS) Open(name string) (fs.File, error) {
	if name == f.failing {
		return nil, errors.New("disk failure")
	}
	return f.files.Open(name)
}

func TestCatalog(t *testing.T) {
	t.Run("starter", func(t *testing.T) {
		t.Run("an unknown name stops, and the message lists the starters of the catalog", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/alpha/starter.yml": &fstest.MapFile{Data: []byte("summary: Alpha.\nfolders:\n  go:\n    path: .\n")},
				"starters/beta/starter.yml":  &fstest.MapFile{Data: []byte("summary: Beta.\nfolders:\n  go:\n    path: .\n")},
			}

			_, err := catalog.New(files).Starter("gamma")

			require.ErrorContains(t, err, `"gamma"`)
			require.ErrorContains(t, err, "alpha, beta")
		})

		t.Run("a name that is a path does not reach a starter", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/alpha/starter.yml": &fstest.MapFile{Data: []byte("summary: Alpha.\nfolders:\n  go:\n    path: .\n")},
				"starters/beta/starter.yml":  &fstest.MapFile{Data: []byte("summary: Beta.\nfolders:\n  go:\n    path: .\n")},
			}

			_, err := catalog.New(files).Starter("alpha/../beta")

			require.ErrorContains(t, err, "alpha, beta")
		})

		t.Run("a starter.yml that is not valid YAML stops, and the message names the file", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders: [go\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}

			_, err := catalog.New(files).Starter("service")

			require.ErrorContains(t, err, "starters/service/starter.yml")
		})

		t.Run("a template outside the folders of the starter stops, and the message names the template", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":          &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go":     &fstest.MapFile{Data: []byte("package main\n")},
				"starters/service/files/web/index.html": &fstest.MapFile{Data: []byte("<html></html>\n")},
			}

			_, err := catalog.New(files).Starter("service")

			require.ErrorContains(t, err, "web/index.html")
		})

		t.Run("a template with the name of a folder directly under the files folder stops, and the message names the template", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml": &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go":    &fstest.MapFile{Data: []byte("package main\n")},
			}

			_, err := catalog.New(files).Starter("service")

			require.ErrorContains(t, err, "template go ")
		})

		t.Run("a pattern that is not a valid regular expression stops, and the message names the option", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("options:\n  go-version:\n    pattern: '^1\\.(2[7-9]$'\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}

			_, err := catalog.New(files).Starter("service")

			require.ErrorContains(t, err, "go-version")
			require.ErrorContains(t, err, "missing closing )")
		})

		t.Run("a delimiters glob that is not valid stops, and the message names the glob", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\ndelimiters:\n  - glob: \"go/**/[Task\"\n    left: \"[[\"\n    right: \"]]\"\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}

			_, err := catalog.New(files).Starter("service")

			require.ErrorContains(t, err, `"go/**/[Task"`)
		})

		t.Run("a delimiters glob with an empty segment stops, and the message names the glob", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\ndelimiters:\n  - glob: \"go//Taskfile.yml\"\n    left: \"[[\"\n    right: \"]]\"\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}

			_, err := catalog.New(files).Starter("service")

			require.ErrorContains(t, err, `"go//Taskfile.yml"`)
		})

		for name, pair := range map[string]string{"left": `left: "[["`, "right": `right: "]]"`} {
			t.Run("delimiters with only "+name+" stop, and the message names the glob", func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\ndelimiters:\n  - glob: \"go/Taskfile.yml\"\n    " + pair + "\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				}

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, `"go/Taskfile.yml"`)
			})
		}

		t.Run("a folder whose path leads out of the repo stops, and the message names the folder", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n  web:\n    path: ../web\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}

			_, err := catalog.New(files).Starter("service")

			require.ErrorContains(t, err, "web")
			require.ErrorContains(t, err, `"../web"`)
		})

		for glob, definition := range map[string]string{
			"go/[Task": "targets: [service]\nfiles:\n  - path: files/\ndelimiters:\n  - glob: \"go/[Task\"\n    left: \"[[\"\n    right: \"]]\"\n",
			"go/a.md":  "targets: [service]\nfiles:\n  - path: files/\ndelimiters:\n  - glob: go/a.md\n    left: \"[[\"\n",
		} {
			t.Run("delimiters of a feature that are not valid stop, and the message names the feature and the glob "+glob, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
					"features/cli/feature.yml":          &fstest.MapFile{Data: []byte(definition)},
					"features/cli/files/go/a.md":        &fstest.MapFile{Data: []byte("a\n")},
				}

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "features/cli/feature.yml")
				require.ErrorContains(t, err, strconv.Quote(glob))
			})
		}

		for name, features := range map[string]fstest.MapFS{
			"a feature.yml that is not valid YAML": {
				"features/cli/feature.yml": &fstest.MapFile{Data: []byte("targets: [service\n")},
			},
			"a files entry of a feature with no folder": {
				"features/cli/feature.yml": &fstest.MapFile{Data: []byte("targets: [service]\nfiles:\n  - path: missing/\n")},
			},
			"an option of a feature with a pattern that is not valid": {
				"features/cli/feature.yml": &fstest.MapFile{Data: []byte("targets: [service]\noptions:\n  port:\n    pattern: '[0-9'\n")},
			},
			"a files entry that leads into a sibling feature whose name starts with the name of the feature": {
				"features/cli/feature.yml":         &fstest.MapFile{Data: []byte("targets: [service]\nfiles:\n  - path: ../cli-extra/files/\n")},
				"features/cli-extra/feature.yml":   &fstest.MapFile{Data: []byte("targets: [other]\nfiles:\n  - path: files/\n")},
				"features/cli-extra/files/go/a.md": &fstest.MapFile{Data: []byte("a\n")},
			},
			"a files entry that leads out of the feature": {
				"features/cli/feature.yml":    &fstest.MapFile{Data: []byte("targets: [service]\nfiles:\n  - path: ../docs/files/\n")},
				"features/docs/feature.yml":   &fstest.MapFile{Data: []byte("targets: [other]\nfiles:\n  - path: files/\n")},
				"features/docs/files/go/a.md": &fstest.MapFile{Data: []byte("a\n")},
			},
		} {
			t.Run(name+" stops, and the message names the feature", func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				}
				maps.Copy(files, features)

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "features/cli")
			})
		}

		t.Run("a file directly in the features folder is not a feature", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/README.md":                &fstest.MapFile{Data: []byte("the features of the catalog\n")},
			}

			_, err := catalog.New(files).Starter("service")

			require.NoError(t, err)
		})

		for _, failing := range []string{
			"starters",
			"starters/service/starter.yml",
			"starters/service/files",
			"starters/service/files/go/main.go",
			"features",
			"features/cli/feature.yml",
		} {
			t.Run("a read of "+failing+" that fails stops with its cause", func(t *testing.T) {
				files := failingFS{failing: failing, files: fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
					"features/cli/feature.yml":          &fstest.MapFile{Data: []byte("targets: [service]\nfiles:\n  - path: files/\n")},
					"features/cli/files/go/cli.go":      &fstest.MapFile{Data: []byte("package cli\n")},
				}}

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "disk failure")
			})
		}
	})

	t.Run("embedded", func(t *testing.T) {
		writtenStarter := func(t *testing.T, given map[string]string) (string, map[string]string) {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", given, nil)
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return output, filesIn(t, output)
		}

		t.Run("the catalog in the binary holds go-event-sourcing-service and its files whose names start with a dot, from any working folder", func(t *testing.T) {
			t.Chdir(t.TempDir())

			_, files := writtenStarter(t, nil)

			require.Subset(t, slices.Collect(maps.Keys(files)), []string{
				".github/workflows/ci.yml", ".gitignore", ".env.local", ".env.integration", ".tmuxp.yaml", "go.mod", "cmd/api/main.go",
			})
		})

		t.Run("with no given values, the repo has the name, the default module and the default Go version", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.Contains(t, files["go.mod"], "module github.com/hpcsc/my-service\n")
			require.Contains(t, files["go.mod"], "go 1.27.1\n")
			require.Contains(t, files["internal/domain/reaction/conflict_retry.go"], `"github.com/hpcsc/my-service/internal/domain/event"`)
			for _, path := range []string{"mise.toml", "Dockerfile.api", "Dockerfile.catchup"} {
				require.Contains(t, files[path], "1.27.1", path)
			}
			require.NotContains(t, files, "go.sum")
		})

		t.Run("each file where the starter uses the name has the name, also with another module", func(t *testing.T) {
			_, files := writtenStarter(t, map[string]string{"module": "example.com/acme/ledger"})

			for _, path := range []string{
				"README.md", ".tmuxp.yaml", "Taskfile.api.yml", "Taskfile.catchup.yml", "Taskfile.yml", "scripts/test/start-test-dependencies.sh",
				"internal/domain/event/constants.go", "internal/domain/event/registry/registry_test.go",
			} {
				require.Contains(t, files[path], "my-service", path)
			}
		})

		t.Run("a module value that a Go module path cannot hold stops the selection", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", map[string]string{"module": "example.com/acme ledger;rm -rf ~"}, nil)

			require.ErrorContains(t, err, "module")
		})

		t.Run("a name with capitals and dots gives a kebab-case name in the Docker names of the repo", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("MyService.v2", nil, nil)
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "repo")
			require.NoError(t, selection.Write(output))
			files := filesIn(t, output)

			require.Equal(t, 5, strings.Count(files["Taskfile.yml"], "-p my-service-v2-local "))
			require.Contains(t, files["Taskfile.api.yml"], "-t my-service-v2-api:")
			require.Contains(t, files["Taskfile.catchup.yml"], "-t my-service-v2-catchup:")
			require.Contains(t, files["scripts/test/start-test-dependencies.sh"], "COMPOSE_PROJECT_NAME=my-service-v2-integration-test")
			require.Contains(t, files["internal/domain/event/constants.go"], `Namespace     = "my-service-v2"`)
			require.Contains(t, files[".tmuxp.yaml"], "session_name: my-service-v2\n")
			require.True(t, strings.HasPrefix(files["README.md"], "# my-service-v2\n"))
		})

		t.Run("given values for the module and the Go version replace the defaults in each file", func(t *testing.T) {
			_, files := writtenStarter(t, map[string]string{"module": "example.com/acme/ledger", "go-version": "1.28.0"})

			require.Contains(t, files["go.mod"], "module example.com/acme/ledger\n")
			require.Contains(t, files["go.mod"], "go 1.28.0\n")
			require.Contains(t, files["cmd/api/main.go"], `"example.com/acme/ledger/internal/`)
			for _, path := range []string{"Dockerfile.api", "Dockerfile.catchup"} {
				require.Contains(t, files[path], "-X 'example.com/acme/ledger/internal/api/use_cases/root.Version=${VERSION}'", path)
			}
			for path, contents := range files {
				require.NotContains(t, contents, "github.com/hpcsc/my-service", path)
				require.NotContains(t, contents, "1.27.1", path)
			}
		})

		t.Run("the Taskfiles and the GitHub workflow keep the template syntax of Task and of GitHub Actions", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.Contains(t, files["Taskfile.yml"], "{{.CLI_ARGS}}")
			require.Contains(t, files["Taskfile.api.yml"], `-t my-service-api:{{ default "latest" .TAG }}`)
			require.Contains(t, files[".github/workflows/ci.yml"], "${{ secrets.GITHUB_TOKEN }}")
		})

		t.Run("no written file holds a value of scaffold", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			for path, contents := range files {
				for _, value := range []string{".Scaffold.", ".ProjectKebab", ".ModulePath", "<no value>", "zzsentinel"} {
					require.NotContains(t, contents, value, path)
				}
			}
		})

		t.Run("the repo has no file and no edit of the process manager example", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			processManager := regexp.MustCompile(`^(internal/use_cases/payment/|internal/api/use_cases/(payment|webhook)/|internal/use_cases/account/(holds/|hold(_test)?\.go$|event/hold_|command/[^/]*_hold\.go$)|internal/fakeservices/|cmd/fakeservices/|Taskfile\.fakeservices\.yml$|Dockerfile\.fakeservices$|migrations/[^/]*_create-payment(s|-deadlines)-table\.|docs/process-manager-example\.md$|docs/event-model-payment\.)`)
			for path, contents := range files {
				require.False(t, processManager.MatchString(path), path)
				require.NotContains(t, contents, "fakeservices", path)
				require.NotContains(t, contents, "HoldPlaced", path)
			}
		})

		t.Run("the repo has no shared-token check: no token file, and no file reads, sets, sends or describes a token", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			for _, path := range []string{"internal/api/middleware/auth.go", "internal/api/middleware/auth_test.go", "internal/api/middleware/testdata/valid-token", "local/api/token"} {
				require.NotContains(t, files, path)
			}
			for path, contents := range files {
				withoutGitHubToken := strings.NewReplacer("github_token", "", "repo-token", "", "GITHUB_TOKEN", "").Replace(contents)
				for _, word := range []string{"token", "bearer", "authorization"} {
					require.NotContains(t, strings.ToLower(withoutGitHubToken), word, path)
				}
			}
		})

		writtenWithGuard := func(t *testing.T) map[string]string {
			t.Helper()
			files := catalogOnDisk(t)
			files["features/guard/feature.yml"] = &fstest.MapFile{Data: []byte(`summary: Guard.
targets: [go-event-sourcing-service]
add_to:
  api.middleware:
    - code: guard.Middleware(cfg.Guard)
      import: internal/guard
    - code: guard.Audit(cfg.Guard)
      import: internal/guard
  api.config:
    - name: Guard
      type: guard.Config
      import: internal/guard
`)}
			starter, err := catalog.New(files).Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues("guard"))
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		t.Run("the server gives the items of api.middleware, in their order, to the protected routes, and imports their package one time", func(t *testing.T) {
			files := writtenWithGuard(t)

			require.Contains(t, files["internal/api/server/server.go"], "\t\tguard.Middleware(cfg.Guard),\n\t\tguard.Audit(cfg.Guard),\n\t}")
			require.Equal(t, 1, strings.Count(files["internal/api/server/server.go"], `"github.com/hpcsc/my-service/internal/guard"`))
		})

		t.Run("the config of the API has a field for each item of api.config, and imports its package one time", func(t *testing.T) {
			files := writtenWithGuard(t)

			require.Contains(t, files["internal/api/config.go"], "\tGuard guard.Config\n}")
			require.Equal(t, 1, strings.Count(files["internal/api/config.go"], `"github.com/hpcsc/my-service/internal/guard"`))
		})

		t.Run("with no item, the server gives no middleware to the protected routes, and the config has only its own fields", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.Contains(t, files["internal/api/server/server.go"], "return []func(http.Handler) http.Handler{\n\t}")
			require.Contains(t, files["internal/api/server/server.go"], "\t\"github.com/hpcsc/my-service/internal/api/use_cases\"\n\t\"github.com/jackc/pgx/v5/pgxpool\"\n)")
			require.Contains(t, files["internal/api/config.go"], "import (\n\t\"github.com/hpcsc/my-service/internal/common/config\"\n)\n")
			require.Contains(t, files["internal/api/config.go"], "\tDB   config.DB\n}")
		})

		t.Run("no written file holds the name of an extension point, with or without items", func(t *testing.T) {
			_, withoutItems := writtenStarter(t, nil)

			for _, files := range []map[string]string{withoutItems, writtenWithGuard(t)} {
				for path, contents := range files {
					for _, extensionPoint := range []string{"api.middleware", "api.config"} {
						require.NotContains(t, contents, extensionPoint, path)
					}
				}
			}
		})

		writtenWithAPIAuth := func(t *testing.T, given map[string]string, apiAuthValues map[string]string) map[string]string {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", given, map[string]map[string]string{"api-auth": apiAuthValues})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		t.Run("api-auth adds its files, and each file of the starter stays as without it except the server and the config of the API", func(t *testing.T) {
			_, withoutAPIAuth := writtenStarter(t, nil)

			withAPIAuth := writtenWithAPIAuth(t, nil, nil)

			added := map[string]string{}
			var changed []string
			for path, contents := range withAPIAuth {
				before, ok := withoutAPIAuth[path]
				switch {
				case !ok:
					added[path] = contents
				case before != contents:
					changed = append(changed, path)
				}
			}
			require.ElementsMatch(t, []string{
				"internal/auth/config.go", "internal/auth/key_hashes.go", "internal/auth/key_hashes_test.go",
				"internal/auth/middleware.go", "internal/auth/middleware_test.go", "docs/api-auth.md",
			}, slices.Collect(maps.Keys(added)))
			require.ElementsMatch(t, []string{"internal/api/server/server.go", "internal/api/config.go"}, changed)
			require.Len(t, withAPIAuth, len(withoutAPIAuth)+len(added))
		})

		for module, auth := range map[string]string{
			"":                       "github.com/hpcsc/my-service/internal/auth",
			"github.com/acme/ledger": "github.com/acme/ledger/internal/auth",
		} {
			t.Run("the server and the config of the API hold the items of api-auth and import "+auth, func(t *testing.T) {
				given := map[string]string{}
				if module != "" {
					given["module"] = module
				}

				files := writtenWithAPIAuth(t, given, nil)

				require.Contains(t, files["internal/api/server/server.go"], "\t\tauth.Middleware(cfg.Auth),\n\t}")
				require.Contains(t, files["internal/api/config.go"], "\tAuth auth.Config\n}")
				for _, path := range []string{"internal/api/server/server.go", "internal/api/config.go"} {
					require.Equal(t, 1, strings.Count(files[path], strconv.Quote(auth)), path)
				}
				if module != "" {
					for path, contents := range files {
						require.NotContains(t, contents, "github.com/hpcsc/my-service", path)
					}
				}
			})
		}

		t.Run("with the value api-key for method, api-auth writes the same files as with no value", func(t *testing.T) {
			require.Equal(t, writtenWithAPIAuth(t, nil, nil), writtenWithAPIAuth(t, nil, map[string]string{"method": "api-key"}))
		})

		t.Run("a value that method does not have stops, and the message names method and its values", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, map[string]map[string]string{"api-auth": {"method": "oauth"}})

			require.ErrorContains(t, err, `the feature api-auth: the value "oauth" of the option method is not one of its values: api-key`)
		})

		t.Run("no Go file of api-auth imports a package of the module other than the packages under internal/auth", func(t *testing.T) {
			files := writtenWithAPIAuth(t, nil, nil)

			for path, contents := range files {
				if !strings.HasPrefix(path, "internal/auth/") {
					continue
				}
				parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.ImportsOnly)
				require.NoError(t, err, path)
				for _, imported := range parsed.Imports {
					importPath, err := strconv.Unquote(imported.Path.Value)
					require.NoError(t, err)
					if strings.HasPrefix(importPath, "github.com/hpcsc/my-service/") {
						require.True(t, strings.HasPrefix(importPath, "github.com/hpcsc/my-service/internal/auth"), path+" imports "+importPath)
					}
				}
			}
		})

		t.Run("the docs of api-auth name the setting of the key hashes and the answers of the API, and only files that the repo has", func(t *testing.T) {
			files := writtenWithAPIAuth(t, nil, nil)
			docs := files["docs/api-auth.md"]

			for _, part := range []string{
				"API_KEY_HASHES", "Authorization: Bearer", "401", "WWW-Authenticate: Bearer", "403",
				"With the method `api-key`", "API_KEY_HASHES=<hash of your key> task api:run",
				"`Taskfile.api.yml`", "`README.md`", "`docker-compose.local.yml`",
			} {
				require.Contains(t, docs, part)
			}
			for _, path := range regexp.MustCompile("`([A-Za-z0-9_./-]+\\.(?:ya?ml|go|md|local))`").FindAllStringSubmatch(docs, -1) {
				require.Contains(t, files, path[1])
			}
		})

		writtenWithCLI := func(t *testing.T, name string, given map[string]string) (string, map[string]string) {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select(name, given, withoutValues("cli"))
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), name)
			require.NoError(t, selection.Write(output))
			return output, filesIn(t, output)
		}

		t.Run("cli adds each of its files at its path, and the files of the starter stay as without cli", func(t *testing.T) {
			_, withoutCLI := writtenStarter(t, nil)

			_, withCLI := writtenWithCLI(t, "my-service", nil)

			added := map[string]string{}
			for path, contents := range withCLI {
				if before, ok := withoutCLI[path]; ok {
					require.Equal(t, before, contents, path)
				} else {
					added[path] = contents
				}
			}
			require.ElementsMatch(t, []string{
				".dockerignore", ".goreleaser.yaml", "scripts/install.sh",
				"cmd/my-service/main.go",
				"internal/cli/adapter.go", "internal/cli/adapter_test.go", "internal/cli/root.go", "internal/cli/update.go",
				"internal/cli/progress/line.go", "internal/cli/progress/line_test.go",
				"internal/cli/release/client.go", "internal/cli/release/updater.go", "internal/cli/release/updater_test.go",
				"internal/cli/version/version.go", "internal/cli/version/version_test.go",
				"docs/cli.md", "docs/e2e-tests.md",
				"e2e/Dockerfile", "e2e/package.json", "e2e/testUtils.ts", "e2e/tsconfig.json", "e2e/vitest.config.ts",
				"e2e/tests/cli.test.ts", "e2e/tests/update.test.ts",
			}, slices.Collect(maps.Keys(added)))
			require.Len(t, withCLI, len(withoutCLI)+len(added))
		})

		t.Run("the files of cli have the name of the repo in each place where go-cli has the project name", func(t *testing.T) {
			_, files := writtenWithCLI(t, "my-service", nil)

			require.Contains(t, files["internal/cli/root.go"], `Name:                  "my-service",`)
			require.Contains(t, files["internal/cli/root.go"], `print the tag my-service was built from`)
			require.Contains(t, files[".goreleaser.yaml"], "project_name: my-service")
			require.Contains(t, files["e2e/package.json"], `"name": "my-service-e2e"`)
			require.Contains(t, files["e2e/tests/cli.test.ts"], "my-service version")
		})

		t.Run("a name with capitals and dots gives a kebab-case name in the binary, the release config and the e2e image of cli", func(t *testing.T) {
			_, files := writtenWithCLI(t, "MyService.v2", nil)

			require.Contains(t, files, "cmd/my-service-v2/main.go")
			require.Contains(t, files["internal/cli/root.go"], `Name:                  "my-service-v2",`)
			require.Contains(t, files[".goreleaser.yaml"], "project_name: my-service-v2\n")
			require.Contains(t, files[".goreleaser.yaml"], "main: ./cmd/my-service-v2\n")
			require.Contains(t, files["e2e/package.json"], `"name": "my-service-v2-e2e"`)
		})

		t.Run("the files of cli have the module and the Go version in each import, ldflag and image", func(t *testing.T) {
			_, files := writtenWithCLI(t, "my-service", map[string]string{"module": "github.com/acme/ledger", "go-version": "1.28.0"})

			require.Contains(t, files["cmd/my-service/main.go"], `"github.com/acme/ledger/internal/cli"`)
			require.Contains(t, files["internal/cli/update.go"], `"github.com/acme/ledger/internal/cli/release"`)
			require.Contains(t, files["internal/cli/version/version_test.go"], `Path: "github.com/acme/ledger"`)
			require.Contains(t, files[".goreleaser.yaml"], "-X github.com/acme/ledger/internal/cli/version.releaseTag={{ .Tag }}")
			require.Contains(t, files["e2e/Dockerfile"], "-X github.com/acme/ledger/internal/cli/version.releaseTag=${BUILD_TAG}")
			require.Contains(t, files["e2e/Dockerfile"], "FROM golang:1.28.0")
			for path, contents := range files {
				require.NotContains(t, contents, "github.com/hpcsc/my-service", path)
			}
		})

		t.Run(".goreleaser.yaml keeps the template syntax of GoReleaser", func(t *testing.T) {
			_, files := writtenWithCLI(t, "my-service", nil)

			require.Contains(t, files[".goreleaser.yaml"], "{{ .Tag }}")
			require.Contains(t, files[".goreleaser.yaml"], "'{{ .ProjectName }}-{{ .Os }}-{{ .Arch }}'")
			require.Contains(t, files[".goreleaser.yaml"], "'{{ incpatch .Version }}-next'")
		})

		t.Run("the root command of the CLI lists version and update with no item of cli.commands, and has no greeting", func(t *testing.T) {
			_, files := writtenWithCLI(t, "my-service", nil)

			require.Contains(t, files["internal/cli/root.go"], "\t\t\tnewVersionCommand(),\n\t\t\tnewUpdateCommand(),\n\t\t},")
			require.NotContains(t, files["internal/cli/root.go"], `"Name: "`)
			require.NotContains(t, files["internal/cli/root.go"], "bufio")
			require.NotContains(t, files["e2e/tests/cli.test.ts"], "Name:")
			require.NotContains(t, files["e2e/tests/cli.test.ts"], "hello world")
		})

		t.Run("the root command of the CLI lists each command that a selected feature adds to cli.commands after version and update, and root.go parses as Go", func(t *testing.T) {
			files := catalogOnDisk(t)
			files["features/accounts/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Accounts.\ntargets: [go-event-sourcing-service]\nadd_to:\n  cli.commands:\n    - code: newOpenCommand()\n    - code: newCloseCommand()\n")}
			starter, err := catalog.New(files).Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues("cli", "accounts"))
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			require.NoError(t, selection.Write(output))

			root, err := os.ReadFile(filepath.Join(output, "internal", "cli", "root.go"))
			require.NoError(t, err)
			require.Contains(t, string(root), "\t\t\tnewVersionCommand(),\n\t\t\tnewUpdateCommand(),\n\t\t\tnewOpenCommand(),\n\t\t\tnewCloseCommand(),\n\t\t},")
			_, err = parser.ParseFile(token.NewFileSet(), "root.go", root, parser.AllErrors)
			require.NoError(t, err)
		})

		writtenWithFeatures := func(t *testing.T, givenByFeature map[string]map[string]string) map[string]string {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, givenByFeature)
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		t.Run("cli-auth adds its files to the repo with cli and api-auth, and only the root command of the CLI changes, which lists auth after version and update", func(t *testing.T) {
			withoutCLIAuth := writtenWithFeatures(t, withoutValues("cli", "api-auth"))

			withCLIAuth := writtenWithFeatures(t, withoutValues("cli-auth"))

			added := map[string]string{}
			var changed []string
			for path, contents := range withCLIAuth {
				before, ok := withoutCLIAuth[path]
				switch {
				case !ok:
					added[path] = contents
				case before != contents:
					changed = append(changed, path)
				}
			}
			require.ElementsMatch(t, []string{
				"internal/credential/credential.go", "internal/credential/command.go", "internal/credential/config.go",
				"internal/credential/credential_test.go", "internal/cli/auth.go", "e2e/tests/auth.test.ts", "docs/cli-auth.md",
			}, slices.Collect(maps.Keys(added)))
			require.Equal(t, []string{"internal/cli/root.go"}, changed)
			require.Len(t, withCLIAuth, len(withoutCLIAuth)+len(added))
			require.Contains(t, withCLIAuth["internal/cli/root.go"], "\t\t\tnewVersionCommand(),\n\t\t\tnewUpdateCommand(),\n\t\t\tnewAuthCommand(),\n\t\t},")
		})

		t.Run("cli-auth alone, and cli-auth with cli and api-auth, give the same files", func(t *testing.T) {
			require.Equal(t, writtenWithFeatures(t, withoutValues("cli-auth")), writtenWithFeatures(t, withoutValues("cli-auth", "cli", "api-auth")))
		})

		t.Run("cli-auth has no option, so a value of method for it stops, and the message says that it has no option", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": {"method": "api-key"}})

			require.ErrorContains(t, err, `unknown option "method": the feature cli-auth has no option`)
		})

		t.Run("the files of a repo with cli-auth hold no value of scaffold", func(t *testing.T) {
			files := writtenWithFeatures(t, withoutValues("cli-auth"))

			for path, contents := range files {
				for _, value := range []string{".Scaffold.", ".Computed.", "go-cli-project", "GO_CLI_PROJECT", "zzsentinel", "<no value>"} {
					require.NotContains(t, contents, value, path)
				}
			}
		})

		t.Run("the Go files of cli-auth import only packages of the module that cli-auth, cli or api-auth writes", func(t *testing.T) {
			files := writtenWithFeatures(t, withoutValues("cli-auth"))

			for _, path := range []string{"internal/credential/credential.go", "internal/credential/command.go", "internal/credential/config.go", "internal/credential/credential_test.go", "internal/cli/auth.go"} {
				parsed, err := parser.ParseFile(token.NewFileSet(), path, files[path], parser.ImportsOnly)
				require.NoError(t, err, path)
				for _, imported := range parsed.Imports {
					importPath, err := strconv.Unquote(imported.Path.Value)
					require.NoError(t, err)
					if inModule, ok := strings.CutPrefix(importPath, "github.com/hpcsc/my-service/"); ok {
						require.Regexp(t, `^internal/(credential|cli|auth)(/|$)`, inModule, path+" imports "+importPath)
					}
				}
			}
		})

		t.Run("the docs of cli-auth name the variable, credentialCommand, auth status and the method of api-auth, and only paths that the repo has", func(t *testing.T) {
			files := writtenWithFeatures(t, withoutValues("cli-auth"))
			docs := files["docs/cli-auth.md"]

			for _, part := range []string{
				"MY_SERVICE_API_KEY", "credentialCommand", "my-service auth status", "Authorization: Bearer <key>",
				"the method `api-key`", "the CLI follows the method of the API", "[api-auth.md](api-auth.md)",
			} {
				require.Contains(t, docs, part)
			}
			for _, path := range regexp.MustCompile("`([A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+)`").FindAllStringSubmatch(docs, -1) {
				require.True(t, slices.ContainsFunc(slices.Collect(maps.Keys(files)), func(written string) bool {
					return written == path[1] || strings.HasPrefix(written, path[1]+"/")
				}), path[1])
			}
		})

		t.Run("the docs of cli-auth name the method that api-auth has in the selection", func(t *testing.T) {
			files := catalogOnDisk(t)
			apiAuth := string(files["features/api-auth/feature.yml"].Data)
			require.Contains(t, apiAuth, "values: [api-key]")
			files["features/api-auth/feature.yml"] = &fstest.MapFile{Data: []byte(strings.Replace(apiAuth, "values: [api-key]", "values: [api-key, zzmethod]", 1))}
			starter, err := catalog.New(files).Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": nil, "api-auth": {"method": "zzmethod"}})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))

			require.Contains(t, filesIn(t, output)["docs/cli-auth.md"], "the method `zzmethod`")
		})

		t.Run("the files of cli hold no value of scaffold", func(t *testing.T) {
			_, files := writtenWithCLI(t, "my-service", nil)

			for path, contents := range files {
				for _, value := range []string{".Scaffold.", ".ProjectKebab", "<no value>", "zzsentinel"} {
					require.NotContains(t, contents, value, path)
				}
			}
		})

		t.Run("the docs of cli name no task or workflow that the repo lacks", func(t *testing.T) {
			_, files := writtenWithCLI(t, "my-service", nil)

			for _, path := range []string{"docs/cli.md", "docs/e2e-tests.md"} {
				for _, missing := range []string{"task ", "Taskfile", ".github/workflows", "release:local"} {
					require.NotContains(t, files[path], missing, path)
				}
			}
		})

		for module, repository := range map[string][2]string{
			"":                             {"hpcsc", "my-service"},
			"github.com/acme/ledger":       {"acme", "ledger"},
			"github.com/acme/ledger/tools": {"acme", "ledger"},
		} {
			t.Run("each file of cli that names the release repository names "+repository[0]+"/"+repository[1]+" for the module "+module, func(t *testing.T) {
				given := map[string]string{}
				if module != "" {
					given["module"] = module
				}

				_, files := writtenWithCLI(t, "my-service", given)

				full := repository[0] + "/" + repository[1]
				require.Contains(t, files["internal/cli/root.go"], `const releaseRepository = "`+full+`"`)
				require.Contains(t, files["scripts/install.sh"], `repo="`+full+`"`)
				require.Contains(t, files[".goreleaser.yaml"], "    owner: "+repository[0]+"\n    name: "+repository[1]+"\n")
				require.Contains(t, files["docs/cli.md"], "https://raw.githubusercontent.com/"+full+"/main/scripts/install.sh")
				require.Contains(t, files["e2e/tests/update.test.ts"], "'/repos/"+full+"/releases/latest'")
				require.Contains(t, files["internal/cli/release/updater_test.go"], `"/repos/`+full+`/releases/latest"`)
				if module != "" {
					_, withoutCLI := writtenStarter(t, given)
					for path, contents := range files {
						if _, ofStarter := withoutCLI[path]; !ofStarter {
							require.NotContains(t, contents, "hpcsc", path)
						}
					}
				}
			})
		}

		t.Run("cli with a module that is not on github.com stops the write before the first file, and the message names the module", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"module": "example.com/acme/ledger"}, withoutValues("cli"))
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			err = selection.Write(output)

			require.ErrorContains(t, err, `"example.com/acme/ledger"`)
			require.ErrorContains(t, err, "GitHub")
			require.NoDirExists(t, output)
		})

		t.Run("the installer of cli can be run", func(t *testing.T) {
			output, _ := writtenWithCLI(t, "my-service", nil)

			require.Equal(t, 0o755&^umask(), modeOf(t, filepath.Join(output, "scripts", "install.sh")))
		})

		t.Run("the installer of cli sends the token in one Authorization header, and never as an argument of its own", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output, _ := writtenWithCLI(t, "my-service", nil)
			stubs := t.TempDir()
			recorded := filepath.Join(stubs, "curl-arguments")
			require.NoError(t, os.WriteFile(filepath.Join(stubs, "curl"), []byte("#!/bin/sh\nfor argument in \"$@\"; do printf '%s\\n' \"$argument\" >> \""+recorded+"\"; done\nexit 22\n"), 0o755))
			for _, tool := range []string{"jq", "tar", "gzip"} {
				require.NoError(t, os.WriteFile(filepath.Join(stubs, tool), []byte("#!/bin/sh\nexit 0\n"), 0o755))
			}
			installer := exec.Command("sh", filepath.Join(output, "scripts", "install.sh"), "--channel", "release", "--dir", t.TempDir())
			installer.Env = append(os.Environ(), "PATH="+stubs+string(os.PathListSeparator)+os.Getenv("PATH"), "GITHUB_TOKEN=secret-token", "GH_TOKEN=")

			printed, err := installer.CombinedOutput()

			require.Error(t, err, string(printed))
			arguments, readErr := os.ReadFile(recorded)
			require.NoError(t, readErr, string(printed))
			lines := strings.Split(strings.TrimSpace(string(arguments)), "\n")
			require.Contains(t, lines, "Authorization: Bearer secret-token")
			require.NotContains(t, lines, "secret-token")
			require.NotContains(t, lines, "Bearer")
		})

		t.Run("a repo named api with cli stops the write, because the starter and cli both write cmd/api/main.go", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("api", nil, withoutValues("cli"))
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "api")

			err = selection.Write(output)

			require.ErrorContains(t, err, "of the starter")
			require.ErrorContains(t, err, "of the feature cli")
			require.ErrorContains(t, err, "both write cmd/api/main.go")
			require.NoDirExists(t, output)
		})

		t.Run("hooks/pre-push and the scripts can be run", func(t *testing.T) {
			output, _ := writtenStarter(t, nil)

			for _, path := range []string{"hooks/pre-push", "scripts/libs/git-hook-prepush.sh", "scripts/local/migrate/entrypoint.sh", "scripts/test/start-test-dependencies.sh"} {
				require.Equal(t, 0o755&^umask(), modeOf(t, filepath.Join(output, path)), path)
			}
		})
	})

	t.Run("combinations", func(t *testing.T) {
		namesOf := func(t *testing.T, files fs.FS) []string {
			t.Helper()
			combinations, err := catalog.New(files).Combinations()
			require.NoError(t, err)
			names := make([]string, 0, len(combinations))
			for _, combination := range combinations {
				names = append(names, combination.String())
			}
			return names
		}
		oneFolder := &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
		mainGo := &fstest.MapFile{Data: []byte("package main\n")}

		t.Run("each starter, in the order of the names, gives a combination of the starter alone and one of each feature that targets it alone", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/worker/starter.yml":         oneFolder,
				"starters/worker/files/go/main.go":    mainGo,
				"starters/api/starter.yml":            oneFolder,
				"starters/api/files/go/main.go":       mainGo,
				"starters/static/starter.yml":         oneFolder,
				"starters/static/files/go/index.html": &fstest.MapFile{Data: []byte("<html></html>\n")},
				"features/docs/feature.yml":           &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [api]\n")},
				"features/cli/feature.yml":            &fstest.MapFile{Data: []byte("summary: CLI.\ntargets: [worker, api]\n")},
			}

			names := namesOf(t, files)

			require.Equal(t, []string{
				"api",
				"api --feature cli",
				"api --feature docs",
				"static",
				"worker",
				"worker --feature cli",
			}, names)
		})

		withValues := fstest.MapFS{
			"starters/api/starter.yml":         &fstest.MapFile{Data: []byte("options:\n  region:\n    values: [eu, us]\n    default: us\n  module:\n    default: example.com/{{ .Name }}\n  tier:\n    values: [standard]\n    default: standard\n  database:\n    values: [postgres, sqlite, mysql]\n    default: postgres\nfolders:\n  go:\n    path: .\n")},
			"starters/api/files/go/values.txt": &fstest.MapFile{Data: []byte(`{{ option "starter.database" }} {{ option "starter.region" }} {{ option "starter.tier" }}`)},
			"features/plain/feature.yml":       &fstest.MapFile{Data: []byte("summary: Plain.\ntargets: [api]\n")},
			"features/auth/feature.yml":        &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\nfiles:\n  - path: files/\noptions:\n  store:\n    values: [file, keychain]\n    default: file\n  realm:\n    default: \"{{ .Name }}-api\"\n  method:\n    values: [api-key, oauth, none]\n    default: oauth\n")},
			"features/auth/files/go/auth.txt":  &fstest.MapFile{Data: []byte(`{{ option "method" }} {{ option "store" }} {{ option "realm" }}`)},
		}
		defaults := "api --option database=postgres --option region=us --option tier=standard"

		t.Run("each other value of an option with values gives one combination, with the other options at their defaults", func(t *testing.T) {
			names := namesOf(t, withValues)

			require.Equal(t, []string{
				defaults,
				"api --option database=sqlite --option region=us --option tier=standard",
				"api --option database=mysql --option region=us --option tier=standard",
				"api --option database=postgres --option region=eu --option tier=standard",
				defaults + " --feature auth:method=oauth,store=file",
				defaults + " --feature auth:method=api-key,store=file",
				defaults + " --feature auth:method=none,store=file",
				defaults + " --feature auth:method=oauth,store=keychain",
				defaults + " --feature plain",
			}, names)
		})

		t.Run("the repo of each combination holds the values that its name gives", func(t *testing.T) {
			combinations, err := catalog.New(withValues).Combinations()
			require.NoError(t, err)
			written := map[string]map[string]string{}

			for _, combination := range combinations {
				starter, err := catalog.New(withValues).Starter(combination.StarterName)
				require.NoError(t, err)
				selection, err := starter.Select("my-service", combination.StarterValues, combination.FeatureValues)
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "my-service")
				require.NoError(t, selection.Write(output))
				written[combination.String()] = filesIn(t, output)
			}

			require.Equal(t, map[string]map[string]string{
				defaults: {"values.txt": "postgres us standard"},
				"api --option database=sqlite --option region=us --option tier=standard":   {"values.txt": "sqlite us standard"},
				"api --option database=mysql --option region=us --option tier=standard":    {"values.txt": "mysql us standard"},
				"api --option database=postgres --option region=eu --option tier=standard": {"values.txt": "postgres eu standard"},
				defaults + " --feature auth:method=oauth,store=file":                       {"values.txt": "postgres us standard", "auth.txt": "oauth file my-service-api"},
				defaults + " --feature auth:method=api-key,store=file":                     {"values.txt": "postgres us standard", "auth.txt": "api-key file my-service-api"},
				defaults + " --feature auth:method=none,store=file":                        {"values.txt": "postgres us standard", "auth.txt": "none file my-service-api"},
				defaults + " --feature auth:method=oauth,store=keychain":                   {"values.txt": "postgres us standard", "auth.txt": "oauth keychain my-service-api"},
				defaults + " --feature plain":                                              {"values.txt": "postgres us standard"},
			}, written)
		})

		t.Run("each pair of features that add items to one extension point gives one combination with both", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/files/go/main.go": mainGo,
				"starters/api/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n  api.config:\n    file: go/config.go\n    item: [name]\n")},
				"features/auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n  api.config:\n    - name: Auth\n")},
				"features/audit/feature.yml":    &fstest.MapFile{Data: []byte("summary: Audit.\ntargets: [api]\nadd_to:\n  api.config:\n    - name: Audit\n  api.middleware:\n    - code: audit.Middleware\n")},
				"features/trace/feature.yml":    &fstest.MapFile{Data: []byte("summary: Trace.\ntargets: [api]\nadd_to:\n  api.middleware:\n    - code: trace.Middleware\n")},
				"features/metrics/feature.yml":  &fstest.MapFile{Data: []byte("summary: Metrics.\ntargets: [api]\nadd_to:\n  metrics.collectors:\n    - code: runtime.Collector\n")},
				"features/cli/feature.yml":      &fstest.MapFile{Data: []byte("summary: CLI.\ntargets: [api]\nextension_points:\n  cli.commands:\n    file: go/root.go\n    item: [code]\n")},
				"features/greet/feature.yml":    &fstest.MapFile{Data: []byte("summary: Greet.\ntargets: [api]\nadd_to:\n  cli.commands:\n    - code: greet.Command\n")},
				"features/hello/feature.yml":    &fstest.MapFile{Data: []byte("summary: Hello.\ntargets: [api]\nadd_to:\n  cli.commands:\n    - code: hello.Command\n")},
				"features/other/feature.yml":    &fstest.MapFile{Data: []byte("summary: Other.\ntargets: [worker]\nadd_to:\n  api.middleware:\n    - code: other.Middleware\n")},
			}

			names := namesOf(t, files)

			require.Equal(t, []string{
				"api",
				"api --feature audit",
				"api --feature auth:method=api-key",
				"api --feature auth:method=oauth",
				"api --feature cli",
				"api --feature greet",
				"api --feature hello",
				"api --feature metrics",
				"api --feature trace",
				"api --feature audit --feature auth:method=api-key",
				"api --feature audit --feature trace",
				"api --feature auth:method=api-key --feature trace",
				"api --feature greet --feature hello",
			}, names)
		})

		t.Run("a pair in which one feature needs the other, also through a need of a need, gives no combination, and the repo of a feature holds its needs", func(t *testing.T) {
			adds := func(featureName, needs string) *fstest.MapFile {
				return &fstest.MapFile{Data: []byte("summary: A feature.\ntargets: [api]\n" + needs + "add_to:\n  api.middleware:\n    - code: " + featureName + "\n")}
			}
			files := fstest.MapFS{
				"starters/api/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n")},
				"starters/api/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }};{{ end }}`)},
				"features/app/feature.yml":        adds("app", "needs: [side, mid]\n"),
				"features/side/feature.yml":       &fstest.MapFile{Data: []byte("summary: Side.\ntargets: [api]\n")},
				"features/mid/feature.yml":        &fstest.MapFile{Data: []byte("summary: Mid.\ntargets: [api]\nneeds: [base]\n")},
				"features/base/feature.yml":       adds("base", ""),
				"features/other/feature.yml":      adds("other", ""),
				"features/zone/feature.yml":       adds("zone", "needs: [base]\n"),
			}

			names := namesOf(t, files)
			starter, err := catalog.New(files).Starter("api")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"app": {}})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))

			require.Equal(t, []string{
				"api",
				"api --feature app",
				"api --feature base",
				"api --feature mid",
				"api --feature other",
				"api --feature side",
				"api --feature zone",
				"api --feature app --feature other",
				"api --feature app --feature zone",
				"api --feature base --feature other",
				"api --feature other --feature zone",
			}, names)
			require.Equal(t, map[string]string{"server.go": "base;app;"}, filesIn(t, output))
		})

		for _, failing := range []string{"starters", "starters/api/starter.yml", "features", "features/cli/feature.yml"} {
			t.Run("a read of "+failing+" that fails stops the list with its cause", func(t *testing.T) {
				files := failingFS{failing: failing, files: fstest.MapFS{
					"starters/api/starter.yml":      oneFolder,
					"starters/api/files/go/main.go": mainGo,
					"features/cli/feature.yml":      &fstest.MapFile{Data: []byte("summary: CLI.\ntargets: [api]\n")},
				}}

				_, err := catalog.New(files).Combinations()

				require.ErrorContains(t, err, "disk failure")
			})
		}
	})
}

func catalogOnDisk(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	err := fs.WalkDir(os.DirFS(repoCatalogDir), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !strings.HasPrefix(path, "starters/") && !strings.HasPrefix(path, "features/") {
			return nil
		}
		contents, err := fs.ReadFile(os.DirFS(repoCatalogDir), path)
		files[path] = &fstest.MapFile{Data: contents}
		return err
	})
	require.NoError(t, err)
	return files
}
