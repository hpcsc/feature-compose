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
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestCheckCommand(t *testing.T) {
	runCheck := func(t *testing.T, files fstest.MapFS, arguments ...string) (string, error) {
		t.Helper()
		var printed, stderr bytes.Buffer
		command := newCheckCommand(func(context.Context, string) (catalog.Catalogs, error) {
			return catalog.NewCatalogs(catalog.New(files)), nil
		})
		command.Writer = &printed
		command.ErrWriter = &stderr
		err := command.Run(context.Background(), append([]string{"check"}, arguments...))
		return printed.String(), err
	}
	withFeatures := func(features map[string]string) fstest.MapFS {
		files := fstest.MapFS{
			"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code, import]\n")},
			"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }}{{ end }}` + "\n")},
		}
		for featureName, definition := range features {
			files["features/"+featureName+"/feature.yml"] = &fstest.MapFile{Data: []byte(definition)}
		}
		return files
	}

	t.Run("check", func(t *testing.T) {
		t.Run("with no mistake, prints that the catalog has no mistakes and stops with no error", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": "targets: [service]\nneeds: [cli]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n  cli.commands:\n    - code: newAuthCommand()\n",
				"cli":  "targets: [service]\nextension_points:\n  cli.commands:\n    file: go/cli.go\n    item: [code]\nfiles:\n  - path: files/\n",
			})
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte(`{{ range items "cli.commands" }}{{ .code }}{{ end }}` + "\n")}

			printed, err := runCheck(t, files)

			require.NoError(t, err)
			require.Equal(t, "The catalog has no mistakes.\n", printed)
		})

		for name, features := range map[string]map[string]string{
			"with no mistake":   {"auth": "targets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n"},
			"with two mistakes": {"auth": "targets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n  cli.commands:\n    - code: newAuthCommand()\n"},
		} {
			t.Run(name+", a write to stdout that fails stops check with its cause", func(t *testing.T) {
				command := newCheckCommand(func(context.Context, string) (catalog.Catalogs, error) {
					return catalog.NewCatalogs(catalog.New(withFeatures(features))), nil
				})
				command.Writer, command.ErrWriter = failingWriter{}, failingWriter{}

				err := command.Run(context.Background(), []string{"check"})

				require.EqualError(t, err, "closed output")
			})
		}

		t.Run("prints each mistake on its own line, and stops with an error that counts the mistakes", func(t *testing.T) {
			printed, err := runCheck(t, withFeatures(map[string]string{
				"auth":  "targets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n",
				"tools": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n",
			}))

			require.EqualError(t, err, "the catalog has 2 mistakes")
			require.Equal(t, `the starter service: the item 1 that the feature auth adds to the extension point "api.middleware" has the fields [code], but the extension point declares the fields [code, import]
the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare
`, printed)
		})

		t.Run("with one mistake, the error counts one mistake", func(t *testing.T) {
			_, err := runCheck(t, withFeatures(map[string]string{"tools": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n"}))

			require.EqualError(t, err, "the catalog has 1 mistake")
		})

		t.Run("prints the mistakes of the folders, the needs, the cycles and the item fields of the targets", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"api":   "targets: [service]\nneeds: [app]\n",
				"app":   "targets: [service]\nneeds: [base]\n",
				"base":  "targets: [service]\nneeds: [app]\n",
				"auth":  "targets: [service, worker]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n",
				"cli":   "targets: [service]\nneeds:\n  jobs:\n  level:\n    mode: loud\n  vault:\n",
				"jobs":  "targets: [worker]\n",
				"level": "targets: [service]\noptions:\n  mode:\n    values: [quiet]\n    default: quiet\n",
				"web":   "targets: [service]\nfiles:\n  - path: files/\n",
			})
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n")}
			files["starters/worker/files/go/server.go"] = &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }}{{ end }}` + "\n")}
			files["features/web/files/README.md"] = &fstest.MapFile{Data: []byte("web\n")}
			files["features/web/files/go/web.go"] = &fstest.MapFile{Data: []byte("package web\n")}
			files["features/web/files/web/index.html"] = &fstest.MapFile{Data: []byte("<html></html>\n")}

			printed, err := runCheck(t, files)

			require.EqualError(t, err, "the catalog has 8 mistakes")
			require.Equal(t, `the starter service: the feature cli needs jobs, which does not target the starter
