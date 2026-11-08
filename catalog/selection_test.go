//go:build unit

package catalog_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/catalog"
	"github.com/stretchr/testify/require"
)

func TestSelection(t *testing.T) {
	selectionOf := func(t *testing.T, files fstest.MapFS, name string) catalog.Selection {
		t.Helper()
		starter, err := catalog.New(files).Starter("service")
		require.NoError(t, err)
		return starter.Select(name)
	}
	oneFolder := &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")}

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

		for _, name := range []string{"order-event-store", "order_event_store", "orderEventStore", "OrderEventStore"} {
			t.Run("the case functions give one result for the name "+name, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":    oneFolder,
					"starters/service/files/go/names": &fstest.MapFile{Data: []byte("{{ kebab .Name }} {{ snake .Name }} {{ camel .Name }} {{ pascal .Name }}")},
				}
				output := filepath.Join(t.TempDir(), "repo")

				err := selectionOf(t, files, name).Write(output)

				require.NoError(t, err)
				require.Equal(t, map[string]string{"names": "order-event-store order_event_store orderEventStore OrderEventStore"}, filesIn(t, output))
			})
		}

		t.Run("an upper-case letter after another upper-case letter stays in the word", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":    oneFolder,
				"starters/service/files/go/names": &fstest.MapFile{Data: []byte("{{ kebab .Name }} {{ snake .Name }} {{ camel .Name }} {{ pascal .Name }}")},
			}
			output := filepath.Join(t.TempDir(), "repo")

			err := selectionOf(t, files, "OrderAPI").Write(output)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"names": "order-api order_api orderApi OrderApi"}, filesIn(t, output))
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

		t.Run("a written file cannot be run", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      oneFolder,
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			output := filepath.Join(t.TempDir(), "my-service")

			err := selectionOf(t, files, "my-service").Write(output)

			require.NoError(t, err)
			info, err := os.Stat(filepath.Join(output, "main.go"))
			require.NoError(t, err)
			require.Zero(t, info.Mode().Perm()&0o111)
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

		t.Run("a name that leads out of the output folder stops the write before the first file", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":                  oneFolder,
				"starters/service/files/go/a.go":                &fstest.MapFile{Data: []byte("package a\n")},
				"starters/service/files/go/{{ .Name }}/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}
			parent := t.TempDir()
			output := filepath.Join(parent, "work", "repo")

			err := selectionOf(t, files, "../../escape").Write(output)

			require.ErrorContains(t, err, "go/{{ .Name }}/main.go")
			require.Empty(t, filesIn(t, parent))
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
