//go:build unit

package catalog_test

import (
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/internal/catalog"
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

		t.Run("a value that is not one of the values of a starter option stops, and the message names the option and its values", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":     &fstest.MapFile{Data: []byte("options:\n  database:\n    values: [postgres, sqlite]\n    default: postgres\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/db.txt": &fstest.MapFile{Data: []byte(`{{ option "starter.database" }}`)},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", map[string]string{"database": "mysql"}, nil)

			require.ErrorContains(t, err, `the starter service: the value "mysql" of the option database is not one of its values: postgres, sqlite`)
		})

		t.Run("a given value for a starter with no option stops, and the message says that the starter has no option", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":    &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/service/files/go/a.txt": &fstest.MapFile{Data: []byte("a\n")},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", map[string]string{"module": "x"}, nil)

			require.ErrorContains(t, err, `unknown option "module": the starter service has no option`)
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

				_, err = starter.Select("my-service", nil, withoutValues(features...))

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

			_, err = starter.Select("my-service", nil, withoutValues("cli"))

			require.ErrorContains(t, err, "no feature targets the starter lonely")
		})

		t.Run("a selected feature with a template outside the folders of the starter stops, and the message names the feature, the template and the folders", func(t *testing.T) {
			starter, err := catalog.New(withFeatures).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, withoutValues("web"))

			require.ErrorContains(t, err, "the template web/index.html of the feature web")
			require.ErrorContains(t, err, "the folders are go")
		})

		t.Run("a feature that is not selected does not stop the selection", func(t *testing.T) {
			starter, err := catalog.New(withFeatures).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, withoutValues("cli"))

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

			_, err = starter.Select("my-service", nil, withoutValues("cli", "tools"))

			require.ErrorContains(t, err, `the feature cli and the feature tools both declare the extension point "cli.commands"`)
		})

		withExtensionPoints := func(addTo string) fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    summary: A middleware.\n    file: go/server.go\n    item: [code, import]\n")},
				"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte("package server\n")},
				"features/auth/feature.yml":           &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [service]\nadd_to:\n" + addTo)},
				"features/docs/feature.yml":           &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service]\nextension_points:\n  docs.pages:\n    summary: A page.\n    file: go/index.md\n    item: [path]\n")},
			}
		}

		for name, item := range map[string]string{
			"an item that lacks a field of its extension point":              "    - code: auth.Middleware\n",
			"an item with a field that its extension point does not declare": "    - code: auth.Middleware\n      import: internal/auth\n      order: first\n",
		} {
			t.Run(name+" stops, and the message names the feature, the extension point and the fields it declares", func(t *testing.T) {
				starter, err := catalog.New(withExtensionPoints("  api.middleware:\n" + item)).Starter("service")
				require.NoError(t, err)

				_, err = starter.Select("my-service", nil, withoutValues("auth"))

				require.ErrorContains(t, err, "the feature auth")
				require.ErrorContains(t, err, `"api.middleware"`)
				require.ErrorContains(t, err, "declares the fields [code, import]")
			})
		}

		t.Run("each item is checked, also after an item with the fields of its extension point, and the message gives the number of the item", func(t *testing.T) {
			starter, err := catalog.New(withExtensionPoints("  docs.pages:\n    - path: auth.md\n    - path: keys.md\n      title: Keys\n")).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, withoutValues("auth", "docs"))

			require.ErrorContains(t, err, `the item 2 that the feature auth adds to the extension point "docs.pages" has the fields [path, title], but the extension point declares the fields [path]`)
		})

		t.Run("a second selection of one starter holds none of the items that the features of the first selection add", func(t *testing.T) {
			files := withExtensionPoints("  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n")
			files["starters/service/files/go/server.go"] = &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }};{{ end }}`)}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			first, err := starter.Select("my-service", nil, withoutValues("auth"))
			require.NoError(t, err)

			second, err := starter.Select("my-service", nil, nil)

			require.NoError(t, err)
			require.Equal(t, map[string]string{"server.go": "auth.Middleware;"}, written(t, first))
			require.Equal(t, map[string]string{"server.go": ""}, written(t, second))
		})

		for name, extensionPoint := range map[string]string{
			"that nothing declares":                             "cli.commands",
			"that only a feature that is not selected declares": "docs.pages",
		} {
			t.Run("an add_to to an extension point "+name+" stops, and the message names the feature and the extension point", func(t *testing.T) {
				starter, err := catalog.New(withExtensionPoints("  " + extensionPoint + ":\n    - path: auth.md\n")).Starter("service")
				require.NoError(t, err)

				_, err = starter.Select("my-service", nil, withoutValues("auth"))

				require.ErrorContains(t, err, `the feature auth adds to the extension point "`+extensionPoint+`", which neither the starter nor a selected feature declares`)
			})
		}

		t.Run("the starter and a selected feature that declare one extension point stop, and the message names it, the starter and the feature", func(t *testing.T) {
			files := withExtensionPoints("  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n")
			files["features/proxy/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Proxy.\ntargets: [service]\nextension_points:\n  api.middleware:\n    summary: A middleware.\n    file: go/proxy.go\n    item: [code]\n")}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, withoutValues("proxy"))

			require.ErrorContains(t, err, `the starter service and the feature proxy both declare the extension point "api.middleware"`)
		})

		withFeatureOptions := func(template string) fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":     &fstest.MapFile{Data: []byte("options:\n  module:\n    default: example.com/{{ .Name }}\n  app:\n    default: api\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/go.mod": &fstest.MapFile{Data: []byte(`module {{ option "starter.module" }}`)},
				"features/auth/feature.yml":        &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [service]\nfiles:\n  - path: files/\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  realm:\n    default: \"{{ .Name }}-api\"\n  port:\n    default: \"8080\"\n    pattern: '^[0-9]+$'\n")},
				"features/auth/files/go/auth.txt":  &fstest.MapFile{Data: []byte(template)},
				"features/docs/feature.yml":        &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [service]\nfiles:\n  - path: files/\noptions:\n  realm:\n    default: docs\n")},
				"features/docs/files/go/docs.txt":  &fstest.MapFile{Data: []byte(`{{ option "realm" }}`)},
				"features/plain/feature.yml":       &fstest.MapFile{Data: []byte("summary: Plain.\ntargets: [service]\n")},
				"features/single/feature.yml":      &fstest.MapFile{Data: []byte("summary: Single.\ntargets: [service]\noptions:\n  mode:\n    values: [one]\n    default: one\n")},
				"features/broken/feature.yml":      &fstest.MapFile{Data: []byte("summary: Broken.\ntargets: [service]\noptions:\n  realm:\n    default: \"{{ .Nmae }}\"\n")},
			}
		}
		authTemplate := `{{ option "method" }} {{ option "realm" }} {{ option "port" }} {{ option "starter.module" }}`

		t.Run("an option of a feature with no given value takes its default, and the default renders with the name", func(t *testing.T) {
			starter, err := catalog.New(withFeatureOptions(authTemplate)).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, withoutValues("auth"))

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module example.com/my-service", "auth.txt": "api-key my-service-api 8080 example.com/my-service"}, written(t, selection))
		})

		t.Run("a given value of a feature option replaces its default", func(t *testing.T) {
			starter, err := catalog.New(withFeatureOptions(authTemplate)).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"auth": {"method": "oauth", "port": "9090"}})

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module example.com/my-service", "auth.txt": "oauth my-service-api 9090 example.com/my-service"}, written(t, selection))
		})

		t.Run("two selected features with an option of one name each read their own value", func(t *testing.T) {
			starter, err := catalog.New(withFeatureOptions(authTemplate)).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"auth": {"realm": "accounts"}, "docs": nil})

			require.NoError(t, err)
			require.Equal(t, map[string]string{"go.mod": "module example.com/my-service", "auth.txt": "api-key accounts 8080 example.com/my-service", "docs.txt": "docs"}, written(t, selection))
		})

		t.Run("a template of a feature reads the options of its feature also in its path", func(t *testing.T) {
			files := withFeatureOptions(authTemplate)
			files[`features/auth/files/go/{{ option "realm" }}/realm.txt`] = &fstest.MapFile{Data: []byte("realm\n")}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, withoutValues("auth"))

			require.NoError(t, err)
			require.Equal(t, "realm\n", written(t, selection)["my-service-api/realm.txt"])
		})

		for name, given := range map[string]struct {
			values  map[string]string
			message []string
		}{
			"a value of a feature option that is not one of its values":   {map[string]string{"method": "basic"}, []string{"the feature auth", `"basic"`, "the option method", "api-key, oauth"}},
			"a value of a feature option that does not match its pattern": {map[string]string{"port": "http"}, []string{"the feature auth", `"http"`, "the option port", `^[0-9]+$`}},
			"a value for an option that the feature does not declare":     {map[string]string{"methd": "oauth"}, []string{`"methd"`, "the feature auth", "method, port, realm"}},
			"a wrong value of a feature option after an accepted value":   {map[string]string{"method": "oauth", "port": "http"}, []string{`"http"`}},
		} {
			t.Run(name+" stops, and the message names the feature and the option", func(t *testing.T) {
				starter, err := catalog.New(withFeatureOptions(authTemplate)).Starter("service")
				require.NoError(t, err)

				_, err = starter.Select("my-service", nil, map[string]map[string]string{"auth": given.values})

				for _, part := range given.message {
					require.ErrorContains(t, err, part)
				}
			})
		}

		t.Run("an option with one value accepts only that value", func(t *testing.T) {
			starter, err := catalog.New(withFeatureOptions(authTemplate)).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, map[string]map[string]string{"single": {"mode": "two"}})

			require.ErrorContains(t, err, `the feature single: the value "two" of the option mode is not one of its values: one`)
		})

		t.Run("a default of a feature option that fails to render stops, and the message names the option and the feature", func(t *testing.T) {
			starter, err := catalog.New(withFeatureOptions(authTemplate)).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, withoutValues("broken"))

			require.ErrorContains(t, err, "the default of the option realm of the feature broken")
			require.ErrorContains(t, err, "Nmae")
		})

		t.Run("a default reads only the name of the repo, so option in a default of a feature stops, and the message names the option and the feature", func(t *testing.T) {
			files := withFeatureOptions(authTemplate)
			files["features/broken/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Broken.\ntargets: [service]\noptions:\n  realm:\n    default: '{{ option \"starter.module\" }}'\n")}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, withoutValues("broken"))

			require.ErrorContains(t, err, "the default of the option realm of the feature broken")
			require.ErrorContains(t, err, `unknown option "starter.module": the template can read no option`)
		})

		t.Run("option in a template of a starter with no option stops the write, and the message says that the template can read no option", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":    &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/service/files/go/a.txt": &fstest.MapFile{Data: []byte(`{{ option "starter.module" }}`)},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, nil)
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "repo")

			err = selection.Write(output)

			require.ErrorContains(t, err, `unknown option "starter.module": the template can read no option`)
			require.NoDirExists(t, output)
		})

		t.Run("a value for a feature with no option stops, and the message says that the feature has no option", func(t *testing.T) {
			starter, err := catalog.New(withFeatureOptions(authTemplate)).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, map[string]map[string]string{"plain": {"method": "oauth"}})

			require.ErrorContains(t, err, `unknown option "method": the feature plain has no option`)
		})

		for _, reference := range []string{"metod", "docs.realm", "module"} {
			t.Run("option "+strconv.Quote(reference)+" in a template of a feature stops the write, and the message names the template and lists the names that it can read", func(t *testing.T) {
				starter, err := catalog.New(withFeatureOptions(`{{ option "` + reference + `" }}`)).Starter("service")
				require.NoError(t, err)
				selection, err := starter.Select("my-service", nil, withoutValues("auth", "docs"))
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "repo")

				err = selection.Write(output)

				require.ErrorContains(t, err, "go/auth.txt of the feature auth")
				require.ErrorContains(t, err, strconv.Quote(reference))
				require.ErrorContains(t, err, "the template can read method, port, realm, starter.app, starter.module")
				require.NoDirExists(t, output)
			})
		}

		t.Run("option with the name of a feature option in a template of the starter stops the write, and the message lists the options of the starter", func(t *testing.T) {
			files := withFeatureOptions(authTemplate)
			files["starters/service/files/go/go.mod"] = &fstest.MapFile{Data: []byte(`module {{ option "method" }}`)}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues("auth"))
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "repo")

			err = selection.Write(output)

			require.ErrorContains(t, err, "go/go.mod of the starter")
			require.ErrorContains(t, err, `unknown option "method": the template can read starter.app, starter.module`)
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

		withNeeds := func(definitions map[string]string) fstest.MapFS {
			files := fstest.MapFS{
				"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    summary: A middleware.\n    file: go/server.go\n    item: [code]\n")},
				"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }};{{ end }}`)},
			}
			for featureName, definition := range definitions {
				files["features/"+featureName+"/feature.yml"] = &fstest.MapFile{Data: []byte("summary: A feature.\ntargets: [service]\nfiles:\n  - path: files/\nadd_to:\n  api.middleware:\n    - code: " + featureName + "\n" + definition)}
				files["features/"+featureName+"/files/go/"+featureName+".txt"] = &fstest.MapFile{Data: []byte(featureName)}
			}
			return files
		}

		t.Run("a feature selects its needs and the needs of its needs, with their files and their items", func(t *testing.T) {
			starter, err := catalog.New(withNeeds(map[string]string{"app": "needs: [mid, side]\n", "mid": "needs: [base]\n", "base": "", "side": "", "other": ""})).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, withoutValues("app"))

			require.NoError(t, err)
			require.Equal(t, map[string]string{"server.go": "base;mid;side;app;", "app.txt": "app", "mid.txt": "mid", "base.txt": "base", "side.txt": "side"}, written(t, selection))
		})

		t.Run("the items of an extension point come after the items of their needs, and otherwise in the order of the feature names", func(t *testing.T) {
			starter, err := catalog.New(withNeeds(map[string]string{"app": "needs: [zone]\n", "zone": "", "base": ""})).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, withoutValues("app", "base"))

			require.NoError(t, err)
			require.Equal(t, "base;zone;app;", written(t, selection)["server.go"])
		})

		withMethod := func() fstest.MapFS {
			files := withNeeds(map[string]string{"cli-auth": "needs: [auth]\n", "auth": "options:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n"})
			files["features/auth/files/go/auth.txt"] = &fstest.MapFile{Data: []byte(`{{ option "method" }}`)}
			return files
		}

		t.Run("a need that no flag gives takes the defaults of its options", func(t *testing.T) {
			starter, err := catalog.New(withMethod()).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, withoutValues("cli-auth"))

			require.NoError(t, err)
			require.Equal(t, "api-key", written(t, selection)["auth.txt"])
		})

		t.Run("a need that a flag also gives keeps the given values, and its items come one time", func(t *testing.T) {
			starter, err := catalog.New(withMethod()).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": nil, "auth": {"method": "oauth"}})

			require.NoError(t, err)
			files := written(t, selection)
			require.Equal(t, []string{"oauth", "auth;cli-auth;"}, []string{files["auth.txt"], files["server.go"]})
		})

		t.Run("a feature that adds items to an extension point that only its need declares selects alone, and its items render in the file of the need", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/cli/feature.yml":          &fstest.MapFile{Data: []byte("summary: CLI.\ntargets: [service]\nfiles:\n  - path: files/\nextension_points:\n  cli.commands:\n    summary: A command.\n    file: go/root.go\n    item: [code]\n")},
				"features/cli/files/go/root.go":     &fstest.MapFile{Data: []byte(`{{ range items "cli.commands" }}{{ .code }};{{ end }}`)},
				"features/cli-auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: CLI auth.\ntargets: [service]\nneeds: [cli]\nadd_to:\n  cli.commands:\n    - code: newAuthCommand()\n")},
			}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, withoutValues("cli-auth"))

			require.NoError(t, err)
			require.Equal(t, map[string]string{"main.go": "package main\n", "root.go": "newAuthCommand();"}, written(t, selection))
		})

		for name, need := range map[string]string{"does not exist": "missing", "targets another starter": "worker-tools"} {
			t.Run("a need that "+name+" stops, and the message names the feature, the need and the features that target the starter", func(t *testing.T) {
				files := withNeeds(map[string]string{"app": "needs: [" + need + "]\n", "base": ""})
				files["features/worker-tools/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Worker tools.\ntargets: [worker]\n")}
				starter, err := catalog.New(files).Starter("service")
				require.NoError(t, err)

				_, err = starter.Select("my-service", nil, withoutValues("app"))

				require.ErrorContains(t, err, `the feature app needs "`+need+`", which is not a feature that targets the starter service: the features that target it are app, base`)
			})
		}

		t.Run("a need of a need that is not a feature of the starter stops, and the message names the feature that needs it", func(t *testing.T) {
			starter, err := catalog.New(withNeeds(map[string]string{"app": "needs: [base]\n", "base": "needs: [missing]\n"})).Starter("service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, withoutValues("app"))

			require.ErrorContains(t, err, `the feature base needs "missing", which is not a feature that targets the starter service`)
		})

		for name, cycle := range map[string]struct {
			definitions map[string]string
			given       []string
			message     string
		}{
			"two features that need each other":                                {map[string]string{"app": "needs: [base]\n", "base": "needs: [app]\n"}, []string{"app"}, "app needs base needs app"},
			"a feature that needs itself":                                      {map[string]string{"app": "needs: [app]\n"}, []string{"app"}, "app needs app"},
			"three features, from a feature that needs the cycle":              {map[string]string{"access": "needs: [app]\n", "app": "needs: [mid]\n", "mid": "needs: [base]\n", "base": "needs: [app]\n"}, []string{"access"}, "app needs mid needs base needs app"},
			"two features, one of which also needs a feature out of the cycle": {map[string]string{"base": "", "mid": "needs: [base, zed]\n", "zed": "needs: [mid]\n"}, []string{"mid"}, "mid needs zed needs mid"},
		} {
			t.Run("the needs of "+name+" stop, and the message names each feature of the cycle", func(t *testing.T) {
				starter, err := catalog.New(withNeeds(cycle.definitions)).Starter("service")
				require.NoError(t, err)

				_, err = starter.Select("my-service", nil, withoutValues(cycle.given...))

				require.ErrorContains(t, err, "the needs of the features form a cycle: "+cycle.message)
			})
		}

		t.Run("a cycle among features that the selection does not hold does not stop it", func(t *testing.T) {
			starter, err := catalog.New(withNeeds(map[string]string{"app": "needs: [base]\n", "base": "needs: [app]\n", "plain": ""})).Starter("service")
			require.NoError(t, err)

			selection, err := starter.Select("my-service", nil, withoutValues("plain"))

			require.NoError(t, err)
			require.Equal(t, map[string]string{"server.go": "plain;", "plain.txt": "plain"}, written(t, selection))
		})
	})

}

func withoutValues(featureNames ...string) map[string]map[string]string {
	features := make(map[string]map[string]string, len(featureNames))
	for _, featureName := range featureNames {
		features[featureName] = nil
	}
	return features
}
