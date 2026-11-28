//go:build unit

package catalog_test

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
	"time"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestSelection(t *testing.T) {
	selectionOf := func(t *testing.T, files fstest.MapFS, name string) catalog.Selection {
		t.Helper()
		starter, err := catalog.New(files).Starter("service")
		require.NoError(t, err)
		selection, err := starter.Select(name, nil, nil, catalog.Answers{})
		require.NoError(t, err)
		return selection
	}
	oneFolder := &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")}
	writtenRepo := func(t *testing.T, starterYML string, output string) catalog.Selection {
		t.Helper()
		files := fstest.MapFS{
			"starters/service/starter.yml":          &fstest.MapFile{Data: []byte(starterYML)},
			"starters/service/files/app/main.go":    &fstest.MapFile{Data: []byte("package main\n")},
			"starters/service/files/web/index.html": &fstest.MapFile{Data: []byte("<html></html>\n")},
		}
		selection := selectionOf(t, files, "my-service")
		require.NoError(t, selection.Write(output))
		return selection
	}

	featureSelection := func(t *testing.T, files fstest.MapFS, features ...string) catalog.Selection {
		t.Helper()
		starter, err := catalog.New(files).Starter("service")
		require.NoError(t, err)
		selection, err := starter.Select("my-service", nil, withoutValues(features...), catalog.Answers{})
		require.NoError(t, err)
		return selection
	}
	withFeatureDefinitions := func(definitions map[string]string) fstest.MapFS {
		files := fstest.MapFS{
			"starters/service/starter.yml":      oneFolder,
			"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
		}
		for featureName, definition := range definitions {
			files["features/"+featureName+"/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A feature.\ntargets: [service]\n" + definition)}
		}
		return files
	}
	withTwoNeeds := func() fstest.MapFS {
		return withFeatureDefinitions(map[string]string{"x": "needs: [b]\n", "y": "needs: [b]\n", "b": "", "c": ""})
	}

	t.Run("write", func(t *testing.T) {
		t.Run("refuses a starter with a generate command, because only MakeRepo runs the command", func(t *testing.T) {
			selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }}
`})
			output := filepath.Join(t.TempDir(), "my-service")

			err := selection.Write(output)

			require.ErrorContains(t, err, "the starter service runs a generate command, so only MakeRepo writes its repo")
			require.NoDirExists(t, output)
		})

		t.Run("writes each template to the path of its folder in the repo", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":                  &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n  web:\n    path: frontend\n")},
				"starters/service/files/go/main.go":             &fstest.MapFile{Data: []byte("package main\n")},
				"starters/service/files/go/internal/app/app.go": &fstest.MapFile{Data: []byte("package app\n")},
				"starters/service/files/web/index.html":         &fstest.MapFile{Data: []byte("<html></html>\n")},
			}
			output := filepath.Join(t.TempDir(), "work", "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":             "package main\n",
				"internal/app/app.go": "package app\n",
				"frontend/index.html": "<html></html>\n",
			}, filesIn(t, output))
		})

		t.Run("renders the name of the repo in the contents and the paths of templates", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":                      oneFolder,
				"starters/service/files/go/cmd/{{ .Name }}/main.go": &fstest.MapFile{Data: []byte("// {{ .Name }} starts here\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"cmd/my-service/main.go": "// my-service starts here\n"}, filesIn(t, output))
		})

		for _, name := range []string{"my-service", "My.Service"} {
			t.Run("upper gives the snake case of the name "+name+" in capitals", func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":       oneFolder,
					"starters/service/files/go/variable": &fstest.MapFile{Data: []byte(`{{ upper (snake .Name) }}_API_KEY`)},
				}
				output := filepath.Join(t.TempDir(), "repo")

				err := selectionOf(t, files, name).Write(output)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"variable": "MY_SERVICE_API_KEY"}, filesIn(t, output))
			})
		}

		t.Run("a template renders the name in kebab, snake, camel and pascal case", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":    oneFolder,
				"starters/service/files/go/names": &fstest.MapFile{Data: []byte("{{ kebab .Name }} {{ snake .Name }} {{ camel .Name }} {{ pascal .Name }}")},
			}
			output := filepath.Join(t.TempDir(), "repo")

			err := selectionOf(t, files, "OrderEventStore").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"names": "order-event-store order_event_store orderEventStore OrderEventStore"}, filesIn(t, output))
		})

		t.Run("a word of one letter at the start of the name is a word", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":    oneFolder,
				"starters/service/files/go/names": &fstest.MapFile{Data: []byte("{{ kebab .Name }}")},
			}
			output := filepath.Join(t.TempDir(), "repo")

			err := selectionOf(t, files, "xRayScan").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"names": "x-ray-scan"}, filesIn(t, output))
		})

		for name, call := range map[string]struct {
			template string
			expected string
		}{
			"puts the spaces before a text of one line":                    {`{{ indent 2 "a: 1" }}`, "  a: 1"},
			"puts the spaces before each line of a text of several lines":  {`{{ indent 4 "a:\n  b: 1\nc: 2" }}`, "    a:\n      b: 1\n    c: 2"},
			"leaves an empty line empty, and the end of the text as it is": {`{{ indent 2 "a:\n\nb:\n" }}`, "  a:\n\n  b:\n"},
			"gives no text for no text":                                    {`{{ indent 2 "" }}`, ""},
			"gives the text as it is for 0 spaces":                         {`{{ indent 0 "a:\n  b: 1" }}`, "a:\n  b: 1"},
		} {
			t.Run("indent "+name, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":   oneFolder,
					"starters/service/files/go/text": &fstest.MapFile{Data: []byte(call.template)},
				}
				output := filepath.Join(t.TempDir(), "repo")

				err := selectionOf(t, files, "my-service").Write(output)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"text": call.expected}, filesIn(t, output))
			})
		}

		t.Run("indent works in a file with the delimiters [[ and ]]", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":           &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\ndelimiters:\n  - glob: \"go/*.yml\"\n    left: \"[[\"\n    right: \"]]\"\n")},
				"starters/service/files/go/Taskfile.yml": &fstest.MapFile{Data: []byte("tasks:\n[[ indent 2 \"run:\\n  cmds: [\\\"{{.CLI_ARGS}}\\\"]\" ]]\n")},
			}
			output := filepath.Join(t.TempDir(), "repo")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"Taskfile.yml": "tasks:\n  run:\n    cmds: [\"{{.CLI_ARGS}}\"]\n"}, filesIn(t, output))
		})

		t.Run("indent with fewer than 0 spaces stops the write, and the message names the template and the spaces", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":   oneFolder,
				"starters/service/files/go/text": &fstest.MapFile{Data: []byte(`{{ indent -1 "a" }}`)},
			}
			output := filepath.Join(t.TempDir(), "repo")

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, "go/text of the starter")
			require.ErrorContains(t, err, "indent -1")
			require.NoDirExists(t, output)
		})

		t.Run("a written file can be read by everyone and cannot be run", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, 0o644&^umask(), modeOf(t, filepath.Join(output, "main.go")))
		})

		t.Run("a rendered file that starts with #! can be run by everyone", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":               oneFolder,
				"starters/service/files/go/scripts/run.sh":   &fstest.MapFile{Data: []byte("#!/bin/sh\necho {{ .Name }}\n")},
				"starters/service/files/go/scripts/notes.sh": &fstest.MapFile{Data: []byte("echo {{ .Name }}\n#!/bin/sh\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]fs.FileMode{
				"scripts/run.sh":   0o755 &^ umask(),
				"scripts/notes.sh": 0o644 &^ umask(),
			}, map[string]fs.FileMode{
				"scripts/run.sh":   modeOf(t, filepath.Join(output, "scripts", "run.sh")),
				"scripts/notes.sh": modeOf(t, filepath.Join(output, "scripts", "notes.sh")),
			})
		})

		t.Run("a template whose name ends in .tmpl is written without the suffix, and other templates keep their name", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":            oneFolder,
				"starters/service/files/go/go.mod.tmpl":   &fstest.MapFile{Data: []byte("module {{ .Name }}\n")},
				"starters/service/files/go/main.go.tmpl":  &fstest.MapFile{Data: []byte("package main\n")},
				"starters/service/files/go/notes.tmpl.md": &fstest.MapFile{Data: []byte("notes\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":        "module my-service\n",
				"main.go":       "package main\n",
				"notes.tmpl.md": "notes\n",
			}, filesIn(t, output))
		})

		t.Run("two templates that write one path stop the write before the first file, and the message names both", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":           oneFolder,
				"starters/service/files/go/a.go":         &fstest.MapFile{Data: []byte("package a\n")},
				"starters/service/files/go/main.go":      &fstest.MapFile{Data: []byte("package main\n")},
				"starters/service/files/go/main.go.tmpl": &fstest.MapFile{Data: []byte("package main\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, "go/main.go ")
			require.ErrorContains(t, err, "go/main.go.tmpl")
			require.NoDirExists(t, output)
		})

		t.Run("writes into an output folder that exists and is empty", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			output := t.TempDir()

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"main.go": "package main\n"}, filesIn(t, output))
		})

		t.Run("refuses an output folder that has an entry, and leaves the folder as it was", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			output := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(output, ".git"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(output, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644))

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, output)
			require.Equal(t, map[string]string{".git/HEAD": "ref: refs/heads/main\n"}, filesIn(t, output))
		})

		t.Run("refuses an output folder that it cannot read, and leaves its files as they were", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			output := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(output, "main.go"), []byte("package mine\n"), 0o644))
			require.NoError(t, os.Chmod(output, 0o300))
			t.Cleanup(func() { _ = os.Chmod(output, 0o755) })

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, output)
			require.NoError(t, os.Chmod(output, 0o755))
			require.Equal(t, map[string]string{"main.go": "package mine\n"}, filesIn(t, output))
		})

		for name, template := range map[string]string{
			"a file at the root of the repo": "starters/service/files/go/main.go",
			"a file in a folder of the repo": "starters/service/files/go/internal/app.go",
		} {
			t.Run("an output folder that it cannot write to stops the write of "+name+" with the cause", func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml": oneFolder,
					template:                       &fstest.MapFile{Data: []byte("package main\n")},
				}
				output := t.TempDir()
				require.NoError(t, os.Chmod(output, 0o555))
				t.Cleanup(func() { _ = os.Chmod(output, 0o755) })

				err := selectionOf(t, files, "my-service").Write(output)

				require.ErrorContains(t, err, "permission denied")
			})
		}

		t.Run("an output path that is a file stops the write, and leaves the file as it was", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			parent := t.TempDir()
			output := filepath.Join(parent, "notes.txt")
			require.NoError(t, os.WriteFile(output, []byte("my notes\n"), 0o644))

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, output)
			require.Equal(t, map[string]string{"notes.txt": "my notes\n"}, filesIn(t, parent))
		})

		t.Run("a template whose path fails to render stops the write before the first file, and names the template", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":               oneFolder,
				"starters/service/files/go/{{ .Nmae }}/a.go": &fstest.MapFile{Data: []byte("package a\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, "go/{{ .Nmae }}/a.go")
			require.NoDirExists(t, output)
		})

		t.Run("a template that does not parse stops the write before the first file, and names the template", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":   oneFolder,
				"starters/service/files/go/a.go": &fstest.MapFile{Data: []byte("package a\n")},
				"starters/service/files/go/b.go": &fstest.MapFile{Data: []byte("package {{ .Name\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, "go/b.go")
			require.NoDirExists(t, output)
		})

		t.Run("a template that fails to render stops the write before the first file, and names the template", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":   oneFolder,
				"starters/service/files/go/a.go": &fstest.MapFile{Data: []byte("package a\n")},
				"starters/service/files/go/b.go": &fstest.MapFile{Data: []byte("package {{ .Nmae }}\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, "go/b.go")
			require.NoDirExists(t, output)
		})

		t.Run("a path that leads out of the output folder stops the write before the first file", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":                                 &fstest.MapFile{Data: []byte("options:\n  app:\n    default: ../../escape\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/a.go":                               &fstest.MapFile{Data: []byte("package a\n")},
				`starters/service/files/go/{{ option "starter.app" }}/main.go`: &fstest.MapFile{Data: []byte("package main\n")},
			}
			parent := t.TempDir()
			output := filepath.Join(parent, "work", "repo")

			err := selectionOf(t, files, "my-service").Write(output)

			require.ErrorContains(t, err, `go/{{ option "starter.app" }}/main.go`)
			require.Empty(t, filesIn(t, parent))
		})

		t.Run("an empty output path is the working folder, so a working folder that has entries is refused", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":     oneFolder,
				"starters/service/files/go/go.mod": &fstest.MapFile{Data: []byte("module {{ .Name }}\n")},
			}
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			require.NoError(t, os.WriteFile("go.mod", []byte("module mine\n"), 0o644))

			err := selectionOf(t, files, "my-service").Write("")

			require.Error(t, err)
			require.Equal(t, map[string]string{"go.mod": "module mine\n"}, filesIn(t, workingFolder))
		})

		t.Run("a folder that the write makes can be read and entered by everyone", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":          oneFolder,
				"starters/service/files/go/cmd/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, 0o755&^umask(), modeOf(t, filepath.Join(output, "cmd")))
		})

		for _, name := range []string{"starter.nope", "module"} {
			t.Run("option with the name "+name+" stops the write before the first file, and the message names the template and the name", func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":   &fstest.MapFile{Data: []byte("options:\n  module:\n    default: example.com/m\nfolders:\n  go:\n    path: .\n")},
					"starters/service/files/go/a.go": &fstest.MapFile{Data: []byte("package a\n")},
					"starters/service/files/go/b.go": &fstest.MapFile{Data: []byte(`// {{ option "` + name + `" }}`)},
				}
				output := filepath.Join(t.TempDir(), "repo")

				err := selectionOf(t, files, "my-service").Write(output)

				require.ErrorContains(t, err, "go/b.go")
				require.ErrorContains(t, err, `"`+name+`"`)
				require.NoDirExists(t, output)
			})
		}

		twoFolders := &fstest.MapFile{Data: []byte("options:\n  module:\n    default: example.com/{{ .Name }}\nfolders:\n  go:\n    path: .\n  web:\n    path: frontend\n")}

		t.Run("writes the templates of each selected feature into the folders of the starter", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":                       twoFolders,
				"starters/service/files/go/main.go":                  &fstest.MapFile{Data: []byte("package main\n")},
				"features/cli/feature.yml":                           &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/cli/files/go/cmd/{{ .Name }}/main.go.tmpl": &fstest.MapFile{Data: []byte("package main // {{ .Name }}\n")},
				"features/cli/files/web/cli.html":                    &fstest.MapFile{Data: []byte("<p>cli</p>\n")},
				"features/docs/feature.yml":                          &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [other, service]\nfiles:\n  - path: files/common/\n  - path: files/extra\n")},
				"features/docs/files/common/go/docs/a.md":            &fstest.MapFile{Data: []byte("a\n")},
				"features/docs/files/extra/go/docs/b.md":             &fstest.MapFile{Data: []byte("b\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files, "cli", "docs").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":                "package main\n",
				"cmd/my-service/main.go": "package main // my-service\n",
				"frontend/cli.html":      "<p>cli</p>\n",
				"docs/a.md":              "a\n",
				"docs/b.md":              "b\n",
			}, filesIn(t, output))
		})

		t.Run("a feature that is not given writes no file", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":        twoFolders,
				"starters/service/files/go/README.md": &fstest.MapFile{Data: []byte("service\n")},
				"features/cli/feature.yml":            &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/cli/files/go/cli.go":        &fstest.MapFile{Data: []byte("package cli\n")},
				"features/docs/feature.yml":           &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/docs/files/go/docs/a.md":    &fstest.MapFile{Data: []byte("a\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files, "cli").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"README.md": "service\n", "cli.go": "package cli\n"}, filesIn(t, output))
		})

		t.Run("a template of a feature renders the name, the options, the case functions, .tmpl and #! as a template of the starter does", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":         twoFolders,
				"starters/service/files/go/README.md":  &fstest.MapFile{Data: []byte("service\n")},
				"features/cli/feature.yml":             &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/cli/files/go/values.tmpl":    &fstest.MapFile{Data: []byte(`{{ .Name }} {{ option "starter.module" }} {{ pascal .Name }}`)},
				"features/cli/files/go/scripts/run.sh": &fstest.MapFile{Data: []byte("#!/bin/sh\necho {{ .Name }}\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files, "cli").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"README.md":      "service\n",
				"values":         "my-service example.com/my-service MyService",
				"scripts/run.sh": "#!/bin/sh\necho my-service\n",
			}, filesIn(t, output))
			require.Equal(t, 0o755&^umask(), modeOf(t, filepath.Join(output, "scripts", "run.sh")))
		})

		t.Run("a template of the starter and a template of a feature that write one path stop the write before the first file, and the message names both", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":       twoFolders,
				"starters/service/files/go/a.go":     &fstest.MapFile{Data: []byte("package a\n")},
				"starters/service/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
				"features/cli/feature.yml":           &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/cli/files/go/main.go.tmpl": &fstest.MapFile{Data: []byte("package cli\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files, "cli").Write(output)

			require.ErrorContains(t, err, "the template go/main.go of the starter")
			require.ErrorContains(t, err, "the template go/main.go.tmpl of the files entry files/ of the feature cli")
			require.ErrorContains(t, err, "both write main.go")
			require.NoDirExists(t, output)
		})

		t.Run("two features that write one path stop the write before the first file, and the message names both", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":        twoFolders,
				"starters/service/files/go/README.md": &fstest.MapFile{Data: []byte("service\n")},
				"features/cli/feature.yml":            &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/cli/files/go/shared.md":     &fstest.MapFile{Data: []byte("cli\n")},
				"features/docs/feature.yml":           &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/docs/files/go/shared.md":    &fstest.MapFile{Data: []byte("docs\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files, "cli", "docs").Write(output)

			require.ErrorContains(t, err, "the feature cli")
			require.ErrorContains(t, err, "the feature docs")
			require.ErrorContains(t, err, "both write shared.md")
			require.NoDirExists(t, output)
		})

		t.Run("a template of a feature that matches a glob of its feature.yml renders with the delimiters of the first matching entry", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":           twoFolders,
				"starters/service/files/go/README.md":    &fstest.MapFile{Data: []byte("service\n")},
				"features/cli/feature.yml":               &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\ndelimiters:\n  - glob: go/.goreleaser.yaml\n    left: \"[[\"\n    right: \"]]\"\n  - glob: go/**\n    left: \"((\"\n    right: \"))\"\n")},
				"features/cli/files/go/.goreleaser.yaml": &fstest.MapFile{Data: []byte(`project_name: [[ .Name ]] [[ option "starter.module" ]] [[ pascal .Name ]]` + "\nversion: {{ .Tag }}\n")},
				"features/cli/files/go/notes.txt":        &fstest.MapFile{Data: []byte("(( .Name )) {{ .Name }}")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files, "cli").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"README.md":        "service\n",
				"notes.txt":        "my-service {{ .Name }}",
				".goreleaser.yaml": "project_name: my-service example.com/my-service MyService\nversion: {{ .Tag }}\n",
			}, filesIn(t, output))
		})

		t.Run("the globs of a starter and of a feature apply only to their own templates", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":                  &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\ndelimiters:\n  - glob: go/*.starter-glob\n    left: \"[[\"\n    right: \"]]\"\n")},
				"starters/service/files/go/of-starter.cli-glob": &fstest.MapFile{Data: []byte("{{ .Name }} [[ .Name ]]")},
				"features/cli/feature.yml":                      &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\ndelimiters:\n  - glob: go/*.cli-glob\n    left: \"[[\"\n    right: \"]]\"\n")},
				"features/cli/files/go/of-cli.starter-glob":     &fstest.MapFile{Data: []byte("{{ .Name }} [[ .Name ]]")},
				"features/cli/files/go/of-cli.cli-glob":         &fstest.MapFile{Data: []byte("{{ .Name }} [[ .Name ]]")},
				"features/docs/feature.yml":                     &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/docs/files/go/of-docs.cli-glob":       &fstest.MapFile{Data: []byte("{{ .Name }} [[ .Name ]]")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files, "cli", "docs").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"of-starter.cli-glob": "my-service [[ .Name ]]",
				"of-cli.starter-glob": "my-service [[ .Name ]]",
				"of-cli.cli-glob":     "{{ .Name }} my-service",
				"of-docs.cli-glob":    "my-service [[ .Name ]]",
			}, filesIn(t, output))
		})

		t.Run("a range over the items of an extension point that a selected feature declares renders nothing, and the rest renders as without it", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":        twoFolders,
				"starters/service/files/go/README.md": &fstest.MapFile{Data: []byte(`{{ range items "cli.commands" }}- {{ .code }}{{ end }}service`)},
				"features/cli/feature.yml":            &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\nextension_points:\n  cli.commands:\n    summary: An expression of type *cli.Command.\n    file: go/root.go\n    item: [code]\n  cli.flags:\n    summary: A flag.\n    file: go/root.go\n    item: [code]\n")},
				"features/cli/files/go/root.go":       &fstest.MapFile{Data: []byte(`commands: [version{{ range items "cli.commands" }}, {{ .code }}{{ end }}] flags: [{{ range items "cli.flags" }}{{ .code }}{{ end }}]`)},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files, "cli").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"README.md": "service", "root.go": "commands: [version] flags: []"}, filesIn(t, output))
		})

		withMiddleware := &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    summary: A middleware.\n    file: go/server.go\n    item: [code, import]\n  api.config:\n    summary: A field of the config.\n    file: go/config.go\n    item: [name, type, import]\n")}

		t.Run("a range over the items of an extension point of the starter with no selected feature renders nothing, and the rest renders as without it", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":        withMiddleware,
				"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }}{{ end }}server`)},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files).Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"server.go": "server"}, filesIn(t, output))
		})

		for _, order := range [][]string{{"log", "auth", "docs"}, {"docs", "auth", "log"}} {
			t.Run(fmt.Sprintf("items gives each item that a selected feature adds, by the name of the feature and then in the order of its add_to, for the selection %v", order), func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":        withMiddleware,
					"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }} from {{ .import }}; {{ end }}| {{ range items "docs.pages" }}{{ .path }} {{ end }}`)},
					"features/log/feature.yml":            &fstest.MapFile{Data: []byte("summary: Log.\ntargets: [service]\nadd_to:\n  api.middleware:\n    - code: log.Middleware\n      import: internal/log\n  docs.pages:\n    - path: log.md\n")},
					"features/auth/feature.yml":           &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n    - code: audit.Middleware\n      import: internal/audit\n")},
					"features/docs/feature.yml":           &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service]\nextension_points:\n  docs.pages:\n    summary: A page.\n    file: go/server.go\n    item: [path]\n")},
					"features/metrics/feature.yml":        &fstest.MapFile{Data: []byte("summary: Metrics.\ntargets: [service]\nadd_to:\n  api.middleware:\n    - code: metrics.Middleware\n      import: internal/metrics\n")},
				}
				output := filepath.Join(t.TempDir(), "my-service")

				err := featureSelection(t, files, order...).Write(output)

				require.NoError(t, err)
				require.Equal(t, map[string]string{
					"server.go": "auth.Middleware from internal/auth; audit.Middleware from internal/audit; log.Middleware from internal/log; | log.md ",
				}, filesIn(t, output))
			})
		}

		for extensionPoints, expected := range map[string]string{
			`"api.middleware" "api.config"`: "internal/auth;internal/log;internal/audit;",
			`"api.config" "api.middleware"`: "internal/audit;internal/auth;internal/log;",
		} {
			t.Run("unique gives each value of a field one time, in the order in which it first comes over the extension points "+extensionPoints, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":        withMiddleware,
					"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range unique "import" ` + extensionPoints + ` }}{{ . }};{{ end }}`)},
					"features/auth/feature.yml":           &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n  api.config:\n    - name: Auth\n      type: auth.Config\n      import: internal/auth\n")},
					"features/log/feature.yml":            &fstest.MapFile{Data: []byte("summary: Log.\ntargets: [service]\nadd_to:\n  api.middleware:\n    - code: log.Middleware\n      import: internal/log\n  api.config:\n    - name: Log\n      type: log.Config\n      import: internal/log\n")},
					"features/audit/feature.yml":          &fstest.MapFile{Data: []byte("summary: Audit.\ntargets: [service]\nadd_to:\n  api.config:\n    - name: Audit\n      type: audit.Config\n      import: internal/audit\n")},
				}
				output := filepath.Join(t.TempDir(), "my-service")

				err := featureSelection(t, files, "log", "auth", "audit").Write(output)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"server.go": expected}, filesIn(t, output))
			})
		}

		for name, selected := range map[string]struct {
			starterValues map[string]string
			featureValues map[string]map[string]string
			expected      string
		}{
			"method oauth and region eu": {
				map[string]string{"region": "eu"}, map[string]map[string]string{"auth": {"method": "oauth"}, "client": {}},
				"always;oauth-only;client-oauth;client-eu;client-fast;|internal/a;internal/o;internal/e;internal/f;",
			},
			"method api-key and region us": {
				map[string]string{"region": "us"}, map[string]map[string]string{"auth": {"method": "api-key"}, "client": {}},
				"always;api-key-only;client-fast;client-us;|internal/a;internal/k;internal/f;internal/u;",
			},
		} {
			t.Run("items and unique give an item with no when and each item whose when is true for the values that the feature that adds it reads, in their order, with "+name, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\noptions:\n  region:\n    default: us\nextension_points:\n  api.middleware:\n    summary: A middleware.\n    file: go/server.go\n    item: [code, import]\n")},
					"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }};{{ end }}|{{ range unique "import" "api.middleware" }}{{ . }};{{ end }}`)},
					"features/auth/feature.yml": &fstest.MapFile{Data: []byte(`summary: Auth.
targets: [service]
options:
  method:
    values: [api-key, oauth]
    default: api-key
  mode:
    default: slow
add_to:
  api.middleware:
    - code: always
      import: internal/a
    - code: oauth-only
      import: internal/o
      when: method == "oauth"
    - code: api-key-only
      import: internal/k
      when: method == "api-key"
    - code: auth-fast
      import: internal/f
      when: mode == "fast"
`)},
					"features/client/feature.yml": &fstest.MapFile{Data: []byte(`summary: Client.
targets: [service]
needs: [auth]
options:
  mode:
    default: fast
add_to:
  api.middleware:
    - code: client-oauth
      import: internal/o
      when: auth.method == "oauth"
    - code: client-eu
      import: internal/e
      when: starter.region == "eu"
    - code: client-fast
      import: internal/f
      when: mode == "fast"
    - code: client-us
      import: internal/u
      when: starter.region == "us"
`)},
				}
				starter, err := catalog.New(files).Starter("service")
				require.NoError(t, err)
				selection, err := starter.Select("my-service", selected.starterValues, selected.featureValues, catalog.Answers{})
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "my-service")

				err = selection.Write(output)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"server.go": selected.expected}, filesIn(t, output))
			})
		}

		for _, call := range []string{`items "api.middleware"`, `unique "import" "api.middleware"`} {
			t.Run("a when of an item that reads a name that the feature cannot read stops the write before the first file, and the message names the when, the feature and the extension point, with "+call, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":        withMiddleware,
					"starters/service/files/go/a.md":      &fstest.MapFile{Data: []byte("a\n")},
					"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range ` + call + ` }}{{ . }}{{ end }}`)},
					"features/auth/feature.yml":           &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n      when: vault.mode == \"fast\"\n")},
				}
				output := filepath.Join(t.TempDir(), "my-service")

				err := featureSelection(t, files, "auth").Write(output)

				require.ErrorContains(t, err, `the when "vault.mode == \"fast\"" of an item that the feature auth adds to the extension point "api.middleware": unknown name "vault.mode"`)
				require.NoDirExists(t, output)
			})
		}

		t.Run("unique over extension points with no item renders nothing, and the rest renders as without it", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":        withMiddleware,
				"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range unique "import" "api.middleware" "api.config" }}{{ . }}{{ end }}server`)},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files).Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"server.go": "server"}, filesIn(t, output))
		})

		for name, call := range map[string]struct {
			template string
			message  []string
		}{
			"an extension point that nothing declares":          {`unique "import" "api.middleware" "api.routes"`, []string{`"api.routes"`, "the starter and the selected features declare api.config, api.middleware"}},
			"a field that one extension point does not declare": {`unique "code" "api.middleware" "api.config"`, []string{`unknown field "code"`, `the extension point "api.config" declares the fields [name, type, import]`}},
			"no extension point":                                {`unique "import"`, []string{`unique "import" names no extension point`}},
		} {
			t.Run("unique with "+name+" stops the write before the first file, and the message names the template", func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":        withMiddleware,
					"starters/service/files/go/a.md":      &fstest.MapFile{Data: []byte("a\n")},
					"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range ` + call.template + ` }}{{ . }}{{ end }}`)},
				}
				output := filepath.Join(t.TempDir(), "my-service")

				err := featureSelection(t, files).Write(output)

				require.ErrorContains(t, err, "go/server.go of the starter")
				for _, part := range call.message {
					require.ErrorContains(t, err, part)
				}
				require.NoDirExists(t, output)
			})
		}

		t.Run("items in a selection with no extension point stops the write, and the message says that neither the starter nor a selected feature declares one", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":        twoFolders,
				"starters/service/files/go/README.md": &fstest.MapFile{Data: []byte(`{{ range items "cli.commands" }}x{{ end }}`)},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := featureSelection(t, files).Write(output)

			require.ErrorContains(t, err, "neither the starter nor a selected feature declares an extension point")
			require.NoDirExists(t, output)
		})

		for name, extensionPoint := range map[string]string{
			"that nothing declares":             "api.routes",
			"of a feature that is not selected": "docs.pages",
		} {
			t.Run("items with an extension point "+name+" stops the write before the first file, and the message names the template and the extension point, and lists the extension points of the selection", func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":         withMiddleware,
					"starters/service/files/go/README.md":  &fstest.MapFile{Data: []byte("service\n")},
					"features/cli/feature.yml":             &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\nextension_points:\n  cli.commands:\n    summary: An expression of type *cli.Command.\n    file: go/root.go\n    item: [code]\n")},
					"features/cli/files/go/root.go":        &fstest.MapFile{Data: []byte(`{{ range items "` + extensionPoint + `" }}x{{ end }}`)},
					"features/docs/feature.yml":            &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service]\nfiles:\n  - path: files/\nextension_points:\n  docs.pages:\n    summary: A page.\n    file: go/docs/index.md\n    item: [path]\n")},
					"features/docs/files/go/docs/index.md": &fstest.MapFile{Data: []byte("docs\n")},
				}
				output := filepath.Join(t.TempDir(), "my-service")

				err := featureSelection(t, files, "cli").Write(output)

				require.ErrorContains(t, err, "go/root.go of the feature cli")
				require.ErrorContains(t, err, strconv.Quote(extensionPoint))
				require.ErrorContains(t, err, "the starter and the selected features declare api.config, api.middleware, cli.commands")
				require.NoDirExists(t, output)
			})
		}

		githubTemplate := `{{ githubRepository (option "starter.module") }} {{ (githubRepository (option "starter.module")).Account }} {{ (githubRepository (option "starter.module")).Name }}`
		t.Run("githubRepository gives the account and the repository of the module, also with a path after the repository", func(t *testing.T) {
			module := "github.com/acme/ledger/tools"
			files := fstest.MapFS{
				"starters/service/starter.yml":     twoFolders,
				"starters/service/files/go/github": &fstest.MapFile{Data: []byte(githubTemplate)},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"module": module}, nil, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			err = selection.Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"github": "acme/ledger acme ledger"}, filesIn(t, output))
		})

		t.Run("githubRepository with a module that is not on GitHub stops the write before the first file, and the message names the module", func(t *testing.T) {
			module := "example.com/acme/ledger"
			files := fstest.MapFS{
				"starters/service/starter.yml":     twoFolders,
				"starters/service/files/go/a.md":   &fstest.MapFile{Data: []byte("a\n")},
				"starters/service/files/go/github": &fstest.MapFile{Data: []byte(githubTemplate)},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"module": module}, nil, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			err = selection.Write(output)

			require.ErrorContains(t, err, strconv.Quote(module))
			require.ErrorContains(t, err, "GitHub")
			require.NoDirExists(t, output)
		})

		withDelimiters := &fstest.MapFile{Data: []byte(`
options:
  module:
    default: github.com/hpcsc/{{ .Name }}
folders:
  go:
    path: .
  web:
    path: web
delimiters:
  - glob: "go/**/Taskfile*.yml"
    left: "[["
    right: "]]"
  - glob: "go/.github/**"
    left: "<%"
    right: "%>"
  - glob: "go/**"
    left: "(("
    right: "))"
