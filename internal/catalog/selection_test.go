//go:build unit

package catalog_test

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
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
)

func TestSelection(t *testing.T) {
	selectionOf := func(t *testing.T, files fstest.MapFS, name string) catalog.Selection {
		t.Helper()
		starter, err := catalog.New(files).Starter("service")
		require.NoError(t, err)
		selection, err := starter.Select(name, nil, nil)
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

	t.Run("write", func(t *testing.T) {
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

		featureSelection := func(t *testing.T, files fstest.MapFS, features ...string) catalog.Selection {
			t.Helper()
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues(features...))
			require.NoError(t, err)
			return selection
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
			require.ErrorContains(t, err, "the template go/main.go.tmpl of the feature cli")
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
			selection, err := starter.Select("my-service", map[string]string{"module": module}, nil)
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
			selection, err := starter.Select("my-service", map[string]string{"module": module}, nil)
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
	})

	t.Run("preview", func(t *testing.T) {
		withFeature := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":                       &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n  web:\n    path: frontend\n")},
				"starters/service/files/go/go.mod.tmpl":              &fstest.MapFile{Data: []byte("module {{ .Name }}\n")},
				"starters/service/files/web/index.html":              &fstest.MapFile{Data: []byte("<html></html>\n")},
				"features/cli/feature.yml":                           &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
				"features/cli/files/go/cmd/{{ .Name }}/main.go.tmpl": &fstest.MapFile{Data: []byte("package main\n")},
			}
		}
		selectionWith := func(t *testing.T, files fstest.MapFS, features ...string) catalog.Selection {
			t.Helper()
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues(features...))
			require.NoError(t, err)
			return selection
		}

		t.Run("gives the name of the repo, its starter, and each file by its path in the repo, in the order of the paths, with its owner", func(t *testing.T) {
			preview, err := selectionWith(t, withFeature(), "cli").Preview()

			require.NoError(t, err)
			require.Equal(t, `Repo: my-service
Starter: service

Features:
  cli: command line

Files:
  cmd/my-service/main.go: the feature cli
  frontend/index.html: the starter
  go.mod: the starter
`, preview)
		})

		t.Run("gives the path of each file that the write of the same selection writes, one time each", func(t *testing.T) {
			selection := selectionWith(t, withFeature(), "cli")
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))

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

		t.Run("a starter with no template gives a repo with no file", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml": oneFolder,
				"starters/service/files":       &fstest.MapFile{Mode: fs.ModeDir | 0o755},
			}

			preview, err := selectionWith(t, files).Preview()

			require.NoError(t, err)
			require.Equal(t, "Repo: my-service\nStarter: service\n\nFeatures: none\n\nFiles: none\n", preview)
		})

		withNeeds := func(definitions map[string]string) fstest.MapFS {
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			for featureName, definition := range definitions {
				files["features/"+featureName+"/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A feature.\ntargets: [service]\n" + definition)}
			}
			return files
		}

		t.Run("gives each feature after its needs and otherwise in the order of the names, a given feature with the source command line, and a feature that a need added with the features that need it", func(t *testing.T) {
			files := withNeeds(map[string]string{"x": "needs: [z, y]\n", "y": "needs: [z]\n", "z": "", "w": ""})

			preview, err := selectionWith(t, files, "x", "w").Preview()

			require.NoError(t, err)
			require.Equal(t, `Repo: my-service
Starter: service

Features:
  w: command line
  z: needed by x, y
  y: needed by x
  x: command line

Files:
  main.go: the starter
`, preview)
		})

		t.Run("a given feature shows the source command line, also when a selected feature needs it", func(t *testing.T) {
			files := withNeeds(map[string]string{"a": "needs: [b]\n", "b": "needs: [c]\n", "c": ""})

			preview, err := selectionWith(t, files, "a", "b").Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "Features:\n  c: needed by b\n  b: command line\n  a: command line\n\n")
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
		} {
			t.Run(name+" stops the preview with the message that stops the write of the same selection", func(t *testing.T) {
				selection := selectionWith(t, test.files, test.features...)
				writeErr := selection.Write(filepath.Join(t.TempDir(), "my-service"))
				require.Error(t, writeErr)

				preview, err := selection.Preview()

				require.EqualError(t, err, writeErr.Error())
				require.Empty(t, preview)
			})
		}
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
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			for featureName, definition := range definitions {
				files["features/"+featureName+"/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A feature.\ntargets: [service]\n" + definition)}
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues(given...))
			require.NoError(t, err)
			return selection
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
