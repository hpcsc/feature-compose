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

		t.Run("gives the name of the repo, its starter, and each file by its path in the repo, in the order of the paths, with its owner", func(t *testing.T) {
			preview, err := featureSelection(t, withFeature(), "cli").Preview()

			require.NoError(t, err)
			require.Equal(t, `Repo: my-service
Starter: service

Features:
  cli: command line

Options: none

Files:
  cmd/my-service/main.go: the feature cli
  frontend/index.html: the starter
  go.mod: the starter
`, preview)
		})

		t.Run("gives the path of each file that the write of the same selection writes, one time each", func(t *testing.T) {
			selection := featureSelection(t, withFeature(), "cli")
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
			require.True(t, strings.HasSuffix(preview, "\nFiles:\n  notes: the starter\n  notes.md: the starter\n"), preview)
		})

		t.Run("a starter with no template gives a repo with no file", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml": oneFolder,
				"starters/service/files":       &fstest.MapFile{Mode: fs.ModeDir | 0o755},
			}

			preview, err := featureSelection(t, files).Preview()

			require.NoError(t, err)
			require.Equal(t, "Repo: my-service\nStarter: service\n\nFeatures: none\n\nOptions: none\n\nFiles: none\n", preview)
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

Files:
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