the starter service: the feature cli needs vault, which is not a feature of the catalog
the starter service: the need of the feature cli: the feature level: the value "loud" of the option mode is not one of its values: quiet
the starter service: the template README.md of the files entry files/ of the feature web is not in a folder
the starter service: the feature web adds files to the folder web, which the starter does not have
the starter service: the needs of the features form a cycle: app needs base needs app
the starter worker: the item 1 that the feature auth adds to the extension point "api.middleware" has the fields [code, import], but the extension point declares the fields [code]
the feature auth adds to the extension point "api.middleware", which its targets declare with different item fields: [code, import] in the starter service, [code] in the starter worker
`, printed)
		})

		t.Run("prints each when that does not parse with the other mistakes", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth":  "targets: [service]\nfiles:\n  - path: files/\n    when: 'method =='\n",
				"tools": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n",
			})
			files["features/auth/files/go/auth.go"] = &fstest.MapFile{Data: []byte("package auth\n")}

			printed, err := runCheck(t, files)

			require.EqualError(t, err, "the catalog has 2 mistakes")
			require.Equal(t, `the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare
the feature auth: the when "method ==" of the files entry "files/": method == needs a quoted value after it
`, printed)
		})

		t.Run("prints each when of an option that does not parse with the other mistakes", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth":  "targets: [service]\noptions:\n  store-key:\n    default: \"false\"\n    when: 'method =='\n",
				"tools": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n",
			})

			printed, err := runCheck(t, files)

			require.EqualError(t, err, "the catalog has 2 mistakes")
			require.Equal(t, `the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare
the feature auth: the when "method ==" of the option "store-key": method == needs a quoted value after it
`, printed)
		})

		t.Run("prints each reference that a when of an option reads and cannot read", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": "targets: [service]\noptions:\n  store-key:\n    default: \"false\"\n    when: 'api-auth.method == \"api-key\"'\n",
			})

			printed, err := runCheck(t, files)

			require.EqualError(t, err, "the catalog has 1 mistake")
			require.Equal(t, `the feature auth: the when "api-auth.method == \"api-key\"" of the option "store-key" reads "api-auth.method", which the when of an option cannot read: it can read no option
`, printed)
		})

		t.Run("prints each reference that a when reads and the feature cannot read", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": "targets: [service]\noptions:\n  method:\n    default: api-key\nfiles:\n  - path: files/\n    when: 'method == \"oauth\" && starter.region == \"eu\"'\n",
			})
			files["features/auth/files/go/auth.go"] = &fstest.MapFile{Data: []byte("package auth\n")}

			printed, err := runCheck(t, files)

			require.EqualError(t, err, "the catalog has 1 mistake")
			require.Equal(t, `the feature auth: the when "method == \"oauth\" && starter.region == \"eu\"" of the files entry "files/" reads "starter.region", which the starter service does not declare
`, printed)
		})

		t.Run("prints the reads and the parse mistakes of the templates of the starter and of a feature", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": "targets: [service]\nfiles:\n  - path: files/\n"})
			files["starters/service/files/go/main.go"] = &fstest.MapFile{Data: []byte(`{{ option "auth.method" }}` + "\n")}
			files["features/auth/files/go/auth.go"] = &fstest.MapFile{Data: []byte(`{{ option "starter.region" }}` + "\n")}
			files["features/auth/files/go/keys.go"] = &fstest.MapFile{Data: []byte("{{ end }}\n")}

			printed, err := runCheck(t, files)

			require.EqualError(t, err, "the catalog has 3 mistakes")
			require.Equal(t, `the starter service: the template go/main.go reads "auth.method", which the starter cannot read: it can read no option
