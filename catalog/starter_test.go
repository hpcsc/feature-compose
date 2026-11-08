//go:build unit

package catalog_test

import (
	"path/filepath"
	"strconv"
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

			selection, err := starter.Select("my-service", nil, nil)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module github.com/hpcsc/my-service"}, written(t, selection))
		})

		t.Run("a given value replaces the default", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `module {{ option "starter.module" }}`})

			selection, err := starter.Select("my-service", map[string]string{"module": "example.com/acme/ledger"}, nil)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module example.com/acme/ledger"}, written(t, selection))
		})

		t.Run("option gives the value of each option in the contents and the paths of templates", func(t *testing.T) {
			starter := starterWith(t, map[string]string{
				`go/cmd/{{ option "starter.app" }}/main.go`: `// {{ option "starter.module" }} {{ option "starter.go-version" }}`,
			})

			selection, err := starter.Select("my-service", map[string]string{"app": "catchup", "go-version": "1.28.0"}, nil)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"cmd/catchup/main.go": "// github.com/hpcsc/my-service 1.28.0"}, written(t, selection))
		})

		t.Run("a given value for an unknown option stops, and the message lists the options of the starter", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `module {{ option "starter.module" }}`})

			_, err := starter.Select("my-service", map[string]string{"modul": "example.com/acme/ledger"}, nil)

			require.ErrorContains(t, err, `"modul"`)
			require.ErrorContains(t, err, "app, go-version, module")
		})

		t.Run("a value that does not match the pattern of its option stops, and the message names the option and the pattern", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `go {{ option "starter.go-version" }}`})

			_, err := starter.Select("my-service", map[string]string{"go-version": "1.20"}, nil)

			require.ErrorContains(t, err, "go-version")
			require.ErrorContains(t, err, `^1\.(2[7-9]|[3-9][0-9])(\.[0-9]+)?$`)
		})

		t.Run("each given value is checked, also after a value that matches", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/go.mod": `go {{ option "starter.go-version" }}`})

			_, err := starter.Select("my-service", map[string]string{"app": "catchup", "go-version": "1.20"}, nil)

			require.ErrorContains(t, err, `"1.20"`)
		})

		for _, name := range []string{"", "../escape", "team/service", "my service", "my-service;rm -rf ~", "-flag", ".hidden"} {
			t.Run("the name "+strconv.Quote(name)+" stops, and the message names the name", func(t *testing.T) {
				starter := starterWith(t, map[string]string{"go/go.mod": `module {{ option "starter.module" }}`})

				_, err := starter.Select(name, nil, nil)

				require.ErrorContains(t, err, "the name "+strconv.Quote(name)+" of the repo")
			})
		}

		t.Run("a name of letters, digits, dots, underscores and hyphens is accepted", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/name": "{{ .Name }}"})

			selection, err := starter.Select("Ledger_2.api-v1", nil, nil)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"name": "Ledger_2.api-v1"}, written(t, selection))
		})

		t.Run("an option without a pattern accepts any text", func(t *testing.T) {
			starter := starterWith(t, map[string]string{"go/app": `{{ option "starter.app" }}`})

			selection, err := starter.Select("my-service", map[string]string{"app": "any text, at all!"}, nil)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"app": "any text, at all!"}, written(t, selection))
		})

		withFeatures := fstest.MapFS{
			"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
			"starters/service/files/go/README.md": &fstest.MapFile{Data: []byte("service\n")},
			"starters/other/starter.yml":          &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
			"starters/other/files/go/README.md":   &fstest.MapFile{Data: []byte("other\n")},
			"features/cli/feature.yml":            &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
			"features/cli/files/go/cli.go":        &fstest.MapFile{Data: []byte("package cli\n")},
			"features/docs/feature.yml":           &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service, other]\nfiles:\n  - path: files/\n")},
			"features/docs/files/go/a.md":         &fstest.MapFile{Data: []byte("a\n")},
			"features/web/feature.yml":            &fstest.MapFile{Data: []byte("summary: Web.\ntargets: [service]\nfiles:\n  - path: files/\n")},
			"features/web/files/web/index.html":   &fstest.MapFile{Data: []byte("<html></html>\n")},
			"features/elsewhere/feature.yml":      &fstest.MapFile{Data: []byte("summary: Elsewhere.\ntargets: [other]\nfiles:\n  - path: files/\n")},
			"features/elsewhere/files/go/e.go":    &fstest.MapFile{Data: []byte("package e\n")},
		}

		for name, features := range map[string][]string{
			"a feature that does not target the starter": {"elsewhere"},
			"an unknown feature, also with a known one":  {"cli", "no-such-feature"},
		} {
			t.Run(name+" stops the selection, and the message lists the features that target the starter", func(t *testing.T) {
				starter, err := catalog.New(withFeatures).Starter("service")
				require.NoError(t, err)

				_, err = starter.Select("my-service", nil, features)

				require.ErrorContains(t, err, strconv.Quote(features[len(features)-1]))
				require.ErrorContains(t, err, "the features that target the starter service are cli, docs, web")
			})
		}

		t.Run("a feature for a starter that no feature targets stops, and the message says so", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/lonely/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/lonely/files/go/README.md": &fstest.MapFile{Data: []byte("lonely\n")},
				"features/cli/feature.yml":           &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
			}
			starter, err := catalog.New(files).Starter("lonely")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, []string{"cli"})

			require.ErrorContains(t, err, "no feature targets the starter lonely")
		})

		t.Run("a selected feature with a template outside the folders of the starter stops, and the message names the feature, the template and the folders", func(t *testing.T) {
			starter, err := catalog.New(withFeatures).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, []string{"web"})

			require.ErrorContains(t, err, "the template web/index.html of the feature web")
			require.ErrorContains(t, err, "the folders are go")
		})

		t.Run("a feature that is not selected does not stop the selection", func(t *testing.T) {
			starter, err := catalog.New(withFeatures).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, []string{"cli"})

			require.NoError(t, err)
			require.Equal(t, map[string]string{"README.md": "service\n", "cli.go": "package cli\n"}, written(t, selection))
		})

		t.Run("two selected features that declare one extension point stop, and the message names it and both features", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/service/files/go/README.md": &fstest.MapFile{Data: []byte("service\n")},
				"features/cli/feature.yml":            &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\nextension_points:\n  cli.commands:\n    file: go/a.go\n    item: [code]\n")},
				"features/cli/files/go/a.go":          &fstest.MapFile{Data: []byte("package a\n")},
				"features/tools/feature.yml":          &fstest.MapFile{Data: []byte("summary: Tools.\ntargets: [service]\nfiles:\n  - path: files/\nextension_points:\n  cli.commands:\n    file: go/b.go\n    item: [code]\n")},
				"features/tools/files/go/b.go":        &fstest.MapFile{Data: []byte("package b\n")},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, []string{"cli", "tools"})

			require.ErrorContains(t, err, `the features cli and tools both declare the extension point "cli.commands"`)
		})

		t.Run("a default that fails to render stops, and the message names the option", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":     &fstest.MapFile{Data: []byte("options:\n  module:\n    default: example.com/{{ .Nmae }}\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/go.mod": &fstest.MapFile{Data: []byte(`module {{ option "starter.module" }}`)},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, nil)

			require.ErrorContains(t, err, "module")
			require.ErrorContains(t, err, "Nmae")
		})
	})

}
