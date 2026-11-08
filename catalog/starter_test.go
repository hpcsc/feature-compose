//go:build unit

package catalog_test

import (
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/catalog"
	"github.com/stretchr/testify/require"
)

func TestStarter(t *testing.T) {
	starterWith := func(t *testing.T, templates map[string]string) catalog.Starter {
		t.Helper()
		files := fstest.MapFS{"starters/service/starter.yml": &fstest.MapFile{Data: []byte(`
options:
  go-version:
    default: "1.27.1"
    pattern: '^1\.(2[7-9]|[3-9][0-9])(\.[0-9]+)?$'
  module:
    default: github.com/hpcsc/{{ .Name }}
  app:
    default: api
folders:
  go:
    path: .
`)}}
		for path, contents := range templates {
			files["starters/service/files/"+path] = &fstest.MapFile{Data: []byte(contents)}
		}
		starter, err := catalog.New(files).Starter("service")
		require.NoError(t, err)
		return starter
	}
	written := func(t *testing.T, selection catalog.Selection) map[string]string {
		t.Helper()
		output := filepath.Join(t.TempDir(), "repo")
		require.NoError(t, selection.Write(output))
		return filesIn(t, output)
	}

	t.Run("select", func(t *testing.T) {
		t.Run("an option with no given value takes its default, and the default renders with the name", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `module {{ option "starter.module" }}`})

			selection, err := starter.Select("my-service", nil)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module github.com/hpcsc/my-service"}, written(t, selection))
		})

		t.Run("a given value replaces the default", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `module {{ option "starter.module" }}`})

			selection, err := starter.Select("my-service", map[string]string{"module": "example.com/acme/ledger"})

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module example.com/acme/ledger"}, written(t, selection))
		})

		t.Run("option gives the value of each option in the contents and the paths of templates", func(t *testing.T) {
			starter := starterWith(t, map[string]string{
				`go/cmd/{{ option "starter.app" }}/main.go`: `// {{ option "starter.module" }} {{ option "starter.go-version" }}`,
			})

			selection, err := starter.Select("my-service", map[string]string{"app": "catchup", "go-version": "1.28.0"})

			require.NoError(t, err)
			require.Equal(t, map[string]string{"cmd/catchup/main.go": "// github.com/hpcsc/my-service 1.28.0"}, written(t, selection))
		})

		t.Run("a given value for an unknown option stops, and the message lists the options of the starter", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `module {{ option "starter.module" }}`})

			_, err := starter.Select("my-service", map[string]string{"modul": "example.com/acme/ledger"})

			require.ErrorContains(t, err, `"modul"`)
			require.ErrorContains(t, err, "app, go-version, module")
		})

		t.Run("a value that does not match the pattern of its option stops, and the message names the option and the pattern", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `go {{ option "starter.go-version" }}`})

			_, err := starter.Select("my-service", map[string]string{"go-version": "1.20"})

			require.ErrorContains(t, err, "go-version")
			require.ErrorContains(t, err, `^1\.(2[7-9]|[3-9][0-9])(\.[0-9]+)?$`)
		})

		t.Run("each given value is checked, also after a value that matches", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `go {{ option "starter.go-version" }}`})

			_, err := starter.Select("my-service", map[string]string{"app": "catchup", "go-version": "1.20"})

			require.ErrorContains(t, err, `"1.20"`)
		})

		t.Run("an option without a pattern accepts any text", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/app": `{{ option "starter.app" }}`})

			selection, err := starter.Select("my-service", map[string]string{"app": "any text, at all!"})

			require.NoError(t, err)
			require.Equal(t, map[string]string{"app": "any text, at all!"}, written(t, selection))
		})

		t.Run("a default that fails to render stops, and the message names the option", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":     &fstest.MapFile{Data: []byte("options:\n  module:\n    default: example.com/{{ .Nmae }}\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/go.mod": &fstest.MapFile{Data: []byte(`module {{ option "starter.module" }}`)},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil)

			require.ErrorContains(t, err, "module")
			require.ErrorContains(t, err, "Nmae")
		})
	})

	t.Run("write", func(t *testing.T) {
		for _, name := range []string{"starter.nope", "module"} {
			t.Run("option with the name "+name+" stops the write before the first file, and the message names the template and the name", func(t *testing.T) {
				starter := starterWith(t, map[string]string{
					"go/a.go": "package a\n",
					"go/b.go": `// {{ option "` + name + `" }}`,
				})
				selection, err := starter.Select("my-service", nil)
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "repo")

				err = selection.Write(output)

				require.ErrorContains(t, err, "go/b.go")
				require.ErrorContains(t, err, `"`+name+`"`)
				require.NoDirExists(t, output)
			})
		}
	})
}