the feature auth: the template go/auth.go of the files entry files/ reads "starter.region", which the starter service does not declare
the feature auth: the template go/keys.go of the files entry files/ does not parse: template: go/keys.go:1: unexpected {{end}}
`, printed)
		})

		t.Run("prints each extension point of the starter and of a feature whose file does not exist", func(t *testing.T) {
			files := withFeatures(map[string]string{"cli": "targets: [service]\nextension_points:\n  cli.commands:\n    file: go/cli.go\n    item: [code]\n"})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.config:\n    file: go/config.go\n    item: [code]\n  api.middleware:\n    file: go/server.go\n    item: [code, import]\n")}

			printed, err := runCheck(t, files)

			require.EqualError(t, err, "the catalog has 2 mistakes")
			require.Equal(t, `the starter service: the file "go/config.go" of the extension point "api.config" does not exist
the feature cli: the file "go/cli.go" of the extension point "cli.commands" does not exist
`, printed)
		})

		t.Run("prints each path that two owners write", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": "targets: [service]\nfiles:\n  - path: files/\n"})
			files["features/auth/files/go/server.go"] = &fstest.MapFile{Data: []byte("package auth\n")}

			printed, err := runCheck(t, files)

			require.EqualError(t, err, "the catalog has 1 mistake")
			require.Equal(t, "the starter service: the starter and the feature auth both write server.go\n", printed)
		})

		t.Run("--catalog checks the catalog in that folder, also with no config file", func(t *testing.T) {
			catalogDir := writtenCatalog(t, map[string]string{
				"starters/service/starter.yml":      "folders:\n  go:\n    path: .\n",
				"starters/service/files/go/main.go": "package main\n",
				"features/tools/feature.yml":        "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n",
			})
			command := newCheckCommand(func(context.Context, string) (catalog.Catalogs, error) {
				return catalog.Catalogs{}, errors.New("no config file")
			})
			var printed bytes.Buffer
			command.Writer, command.ErrWriter = &printed, io.Discard

			err := command.Run(context.Background(), []string{"check", "--catalog", catalogDir})

			require.EqualError(t, err, "the catalog has 1 mistake")
			require.Equal(t, "the starter service: the feature tools adds to the extension point \"cli.commands\", which the starter, the feature and its needs do not declare\n", printed.String())
		})
	})

	t.Run("build", func(t *testing.T) {
		withCommands := func(afterRender string) fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":         &fstest.MapFile{Data: []byte("folders:\n  app:\n    path: .\n    after_render: [" + afterRender + "]\n    test: [grep -q my-repo README.md, test ! -f broken.txt]\n")},
				"starters/service/files/app/README.md": &fstest.MapFile{Data: []byte("# {{ .Name }}\n")},
			}
		}
		withBroken := func() fstest.MapFS {
			files := withCommands("test -f README.md")
			files["features/broken/feature.yml"] = &fstest.MapFile{Data: []byte("targets: [service]\nfiles:\n  - path: files/\n")}
			files["features/broken/files/app/broken.txt"] = &fstest.MapFile{Data: []byte("broken\n")}
			return files
		}
		durations := regexp.MustCompile(`\(\d+s\)`)

		t.Run("--build makes the repo of each combination and runs its commands, prints FAIL with the error of the command that fails, and stops with an error that names each combination that fails", func(t *testing.T) {
			printed, err := runCheck(t, withBroken(), "--build")

			require.EqualError(t, err, "1 of 2 combinations failed:\n  service --feature broken")
			require.Equal(t, `The catalog has no mistakes.
The catalog has 2 combinations. Make the repo of each, and run its after_render and test commands.
[1/2] ok    service (…)
[2/2] FAIL  service --feature broken (…)
      the test command "test ! -f broken.txt" of the folder app failed: exit status 1
`, durations.ReplaceAllString(printed, "(…)"))
		})

		t.Run("--build says that each repo passes, and stops with no error, when each command passes", func(t *testing.T) {
			printed, err := runCheck(t, withCommands("test -f README.md"), "--build")

			require.NoError(t, err)
			require.Equal(t, `The catalog has no mistakes.