`)}

		t.Run("a template that matches a delimiters glob renders with the delimiters of its first matching entry, and writes {{ }} as it is", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":                       withDelimiters,
				"starters/service/files/go/Taskfile.yml":             &fstest.MapFile{Data: []byte("name: [[ .Name ]]\ncmd: go run . {{.CLI_ARGS}}\n")},
				"starters/service/files/go/.github/workflows/ci.yml": &fstest.MapFile{Data: []byte("name: <% .Name %>\ntoken: ${{ secrets.GITHUB_TOKEN }}\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"Taskfile.yml":             "name: my-service\ncmd: go run . {{.CLI_ARGS}}\n",
				".github/workflows/ci.yml": "name: my-service\ntoken: ${{ secrets.GITHUB_TOKEN }}\n",
			}, filesIn(t, output))
		})

		for catalogPath, expected := range map[string]string{
			"go/Taskfile.yml":           "my-service {{ .Name }} (( .Name ))",
			"go/a/b/Taskfile.test.yml":  "my-service {{ .Name }} (( .Name ))",
			"web/Taskfile.yml":          "[[ .Name ]] my-service (( .Name ))",
			"go/a/b/Makefile":           "[[ .Name ]] {{ .Name }} my-service",
			"go/a/b/Taskfile.yml/other": "[[ .Name ]] {{ .Name }} my-service",
		} {
			t.Run("the glob ** matches any number of path segments, so "+catalogPath+" renders "+expected, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":          withDelimiters,
					"starters/service/files/" + catalogPath: &fstest.MapFile{Data: []byte("[[ .Name ]] {{ .Name }} (( .Name ))")},
				}
				output := filepath.Join(t.TempDir(), "my-service")

				err := selectionOf(t, files, "my-service").Write(output)

				require.NoError(t, err)
				require.Equal(t, []string{expected}, slices.Collect(maps.Values(filesIn(t, output))))
			})
		}

		t.Run("other delimiters give the same name, options and case functions as {{ and }}", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":           withDelimiters,
				"starters/service/files/go/Taskfile.yml": &fstest.MapFile{Data: []byte(`[[ .Name ]] [[ option "starter.module" ]] [[ pascal .Name ]]`)},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"Taskfile.yml": "my-service github.com/hpcsc/my-service MyService"}, filesIn(t, output))
		})

		t.Run("the path of a template renders with {{ and }} also when its contents use other delimiters", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":                           withDelimiters,
				"starters/service/files/go/cmd/{{ .Name }}/Taskfile.yml": &fstest.MapFile{Data: []byte("name: [[ .Name ]]\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"cmd/my-service/Taskfile.yml": "name: my-service\n"}, filesIn(t, output))
		})

		withoutDefaults := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":         &fstest.MapFile{Data: []byte("options:\n  zone:\n    default: \"\"\n  region:\n    pattern: '^[a-z]*$'\n  module:\n    default: example.com/{{ .Name }}\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/region.txt": &fstest.MapFile{Data: []byte(`{{ option "starter.region" }}|{{ option "starter.zone" }}|{{ option "starter.module" }}`)},
				"features/a/feature.yml":               &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds: [b]\noptions:\n  level:\n    default: \"\"\n")},
				"features/b/feature.yml":               &fstest.MapFile{Data: []byte("summary: B.\ntargets: [service]\noptions:\n  token:\n    pattern: '^.*$'\n  key:\n    default: \"\"\n")},
			}
		}

		t.Run("each option with no default and no value stops the write before the first file, and the message names each one in the order of the preview, with the flag that gives it", func(t *testing.T) {
			starter, err := catalog.New(withoutDefaults()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"a": {}}, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			err = selection.Write(output)

			require.EqualError(t, err, `the option region of the starter service has no default and no value: give it with --option region=<value>
