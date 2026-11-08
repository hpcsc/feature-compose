//go:build unit

package catalog_test

import (
	"errors"
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
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
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
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", map[string]string{"module": "example.com/acme ledger;rm -rf ~"}, nil)

			require.ErrorContains(t, err, "module")
		})

		t.Run("a name with capitals and dots gives a kebab-case name in the Docker names of the repo", func(t *testing.T) {
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
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

		writtenWithCLI := func(t *testing.T, name string, given map[string]string) (string, map[string]string) {
			t.Helper()
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select(name, given, []string{"cli"})
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
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"module": "example.com/acme/ledger"}, []string{"cli"})
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
			starter, err := catalog.Embedded().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("api", nil, []string{"cli"})
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
}