The catalog has 1 combination. Make the repo of each, and run its after_render and test commands.
[1/1] ok    service (…)
The repo of each combination passes its commands.
`, durations.ReplaceAllString(printed, "(…)"))
		})

		t.Run("--build removes the repo of each combination when it ends", func(t *testing.T) {
			folderFile := filepath.Join(t.TempDir(), "folder")

			_, err := runCheck(t, withCommands("pwd > "+folderFile), "--build")

			require.NoError(t, err)
			folder, err := os.ReadFile(folderFile)
			require.NoError(t, err)
			require.NoDirExists(t, strings.TrimSpace(string(folder)))
		})

		t.Run("--build with a mistake prints the mistakes and makes no repo", func(t *testing.T) {
			madeFile := filepath.Join(t.TempDir(), "made")
			files := withCommands("touch " + madeFile)
			files["features/tools/feature.yml"] = &fstest.MapFile{Data: []byte("targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n")}

			_, err := runCheck(t, files, "--build")

			require.EqualError(t, err, "the catalog has 1 mistake")
			require.NoFileExists(t, madeFile)
		})
	})

	t.Run("help", func(t *testing.T) {
		t.Run("--help names --build and what it does", func(t *testing.T) {
			printed, err := runCheck(t, withFeatures(nil), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "--build")
			require.Contains(t, printed, "also make the repo of each combination of the catalog in a temporary folder, and run its after_render and test commands")
		})

		t.Run("--help names --catalog and the folder that it checks", func(t *testing.T) {
			printed, err := runCheck(t, withFeatures(nil), "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "--catalog")
			require.Contains(t, printed, "check the catalog in this folder, or only this catalog of the config file")
		})
	})

	t.Run("stop", func(t *testing.T) {
		t.Run("an argument stops check before it reads the catalog", func(t *testing.T) {
			command := newCheckCommand(func(context.Context, string) (catalog.Catalogs, error) {
				require.FailNow(t, "check read the catalog")
				return catalog.Catalogs{}, nil
			})
			var printed bytes.Buffer
			command.Writer, command.ErrWriter = &printed, &printed

			err := command.Run(context.Background(), []string{"check", "go-event-sourcing-service"})

			require.EqualError(t, err, "check takes no argument")
		})

		for name, folder := range map[string]func(t *testing.T) string{
			"that does not exist": func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") },
			"that is a file": func(t *testing.T) string {
				file := filepath.Join(t.TempDir(), "catalog.yml")
				require.NoError(t, os.WriteFile(file, []byte("starters: []\n"), 0o644))
				return file
			},
		} {
			t.Run("a --catalog path "+name+" stops check, and the message names it", func(t *testing.T) {
				path := folder(t)
				command := newCheckCommand(func(context.Context, string) (catalog.Catalogs, error) {
					return catalog.Catalogs{}, errors.New("no config file")
				})
				command.Writer, command.ErrWriter = io.Discard, io.Discard

				err := command.Run(context.Background(), []string{"check", "--catalog", path})

				require.EqualError(t, err, "the catalog folder "+path+" is not a folder")
			})
		}

		t.Run("a catalog that check cannot read stops it with the cause, and prints nothing", func(t *testing.T) {
			files := withFeatures(nil)
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders: [go\n")}

			printed, err := runCheck(t, files)

			require.ErrorContains(t, err, "starters/service/starter.yml")
			require.Empty(t, printed)
		})

		t.Run("a files entry that leads out of the feature stops check, and the message names the feature.yml and the files entry", func(t *testing.T) {
			printed, err := runCheck(t, withFeatures(map[string]string{"cli": "targets: [service]\nfiles:\n  - path: ../docs/files/\n"}))

			require.ErrorContains(t, err, `read features/cli/feature.yml: the files entry "../docs/files/" is outside the folder of the feature`)
			require.Empty(t, printed)
		})
	})
}