the option zone of the starter service has no default and no value: give it with --option zone=<value>
the option key of the feature b has no default and no value: give it with --feature b:key=<value>
the option token of the feature b has no default and no value: give it with --feature b:token=<value>
the option level of the feature a has no default and no value: give it with --feature a:level=<value>`)
			require.NoDirExists(t, output)
		})

		t.Run("a selection with no name stops the write with the message of the name, and does not read the output folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			require.NoError(t, os.WriteFile("go.mod", []byte("module mine\n"), 0o644))
			files := fstest.MapFS{"starters/service/starter.yml": oneFolder, "starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")}}

			err := selectionOf(t, files, "").Write("")

			require.EqualError(t, err, `the name "" of the repo must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'`)
			require.Equal(t, map[string]string{"go.mod": "module mine\n"}, filesIn(t, workingFolder))
		})

		t.Run("a selection with no name and an option with no default and no value stops the write with the message of the name only", func(t *testing.T) {
			starter, err := catalog.New(withoutDefaults()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("", nil, nil, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "repo")

			err = selection.Write(output)

			require.EqualError(t, err, `the name "" of the repo must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'`)
			require.NoDirExists(t, output)
		})

		t.Run("a value that a flag gives to an option with no default is the value that the templates read", func(t *testing.T) {
			starter, err := catalog.New(withoutDefaults()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"region": "eu", "zone": "west"}, map[string]map[string]string{"a": {"level": "1"}, "b": {"token": "t", "key": "k"}}, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			err = selection.Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"region.txt": "eu|west|example.com/my-service"}, filesIn(t, output))
		})

		t.Run("the empty text from a flag is no value for an option with no default", func(t *testing.T) {
			starter, err := catalog.New(withoutDefaults()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"region": "", "zone": "west"}, nil, catalog.Answers{})
			require.NoError(t, err)

			err = selection.Write(filepath.Join(t.TempDir(), "my-service"))

			require.EqualError(t, err, "the option region of the starter service has no default and no value: give it with --option region=<value>")
		})

		t.Run("a flag can give the empty text to an option that has a default", func(t *testing.T) {
			starter, err := catalog.New(withoutDefaults()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"region": "eu", "zone": "west", "module": ""}, nil, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			err = selection.Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"region.txt": "eu|west|"}, filesIn(t, output))
		})

		t.Run("the features that answers select, with their needs and an answered value, give the files of the same selection from flags", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/a/feature.yml":            &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds: [b]\nfiles:\n  - path: files/\n")},
				"features/a/files/go/a.txt":         &fstest.MapFile{Data: []byte(`{{ option "b.mode" }}`)},
				"features/b/feature.yml":            &fstest.MapFile{Data: []byte("summary: B.\ntargets: [service]\nfiles:\n  - path: files/\noptions:\n  mode:\n    values: [fast, slow]\n    default: fast\n")},
				"features/b/files/go/b.txt":         &fstest.MapFile{Data: []byte("b\n")},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			fromFlags, err := starter.Select("my-service", nil, map[string]map[string]string{"a": {}, "b": {"mode": "slow"}}, catalog.Answers{})
			require.NoError(t, err)
			fromAnswers, err := starter.Select("my-service", nil, nil, catalog.Answers{Features: []string{"a"}, FeatureValues: map[string]map[string]string{"b": {"mode": "slow"}}})
			require.NoError(t, err)
			flagsOutput := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, fromFlags.Write(flagsOutput))
			answersOutput := filepath.Join(t.TempDir(), "my-service")

			err = fromAnswers.Write(answersOutput)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"main.go": "package main\n", "a.txt": "slow", "b.txt": "b\n"}, filesIn(t, answersOutput))
			require.Equal(t, filesIn(t, flagsOutput), filesIn(t, answersOutput))
		})
	})

	t.Run("make repo", func(t *testing.T) {
		selectionWith := func(t *testing.T, starterYML string, name string, given map[string]string, answers catalog.Answers) catalog.Selection {
			t.Helper()
			files := fstest.MapFS{
				"starters/service/starter.yml":          &fstest.MapFile{Data: []byte(starterYML)},
				"starters/service/files/go/go.mod.tmpl": &fstest.MapFile{Data: []byte("module {{ .Name }}\n")},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select(name, given, nil, answers)
			require.NoError(t, err)
			return selection
		}
		threeOptions := "folders:\n  go:\n    path: .\noptions:\n  region:\n    default: us\n  module:\n    default: example.com/{{ .Name }}\n  tier:\n    default: free\n"

		t.Run("writes .feature-compose.yml with the version that it gets, the starter, the name, and each option of the starter in the order of the names with the value that the templates read", func(t *testing.T) {
			selection := selectionWith(t, threeOptions, "my-service", map[string]string{"region": "eu"}, catalog.Answers{StarterValues: map[string]string{"tier": "gold"}})
			output := filepath.Join(t.TempDir(), "my-service")

			err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module my-service\n",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions:\n  module: example.com/my-service\n  region: eu\n  tier: gold\nfeatures: {}\nrequested: []\n",
			}, filesIn(t, output))
		})

		t.Run("runs the after_render commands of each folder before it writes .feature-compose.yml, so no command finds the file", func(t *testing.T) {
			t.Chdir(t.TempDir())
			selection := selectionWith(t, "folders:\n  go:\n    path: .\n    after_render: [\"ls -A > listed\"]\n", "ledger", nil, catalog.Answers{})
			output := filepath.Join(t.TempDir(), "ledger")

			err := selection.MakeRepo(context.Background(), output, "v0.1.0", catalog.NoSteps{})

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"go.mod":               "module ledger\n",
				"listed":               "go.mod\nlisted\n",
				".feature-compose.yml": "version: v0.1.0\nstarter: service\nname: ledger\noptions: {}\nfeatures: {}\nrequested: []\n",
			}, filesIn(t, output))
		})

		t.Run("a starter with no option gives an empty map of options", func(t *testing.T) {
			selection := selectionWith(t, "folders:\n  go:\n    path: .\n", "api", nil, catalog.Answers{})
			output := t.TempDir()

			err := selection.MakeRepo(context.Background(), output, "0123abc-dirty", catalog.NoSteps{})

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module api\n", ".feature-compose.yml": "version: 0123abc-dirty\nstarter: service\nname: api\noptions: {}\nfeatures: {}\nrequested: []\n"}, filesIn(t, output))
		})

		t.Run("each value reads back with a YAML parser as the same text, also a value that YAML reads as a number, a bool or null", func(t *testing.T) {
			starterYML := "folders:\n  go:\n    path: .\noptions:\n"
			for _, optionName := range []string{"float", "bool", "on", "empty", "tilde"} {
				starterYML += "  " + optionName + ":\n    default: x\n"
			}
			selection := selectionWith(t, starterYML, "my-service", map[string]string{"float": "1.28", "bool": "true", "on": "on", "empty": "", "tilde": "~"}, catalog.Answers{})
			output := t.TempDir()
			require.NoError(t, selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{}))
			var readBack map[string]any

			err := yaml.Unmarshal([]byte(filesIn(t, output)[".feature-compose.yml"]), &readBack)

			require.NoError(t, err)
			require.Equal(t, map[string]any{"float": "1.28", "bool": "true", "on": "on", "empty": "", "tilde": "~"}, readBack["options"])
		})

		selectionOfFeatures := func(t *testing.T, definitions map[string]string, givenByFeature map[string]map[string]string, answers catalog.Answers) catalog.Selection {
			t.Helper()
			starter, err := catalog.New(withFeatureDefinitions(definitions)).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, givenByFeature, answers)
			require.NoError(t, err)
			return selection
		}

		t.Run("features holds each feature in the order of the names, also a feature that only a need added, each with its options and the value that the templates read from a flag, a need, an answer or the default", func(t *testing.T) {
			selection := selectionOfFeatures(t, map[string]string{
				"auth-cli": "needs:\n  api:\n    method: oauth\n",
				"api":      "options:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  realm:\n    default: api\n",
				"zone":     "options:\n  region:\n    default: eu\n  tier:\n    default: free\n",
				"docs":     "",
			}, map[string]map[string]string{"auth-cli": {}, "zone": {"tier": "gold"}}, catalog.Answers{Features: []string{"docs"}, FeatureValues: map[string]map[string]string{"zone": {"region": "us"}}})
			output := t.TempDir()

			err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":              "package main\n",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions: {}\nfeatures:\n  api:\n    method: oauth\n    realm: api\n  auth-cli: {}\n  docs: {}\n  zone:\n    region: us\n    tier: gold\nrequested: [auth-cli, docs, zone]\n",
			}, filesIn(t, output))
		})

		t.Run("requested lists in the order of the names each feature that a flag gives or that the form selects, also one that another feature needs, and not a feature that only a need added, also when an answer gives a value to its option", func(t *testing.T) {
			selection := selectionOfFeatures(t, map[string]string{
				"a": "needs: [b]\n",
				"b": "",
				"c": "needs: [d]\n",
				"d": "options:\n  mode:\n    values: [fast, slow]\n    default: fast\n",
			}, map[string]map[string]string{"a": {}, "b": {}}, catalog.Answers{Features: []string{"c"}, FeatureValues: map[string]map[string]string{"d": {"mode": "slow"}}})
			output := t.TempDir()

			err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":              "package main\n",
				".feature-compose.yml": "version: v1.2.3\nstarter: service\nname: my-service\noptions: {}\nfeatures:\n  a: {}\n  b: {}\n  c: {}\n  d:\n    mode: slow\nrequested: [a, b, c]\n",
			}, filesIn(t, output))
		})

		t.Run("an output folder that has entries stops it before the first file, so it runs no command and writes no .feature-compose.yml", func(t *testing.T) {
			t.Chdir(t.TempDir())
			selection := selectionWith(t, "folders:\n  go:\n    path: .\n    after_render: [\"touch ran\"]\n", "my-service", nil, catalog.Answers{})
			output := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(output, "notes.txt"), []byte("my notes\n"), 0o644))

			err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.ErrorContains(t, err, "is not empty")
			require.Equal(t, map[string]string{"notes.txt": "my notes\n"}, filesIn(t, output))
		})

		t.Run("an after_render command that fails stops it before it writes .feature-compose.yml", func(t *testing.T) {
			t.Chdir(t.TempDir())
			selection := selectionWith(t, "folders:\n  go:\n    path: .\n    after_render: [\"exit 3\"]\n", "my-service", nil, catalog.Answers{})
			output := filepath.Join(t.TempDir(), "my-service")

			err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.ErrorContains(t, err, "exit 3")
			require.Equal(t, map[string]string{"go.mod": "module my-service\n"}, filesIn(t, output))
		})

		t.Run("an after_render command that makes a folder at .feature-compose.yml stops it with an error that names the file", func(t *testing.T) {
			t.Chdir(t.TempDir())
			selection := selectionWith(t, "folders:\n  go:\n    path: .\n    after_render: [\"mkdir .feature-compose.yml\"]\n", "my-service", nil, catalog.Answers{})
			output := filepath.Join(t.TempDir(), "my-service")

			err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.ErrorContains(t, err, "write .feature-compose.yml")
		})

		t.Run("reports the write and each after_render command as a step, in the order in which they run", func(t *testing.T) {
			selection := selectionWith(t, "folders:\n  go:\n    path: .\n    after_render: [\"true\", \"echo done\"]\n", "my-service", nil, catalog.Answers{})
			steps := &recordedSteps{}

			err := selection.MakeRepo(context.Background(), filepath.Join(t.TempDir(), "my-service"), "v1.2.3", steps)

			require.NoError(t, err)
			require.Equal(t, []string{
				"start Write the files of the starter and the features", "end Write the files of the starter and the features",
				"start go: true", "end go: true",
				"start go: echo done", "end go: echo done",
			}, steps.lines)
		})

		t.Run("ends the step that fails with its error, and starts no step after it", func(t *testing.T) {
			selection := selectionWith(t, "folders:\n  go:\n    path: .\n    after_render: [\"false\", \"true\"]\n", "my-service", nil, catalog.Answers{})
			steps := &recordedSteps{}

			err := selection.MakeRepo(context.Background(), filepath.Join(t.TempDir(), "my-service"), "v1.2.3", steps)

			require.ErrorContains(t, err, `the after_render command "false" of the folder go failed`)
			require.Equal(t, []string{
				"start Write the files of the starter and the features", "end Write the files of the starter and the features",
				"start go: false", "fail go: false",
			}, steps.lines)
		})

		t.Run("with a generator", func(t *testing.T) {
			made := func(t *testing.T, selection catalog.Selection) (string, error) {
				t.Helper()
				output := filepath.Join(t.TempDir(), "my-service")
				return output, selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})
			}
			repoFiles := func(t *testing.T, output string) map[string]string {
				t.Helper()
				files := filesIn(t, output)
				delete(files, ".feature-compose.yml")
				return files
			}
			const generatedDeps = `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      cat > {{ .Folder }}/mix.exs <<'EOF'
      defp deps do
        [
          {:phoenix, "~> 1.8"}
        ]
      end
      EOF
extension_points:
  mix.deps:
    summary: A dependency.
    file: app/mix.exs
    item: [dep]
    language: elixir
    pattern: defp deps do [$$$DEPS] end
    after: DEPS
    separator: ","
    insert: "{{ .dep }}"
`

			t.Run("reports the generate command, the removals, the write, the inserts and each after_render command as steps, in the order in which they run", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'page\n' > {{ .Folder }}/page.ex
      printf 'deps = [\n  :phoenix\n]\n' > {{ .Folder }}/deps.ex
    after_render: ["true"]
remove:
  - file: app/page.ex
extension_points:
  deps.items:
    summary: A dependency.
    file: app/deps.ex
    item: [dep]
    language: elixir
    pattern: deps = [$$$DEPS]
    after: DEPS
    separator: ","
    insert: "{{ .dep }}"
add_to:
  deps.items:
    - dep: ":jason"
`})
				steps := &recordedSteps{}

				err := selection.MakeRepo(context.Background(), filepath.Join(t.TempDir(), "my-service"), "v1.2.3", steps)

				require.NoError(t, err)
				require.Equal(t, []string{
					"start Generate the folder app", "end Generate the folder app",
					"start Remove the generated files and code that the starter names", "end Remove the generated files and code that the starter names",
					"start Write the files of the starter and the features", "end Write the files of the starter and the features",
					"start Add the items to the generated files", "end Add the items to the generated files",
					"start app: true", "end app: true",
				}, steps.lines)
			})

			t.Run("writes the files of the generate command, and a file of the starter replaces the generated file at its path", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{
					"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && printf 'generated readme\n' > {{ .Folder }}/README.md && printf 'kept\n' > {{ .Folder }}/kept.txt
`,
					"starters/service/files/app/README.md": "readme of the starter\n",
				})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"README.md": "readme of the starter\n", "kept.txt": "kept\n"}, repoFiles(t, output))
			})

			t.Run("makes the repo in a relative output folder, which is the default output folder of new", func(t *testing.T) {
				t.Chdir(t.TempDir())
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && printf 'generated\n' > {{ .Folder }}/README.md
`})

				err := selection.MakeRepo(context.Background(), "my-service", "v1.2.3", catalog.NoSteps{})

				require.NoError(t, err)
				require.Equal(t, map[string]string{"README.md": "generated\n"}, repoFiles(t, "my-service"))
			})

			t.Run("gives the generate command a folder that does not exist yet, whose last part is the repo name", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && basename {{ .Folder }} > {{ .Folder }}/name.txt
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"name.txt": "my-service\n"}, repoFiles(t, output))
			})

			t.Run("removes a generated file and each folder that the removal leaves empty", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir -p {{ .Folder }}/lib/page && printf 'home' > {{ .Folder }}/lib/page/home.html && printf 'app' > {{ .Folder }}/lib/app.ex
remove:
  - file: app/lib/page/home.html
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"lib/app.ex": "app"}, repoFiles(t, output))
				require.NoDirExists(t, filepath.Join(output, "lib", "page"))
			})

			t.Run("removes the one match of a pattern, with the line that it leaves empty", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      cat > {{ .Folder }}/router.ex <<'EOF'
      scope "/" do
        pipe_through :browser
        get "/", PageController, :home
      end
      EOF
remove:
  - file: app/router.ex
    language: elixir
    pattern: get "/", PageController, :home
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"router.ex": "scope \"/\" do\n  pipe_through :browser\nend\n"}, repoFiles(t, output))
			})

			t.Run("inserts the items of the starter, then of a feature, after the last node of the metavariable, with the separator and the indentation of that node", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{
					"starters/service/starter.yml": generatedDeps + "add_to:\n  mix.deps:\n    - dep: '{:jason, \"~> 1.2\"}'\n",
					"features/oban/feature.yml":    "summary: Oban.\ntargets: [service]\nadd_to:\n  mix.deps:\n    - dep: '{:oban, \"~> 2.24\"}'\n",
				}, "oban")

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"mix.exs": "defp deps do\n  [\n    {:phoenix, \"~> 1.8\"},\n    {:jason, \"~> 1.2\"},\n    {:oban, \"~> 2.24\"}\n  ]\nend\n"}, repoFiles(t, output))
			})

			t.Run("inserts the items of an extension point that a feature declares in a generated file", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{
					"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf '[\n  :repo\n]\n' > {{ .Folder }}/children.ex
`,
					"features/oban/feature.yml": `summary: Oban.
targets: [service]
extension_points:
  oban.children:
    summary: A child.
    file: app/children.ex
    item: [child]
    language: elixir
    pattern: "[$$$CHILDREN]"
    after: CHILDREN
    separator: ","
    insert: "{{ .child }}"
add_to:
  oban.children:
    - child: ":oban"
`,
				}, "oban")

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"children.ex": "[\n  :repo,\n  :oban\n]\n"}, repoFiles(t, output))
			})

			t.Run("renders the value of an item with the options of its feature", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{
					"starters/service/starter.yml": generatedDeps,
					"features/oban/feature.yml":    "summary: Oban.\ntargets: [service]\noptions:\n  version:\n    default: \"2.24\"\nadd_to:\n  mix.deps:\n    - dep: '{:oban, \"~> {{ option \"version\" }}\"}'\n",
				}, "oban")

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"mix.exs": "defp deps do\n  [\n    {:phoenix, \"~> 1.8\"},\n    {:oban, \"~> 2.24\"}\n  ]\nend\n"}, repoFiles(t, output))
			})

			t.Run("inserts the items before the node of a rule, with the separator and the indentation of that node", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      cat > {{ .Folder }}/application.ex <<'EOF'
      children = [
        Repo,
        Endpoint
      ]
      EOF
extension_points:
  application.children:
    summary: A child.
    file: app/application.ex
    item: [child]
    language: elixir
    rule:
      pattern: Endpoint
      inside:
        pattern: children = [$$$CHILDREN]
        stopBy: end
    before: match
    separator: ","
    insert: "{{ .child }}"
add_to:
  application.children:
    - child: Oban
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"application.ex": "children = [\n  Repo,\n  Oban,\n  Endpoint\n]\n"}, repoFiles(t, output))
			})

			t.Run("inserts the items before a node above the comment lines directly above that node, because those comments describe the node", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      cat > {{ .Folder }}/application.ex <<'EOF'
      children = [
        Repo,
        # Start to serve requests,
        # typically the last entry
        Endpoint
      ]
      EOF
extension_points:
  application.children:
    summary: A child.
    file: app/application.ex
    item: [child]
    language: elixir
    rule:
      pattern: Endpoint
      inside:
        pattern: children = [$$$CHILDREN]
        stopBy: end
    before: match
    separator: ","
    insert: "{{ .child }}"
add_to:
  application.children:
    - child: Oban
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"application.ex": "children = [\n  Repo,\n  Oban,\n  # Start to serve requests,\n  # typically the last entry\n  Endpoint\n]\n"}, repoFiles(t, output))
			})

			t.Run("stops when the pattern of an extension point matches no place, and names the extension point and the file", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{
					"starters/service/starter.yml": strings.Replace(generatedDeps, "defp deps do [$$$DEPS] end", "defp other_deps do [$$$DEPS] end", 1) + "add_to:\n  mix.deps:\n    - dep: '{:jason, \"~> 1.2\"}'\n",
				})

				_, err := made(t, selection)

				require.ErrorContains(t, err, `the extension point "mix.deps" matches 0 places in mix.exs, and it must match exactly one`)
			})

			t.Run("stops when a file of a feature has the path of a generated file", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{
					"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && printf 'generated readme\n' > {{ .Folder }}/README.md
`,
					"features/docs/feature.yml":         "summary: Docs.\ntargets: [service]\nfiles:\n  - path: files/\n",
					"features/docs/files/app/README.md": "readme of the feature\n",
				}, "docs")

				_, err := made(t, selection)

				require.ErrorContains(t, err, "the feature docs writes README.md, which the generate command also writes")
			})

			removesTheWrongFile := `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && printf 'generated\n' > {{ .Folder }}/README.md
remove:
  - file: app/lib/page.ex
`

			t.Run("removes the output folder when a step after the generate command fails, so that new can make the repo again", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": removesTheWrongFile})

				output, err := made(t, selection)

				require.ErrorContains(t, err, "removes lib/page.ex, which the generate command does not write")
				require.NoDirExists(t, output)
			})

			t.Run("keeps an empty output folder that existed before, and removes what it wrote into it, when a step after the generate command fails", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": removesTheWrongFile})
				output := filepath.Join(t.TempDir(), "my-service")
				require.NoError(t, os.Mkdir(output, 0o755))

				err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

				require.ErrorContains(t, err, "removes lib/page.ex, which the generate command does not write")
				require.Equal(t, map[string]string{}, filesIn(t, output))
			})

			t.Run("stops before the generate command runs when the command reads an option whose value is not one shell word", func(t *testing.T) {
				injected := filepath.Join(t.TempDir(), "injected")
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
options:
  app:
    default: "x;touch ` + injected + `"
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && echo {{ option "starter.app" }}
`})

				_, err := made(t, selection)

				require.ErrorContains(t, err, `the generate command of the folder app reads the option starter.app, whose value "x;touch `+injected+`" is not one shell word`)
				require.NoFileExists(t, injected)
			})

			t.Run("stops when the generate command fails, and shows what the command printed", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: echo "phx.new is not installed" && exit 3
`})

				_, err := made(t, selection)

				require.ErrorContains(t, err, "phx.new is not installed")
			})
			t.Run("finds the place by the declared language, also in a file whose name ast-grep does not link to that language", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'deps = [\n  :a\n]\n' > {{ .Folder }}/Depsfile
extension_points:
  deps.items:
    summary: A dependency.
    file: app/Depsfile
    item: [dep]
    language: elixir
    pattern: deps = [$$$DEPS]
    after: DEPS
    separator: ","
    insert: "{{ .dep }}"
add_to:
  deps.items:
    - dep: ":b"
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"Depsfile": "deps = [\n  :a,\n  :b\n]\n"}, repoFiles(t, output))
			})

			t.Run("stops when the metavariable of the position matches no node, and names the metavariable", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && printf 'deps = []\n' > {{ .Folder }}/deps.ex
extension_points:
  deps.items:
    summary: A dependency.
    file: app/deps.ex
    item: [dep]
    language: elixir
    pattern: deps = [$$$DEPS]
    after: DEPS
    separator: ","
    insert: "{{ .dep }}"
add_to:
  deps.items:
    - dep: ":a"
`})

				_, err := made(t, selection)

				require.ErrorContains(t, err, "DEPS matches no node, so the items cannot go next to it")
			})

			t.Run("stops when the file of an extension point is not a file that the generate command writes", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": strings.Replace(generatedDeps, "file: app/mix.exs", "file: app/deps.exs", 1) + "add_to:\n  mix.deps:\n    - dep: '{:jason, \"~> 1.2\"}'\n"})

				_, err := made(t, selection)

				require.ErrorContains(t, err, `the extension point "mix.deps" is in deps.exs, which the generate command does not write`)
			})

			t.Run("inserts the items before the first node of a metavariable that matches more than one node", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'quote do\n  alias Repo\n  import Ecto\nend\n' > {{ .Folder }}/case.ex
extension_points:
  case.using:
    summary: A line.
    file: app/case.ex
    item: [code]
    language: elixir
    pattern: quote do $$$BODY end
    before: BODY
    insert: "{{ .code }}"
add_to:
  case.using:
    - code: use Oban.Testing
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"case.ex": "quote do\n  use Oban.Testing\n  alias Repo\n  import Ecto\nend\n"}, repoFiles(t, output))
			})

			t.Run("removes a match that shares its line with other code, and keeps that code", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'x = 1 # remove me\n' > {{ .Folder }}/x.ex
remove:
  - file: app/x.ex
    language: elixir
    rule: {kind: comment, regex: remove me}
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"x.ex": "x = 1 \n"}, repoFiles(t, output))
			})

			t.Run("stops when the generate command does not render, and writes no repo", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && echo {{ .Nmae }}
`})

				output, err := made(t, selection)

				require.ErrorContains(t, err, "the generate command of the folder app")
				require.NoDirExists(t, output)
			})

			t.Run("removes only the entries that it wrote when a step after the generate command fails, and keeps an entry that another writer put into the folder", func(t *testing.T) {
				output := filepath.Join(t.TempDir(), "my-service")
				require.NoError(t, os.Mkdir(output, 0o755))
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && printf 'generated\n' > {{ .Folder }}/README.md && printf 'other\n' > ` + output + `/other.txt
remove:
  - file: app/lib/page.ex
`})

				err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

				require.ErrorContains(t, err, "removes lib/page.ex, which the generate command does not write")
				require.Equal(t, map[string]string{"other.txt": "other\n"}, filesIn(t, output))
			})

			t.Run("inserts the items above a block comment directly above the node, also when the language has an earlier comment kind", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'fn main() {\n    let v = [\n        1,\n        /* the last one */\n        2,\n    ];\n}\n' > {{ .Folder }}/main.rs
extension_points:
  main.numbers:
    summary: A number.
    file: app/main.rs
    item: [number]
    language: rust
    rule:
      pattern: "2"
      inside:
        pattern: let v = [$$$NUMBERS];
        stopBy: end
    before: match
    separator: ","
    insert: "{{ .number }}"
add_to:
  main.numbers:
    - number: "9"
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"main.rs": "fn main() {\n    let v = [\n        1,\n        9,\n        /* the last one */\n        2,\n    ];\n}\n"}, repoFiles(t, output))
			})

			t.Run("stops when the output folder is not empty, and leaves the files in it as they are", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && printf 'generated\n' > {{ .Folder }}/README.md
`})
				output := filepath.Join(t.TempDir(), "my-service")
				require.NoError(t, os.MkdirAll(output, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(output, "notes.txt"), []byte("mine\n"), 0o644))

				err := selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

				require.ErrorContains(t, err, "is not empty")
				require.Equal(t, map[string]string{"notes.txt": "mine\n"}, filesIn(t, output))
			})

			t.Run("stops when the pattern of a removal matches no place, and names the removal and the file", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: mkdir {{ .Folder }} && printf 'scope do\n  get :home\nend\n' > {{ .Folder }}/router.ex
remove:
  - file: app/router.ex
    language: elixir
    pattern: get :index
`})

				_, err := made(t, selection)

				require.ErrorContains(t, err, "the removal 1 of the starter matches 0 places in router.ex, and it must match exactly one")
			})

			t.Run("renders the options of the starter in the file and in the pattern of a removal", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
options:
  app:
    default: shop
  function:
    default: b
folders:
  app:
    path: .
    generate: |
      mkdir -p {{ .Folder }}/lib
      printf 'defmodule Shop do\n  def a, do: 1\n  def b, do: 2\nend\n' > {{ .Folder }}/lib/shop.ex
remove:
  - file: app/lib/{{ option "starter.app" }}.ex
    language: elixir
    pattern: 'def {{ option "starter.function" }}, do: $VALUE'
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"lib/shop.ex": "defmodule Shop do\n  def a, do: 1\nend\n"}, repoFiles(t, output))
			})

			t.Run("renders the options of the starter in the strings of a rule, also inside a list of the rule", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
options:
  app:
    default: shop
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'children = [\n  Shop.Repo,\n  Shop.Endpoint\n]\n' > {{ .Folder }}/application.ex
extension_points:
  application.children:
    summary: A child.
    file: app/application.ex
    item: [child]
    language: elixir
    rule:
      any:
        - pattern: '{{ pascal (option "starter.app") }}.Endpoint'
      inside:
        pattern: children = [$$$CHILDREN]
        stopBy: end
    before: match
    separator: ","
    insert: "{{ .child }}"
add_to:
  application.children:
    - child: Oban
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"application.ex": "children = [\n  Shop.Repo,\n  Oban,\n  Shop.Endpoint\n]\n"}, repoFiles(t, output))
			})

			t.Run("stops when the generate command writes nothing at the folder that it gets", func(t *testing.T) {
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: echo "the generator wrote nothing"
`})

				_, err := made(t, selection)

				require.ErrorContains(t, err, "wrote no folder at the path that {{ .Folder }} gives")
			})

			t.Run("inserts the items before a node that has other code before it on its line, and starts a new line at the node", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'steps = ["compile", "test"]\n' > {{ .Folder }}/mix.exs
extension_points:
  mix.steps:
    summary: A step.
    file: app/mix.exs
    item: [step]
    language: elixir
    rule:
      pattern: '"test"'
      inside:
        pattern: steps = [$$$STEPS]
        stopBy: end
    before: match
    separator: ","
    insert: "{{ .step }}"
add_to:
  mix.steps:
    - step: '"xref"'
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"mix.exs": "steps = [\"compile\", \"xref\",\n\"test\"]\n"}, repoFiles(t, output))
			})

			t.Run("inserts the items after a single metavariable", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'x = [:a, :b]\n' > {{ .Folder }}/list.ex
extension_points:
  list.items:
    summary: An item.
    file: app/list.ex
    item: [atom]
    language: elixir
    pattern: x = [$FIRST, $$$REST]
    after: FIRST
    separator: ","
    insert: "{{ .atom }}"
add_to:
  list.items:
    - atom: ":c"
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"list.ex": "x = [:a,\n:c, :b]\n"}, repoFiles(t, output))
			})

			t.Run("inserts the items above the comment lines of a language whose comments have the kind line_comment", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'fn main() {\n    let v = [\n        1,\n        // the last one\n        2,\n    ];\n}\n' > {{ .Folder }}/main.rs
extension_points:
  main.numbers:
    summary: A number.
    file: app/main.rs
    item: [number]
    language: rust
    rule:
      pattern: "2"
      inside:
        pattern: let v = [$$$NUMBERS];
        stopBy: end
    before: match
    separator: ","
    insert: "{{ .number }}"
add_to:
  main.numbers:
    - number: "9"
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"main.rs": "fn main() {\n    let v = [\n        1,\n        9,\n        // the last one\n        2,\n    ];\n}\n"}, repoFiles(t, output))
			})

			t.Run("inserts the items after the separator when the last node of the metavariable is a separator", func(t *testing.T) {
				requireASTGrep(t)
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: |
      mkdir {{ .Folder }}
      printf 'deps = [\n  :a,\n  :b,\n]\n' > {{ .Folder }}/deps.ex
extension_points:
  deps.items:
    summary: A dependency.
    file: app/deps.ex
    item: [dep]
    language: elixir
    pattern: deps = [$$$DEPS]
    after: DEPS
    separator: ","
    insert: "{{ .dep }}"
add_to:
  deps.items:
    - dep: ":c"
`})

				output, err := made(t, selection)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"deps.ex": "deps = [\n  :a,\n  :b,\n  :c,\n]\n"}, repoFiles(t, output))
			})

			t.Run("stops before the generate command runs when ast-grep is not on the PATH, and says where to get it", func(t *testing.T) {
				ran := filepath.Join(t.TempDir(), "ran")
				selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
folders:
  app:
    path: .
    generate: touch ` + ran + ` && mkdir {{ .Folder }}
extension_points:
  mix.deps:
    summary: A dependency.
    file: app/mix.exs
    item: [dep]
    language: elixir
    pattern: defp deps do [$$$DEPS] end
    after: DEPS
    insert: "{{ .dep }}"
`})
				t.Setenv("PATH", pathWithOnly(t, "sh", "touch", "mkdir"))

				_, err := made(t, selection)

				require.ErrorContains(t, err, "the starter changes the files of its generator with ast-grep: ast-grep is not on the PATH: install it from https://ast-grep.github.io")
				require.NoFileExists(t, ran)
			})
		})

		t.Run("make repo writes the catalog of the starter in .feature-compose.yml", func(t *testing.T) {
			selection, err := namedStarter(t).Select("api", nil, nil, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "api")

			err = selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.NoError(t, err)
			require.Equal(t, "version: v1.2.3\ncatalog:\n  name: hpcsc\n  source: https://example.com/hpcsc.git\n  ref: main\n  commit: 0123abc\nstarter: service\nname: api\noptions: {}\nfeatures: {}\nrequested: []\n", filesIn(t, output)[".feature-compose.yml"])
		})

		t.Run("with pinned_tools, runs each after_render command with the tools of the mise.toml of the folder, and mise reads no other mise.toml and trusts or tracks no file of the repo", func(t *testing.T) {
			requireMise(t)
			state := t.TempDir()
			t.Setenv("MISE_STATE_DIR", state)
			parent := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(parent, "mise.toml"), []byte("[env]\nFROM_PARENT = \"yes\"\n"), 0o644))
			output := filepath.Join(parent, "my-service")

			err := generatorSelection(t, map[string]string{
				"starters/service/starter.yml":         "folders:\n  app:\n    path: .\n    pinned_tools: true\n    after_render: ['shellcheck --version > shellcheck.txt', 'echo \"${FROM_PARENT:-none}\" > parent.txt']\n",
				"starters/service/files/app/mise.toml": "[tools]\n\"aqua:koalaman/shellcheck\" = \"0.11.0\"\n",
			}).MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.NoError(t, err)
			files := filesIn(t, output)
			require.Contains(t, files["shellcheck.txt"], "version: 0.11.0")
			require.Equal(t, "none\n", files["parent.txt"])
			realParent, err := filepath.EvalSymlinks(parent)
			require.NoError(t, err)
			for _, target := range linkTargetsIn(t, state) {
				require.False(t, strings.HasPrefix(target, realParent), "mise trusts or tracks %s", target)
			}
		})

		t.Run("with pinned_tools and a relative output folder, mise trusts or tracks no file of the repo", func(t *testing.T) {
			requireMise(t)
			state := t.TempDir()
			t.Setenv("MISE_STATE_DIR", state)
			work := t.TempDir()
			t.Chdir(work)

			err := generatorSelection(t, map[string]string{
				"starters/service/starter.yml":         "folders:\n  app:\n    path: .\n    pinned_tools: true\n    after_render: ['shellcheck --version > shellcheck.txt']\n",
				"starters/service/files/app/mise.toml": "[tools]\n\"aqua:koalaman/shellcheck\" = \"0.11.0\"\n",
			}).MakeRepo(context.Background(), "my-service", "v1.2.3", catalog.NoSteps{})

			require.NoError(t, err)
			realWork, err := filepath.EvalSymlinks(work)
			require.NoError(t, err)
			for _, target := range linkTargetsIn(t, state) {
				require.False(t, strings.HasPrefix(target, realWork), "mise trusts or tracks %s", target)
			}
		})

		t.Run("with pinned_tools, runs the generate command with the tools of the mise.toml of the folder", func(t *testing.T) {
			requireMise(t)
			output := filepath.Join(t.TempDir(), "my-service")

			err := generatorSelection(t, map[string]string{
				"starters/service/starter.yml":         "folders:\n  app:\n    path: .\n    pinned_tools: true\n    generate: mkdir {{ .Folder }} && shellcheck --version > {{ .Folder }}/generated.txt\n",
				"starters/service/files/app/mise.toml": "[tools]\n\"aqua:koalaman/shellcheck\" = \"0.11.0\"\n",
			}).MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.NoError(t, err)
			require.Contains(t, filesIn(t, output)["generated.txt"], "version: 0.11.0")
		})

		t.Run("with pinned_tools, gives mise each version of a core tool as MISE_<TOOL>_VERSION, so that the install of a tool finds the tools that it needs", func(t *testing.T) {
			bin := pathWithOnly(t, "sh", "grep", "env")
			variables := filepath.Join(t.TempDir(), "variables.txt")
			fakeMise := "#!/bin/sh\nenv | grep '^MISE_.*_VERSION=' >> '" + variables + "'\nwhile [ \"$1\" != -- ]; do shift; done\nshift\nexec \"$@\"\n"
			require.NoError(t, os.WriteFile(filepath.Join(bin, "mise"), []byte(fakeMise), 0o755))
			t.Setenv("PATH", bin)
			output := filepath.Join(t.TempDir(), "my-service")

			err := generatorSelection(t, map[string]string{
				"starters/service/starter.yml":         "folders:\n  app:\n    path: .\n    pinned_tools: true\n    after_render: ['echo ran > ran.txt']\n",
				"starters/service/files/app/mise.toml": "[tools]\nerlang = \"29.1\"\nelixir = \"1.20.4-otp-29\"\ngo-task = \"3.53.1\"\n\"aqua:koalaman/shellcheck\" = \"0.11.0\"\n",
			}).MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.NoError(t, err)
			recorded, err := os.ReadFile(variables)
			require.NoError(t, err)
			require.Equal(t, []string{"MISE_ELIXIR_VERSION=1.20.4-otp-29", "MISE_ERLANG_VERSION=29.1", "MISE_GO_TASK_VERSION=3.53.1"}, slices.Sorted(slices.Values(strings.Split(strings.TrimSpace(string(recorded)), "\n"))))
			require.Equal(t, "ran\n", filesIn(t, output)["ran.txt"])
		})

		t.Run("with pinned_tools and no mise on the PATH, stops, and the message names mise", func(t *testing.T) {
			t.Setenv("PATH", pathWithOnly(t, "sh"))
			output := filepath.Join(t.TempDir(), "my-service")

			err := generatorSelection(t, map[string]string{
				"starters/service/starter.yml":         "folders:\n  app:\n    path: .\n    pinned_tools: true\n    after_render: ['shellcheck --version > shellcheck.txt', 'echo \"${FROM_PARENT:-none}\" > parent.txt']\n",
				"starters/service/files/app/mise.toml": "[tools]\n\"aqua:koalaman/shellcheck\" = \"0.11.0\"\n",
			}).MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.ErrorContains(t, err, "the folder pins its tools in mise.toml, so its commands run with mise, and mise is not on the PATH: install it from https://mise.jdx.dev")
		})

		t.Run("with pinned_tools, a tool of the mise.toml with no version as a text stops before a command runs", func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "my-service")

			err := generatorSelection(t, map[string]string{
				"starters/service/starter.yml":         "folders:\n  app:\n    path: .\n    pinned_tools: true\n    after_render: [touch ran.txt]\n",
				"starters/service/files/app/mise.toml": "[tools]\nshellcheck = { version = \"0.11.0\" }\n",
			}).MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{})

			require.EqualError(t, err, `read mise.toml of the folder app, which the starter writes: the tool shellcheck has no version as a text, for example shellcheck = "1.0.0"`)
			require.NoFileExists(t, filepath.Join(output, "ran.txt"))
		})
	})

	t.Run("preview", func(t *testing.T) {
		t.Run("lists the generate command and each after_render command of each folder in the order in which new runs them, with <new folder> for the folder that the generate command gets", func(t *testing.T) {
			selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
options:
  app:
    default: shop
folders:
  web:
    path: web
    generate: npm create vite {{ .Folder }}
    after_render: [npm install]
  app:
    path: app
    generate: mix phx.new {{ .Folder }} --app {{ option "starter.app" }} --no-install
    after_render: [mix deps.get, mix format]
`})

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "\nCommands:\n  app: mix phx.new <new folder> --app shop --no-install\n  web: npm create vite <new folder>\n  app: mix deps.get\n  app: mix format\n  web: npm install\n\n")
		})

		t.Run("stops when the generate command reads an option whose value is not one shell word, as new does", func(t *testing.T) {
			selection := generatorSelection(t, map[string]string{"starters/service/starter.yml": `
options:
  app:
    default: "shop; rm -rf ~"
folders:
  app:
    path: .
    generate: mix phx.new {{ .Folder }} --app {{ option "starter.app" }}
`})

			_, err := selection.Preview()

			require.ErrorContains(t, err, `reads the option starter.app, whose value "shop; rm -rf ~" is not one shell word`)
		})

		withFeature := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":                       &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n  web:\n    path: frontend\n")},
				"starters/service/files/go/go.mod.tmpl":              &fstest.MapFile{Data: []byte("module {{ .Name }}\n")},
				"starters/service/files/web/index.html":              &fstest.MapFile{Data: []byte("<html></html>\n")},
				"features/cli/feature.yml":                           &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/cli/files/go/cmd/{{ .Name }}/main.go.tmpl": &fstest.MapFile{Data: []byte("package main\n")},
			}
		}

		t.Run("gives the name of the repo, its starter, and each file by its path in the repo, in the order of the paths, with its owner", func(t *testing.T) {
			preview, err := featureSelection(t, withFeature(), "cli").Preview()

			require.NoError(t, err)
			require.Equal(t, `Repo: my-service
Starter: service

Features:
  cli: command line

Options: none

Commands: none

Files:
  .feature-compose.yml: feature-compose
  cmd/my-service/main.go: the feature cli
  frontend/index.html: the starter
  go.mod: the starter
`, preview)
		})

		t.Run("gives the path of each file that the write of the same selection writes, one time each", func(t *testing.T) {
			selection := featureSelection(t, withFeature(), "cli")
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{}))

			preview, err := selection.Preview()

			require.NoError(t, err)
			_, filesPart, _ := strings.Cut(preview, "Files:\n")
			var previewed []string
			for _, line := range strings.Split(strings.TrimSuffix(filesPart, "\n"), "\n") {
				path, _, _ := strings.Cut(strings.TrimPrefix(line, "  "), ": ")
				previewed = append(previewed, path)
			}
			require.Equal(t, slices.Sorted(maps.Keys(filesIn(t, output))), previewed)
		})

		t.Run("a selection with no name stops the preview with the message of the name only, also with an option that has no default and no value", func(t *testing.T) {
			files := withFeature()
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("summary: A service.\noptions:\n  region:\n    pattern: '^[a-z]*$'\nfolders:\n  go:\n    path: .\n  web:\n    path: frontend\n")}

			_, err := selectionOf(t, files, "").Preview()

			require.EqualError(t, err, `the name "" of the repo must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'`)
		})

		t.Run("a path that starts with the text of another path comes after it", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":       oneFolder,
				"starters/service/files/go/notes.md": &fstest.MapFile{Data: []byte("notes\n")},
				"starters/service/files/go/notes":    &fstest.MapFile{Data: []byte("notes\n")},
			}

			preview, err := featureSelection(t, files).Preview()

			require.NoError(t, err)
			require.True(t, strings.HasSuffix(preview, "\nFiles:\n  .feature-compose.yml: feature-compose\n  notes: the starter\n  notes.md: the starter\n"), preview)
		})

		t.Run("a starter with no template gives a repo with the selection file as its one file", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml": oneFolder,
				"starters/service/files":       &fstest.MapFile{Mode: fs.ModeDir | 0o755},
			}

			preview, err := featureSelection(t, files).Preview()

			require.NoError(t, err)
			require.Equal(t, "Repo: my-service\nStarter: service\n\nFeatures: none\n\nOptions: none\n\nCommands: none\n\nFiles:\n  .feature-compose.yml: feature-compose\n", preview)
		})

		t.Run("gives each feature after its needs and otherwise in the order of the names, a given feature with the source command line, and a feature that a need added with the features that need it", func(t *testing.T) {
			files := withFeatureDefinitions(map[string]string{"x": "needs: [z, y]\n", "y": "needs: [z]\n", "z": "", "w": ""})

			preview, err := featureSelection(t, files, "x", "w").Preview()

			require.NoError(t, err)
			require.Equal(t, `Repo: my-service
Starter: service

Features:
  w: command line
  z: needed by x, y
  y: needed by x
  x: command line

Options: none

Commands: none

Files:
  .feature-compose.yml: feature-compose
  main.go: the starter
`, preview)
		})

		t.Run("a given feature that selected features need shows the source command line and the features that need it", func(t *testing.T) {
			files := withFeatureDefinitions(map[string]string{"a": "needs: [c, d]\n", "b": "needs: [c]\n", "c": "", "d": ""})

			preview, err := featureSelection(t, files, "a", "b", "c", "d").Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "Features:\n  c: command line, needed by a, b\n  b: command line\n  d: command line, needed by a\n  a: command line\n\n")
		})

		withOptions := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("options:\n  module:\n    default: example.com/{{ .Name }}\n  go-version:\n    default: 1.27.1\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/a/feature.yml":            &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds: [b]\noptions:\n  level:\n    default: \"1\"\n")},
				"features/b/feature.yml":            &fstest.MapFile{Data: []byte("summary: B.\ntargets: [service]\noptions:\n  zeta:\n    default: z\n  mode:\n    values: [fast, slow]\n    default: fast\n")},
				"features/c/feature.yml":            &fstest.MapFile{Data: []byte("summary: C.\ntargets: [service]\n")},
			}
		}

		t.Run("gives each option of the starter as starter.<option>, then each option of each feature as <feature>.<option> with the features in their order, each with the value that the templates read and its source", func(t *testing.T) {
			starter, err := catalog.New(withOptions()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"go-version": "1.28"}, map[string]map[string]string{"a": {"level": "2"}, "c": {}}, catalog.Answers{})
			require.NoError(t, err)

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, `
Options:
  starter.go-version: 1.28 (command line)
  starter.module: example.com/my-service (default)
  b.mode: fast (default)
  b.zeta: z (default)
  a.level: 2 (command line)

`)
		})

		withNeedValues := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/a/feature.yml":            &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds:\n  b:\n    mode: slow\n    zeta: z\n")},
				"features/b/feature.yml":            &fstest.MapFile{Data: []byte("summary: B.\ntargets: [service]\noptions:\n  zeta:\n    default: z\n  mode:\n    values: [fast, slow]\n    default: fast\n")},
				"features/c/feature.yml":            &fstest.MapFile{Data: []byte("summary: C.\ntargets: [service]\nneeds:\n  b:\n    mode: slow\n")},
			}
		}

		for name, test := range map[string]struct {
			givenByFeature map[string]map[string]string
			options        string
		}{
			"one need sets each value, also a value that equals the default": {map[string]map[string]string{"a": {}}, "  b.mode: slow (need of a)\n  b.zeta: z (need of a)\n"},
			"the needs of two features set one value":                        {map[string]map[string]string{"c": {}, "a": {}}, "  b.mode: slow (need of a, c)\n  b.zeta: z (need of a)\n"},
			"a flag gives the value that a need sets":                        {map[string]map[string]string{"a": {}, "b": {"mode": "slow"}}, "  b.mode: slow (command line, need of a)\n  b.zeta: z (need of a)\n"},
		} {
			t.Run("a value that a need sets shows the source need of the features whose needs set it, in the order of their names: "+name, func(t *testing.T) {
				starter, err := catalog.New(withNeedValues()).Starter("service")
				require.NoError(t, err)
				selection, err := starter.Select("my-service", nil, test.givenByFeature, catalog.Answers{})
				require.NoError(t, err)

				preview, err := selection.Preview()

				require.NoError(t, err)
				require.Contains(t, preview, "\nOptions:\n"+test.options+"\n")
			})
		}

		t.Run("the features whose needs set a value come in the order of their names, also when one of them needs the other, and each need of a feature sets its values", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/a/feature.yml":            &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds:\n  b:\n    mode: slow\n  d:\n    tone: loud\n  z:\n")},
				"features/b/feature.yml":            &fstest.MapFile{Data: []byte("summary: B.\ntargets: [service]\noptions:\n  mode:\n    values: [fast, slow]\n    default: fast\n")},
				"features/d/feature.yml":            &fstest.MapFile{Data: []byte("summary: D.\ntargets: [service]\noptions:\n  tone:\n    default: soft\n")},
				"features/z/feature.yml":            &fstest.MapFile{Data: []byte("summary: Z.\ntargets: [service]\nneeds:\n  b:\n    mode: slow\n")},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"a": {}}, catalog.Answers{})
			require.NoError(t, err)

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "\nOptions:\n  b.mode: slow (need of a, z)\n  d.tone: loud (need of a)\n\n")
		})

		t.Run("a value that a flag gives with the text of the default shows the source command line", func(t *testing.T) {
			starter, err := catalog.New(withOptions()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"module": "example.com/my-service"}, map[string]map[string]string{"b": {"mode": "fast"}}, catalog.Answers{})
			require.NoError(t, err)

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "  starter.module: example.com/my-service (command line)\n")
			require.Contains(t, preview, "  b.mode: fast (command line)\n")
			require.Contains(t, preview, "  b.zeta: z (default)\n")
		})

		t.Run("a change to the given values after the select does not change the sources of the preview", func(t *testing.T) {
			starter, err := catalog.New(withOptions()).Starter("service")
			require.NoError(t, err)
			given := map[string]string{}
			givenByFeature := map[string]map[string]string{"b": {}}
			selection, err := starter.Select("my-service", given, givenByFeature, catalog.Answers{})
			require.NoError(t, err)

			given["module"] = "example.com/my-service"
			givenByFeature["b"]["mode"] = "fast"
			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "  starter.module: example.com/my-service (default)\n")
			require.Contains(t, preview, "  b.mode: fast (default)\n")
		})

		t.Run("a feature that an answer selects shows the source answered, and the features that need it, next to a given feature and a feature that only a need adds", func(t *testing.T) {
			starter, err := catalog.New(withFeatureDefinitions(map[string]string{"a": "needs: [c, d]\n", "b": "", "c": "", "d": ""})).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"b": {}}, catalog.Answers{Features: []string{"a", "c"}})
			require.NoError(t, err)

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "Features:\n  b: command line\n  c: answered, needed by a\n  d: needed by a\n  a: answered\n\n")
		})

		t.Run("each value that an answer gives shows the source answered, also when it equals the default, next to a value of a flag and a default", func(t *testing.T) {
			starter, err := catalog.New(withOptions()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"go-version": "1.28"}, nil, catalog.Answers{
				Features:      []string{"a"},
				StarterValues: map[string]string{"module": "example.com/my-service"},
				FeatureValues: map[string]map[string]string{"b": {"mode": "slow"}, "a": {"level": "2"}},
			})
			require.NoError(t, err)

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, `
Options:
  starter.go-version: 1.28 (command line)
  starter.module: example.com/my-service (answered)
  b.mode: slow (answered)
  b.zeta: z (default)
  a.level: 2 (answered)

`)
		})

		t.Run("a flag and an answer for one option give the value of the flag with the source command line, and a feature that a flag and an answer select shows command line", func(t *testing.T) {
			starter, err := catalog.New(withOptions()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"module": "example.com/flag"}, map[string]map[string]string{"b": {"mode": "fast"}}, catalog.Answers{
				Features:      []string{"b"},
				StarterValues: map[string]string{"module": "example.com/answer"},
				FeatureValues: map[string]map[string]string{"b": {"mode": "slow"}},
			})
			require.NoError(t, err)

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "Features:\n  b: command line\n\n")
			require.Contains(t, preview, "  starter.module: example.com/flag (command line)\n")
			require.Contains(t, preview, "  b.mode: fast (command line)\n")
		})

		for name, test := range map[string]struct {
			files    fstest.MapFS
			features []string
		}{
			"a template whose path fails to render": {files: fstest.MapFS{
				"starters/service/starter.yml":               oneFolder,
				"starters/service/files/go/{{ .Nmae }}/a.go": &fstest.MapFile{Data: []byte("package a\n")},
			}},
			"a template that does not parse": {files: fstest.MapFS{
				"starters/service/starter.yml":   oneFolder,
				"starters/service/files/go/b.go": &fstest.MapFile{Data: []byte("package {{ .Name\n")},
			}},
			"a template that fails to render": {files: fstest.MapFS{
				"starters/service/starter.yml":   oneFolder,
				"starters/service/files/go/b.go": &fstest.MapFile{Data: []byte("package {{ .Nmae }}\n")},
			}},
			"a path that leads out of the repo": {files: fstest.MapFS{
				"starters/service/starter.yml":                                 &fstest.MapFile{Data: []byte("options:\n  app:\n    default: ../../escape\nfolders:\n  go:\n    path: .\n")},
				`starters/service/files/go/{{ option "starter.app" }}/main.go`: &fstest.MapFile{Data: []byte("package main\n")},
			}},
			"two templates of the starter that write one path": {files: fstest.MapFS{
				"starters/service/starter.yml":           oneFolder,
				"starters/service/files/go/main.go":      &fstest.MapFile{Data: []byte("package main\n")},
				"starters/service/files/go/main.go.tmpl": &fstest.MapFile{Data: []byte("package main\n")},
			}},
			"a template of the starter and a template of a feature that write one path": {files: fstest.MapFS{
				"starters/service/starter.yml":       oneFolder,
				"starters/service/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
				"features/cli/feature.yml":           &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/cli/files/go/main.go.tmpl": &fstest.MapFile{Data: []byte("package cli\n")},
			}, features: []string{"cli"}},
			"an option with no default and no value": {files: fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("options:\n  region:\n    default: \"\"\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}},
		} {
			t.Run(name+" stops the preview with the message that stops the write of the same selection", func(t *testing.T) {
				selection := featureSelection(t, test.files, test.features...)
				writeErr := selection.Write(filepath.Join(t.TempDir(), "my-service"))
				require.Error(t, writeErr)

				preview, err := selection.Preview()

				require.EqualError(t, err, writeErr.Error())
				require.Empty(t, preview)
			})
		}

		t.Run("with pinned_tools, shows each command with mise exec and the tools of the mise.toml of the folder", func(t *testing.T) {
			preview, err := generatorSelection(t, map[string]string{
				"starters/service/starter.yml":         "folders:\n  app:\n    path: .\n    pinned_tools: true\n    generate: mix phx.new {{ .Folder }}\n    after_render: [mix deps.get]\n",
				"starters/service/files/app/mise.toml": "[tools]\nerlang = \"29.1\"\nelixir = \"1.20.4-otp-29\"\n",
			}).Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "\nCommands:\n  app: mise exec elixir@1.20.4-otp-29 erlang@29.1 -- mix phx.new <new folder>\n  app: mise exec elixir@1.20.4-otp-29 erlang@29.1 -- mix deps.get\n\n")
		})
	})

	t.Run("questions", func(t *testing.T) {
		withQuestions := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":       &fstest.MapFile{Data: []byte("options:\n  module:\n    default: example.com/{{ .Name }}\n    pattern: '^[a-z./-]+$'\n  region:\n    pattern: '^[a-z]+$'\n  label:\n    default: x\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.txt": &fstest.MapFile{Data: []byte(`{{ option "starter.module" }} {{ option "starter.region" }} {{ option "starter.label" }}`)},
				"features/alpha/feature.yml":         &fstest.MapFile{Data: []byte("summary: Alpha.\ntargets: [service]\nneeds: [zed]\nfiles:\n  - path: files/\noptions:\n  level:\n    default: \"1\"\n")},
				"features/alpha/files/go/alpha.txt":  &fstest.MapFile{Data: []byte(`{{ option "level" }} {{ option "zed.mode" }} {{ option "zed.token" }}`)},
				"features/beta/feature.yml":          &fstest.MapFile{Data: []byte("summary: Beta.\ntargets: [service]\nneeds: [zed]\n")},
				"features/zed/feature.yml":           &fstest.MapFile{Data: []byte("summary: Zed.\ntargets: [service]\noptions:\n  token:\n    pattern: '^.*$'\n  mode:\n    values: [slow, fast]\n    default: fast\n")},
			}
		}
		selectionWith := func(t *testing.T, given map[string]string, givenByFeature map[string]map[string]string) catalog.Selection {
			t.Helper()
			starter, err := catalog.New(withQuestions()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", given, givenByFeature, catalog.Answers{})
			require.NoError(t, err)
			return selection
		}
		type question struct {
			feature, option, value, source, defaultValue, pattern string
			given, missing                                        bool
			accepted, needOf                                      []string
		}
		contentsOf := func(questions []catalog.Question) []question {
			contents := make([]question, 0, len(questions))
			for _, q := range questions {
				contents = append(contents, question{feature: q.Feature, option: q.Option, value: q.Value, source: q.Source, defaultValue: q.Default, pattern: q.Pattern, given: q.Given, missing: q.Missing, accepted: q.Accepted, needOf: q.NeedOf})
			}
			return contents
		}
		questionOf := func(t *testing.T, selection catalog.Selection, featureName, optionName string) catalog.Question {
			t.Helper()
			for _, q := range selection.Questions() {
				if q.Feature == featureName && q.Option == optionName {
					return q
				}
			}
			require.Failf(t, "no question", "%s %s", featureName, optionName)
			return catalog.Question{}
		}

		t.Run("gives the options of the starter in the order of their names, then the options of each feature, each feature after the features that it needs, each option one time, with its value and its source, the default rendered with the name, and the values in the order of the file or the pattern", func(t *testing.T) {
			selection := selectionWith(t, nil, map[string]map[string]string{"alpha": {}, "beta": {}})

			questions := selection.Questions()

			require.Equal(t, []question{
				{option: "label", value: "x", source: "default", defaultValue: "x"},
				{option: "module", value: "example.com/my-service", source: "default", defaultValue: "example.com/my-service", pattern: "^[a-z./-]+$"},
				{option: "region", value: "", source: "default", defaultValue: "", pattern: "^[a-z]+$", missing: true},
				{feature: "zed", option: "mode", value: "fast", source: "default", defaultValue: "fast", accepted: []string{"slow", "fast"}},
				{feature: "zed", option: "token", value: "", source: "default", defaultValue: "", pattern: "^.*$", missing: true},
				{feature: "alpha", option: "level", value: "1", source: "default", defaultValue: "1"},
			}, contentsOf(questions))
		})

		t.Run("an option that a flag gives is a given question with the value of the flag and the source command line, next to the other options", func(t *testing.T) {
			selection := selectionWith(t, map[string]string{"region": "eu", "label": "y"}, map[string]map[string]string{"zed": {"mode": "slow"}})

			questions := selection.Questions()

			require.Equal(t, []question{
				{option: "label", value: "y", source: "command line", defaultValue: "x", given: true},
				{option: "module", value: "example.com/my-service", source: "default", defaultValue: "example.com/my-service", pattern: "^[a-z./-]+$"},
				{option: "region", value: "eu", source: "command line", defaultValue: "", pattern: "^[a-z]+$", given: true},
				{feature: "zed", option: "mode", value: "slow", source: "command line", defaultValue: "fast", given: true, accepted: []string{"slow", "fast"}},
				{feature: "zed", option: "token", value: "", source: "default", defaultValue: "", pattern: "^.*$", missing: true},
			}, contentsOf(questions))
		})

		t.Run("an option that needs set has the value of the needs, the source of the preview and the features whose needs set it, and only a flag makes it given", func(t *testing.T) {
			files := withQuestions()
			files["features/alpha/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Alpha.\ntargets: [service]\nneeds:\n  zed:\n    mode: slow\n")}
			files["features/beta/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Beta.\ntargets: [service]\nneeds:\n  zed:\n    mode: slow\n")}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			needed, err := starter.Select("my-service", nil, map[string]map[string]string{"alpha": {}, "beta": {}}, catalog.Answers{})
			require.NoError(t, err)
			alsoGiven, err := starter.Select("my-service", nil, map[string]map[string]string{"alpha": {}, "beta": {}, "zed": {"mode": "slow"}}, catalog.Answers{})
			require.NoError(t, err)

			require.Equal(t, []question{
				{feature: "zed", option: "mode", value: "slow", source: "need of alpha, beta", defaultValue: "fast", accepted: []string{"slow", "fast"}, needOf: []string{"alpha", "beta"}},
				{feature: "zed", option: "token", value: "", source: "default", defaultValue: "", pattern: "^.*$", missing: true},
				{feature: "zed", option: "mode", value: "slow", source: "command line, need of alpha, beta", defaultValue: "fast", given: true, accepted: []string{"slow", "fast"}, needOf: []string{"alpha", "beta"}},
			}, contentsOf([]catalog.Question{questionOf(t, needed, "zed", "mode"), questionOf(t, needed, "zed", "token"), questionOf(t, alsoGiven, "zed", "mode")}))
		})

		t.Run("an option is fixed when a flag gives it or a need sets it, and not when it has its default", func(t *testing.T) {
			files := withQuestions()
			files["features/alpha/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Alpha.\ntargets: [service]\nneeds:\n  zed:\n    mode: slow\n")}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"region": "eu"}, map[string]map[string]string{"alpha": {}}, catalog.Answers{})
			require.NoError(t, err)

			require.Equal(t, []bool{true, true, false}, []bool{questionOf(t, selection, "", "region").Fixed(), questionOf(t, selection, "zed", "mode").Fixed(), questionOf(t, selection, "zed", "token").Fixed()})
		})

		t.Run("an option that the answers give has the source answered", func(t *testing.T) {
			starter, err := catalog.New(withQuestions()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, nil, catalog.Answers{StarterValues: map[string]string{"label": "z"}})
			require.NoError(t, err)

			require.Equal(t, question{option: "label", value: "z", source: "answered", defaultValue: "x"}, contentsOf(selection.Questions())[0])
		})

		t.Run("each question of a selection whose each value a flag gives is given", func(t *testing.T) {
			selection := selectionWith(t, map[string]string{"region": "eu", "label": "y", "module": "example.com/a"}, map[string]map[string]string{"zed": {"mode": "slow", "token": "t"}})

			var given []bool
			for _, q := range selection.Questions() {
				given = append(given, q.Given)
			}
			require.Equal(t, []bool{true, true, true, true, true}, given)
		})

		for name, test := range map[string]struct{ featureName, optionName, value string }{
			"a value that does not match the pattern of its option": {featureName: "", optionName: "region", value: "EU"},
			"a value that is not one of the values of its option":   {featureName: "zed", optionName: "mode", value: "medium"},
		} {
			t.Run("check refuses "+name+" with the message that the selection gives for the same answer", func(t *testing.T) {
				selection := selectionWith(t, nil, map[string]map[string]string{"zed": {}})
				q := questionOf(t, selection, test.featureName, test.optionName)
				starter, err := catalog.New(withQuestions()).Starter("service")
				require.NoError(t, err)
				_, selectErr := starter.Select("my-service", nil, map[string]map[string]string{"zed": {}}, catalog.Answers{}.WithValue(q.Feature, q.Option, test.value))
				require.Error(t, selectErr)

				err = q.Check(test.value)

				require.EqualError(t, err, selectErr.Error())
			})
		}

		t.Run("check refuses the empty text for an option with no default, and names the option", func(t *testing.T) {
			selection := selectionWith(t, nil, map[string]map[string]string{"zed": {}})

			err := questionOf(t, selection, "zed", "token").Check("")

			require.EqualError(t, err, "the option token of the feature zed has no default, so it needs a value")
		})

		t.Run("check accepts the empty text for an option with a default when the option accepts it", func(t *testing.T) {
			selection := selectionWith(t, nil, nil)

			err := questionOf(t, selection, "", "label").Check("")

			require.NoError(t, err)
		})

		t.Run("check refuses a comma in a value of an option of a feature, because --feature cannot give it", func(t *testing.T) {
			selection := selectionWith(t, nil, map[string]map[string]string{"zed": {}})

			err := questionOf(t, selection, "zed", "token").Check("a,b")

			require.EqualError(t, err, `the value "a,b" of the option token of the feature zed holds a comma, which --feature cannot give`)
		})

		t.Run("each answer that check accepts, given back to the selection, gives the files with that value, and a value of the starter keeps its commas", func(t *testing.T) {
			selection := selectionWith(t, nil, map[string]map[string]string{"alpha": {}})
			answerOf := map[string]string{
				"module": "example.com/acme", "region": "eu", "label": "a,b",
				"zed mode": "slow", "zed token": "t", "alpha level": "2",
			}
			answers := catalog.Answers{}
			for _, q := range selection.Questions() {
				value := answerOf[strings.TrimSpace(q.Feature+" "+q.Option)]
				require.NoError(t, q.Check(value), q.Option)
				answers = answers.WithValue(q.Feature, q.Option, value)
			}
			starter, err := catalog.New(withQuestions()).Starter("service")
			require.NoError(t, err)
			answered, err := starter.Select("my-service", nil, map[string]map[string]string{"alpha": {}}, answers)
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			err = answered.Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"main.txt": "example.com/acme eu a,b", "alpha.txt": "2 slow t"}, filesIn(t, output))
		})
	})

	t.Run("arguments", func(t *testing.T) {
		withArguments := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("options:\n  app:\n    default: api\n  module:\n    default: example.com/{{ .Name }}\n  region:\n    default: us\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/a/feature.yml":            &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds: [x, y]\n")},
				"features/c/feature.yml":            &fstest.MapFile{Data: []byte("summary: C.\ntargets: [service]\noptions:\n  tone:\n    default: soft\n")},
				"features/x/feature.yml":            &fstest.MapFile{Data: []byte("summary: X.\ntargets: [service]\n")},
				"features/y/feature.yml":            &fstest.MapFile{Data: []byte("summary: Y.\ntargets: [service]\nneeds: [z]\noptions:\n  mode:\n    values: [fast, slow]\n    default: fast\n  zone:\n    default: z\n")},
				"features/z/feature.yml":            &fstest.MapFile{Data: []byte("summary: Z.\ntargets: [service]\noptions:\n  level:\n    default: \"1\"\n")},
			}
		}
		argumentsOf := func(t *testing.T, given map[string]string, givenByFeature map[string]map[string]string, answers catalog.Answers, output string) string {
			t.Helper()
			starter, err := catalog.New(withArguments()).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", given, givenByFeature, answers)
			require.NoError(t, err)
			return selection.Arguments(output)
		}

		t.Run("an answer that selects a feature and an answer that gives its need a value that is not the default give the feature by its name and the need with that value", func(t *testing.T) {
			arguments := argumentsOf(t, nil, nil, catalog.Answers{Features: []string{"a"}, FeatureValues: map[string]map[string]string{"y": {"mode": "slow"}}}, "my-service")

			require.Equal(t, "service my-service --feature a --feature y:mode=slow", arguments)
		})

		t.Run("a value that a need sets is not in the command, because the need sets it again", func(t *testing.T) {
			files := withArguments()
			files["features/a/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\nneeds:\n  x:\n  y:\n    mode: slow\n")}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"a": {}}, catalog.Answers{})
			require.NoError(t, err)

			require.Equal(t, "service my-service --feature a", selection.Arguments("my-service"))
		})

		t.Run("give --output-dir only when the output folder is not the name of the repo", func(t *testing.T) {
			require.Equal(t, "service my-service", argumentsOf(t, nil, nil, catalog.Answers{}, "my-service"))
			require.Equal(t, "service my-service --output-dir ../elsewhere", argumentsOf(t, nil, nil, catalog.Answers{}, "../elsewhere"))
		})

		t.Run("give --option for each option of the starter whose value is not its default, in the order of the names, from a flag or an answer", func(t *testing.T) {
			arguments := argumentsOf(t, map[string]string{"region": "eu"}, nil, catalog.Answers{StarterValues: map[string]string{"app": "web", "module": "example.com/acme"}}, "my-service")

			require.Equal(t, "service my-service --option app=web --option module=example.com/acme --option region=eu", arguments)
		})

		t.Run("a value that equals its default is not in the arguments, from a flag or an answer", func(t *testing.T) {
			arguments := argumentsOf(t, map[string]string{"region": "us"}, map[string]map[string]string{"c": {"tone": "soft"}}, catalog.Answers{StarterValues: map[string]string{"app": "web", "module": "example.com/my-service"}}, "my-service")

			require.Equal(t, "service my-service --option app=web --feature c", arguments)
		})

		t.Run("give the features that a flag or an answer selects in the order of the names, then each other feature with a value that is not its default, each after the features that it needs, each with its values that are not their default in the order of the names", func(t *testing.T) {
			arguments := argumentsOf(t, nil, map[string]map[string]string{"c": {"tone": "loud"}}, catalog.Answers{
				Features:      []string{"a"},
				FeatureValues: map[string]map[string]string{"y": {"mode": "slow", "zone": "w"}, "z": {"level": "2"}},
			}, "my-service")

			require.Equal(t, "service my-service --feature a --feature c:tone=loud --feature z:level=2 --feature y:mode=slow,zone=w", arguments)
		})

		t.Run("a word that a shell splits, expands or matches against files is quoted, so sh gives each word as the selection has it", func(t *testing.T) {
			arguments := argumentsOf(t, nil, nil, catalog.Answers{StarterValues: map[string]string{"app": "a b", "module": "$HOME", "region": "it's"}}, "*")

			printed, err := exec.Command("sh", "-c", "printf '%s\\n' "+arguments).Output()

			require.NoError(t, err)
			require.Equal(t, "service\nmy-service\n--output-dir\n*\n--option\napp=a b\n--option\nmodule=$HOME\n--option\nregion=it's\n", string(printed))
		})

		t.Run("the selection names the full name of the starter, and its arguments start with it", func(t *testing.T) {
			selection, err := namedStarter(t).Select("api", nil, map[string]map[string]string{"cli": {}}, catalog.Answers{})

			require.NoError(t, err)
			require.Equal(t, []string{"hpcsc/service", "hpcsc/service api --feature cli"}, []string{selection.StarterFullName(), selection.Arguments("api")})
		})
	})

	t.Run("run after render", func(t *testing.T) {
		t.Run("runs the commands of each folder in that folder of the repo, in their order", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    after_render: ["echo one >> order", "echo two >> order"]
  web:
    path: frontend
    after_render: ["echo web > made-here"]
`, output)

			err := selection.RunAfterRender(context.Background(), output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":             "package main\n",
				"order":               "one\ntwo\n",
				"frontend/index.html": "<html></html>\n",
				"frontend/made-here":  "web\n",
			}, filesIn(t, output))
		})

		t.Run("a command that fails stops the run, and the message names the folder, the command and its output", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    after_render: ["echo before >> log", "echo broken module >&2; exit 3", "echo after >> log"]
  web:
    path: frontend
`, output)

			err := selection.RunAfterRender(context.Background(), output)

			require.ErrorContains(t, err, "the after_render command")
			require.ErrorContains(t, err, "folder app")
			require.ErrorContains(t, err, "echo broken module >&2; exit 3")
			require.ErrorContains(t, err, "broken module")
			require.Equal(t, "before\n", filesIn(t, output)["log"])
		})

		t.Run("a program that is not on PATH stops the run, and the message names the folder, the command and the cause", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    after_render: ["no-such-program-of-feature-compose --flag"]
  web:
    path: frontend
`, output)

			err := selection.RunAfterRender(context.Background(), output)

			require.ErrorContains(t, err, "folder app")
			require.ErrorContains(t, err, "no-such-program-of-feature-compose --flag")
			require.ErrorContains(t, err, "not found")
		})

		t.Run("the test commands of a folder do not run", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    after_render: ["touch rendered"]
    test: ["touch tested"]
  web:
    path: frontend
`, output)

			err := selection.RunAfterRender(context.Background(), output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":             "package main\n",
				"rendered":            "",
				"frontend/index.html": "<html></html>\n",
			}, filesIn(t, output))
		})

		t.Run("runs each command in the repo when the tool runs from another folder", func(t *testing.T) {
			workingFolder := t.TempDir()
			t.Chdir(workingFolder)
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    after_render: ["touch marker"]
  web:
    path: frontend
`, "my-service")

			err := selection.RunAfterRender(context.Background(), "my-service")

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"my-service/main.go":             "package main\n",
				"my-service/marker":              "",
				"my-service/frontend/index.html": "<html></html>\n",
			}, filesIn(t, workingFolder))
		})

		t.Run("a context that ends stops the command and its children, and the run returns an error", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    after_render: ["sh -c 'sleep 1; touch finished'; true"]
  web:
    path: frontend
`, output)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			err := selection.RunAfterRender(ctx, output)

			require.Error(t, err)
			time.Sleep(2 * time.Second)
			require.NoFileExists(t, filepath.Join(output, "finished"))
		})
	})

	t.Run("added by needs", func(t *testing.T) {
		selectionWith := func(t *testing.T, definitions map[string]string, given ...string) catalog.Selection {
			t.Helper()
			return featureSelection(t, withFeatureDefinitions(definitions), given...)
		}

		t.Run("gives each feature that a need added, each after its needs and otherwise in the order of the names, with the selected features that need it", func(t *testing.T) {
			selection := selectionWith(t, map[string]string{"x": "needs: [z, y]\n", "y": "needs: [z]\n", "z": ""}, "x")

			require.Equal(t, []catalog.Need{{Feature: "z", NeededBy: []string{"x", "y"}}, {Feature: "y", NeededBy: []string{"x"}}}, selection.AddedByNeeds())
		})

		t.Run("a need of a need names the feature that needs it", func(t *testing.T) {
			selection := selectionWith(t, map[string]string{"a": "needs: [b]\n", "b": "needs: [c]\n", "c": ""}, "a")

			require.Equal(t, []catalog.Need{{Feature: "c", NeededBy: []string{"b"}}, {Feature: "b", NeededBy: []string{"a"}}}, selection.AddedByNeeds())
		})

		t.Run("a feature that a flag gives is not in the list, also when a selected feature needs it", func(t *testing.T) {
			selection := selectionWith(t, map[string]string{"a": "needs: [b]\n", "b": "needs: [c]\n", "c": ""}, "a", "b")

			require.Equal(t, []catalog.Need{{Feature: "c", NeededBy: []string{"b"}}}, selection.AddedByNeeds())
		})

		t.Run("a feature that an answer selects is not in the list, also when a selected feature needs it", func(t *testing.T) {
			starter, err := catalog.New(withFeatureDefinitions(map[string]string{"a": "needs: [b]\n", "b": "needs: [c]\n", "c": ""})).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, nil, catalog.Answers{Features: []string{"a", "b"}})

			require.NoError(t, err)
			require.Equal(t, []catalog.Need{{Feature: "c", NeededBy: []string{"b"}}}, selection.AddedByNeeds())
		})

		t.Run("a change to the list that it gives does not change the next list", func(t *testing.T) {
			selection := selectionWith(t, map[string]string{"x": "needs: [y]\n", "w": "needs: [y]\n", "y": ""}, "x", "w")

			given := selection.AddedByNeeds()
			given[0].NeededBy[0] = "changed"
			given[0].Feature = "changed"

			require.Equal(t, []catalog.Need{{Feature: "y", NeededBy: []string{"w", "x"}}}, selection.AddedByNeeds())
		})

		t.Run("a selection with no need gives no feature", func(t *testing.T) {
			selection := selectionWith(t, map[string]string{"a": "", "b": ""}, "a", "b")

			require.Empty(t, selection.AddedByNeeds())
		})
	})

	t.Run("run tests", func(t *testing.T) {
		t.Run("runs the test commands of each folder in that folder of the repo, in their order, with the folders in the order of their names", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    test: ["echo app-one >> order", "echo app-two >> order"]
  web:
    path: frontend
    test: ["echo web >> ../order", "echo web > tested-here"]
`, output)

			err := selection.RunTests(context.Background(), output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":              "package main\n",
				"order":                "app-one\napp-two\nweb\n",
				"frontend/index.html":  "<html></html>\n",
				"frontend/tested-here": "web\n",
			}, filesIn(t, output))
		})

		t.Run("the after_render commands of a folder do not run", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    after_render: ["touch rendered"]
    test: ["touch tested"]
  web:
    path: frontend
`, output)

			err := selection.RunTests(context.Background(), output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":             "package main\n",
				"tested":              "",
				"frontend/index.html": "<html></html>\n",
			}, filesIn(t, output))
		})

		t.Run("a test command that fails stops the run, and the message says that a test command failed, with the folder, the command and its output", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    test: ["echo before >> log", "echo FAIL TestAccounts >&2; exit 1", "echo after >> log"]
  web:
    path: frontend
    test: ["echo web >> ../log"]
`, output)

			err := selection.RunTests(context.Background(), output)

			require.ErrorContains(t, err, "the test command")
			require.ErrorContains(t, err, "folder app")
			require.ErrorContains(t, err, "echo FAIL TestAccounts >&2; exit 1")
			require.ErrorContains(t, err, "FAIL TestAccounts")
			require.Equal(t, "before\n", filesIn(t, output)["log"])
		})

		t.Run("a program that is not on PATH stops the run, and the message names the folder, the command and the cause", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    test: ["no-such-program-of-feature-compose --flag"]
  web:
    path: frontend
`, output)

			err := selection.RunTests(context.Background(), output)

			require.ErrorContains(t, err, "the test command")
			require.ErrorContains(t, err, "folder app")
			require.ErrorContains(t, err, "no-such-program-of-feature-compose --flag")
			require.ErrorContains(t, err, "not found")
		})

		t.Run("folders with no test command run nothing", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    after_render: ["touch rendered"]
  web:
    path: frontend
`, output)

			err := selection.RunTests(context.Background(), output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"main.go":             "package main\n",
				"frontend/index.html": "<html></html>\n",
			}, filesIn(t, output))
		})

		t.Run("a context that ends stops the command and its children, and the run returns an error", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := filepath.Join(t.TempDir(), "my-service")
			selection := writtenRepo(t, `
folders:
  app:
    path: .
    test: ["sh -c 'sleep 1; touch finished'; true"]
  web:
    path: frontend
`, output)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			err := selection.RunTests(ctx, output)

			require.Error(t, err)
			time.Sleep(2 * time.Second)
			require.NoFileExists(t, filepath.Join(output, "finished"))
		})
	})

	t.Run("write and preview", func(t *testing.T) {
		withWhen := func(when string) fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":          &fstest.MapFile{Data: []byte("options:\n  region:\n    default: eu\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go":     &fstest.MapFile{Data: []byte("package main\n")},
				"features/need/feature.yml":             &fstest.MapFile{Data: []byte("summary: Need.\ntargets: [service]\noptions:\n  tier:\n    default: gold\n  v2_mode:\n    default: \"on\"\n")},
				"features/x/feature.yml":                &fstest.MapFile{Data: []byte("summary: X.\ntargets: [service]\nneeds: [need]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  store-key:\n    default: \"false\"\nfiles:\n  - path: files/always/\n  - path: files/when/\n    when: '" + when + "'\n")},
				"features/x/files/always/go/always.txt": &fstest.MapFile{Data: []byte("always\n")},
				"features/x/files/when/go/when.txt":     &fstest.MapFile{Data: []byte("when\n")},
				"features/x/files/when/go/when2.txt":    &fstest.MapFile{Data: []byte("when 2\n")},
			}
		}
		writtenAndPreviewed := func(t *testing.T, files fstest.MapFS, givenByFeature map[string]map[string]string, answers catalog.Answers) []string {
			t.Helper()
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, givenByFeature, answers)
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "repo")
			require.NoError(t, selection.MakeRepo(context.Background(), output, "v1.2.3", catalog.NoSteps{}))
			preview, err := selection.Preview()
			require.NoError(t, err)
			return append(slices.Sorted(maps.Keys(filesIn(t, output))), strings.Split(strings.TrimSpace(preview[strings.Index(preview, "Files:\n"):]), "\n  ")[1:]...)
		}
		writtenOnlyAlways := []string{".feature-compose.yml", "always.txt", "main.go", ".feature-compose.yml: feature-compose", "always.txt: the feature x", "main.go: the starter"}
		writtenWithWhen := []string{".feature-compose.yml", "always.txt", "main.go", "when.txt", "when2.txt", ".feature-compose.yml: feature-compose", "always.txt: the feature x", "main.go: the starter", "when.txt: the feature x", "when2.txt: the feature x"}

		for name, test := range map[string]struct {
			when    string
			values  map[string]string
			written bool
		}{
			"== holds when the option has the value":                 {`method == "oauth"`, map[string]string{"method": "oauth"}, true},
			"== does not hold for another value":                     {`method == "oauth"`, nil, false},
			"!= holds for another value":                             {`method != "oauth"`, nil, true},
			"!= does not hold when the option has the value":         {`method != "oauth"`, map[string]string{"method": "oauth"}, false},
			"a bare name holds for the value true":                   {`store-key`, map[string]string{"store-key": "true"}, true},
			"a bare name does not hold for another value":            {`store-key`, map[string]string{"store-key": "yes"}, false},
			"! turns the result around":                              {`!store-key`, nil, true},
			"! does not hold when its operand holds":                 {`!store-key`, map[string]string{"store-key": "true"}, false},
			"&& holds when both sides hold":                          {`store-key && method == "oauth"`, map[string]string{"store-key": "true", "method": "oauth"}, true},
			"&& does not hold when its right side does not hold":     {`store-key && method == "oauth"`, map[string]string{"store-key": "true"}, false},
			"&& does not hold when its left side does not hold":      {`store-key && method == "oauth"`, map[string]string{"method": "oauth"}, false},
			"|| holds when one side holds":                           {`store-key || method == "oauth"`, map[string]string{"method": "oauth"}, true},
			"|| does not hold when no side holds":                    {`store-key || method == "oauth"`, nil, false},
			"&& comes before ||":                                     {`store-key || method != "oauth" && method == "oauth"`, map[string]string{"store-key": "true"}, true},
			"! comes before &&":                                      {`!store-key && method == "oauth"`, nil, false},
			"parentheses group":                                      {`(store-key || method == "oauth") && method == "api-key"`, map[string]string{"store-key": "true", "method": "oauth"}, false},
			"spaces between the parts do not change the result":      {`  ( store-key&&method=="oauth" )  `, map[string]string{"store-key": "true", "method": "oauth"}, true},
			"a name of an option of a need is <need>.<option>":       {`need.tier == "gold"`, nil, true},
			"a name can hold digits and _":                           {`need.v2_mode == "on"`, nil, true},
			"a value can be the empty text":                          {`method != ""`, nil, true},
			"a name of an option of the starter is starter.<option>": {`starter.region != "eu"`, nil, false},
		} {
			t.Run("the files of an entry are written and previewed only while its when holds: "+name, func(t *testing.T) {
				expected := writtenOnlyAlways
				if test.written {
					expected = writtenWithWhen
				}

				require.Equal(t, expected, writtenAndPreviewed(t, withWhen(test.when), map[string]map[string]string{"x": test.values}, catalog.Answers{}))
			})
		}

		for name, block := range map[string]string{
			"a folded block":               "    when: >\n      method == \"oauth\"\n",
			"a literal block on two lines": "    when: |-\n      method == \"oauth\" &&\n      store-key\n",
		} {
			t.Run("a when written as "+name+" of YAML holds as the same when on one line", func(t *testing.T) {
				files := withWhen("x")
				definition := string(files["features/x/feature.yml"].Data)
				files["features/x/feature.yml"] = &fstest.MapFile{Data: []byte(strings.Replace(definition, "    when: 'x'\n", block, 1))}

				require.Equal(t, writtenWithWhen, writtenAndPreviewed(t, files, map[string]map[string]string{"x": {"method": "oauth", "store-key": "true"}}, catalog.Answers{}))
			})
		}

		t.Run("a when reads the value that a need sets and the value that an answer gives", func(t *testing.T) {
			files := withWhen(`method == "oauth"`)
			files["features/y/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Y.\ntargets: [service]\nneeds:\n  x:\n    method: oauth\n")}

			fromNeed := writtenAndPreviewed(t, files, map[string]map[string]string{"y": {}}, catalog.Answers{})
			fromAnswer := writtenAndPreviewed(t, files, nil, catalog.Answers{Features: []string{"x"}, FeatureValues: map[string]map[string]string{"x": {"method": "oauth"}}})

			require.Equal(t, [][]string{writtenWithWhen, writtenWithWhen}, [][]string{fromNeed, fromAnswer})
		})

		t.Run("two entries whose whens never hold together each write one path, and when both hold the write stops with the message of two templates that write one path", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":         &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go":    &fstest.MapFile{Data: []byte("package main\n")},
				"features/x/feature.yml":               &fstest.MapFile{Data: []byte("summary: X.\ntargets: [service]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  extra:\n    default: \"false\"\nfiles:\n  - path: files/api-key/\n    when: method == \"api-key\"\n  - path: files/oauth/\n    when: method == \"oauth\"\n  - path: files/extra/\n    when: extra\n")},
				"features/x/files/api-key/go/auth.txt": &fstest.MapFile{Data: []byte("api-key")},
				"features/x/files/oauth/go/auth.txt":   &fstest.MapFile{Data: []byte("oauth")},
				"features/x/files/extra/go/auth.txt":   &fstest.MapFile{Data: []byte("extra")},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			writtenAuth := func(values map[string]string) string {
				selection, err := starter.Select("my-service", nil, map[string]map[string]string{"x": values}, catalog.Answers{})
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "repo")
				require.NoError(t, selection.Write(output))
				return filesIn(t, output)["auth.txt"]
			}
			both, err := starter.Select("my-service", nil, map[string]map[string]string{"x": {"extra": "true"}}, catalog.Answers{})
			require.NoError(t, err)

			_, previewErr := both.Preview()

			require.Equal(t, []string{"api-key", "oauth"}, []string{writtenAuth(nil), writtenAuth(map[string]string{"method": "oauth"})})
			require.ErrorContains(t, previewErr, "the template go/auth.txt of the files entry files/api-key/ of the feature x and the template go/auth.txt of the files entry files/extra/ of the feature x both write auth.txt")
		})

		for name, test := range map[string]struct {
			path    string
			message string
		}{
			"a template of the starter": {
				path:    "starters/service/files/go/.feature-compose.yml",
				message: "the template go/.feature-compose.yml of the starter and feature-compose both write .feature-compose.yml",
			},
			"a template of a feature whose path renders to it": {
				path:    `features/x/files/always/go/{{ ".feature-compose" }}.yml`,
				message: `the template go/{{ ".feature-compose" }}.yml of the files entry files/always/ of the feature x and feature-compose both write .feature-compose.yml`,
			},
		} {
			t.Run(name+" that writes .feature-compose.yml at the root of the repo stops Write and Preview before the first file, with the message of two owners", func(t *testing.T) {
				files := withWhen("store-key")
				files[test.path] = &fstest.MapFile{Data: []byte("version: mine\n")}
				starter, err := catalog.New(files).Starter("service")
				require.NoError(t, err)
				selection, err := starter.Select("my-service", nil, map[string]map[string]string{"x": {}}, catalog.Answers{})
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "repo")

				writeErr := selection.Write(output)
				_, previewErr := selection.Preview()

				for _, err := range []error{writeErr, previewErr} {
					require.EqualError(t, err, test.message)
				}
				require.NoDirExists(t, output)
			})
		}

		t.Run("a template that writes .feature-compose.yml in a folder below the root of the repo is written and previewed as any other file", func(t *testing.T) {
			files := withWhen("store-key")
			files["features/x/files/always/go/sub/.feature-compose.yml"] = &fstest.MapFile{Data: []byte("version: mine\n")}

			require.Equal(t, []string{
				".feature-compose.yml", "always.txt", "main.go", "sub/.feature-compose.yml",
				".feature-compose.yml: feature-compose", "always.txt: the feature x", "main.go: the starter", "sub/.feature-compose.yml: the feature x",
			}, writtenAndPreviewed(t, files, map[string]map[string]string{"x": {}}, catalog.Answers{}))
		})

		for _, when := range []string{`mode == "x"`, `!mode`, `mode && store-key`, `store-key || mode`} {
			t.Run("a when with a name that the feature cannot read stops Write and Preview before the first file, and the message names the when, the feature and the names that it can read: "+when, func(t *testing.T) {
				starter, err := catalog.New(withWhen(when)).Starter("service")
				require.NoError(t, err)
				selection, err := starter.Select("my-service", nil, map[string]map[string]string{"x": {"store-key": "true"}}, catalog.Answers{})
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "repo")

				writeErr := selection.Write(output)
				_, previewErr := selection.Preview()

				for _, err := range []error{writeErr, previewErr} {
					require.ErrorContains(t, err, strconv.Quote(when)+` of the feature x: unknown name "mode": the names that it can read are method, need.tier, need.v2_mode, starter.region, store-key`)
				}
				require.NoDirExists(t, output)
			})
		}
	})

	t.Run("needed by", func(t *testing.T) {
		t.Run("gives the selected features that need a feature, in the order of their names", func(t *testing.T) {
			require.Equal(t, []string{"x", "y"}, featureSelection(t, withTwoNeeds(), "y", "x").NeededBy("b"))
		})

		t.Run("gives none for a feature that no selected feature needs, and for a feature that the selection does not hold", func(t *testing.T) {
			selection := featureSelection(t, withTwoNeeds(), "x")

			require.Equal(t, [][]string{{"x"}, nil, nil}, [][]string{selection.NeededBy("b"), selection.NeededBy("x"), selection.NeededBy("c")})
		})
	})

	t.Run("holds", func(t *testing.T) {
		t.Run("the selection holds each given feature and each feature that it needs, and no other feature", func(t *testing.T) {
			selection := featureSelection(t, withTwoNeeds(), "x")

			require.Equal(t, []bool{true, true, false, false}, []bool{selection.Holds("x"), selection.Holds("b"), selection.Holds("y"), selection.Holds("c")})
		})
	})

	t.Run("feature source", func(t *testing.T) {
		t.Run("the source of each feature has the text of the preview, and a feature that the selection does not hold has none", func(t *testing.T) {
			starter, err := catalog.New(withTwoNeeds()).Starter("service")
			require.NoError(t, err)
			given, err := starter.Select("my-service", nil, withoutValues("x", "b"), catalog.Answers{Features: []string{"y"}})
			require.NoError(t, err)
			answered, err := starter.Select("my-service", nil, withoutValues("x"), catalog.Answers{Features: []string{"y", "b", "c"}})
			require.NoError(t, err)

			require.Equal(t,
				[]string{"command line", "command line, needed by x, y", "answered", "answered, needed by x, y", "answered", ""},
				[]string{given.FeatureSource("x"), given.FeatureSource("b"), given.FeatureSource("y"), answered.FeatureSource("b"), answered.FeatureSource("c"), given.FeatureSource("c")})
		})
	})
}

func namedStarter(t *testing.T) catalog.Starter {
	t.Helper()
	fsys := fstest.MapFS{
		"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  app:\n    path: .\n")},
		"starters/service/files/app/go.mod": &fstest.MapFile{Data: []byte("module {{ .Name }}\n")},
		"features/cli/feature.yml":          &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\n")},
	}
	starter, err := catalog.NewCatalogs(catalog.FromOrigin(fsys, catalog.Origin{Name: "hpcsc", Source: "https://example.com/hpcsc.git", Ref: "main", Commit: "0123abc"})).Starter("service")
	require.NoError(t, err)
	return starter
}

type recordedSteps struct {
	lines []string
}

func (r *recordedSteps) Start(label string) func(error) {
	r.lines = append(r.lines, "start "+label)
	return func(err error) {
		if err != nil {
			r.lines = append(r.lines, "fail "+label)
			return
		}
		r.lines = append(r.lines, "end "+label)
	}
}

func generatorSelection(t *testing.T, catalogFiles map[string]string, featureNames ...string) catalog.Selection {
	t.Helper()
	fsys := fstest.MapFS{}
	for catalogPath, contents := range catalogFiles {
		fsys[catalogPath] = &fstest.MapFile{Data: []byte(contents)}
	}
	starter, err := catalog.New(fsys).Starter("service")
	require.NoError(t, err)
	givenByFeature := map[string]map[string]string{}
	for _, featureName := range featureNames {
		givenByFeature[featureName] = map[string]string{}
	}
	selection, err := starter.Select("my-service", nil, givenByFeature, catalog.Answers{})
	require.NoError(t, err)
	return selection
}

func pathWithOnly(t *testing.T, tools ...string) string {
	t.Helper()
	bin := t.TempDir()
	for _, tool := range tools {
		found, err := exec.LookPath(tool)
		require.NoError(t, err)
		require.NoError(t, os.Symlink(found, filepath.Join(bin, tool)))
	}
	return bin
}

func linkTargetsIn(t *testing.T, dir string) []string {
	t.Helper()
	var targets []string
	err := filepath.WalkDir(dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.Type()&fs.ModeSymlink == 0 {
			return err
		}
		target, err := os.Readlink(name)
		targets = append(targets, target)
		return err
	})
	require.NoError(t, err)
	return targets
}

func requireMise(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("mise"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("mise is not on the PATH: CI installs it")
		}
		t.Skip("mise is not on the PATH")
	}
}

func requireASTGrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ast-grep"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("ast-grep is not on the PATH: mise.toml installs it in CI")
		}
		t.Skip("ast-grep is not on the PATH")
	}
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

func modeOf(t *testing.T, path string) fs.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info.Mode().Perm()
}

func umask() fs.FileMode {
	current := syscall.Umask(0)
	syscall.Umask(current)
	return fs.FileMode(current)
}
