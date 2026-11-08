//go:build unit

package catalog_test

import (
	"errors"
	"io/fs"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/catalog"
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

		for _, failing := range []string{
			"starters",
			"starters/service/starter.yml",
			"starters/service/files",
			"starters/service/files/go/main.go",
		} {
			t.Run("a read of "+failing+" that fails stops with its cause", func(t *testing.T) {
				files := failingFS{failing: failing, files: fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				}}

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "disk failure")
			})
		}
	})

	t.Run("embedded", func(t *testing.T) {
		writtenStarter := func(t *testing.T, given map[string]string) (string, map[string]string) {
			t.Helper()
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", given)
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
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", map[string]string{"module": "example.com/acme ledger;rm -rf ~"})

			require.ErrorContains(t, err, "module")
		})

		t.Run("a name with capitals and dots gives a kebab-case name in the Docker names of the repo", func(t *testing.T) {
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("MyService.v2", nil)
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

		t.Run("hooks/pre-push and the scripts can be run", func(t *testing.T) {
			output, _ := writtenStarter(t, nil)

			for _, path := range []string{"hooks/pre-push", "scripts/libs/git-hook-prepush.sh", "scripts/local/migrate/entrypoint.sh", "scripts/test/start-test-dependencies.sh"} {
				require.Equal(t, 0o755&^umask(), modeOf(t, filepath.Join(output, path)), path)
			}
		})
	})
}
