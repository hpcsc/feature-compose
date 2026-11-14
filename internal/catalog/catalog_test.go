//go:build unit

package catalog_test

import (
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
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

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
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
			"an option of a feature with a pattern that is not valid": {
				"features/cli/feature.yml": &fstest.MapFile{Data: []byte("targets: [service]\noptions:\n  port:\n    pattern: '[0-9'\n")},
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

		for name, needs := range map[string]string{
			"a text": "needs: cli\n",
			"a list with an entry that is not a name": "needs:\n  - [cli]\n",
			"a need whose values are a list":          "needs:\n  cli: [fast]\n",
			"a need with a value that is not a text":  "needs:\n  cli:\n    mode: {fast: true}\n",
			"a list that names one need two times":    "needs: [docs, docs]\n",
			"a map that names one need two times":     "needs:\n  docs:\n  docs:\n    mode: fast\n",
		} {
			t.Run("needs as "+name+" stops the read of the starter and of the definitions, and the message names the feature.yml and needs", func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
					"features/cli/feature.yml":          &fstest.MapFile{Data: []byte("targets: [service]\n" + needs)},
				}

				_, starterErr := catalog.New(files).Starter("service")
				_, definitionsErr := catalog.New(files).Definitions()

				for _, err := range []error{starterErr, definitionsErr} {
					require.ErrorContains(t, err, "features/cli/feature.yml")
					require.ErrorContains(t, err, "needs")
				}
			})
		}

		whensThatDoNotParse := map[string]string{
			`method ==`:          `method == needs a quoted value after it`,
			`method == oauth`:    `method == needs a quoted value after it`,
			`method = "oauth"`:   `"=" is not a part of an expression`,
			`(method == "oauth"`: `a ( has no closing )`,
			`(method method)`:    `a ( has no closing )`,
			`method == "oauth")`: `) comes after the end of the expression`,
			`a &&`:               `the expression ends where a name is expected`,
			`a ||`:               `the expression ends where a name is expected`,
			`!`:                  `the expression ends where a name is expected`,
			`"oauth"`:            `"oauth" is where a name is expected`,
			`()`:                 `) is where a name is expected`,
			`method == "oauth`:   `the value "oauth has no closing quote`,
		}
		for when, message := range whensThatDoNotParse {
			t.Run("a when that does not parse stops, and the message names the feature.yml, the files entry, the when and why: "+when, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
					"features/cli/feature.yml":          &fstest.MapFile{Data: []byte("targets: [service]\noptions:\n  method:\n    default: api-key\nfiles:\n  - path: files/\n    when: '" + when + "'\n")},
					"features/cli/files/go/cli.txt":     &fstest.MapFile{Data: []byte("cli\n")},
				}

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "features/cli/feature.yml")
				require.ErrorContains(t, err, `the files entry "files/"`)
				require.ErrorContains(t, err, strconv.Quote(when)+` of the files entry "files/": `+message)
			})
		}

		for when, message := range whensThatDoNotParse {
			t.Run("a when of an option that does not parse stops, and the message names the feature.yml, the option, the when and why: "+when, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
					"features/cli/feature.yml":          &fstest.MapFile{Data: []byte("targets: [service]\noptions:\n  store-key:\n    values: [\"true\", \"false\"]\n    default: \"false\"\n    when: '" + when + "'\n")},
				}

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "features/cli/feature.yml")
				require.ErrorContains(t, err, strconv.Quote(when)+` of the option "store-key": `+message)
			})
		}

		for when, message := range whensThatDoNotParse {
			t.Run("a when of an item that does not parse stops, and the message names the feature.yml, the item, the extension point, the when and why: "+when, func(t *testing.T) {
				files := fstest.MapFS{
					"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n")},
					"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }}{{ end }}`)},
					"features/cli/feature.yml":            &fstest.MapFile{Data: []byte("targets: [service]\noptions:\n  method:\n    default: api-key\nadd_to:\n  api.middleware:\n    - code: a\n    - code: b\n      when: '" + when + "'\n")},
				}

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "features/cli/feature.yml")
				require.ErrorContains(t, err, strconv.Quote(when)+` of the item 2 of the extension point "api.middleware": `+message)
			})
		}

		t.Run("of two options of a starter with a when, the message names the first in the order of the names", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("options:\n  zone:\n    default: a\n    when: 'module == \"x\"'\n  region:\n    default: us\n    when: 'module == \"x\"'\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}

			for range 5 {
				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "the option region has a when")
			}
		})

		t.Run("an option of a starter with a when stops the read of the starter and the check, and the message names the starter.yml and the option", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("options:\n  region:\n    default: us\n    when: 'module == \"x\"'\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			}

			_, starterErr := catalog.New(files).Starter("service")
			_, checkErr := catalog.New(files).Check()

			for _, err := range []error{starterErr, checkErr} {
				require.ErrorContains(t, err, "starters/service/starter.yml")
				require.ErrorContains(t, err, "the option region has a when, and only an option of a feature can have a when")
			}
		})

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
			"features/elsewhere/feature.yml",
		} {
			t.Run("a read of "+failing+" that fails stops with its cause", func(t *testing.T) {
				files := failingFS{failing: failing, files: fstest.MapFS{
					"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
					"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
					"features/cli/feature.yml":          &fstest.MapFile{Data: []byte("targets: [service]\nfiles:\n  - path: files/\n")},
					"features/cli/files/go/cli.go":      &fstest.MapFile{Data: []byte("package cli\n")},
					"features/elsewhere/feature.yml":    &fstest.MapFile{Data: []byte("targets: [other]\n")},
				}}

				_, err := catalog.New(files).Starter("service")

				require.ErrorContains(t, err, "disk failure")
			})
		}
	})

	t.Run("definitions", func(t *testing.T) {
		for _, failing := range []string{"starters", "starters/api/starter.yml", "features", "features/cli/feature.yml"} {
			t.Run("a read of "+failing+" that fails stops with its cause", func(t *testing.T) {
				files := failingFS{failing: failing, files: fstest.MapFS{
					"starters/api/starter.yml": &fstest.MapFile{Data: []byte("summary: An API.\n")},
					"features/cli/feature.yml": &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [api]\n")},
				}}

				_, err := catalog.New(files).Definitions()

				require.ErrorContains(t, err, "disk failure")
			})
		}

		t.Run("a field with no value is an empty list, an empty object or an empty text, so the JSON of the definitions holds no null, and each need gives the values that it sets", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/starter.yml":    &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n  web:\n    path: web\noptions:\n  module:\n    default: x\n  region:\n    default: us\nextension_points:\n  api.middleware:\n    file: go/server.go\n  api.config:\n    file: go/config.go\n")},
				"starters/bare/starter.yml":   &fstest.MapFile{Data: []byte("summary: Bare.\n")},
				"features/auth/feature.yml":   &fstest.MapFile{Data: []byte("needs:\n  bare:\n  listed:\n    mode: fast\noptions:\n  method:\n    default: api-key\nextension_points:\n  auth.checks:\n    file: go/auth.go\nadd_to:\n  api.middleware:\n  api.config:\n")},
				"features/bare/feature.yml":   &fstest.MapFile{Data: []byte("summary: Bare.\nneeds:\n")},
				"features/listed/feature.yml": &fstest.MapFile{Data: []byte("needs: [auth, bare]\noptions:\n  mode:\n    default: slow\n    when: 'auth.method == \"api-key\"'\n")},
			}
			definitions, err := catalog.New(files).Definitions()
			require.NoError(t, err)

			encoded, err := json.Marshal(definitions)

			require.NoError(t, err)
			require.JSONEq(t, `{
				"starters": {
					"api": {
						"summary": "",
						"folders": {"go": {"path": ".", "after_render": [], "test": []}, "web": {"path": "web", "after_render": [], "test": []}},
						"options": {"module": {"default": "x", "pattern": "", "values": [], "when": ""}, "region": {"default": "us", "pattern": "", "values": [], "when": ""}},
						"extension_points": {"api.middleware": {"summary": "", "file": "go/server.go", "item": []}, "api.config": {"summary": "", "file": "go/config.go", "item": []}}
					},
					"bare": {"summary": "Bare.", "folders": {}, "options": {}, "extension_points": {}}
				},
				"features": {
					"auth": {
						"summary": "", "targets": [], "needs": {"bare": {}, "listed": {"mode": "fast"}},
						"options": {"method": {"default": "api-key", "pattern": "", "values": [], "when": ""}},
						"extension_points": {"auth.checks": {"summary": "", "file": "go/auth.go", "item": []}},
						"add_to": {"api.middleware": [], "api.config": []}
					},
					"bare": {"summary": "Bare.", "targets": [], "needs": {}, "options": {}, "extension_points": {}, "add_to": {}},
					"listed": {"summary": "", "targets": [], "needs": {"auth": {}, "bare": {}}, "options": {"mode": {"default": "slow", "pattern": "", "values": [], "when": "auth.method == \"api-key\""}}, "extension_points": {}, "add_to": {}}
				}
			}`, string(encoded))
		})

		t.Run("the add_to of a feature gives the when of an item as the key when of the item, and an item with no when has no such key", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/starter.yml":  &fstest.MapFile{Data: []byte("summary: An API.\n")},
				"features/auth/feature.yml": &fstest.MapFile{Data: []byte("targets: [api]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      when: method == \"oauth\"\n    - code: log.Middleware\n")},
			}

			definitions, err := catalog.New(files).Definitions()

			require.NoError(t, err)
			require.Equal(t, map[string][]map[string]string{
				"api.middleware": {
					{"code": "auth.Middleware", "when": `method == "oauth"`},
					{"code": "log.Middleware"},
				},
			}, definitions.Features["auth"].AddTo)
		})
	})

	t.Run("check", func(t *testing.T) {
		rendersItems := func(extensionPoints ...string) *fstest.MapFile {
			var text strings.Builder
			for _, name := range extensionPoints {
				text.WriteString("{{ range items " + strconv.Quote(name) + " }}{{ .code }}{{ end }}\n")
			}
			return &fstest.MapFile{Data: []byte(text.String())}
		}
		withFeatures := func(features map[string]string) fstest.MapFS {
			files := fstest.MapFS{
				"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code, import]\n")},
				"starters/service/files/go/server.go": rendersItems("api.middleware"),
			}
			for featureName, definition := range features {
				files["features/"+featureName+"/feature.yml"] = &fstest.MapFile{Data: []byte(definition)}
			}
			return files
		}

		for name, extensionPoint := range map[string]string{
			"that nothing declares": "cli.commands",
			"that only a feature that the feature does not need declares": "docs.pages",
		} {
			t.Run("an add_to to an extension point "+name+" is a mistake, and the message names the starter, the feature and the extension point", func(t *testing.T) {
				files := withFeatures(map[string]string{
					"auth": "targets: [service]\nadd_to:\n  " + extensionPoint + ":\n    - path: auth.md\n",
					"docs": "targets: [service]\nextension_points:\n  docs.pages:\n    file: go/index.md\n    item: [path]\nfiles:\n  - path: files/\n",
				})
				files["features/docs/files/go/index.md"] = rendersItems("docs.pages")

				mistakes, err := catalog.New(files).Check()

				require.NoError(t, err)
				require.Equal(t, []string{
					`the starter service: the feature auth adds to the extension point "` + extensionPoint + `", which the starter, the feature and its needs do not declare`,
				}, mistakes)
			})
		}

		t.Run("an add_to to an extension point that the starter, the feature itself, a need or a need of a need declares is not a mistake", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"app": `targets: [service]
needs: [cli]
extension_points:
  app.routes:
    file: go/routes.go
    item: [path]
add_to:
  api.middleware:
    - code: app.Middleware()
      import: internal/app
  app.routes:
    - path: /app
  cli.commands:
    - code: newAppCommand()
  cli.flags:
    - name: verbose
  base.hooks:
    - code: appHook()
  docs.pages:
    - path: app.md
files:
  - path: files/
`,
				"cli":  "targets: [service]\nneeds: [base]\nextension_points:\n  cli.commands:\n    file: go/cli.go\n    item: [code]\n  cli.flags:\n    file: go/flags.go\n    item: [name]\nfiles:\n  - path: files/\n",
				"base": "targets: [service]\nextension_points:\n  base.hooks:\n    file: go/base.go\n    item: [code]\nfiles:\n  - path: files/\n",
			})
			files["features/app/files/go/routes.go"] = rendersItems("app.routes")
			files["features/cli/files/go/cli.go"] = rendersItems("cli.commands")
			files["features/cli/files/go/flags.go"] = rendersItems("cli.flags")
			files["features/base/files/go/base.go"] = rendersItems("base.hooks")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the feature app adds to the extension point "docs.pages", which the starter, the feature and its needs do not declare`,
			}, mistakes)
		})

		for name, item := range map[string][2]string{
			"that lacks a field of its extension point":              {"    - code: auth.Middleware\n", "[code]"},
			"with a field that its extension point does not declare": {"    - code: auth.Middleware\n      import: internal/auth\n      order: first\n", "[code, import, order]"},
		} {
			t.Run("an item "+name+" is a mistake, and the message names the starter, the feature, the extension point, the item, its fields and the declared fields", func(t *testing.T) {
				files := withFeatures(map[string]string{"auth": "targets: [service]\nadd_to:\n  api.middleware:\n" + item[0]})

				mistakes, err := catalog.New(files).Check()

				require.NoError(t, err)
				require.Equal(t, []string{
					`the starter service: the item 1 that the feature auth adds to the extension point "api.middleware" has the fields ` + item[1] + `, but the extension point declares the fields [code, import]`,
				}, mistakes)
			})
		}

		t.Run("an item with the fields of its extension point in another order is not a mistake", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": "targets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n  cli.commands:\n    - code: newAuthCommand()\n"})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [import, code]\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the feature auth adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare`,
			}, mistakes)
		})

		t.Run("all mistakes of the catalog are in one list, in the order of the starters, the features, the extension points and the items, and each target of a feature gives its own mistake", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": `targets: [service, worker]
add_to:
  api.middleware:
    - code: auth.Middleware
    - code: auth.Audit
      import: internal/auth
    - code: auth.Log
      import: internal/auth
      order: last
`,
				"tools": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n  admin.pages:\n    - path: tools.md\n",
			})
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
			files["starters/worker/files/go/main.go"] = &fstest.MapFile{Data: []byte("package main\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the item 1 that the feature auth adds to the extension point "api.middleware" has the fields [code], but the extension point declares the fields [code, import]`,
				`the starter service: the item 3 that the feature auth adds to the extension point "api.middleware" has the fields [code, import, order], but the extension point declares the fields [code, import]`,
				`the starter service: the feature tools adds to the extension point "admin.pages", which the starter, the feature and its needs do not declare`,
				`the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare`,
				`the starter worker: the feature auth adds to the extension point "api.middleware", which the starter, the feature and its needs do not declare`,
			}, mistakes)
		})

		t.Run("a feature that adds files to a folder that the starter does not have is a mistake, and a folder with many files gives one mistake", func(t *testing.T) {
			files := withFeatures(map[string]string{"web": "targets: [service]\nfiles:\n  - path: files/\n"})
			files["features/web/files/go/web.go"] = &fstest.MapFile{Data: []byte("package web\n")}
			files["features/web/files/web/index.html"] = &fstest.MapFile{Data: []byte("<html></html>\n")}
			files["features/web/files/web/app.js"] = &fstest.MapFile{Data: []byte("app()\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{"the starter service: the feature web adds files to the folder web, which the starter does not have"}, mistakes)
		})

		t.Run("the files of each files entry are checked, also of an entry whose when is false and of an entry whose when is true", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": `targets: [service]
options:
  method:
    values: [api-key, oauth]
    default: api-key
files:
  - path: files/common/
  - path: files/oauth/
    when: method == "oauth"
  - path: files/api-key/
    when: method == "api-key"
`})
			files["features/auth/files/common/go/auth.go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			files["features/auth/files/oauth/docs/oauth.md"] = &fstest.MapFile{Data: []byte("oauth\n")}
			files["features/auth/files/api-key/web/keys.html"] = &fstest.MapFile{Data: []byte("keys\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the feature auth adds files to the folder docs, which the starter does not have",
				"the starter service: the feature auth adds files to the folder web, which the starter does not have",
			}, mistakes)
		})

		t.Run("a file directly in a files entry is in no folder, and the message names the starter, the feature and the file", func(t *testing.T) {
			files := withFeatures(map[string]string{"cli": "targets: [service]\nfiles:\n  - path: files/\n"})
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte("package cli\n")}
			files["features/cli/files/README.md"] = &fstest.MapFile{Data: []byte("cli\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{"the starter service: the template README.md of the files entry files/ of the feature cli is not in a folder"}, mistakes)
		})

		t.Run("a feature whose targets declare an extension point of its add_to with different item fields is a mistake, the message names each target with its item fields, and in a target where a need declares the extension point, the fields of the need count", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth":  "targets: [service, worker]\nneeds: [queue]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n",
				"queue": "targets: [worker]\nextension_points:\n  api.middleware:\n    file: go/queue.go\n    item: [code]\nfiles:\n  - path: files/\n",
			})
			files["features/queue/files/go/queue.go"] = rendersItems("api.middleware")
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
			files["starters/worker/files/go/main.go"] = &fstest.MapFile{Data: []byte("package main\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the feature auth needs queue, which does not target the starter`,
				`the starter worker: the item 1 that the feature auth adds to the extension point "api.middleware" has the fields [code, import], but the extension point declares the fields [code]`,
				`the feature auth adds to the extension point "api.middleware", which its targets declare with different item fields: [code, import] in the starter service, [code] in the starter worker`,
			}, mistakes)
		})

		t.Run("the mistakes of targets with different item fields come after the mistakes of the starters, in the order of the features and the extension points, and a target between two targets with the same item fields counts", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth":  "targets: [cron, service, worker]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n",
				"tools": "targets: [cron, service, worker]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n  api.middleware:\n    - code: tools.Middleware\n      import: internal/tools\n",
			})
			for starterName, extensionPoints := range map[string]string{
				"api":     "",
				"cron":    "extension_points:\n  api.middleware:\n    file: go/main.go\n    item: [code, import]\n  cli.commands:\n    file: go/main.go\n    item: [code]\n",
				"service": "extension_points:\n  api.middleware:\n    file: go/main.go\n    item: [code]\n  cli.commands:\n    file: go/main.go\n    item: [code]\n",
				"worker":  "extension_points:\n  api.middleware:\n    file: go/main.go\n    item: [code, import]\n  cli.commands:\n    file: go/main.go\n    item: [code, name]\n",
			} {
				files["starters/"+starterName+"/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n" + extensionPoints)}
				files["starters/"+starterName+"/files/go/main.go"] = rendersItems("api.middleware", "cli.commands")
			}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the item 1 that the feature auth adds to the extension point "api.middleware" has the fields [code, import], but the extension point declares the fields [code]`,
				`the starter service: the item 1 that the feature tools adds to the extension point "api.middleware" has the fields [code, import], but the extension point declares the fields [code]`,
				`the starter worker: the item 1 that the feature tools adds to the extension point "cli.commands" has the fields [code], but the extension point declares the fields [code, name]`,
				`the feature auth adds to the extension point "api.middleware", which its targets declare with different item fields: [code, import] in the starter cron, [code] in the starter service, [code, import] in the starter worker`,
				`the feature tools adds to the extension point "api.middleware", which its targets declare with different item fields: [code, import] in the starter cron, [code] in the starter service, [code, import] in the starter worker`,
				`the feature tools adds to the extension point "cli.commands", which its targets declare with different item fields: [code] in the starter cron, [code] in the starter service, [code, name] in the starter worker`,
			}, mistakes)
		})

		t.Run("targets that declare the item fields of an extension point in another order are not a mistake, and a target that does not declare it is not named", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": "targets: [service, worker, cron]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n"})
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [import, code]\n")}
			files["starters/worker/files/go/server.go"] = rendersItems("api.middleware")
			files["starters/cron/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
			files["starters/cron/files/go/main.go"] = &fstest.MapFile{Data: []byte("package main\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter cron: the feature auth adds to the extension point "api.middleware", which the starter, the feature and its needs do not declare`,
			}, mistakes)
		})

		t.Run("a starter that the feature does not target is not in the comparison of the item fields of its targets", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": "targets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n  cli.commands:\n    - code: newAuthCommand()\n"})
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n")}
			files["starters/worker/files/go/server.go"] = rendersItems("api.middleware")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the feature auth adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare`,
			}, mistakes)
		})

		t.Run("an extension point that two targets declare with different item fields is not a mistake when the feature does not add to it", func(t *testing.T) {
			files := withFeatures(map[string]string{"tools": "targets: [service, worker]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n"})
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n")}
			files["starters/worker/files/go/server.go"] = rendersItems("api.middleware")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare`,
				`the starter worker: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare`,
			}, mistakes)
		})

		withWorker := func(files fstest.MapFS) fstest.MapFS {
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
			files["starters/worker/files/go/main.go"] = &fstest.MapFile{Data: []byte("package main\n")}
			return files
		}

		t.Run("a need that is not a feature of the catalog is a mistake for each target of the feature, and the message names the starter, the feature and the need", func(t *testing.T) {
			files := withWorker(withFeatures(map[string]string{"auth": "targets: [service, worker]\nneeds: [vault]\n"}))

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the feature auth needs vault, which is not a feature of the catalog",
				"the starter worker: the feature auth needs vault, which is not a feature of the catalog",
			}, mistakes)
		})

		t.Run("a need that does not target a target of the feature is a mistake for that target only, and the message names the starter, the feature and the need", func(t *testing.T) {
			files := withWorker(withFeatures(map[string]string{
				"auth":  "targets: [service, worker]\nneeds: [cli, vault]\n",
				"cli":   "targets: [service]\n",
				"vault": "targets: [worker]\n",
			}))

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the feature auth needs vault, which does not target the starter",
				"the starter worker: the feature auth needs cli, which does not target the starter",
			}, mistakes)
		})

		t.Run("each value that a need sets and its option does not have is a mistake, and the message names the starter, the feature, the need, the option and the value", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"api-auth": "targets: [service]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  port:\n    pattern: '^[0-9]+$'\n    default: \"80\"\n",
				"plain":    "targets: [service]\n",
				"cli-auth": "targets: [service]\nneeds:\n  api-auth:\n    method: basic\n    port: http\n    region: us\n  plain:\n    mode: fast\n",
			})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the need of the feature cli-auth: the feature api-auth: the value "basic" of the option method is not one of its values: api-key, oauth`,
				`the starter service: the need of the feature cli-auth: the feature api-auth: the value "http" of the option port does not match its pattern ^[0-9]+$`,
				`the starter service: the need of the feature cli-auth: unknown option "region": the options of the feature api-auth are method, port`,
				`the starter service: the need of the feature cli-auth: unknown option "mode": the feature plain has no option`,
			}, mistakes)
		})

		t.Run("a value that a need sets and its option has is not a mistake", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"api-auth": "targets: [service]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n",
				"cli-auth": "targets: [service]\nneeds:\n  api-auth:\n    method: oauth\n  vault:\n    mode: fast\n",
			})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{"the starter service: the feature cli-auth needs vault, which is not a feature of the catalog"}, mistakes)
		})

		t.Run("a wrong value that a need sets is a mistake also in a target that the need does not target", func(t *testing.T) {
			files := withWorker(withFeatures(map[string]string{
				"api-auth": "targets: [service]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n",
				"cli-auth": "targets: [service, worker]\nneeds:\n  api-auth:\n    method: basic\n",
			}))

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the need of the feature cli-auth: the feature api-auth: the value "basic" of the option method is not one of its values: api-key, oauth`,
				`the starter worker: the feature cli-auth needs api-auth, which does not target the starter`,
				`the starter worker: the need of the feature cli-auth: the feature api-auth: the value "basic" of the option method is not one of its values: api-key, oauth`,
			}, mistakes)
		})

		t.Run("a wrong need of a need is the mistake of the feature that needs it, not of the feature that needs that feature", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"app": "targets: [service]\nneeds: [cli]\n",
				"cli": "targets: [service]\nneeds: [vault]\n",
			})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{"the starter service: the feature cli needs vault, which is not a feature of the catalog"}, mistakes)
		})

		t.Run("features whose needs form a cycle are one mistake after the mistakes of the features, and the message names the starter and each feature of the cycle in the order of the needs", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"app":  "targets: [service]\nneeds: [base]\nadd_to:\n  cli.commands:\n    - code: newAppCommand()\n",
				"base": "targets: [service]\nneeds: [app]\n",
			})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the feature app adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare`,
				"the starter service: the needs of the features form a cycle: app needs base needs app",
			}, mistakes)
		})

		t.Run("a feature that needs itself is a cycle", func(t *testing.T) {
			files := withFeatures(map[string]string{"app": "targets: [service]\nneeds: [app]\n"})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{"the starter service: the needs of the features form a cycle: app needs app"}, mistakes)
		})

		t.Run("a cycle of three features that another feature needs names the three features and not the feature that needs them", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"api": "targets: [service]\nneeds: [cli]\n",
				"cli": "targets: [service]\nneeds: [db]\n",
				"db":  "targets: [service]\nneeds: [env]\n",
				"env": "targets: [service]\nneeds: [cli]\n",
			})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{"the starter service: the needs of the features form a cycle: cli needs db needs env needs cli"}, mistakes)
		})

		t.Run("two cycles with no feature in common are two mistakes, and a cycle gives one mistake for each starter whose features hold it", func(t *testing.T) {
			files := withWorker(withFeatures(map[string]string{
				"a": "targets: [service, worker]\nneeds: [b]\n",
				"b": "targets: [service, worker]\nneeds: [a]\n",
				"c": "targets: [service]\nneeds: [d]\n",
				"d": "targets: [service]\nneeds: [c]\n",
			}))

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the needs of the features form a cycle: a needs b needs a",
				"the starter service: the needs of the features form a cycle: c needs d needs c",
				"the starter worker: the needs of the features form a cycle: a needs b needs a",
			}, mistakes)
		})

		t.Run("each feature of two cycles that share a feature is named in a cycle mistake", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"a": "targets: [service]\nneeds: [b]\n",
				"b": "targets: [service]\nneeds: [a, c]\n",
				"c": "targets: [service]\nneeds: [b]\n",
			})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the needs of the features form a cycle: a needs b needs a",
				"the starter service: the needs of the features form a cycle: c needs b needs c",
			}, mistakes)
		})

		t.Run("needs that form a cycle only through a feature that does not target the starter are not a cycle of that starter", func(t *testing.T) {
			files := withWorker(withFeatures(map[string]string{
				"a": "targets: [service, worker]\nneeds: [x]\n",
				"x": "targets: [service]\nneeds: [a]\n",
			}))

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the needs of the features form a cycle: a needs x needs a",
				"the starter worker: the feature a needs x, which does not target the starter",
			}, mistakes)
		})

		t.Run("each files entry whose when does not parse is one mistake of the feature, also with two targets, after its item fields mistakes, and the message names the feature, the files entry, the when and why", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": `targets: [service, worker]
options:
  method:
    default: api-key
add_to:
  api.middleware:
    - code: auth.Middleware
      import: internal/auth
files:
  - path: files/api-key/
    when: method == api-key
  - path: files/common/
  - path: files/oauth/
    when: '(method == "oauth"'
`,
				"cli": "targets: [service]\nfiles:\n  - path: files/\n    when: 'a &&'\n",
			})
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n")}
			files["starters/worker/files/go/server.go"] = rendersItems("api.middleware")
			files["features/auth/files/api-key/go/keys.go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			files["features/auth/files/common/go/auth.go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			files["features/auth/files/oauth/go/tokens.go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte("package cli\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter worker: the item 1 that the feature auth adds to the extension point "api.middleware" has the fields [code, import], but the extension point declares the fields [code]`,
				`the feature auth adds to the extension point "api.middleware", which its targets declare with different item fields: [code, import] in the starter service, [code] in the starter worker`,
				`the feature auth: the when "method == api-key" of the files entry "files/api-key/": method == needs a quoted value after it`,
				`the feature auth: the when "(method == \"oauth\"" of the files entry "files/oauth/": a ( has no closing )`,
				`the feature cli: the when "a &&" of the files entry "files/": the expression ends where a name is expected`,
			}, mistakes)
		})

		t.Run("each option whose when does not parse is one mistake of the feature, also with two targets, after the when of the files entries, in the order of the option names", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": `targets: [service, worker]
options:
  scope:
    default: read
    when: 'a ||'
  method:
    default: api-key
    when: method == api-key
files:
  - path: files/
    when: '(method == "oauth"'
`,
				"cli": "targets: [service]\noptions:\n  level:\n    default: info\n    when: '!'\n",
			})
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
			files["starters/worker/files/go/main.go"] = &fstest.MapFile{Data: []byte("package main\n")}
			files["features/auth/files/go/auth.go"] = &fstest.MapFile{Data: []byte("package auth\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature auth: the when "(method == \"oauth\"" of the files entry "files/": a ( has no closing )`,
				`the feature auth: the when "method == api-key" of the option "method": method == needs a quoted value after it`,
				`the feature auth: the when "a ||" of the option "scope": the expression ends where a name is expected`,
				`the feature cli: the when "!" of the option "level": the expression ends where a name is expected`,
			}, mistakes)
		})

		t.Run("each item whose when does not parse is one mistake of the feature, after the when of its options, in the order of the extension points and the items, and the message names the feature, the item, the extension point, the when and why", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": `targets: [service]
options:
  method:
    default: api-key
    when: 'a ||'
extension_points:
  auth.checks:
    file: go/auth.go
    item: [code]
add_to:
  auth.checks:
    - code: a
      when: '!'
  api.middleware:
    - code: auth.Middleware
      import: internal/auth
    - code: audit.Middleware
      import: internal/audit
      when: method == oauth
files:
  - path: files/
`,
			})
			files["features/auth/files/go/auth.go"] = rendersItems("auth.checks")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature auth: the when "a ||" of the option "method": the expression ends where a name is expected`,
				`the feature auth: the when "method == oauth" of the item 2 of the extension point "api.middleware": method == needs a quoted value after it`,
				`the feature auth: the when "!" of the item 1 of the extension point "auth.checks": the expression ends where a name is expected`,
			}, mistakes)
		})

		t.Run("a when of an item that reads a reference that the feature cannot read is a mistake, and the message names the feature, the item, the extension point, the when, the reference and the references that the feature can read", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"vault": "targets: [service]\noptions:\n  mode:\n    default: slow\n",
				"auth": `targets: [service]
options:
  method:
    default: api-key
extension_points:
  auth.checks:
    file: go/auth.go
    item: [code]
add_to:
  api.middleware:
    - code: auth.Middleware
      import: internal/auth
      when: vault.mode == "fast"
    - code: audit.Middleware
      import: internal/audit
      when: method == "oauth" && starter.region == "eu"
    - code: log.Middleware
      import: internal/log
      when: method == "oauth"
  auth.checks:
    - code: a
      when: level == "high"
files:
  - path: files/
`,
			})
			files["features/auth/files/go/auth.go"] = rendersItems("auth.checks")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature auth: the when "vault.mode == \"fast\"" of the item 1 of the extension point "api.middleware" reads "vault.mode", which the feature cannot read: it can read method`,
				`the feature auth: the when "method == \"oauth\" && starter.region == \"eu\"" of the item 2 of the extension point "api.middleware" reads "starter.region", which the starter service does not declare`,
				`the feature auth: the when "level == \"high\"" of the item 1 of the extension point "auth.checks" reads "level", which the feature cannot read: it can read method`,
			}, mistakes)
		})

		t.Run("an item with the item fields of its extension point and a when is not a mistake, and neither is an item with no when", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": "targets: [service]\noptions:\n  method:\n    default: api-key\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n      when: method == \"oauth\"\n    - code: log.Middleware\n      import: internal/log\n",
			})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Empty(t, mistakes)
		})

		t.Run("an extension point that declares the item field when is a mistake of the starter or of the feature that declares it", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"docs": "targets: [service]\nextension_points:\n  docs.pages:\n    file: go/index.md\n    item: [path, when]\nfiles:\n  - path: files/\n",
			})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code, when]\n")}
			files["features/docs/files/go/index.md"] = rendersItems("docs.pages")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the extension point "api.middleware" declares the item field when, which an item of add_to uses for its own when`,
				`the feature docs: the extension point "docs.pages" declares the item field when, which an item of add_to uses for its own when`,
			}, mistakes)
		})

		t.Run("a feature with a when that does not parse still gives its other mistakes, also of the files of that entry, and the other features give theirs", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth":  "targets: [service]\nneeds: [vault]\nfiles:\n  - path: files/\n    when: 'method ||'\n",
				"tools": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n",
			})
			files["features/auth/files/web/index.html"] = &fstest.MapFile{Data: []byte("<html></html>\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the feature auth adds files to the folder web, which the starter does not have",
				"the starter service: the feature auth needs vault, which is not a feature of the catalog",
				`the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare`,
				`the feature auth: the when "method ||" of the files entry "files/": the expression ends where a name is expected`,
			}, mistakes)
		})

		t.Run("a when that reads a reference that the feature cannot read is a mistake, and the message names the feature, the when, its files entry, the reference and the references that the feature can read", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"base":     "targets: [service]\noptions:\n  level:\n    default: low\n",
				"api-auth": "targets: [service]\nneeds: [base]\noptions:\n  method:\n    default: api-key\n  scope:\n    default: read\n",
				"tokens":   "targets: [service]\noptions:\n  ttl:\n    default: 1h\n",
				"vault":    "targets: [service]\noptions:\n  mode:\n    default: slow\n",
				"cli-auth": `targets: [service]
needs: [api-auth, tokens]
options:
  debug:
    default: "false"
  store:
    default: "false"
files:
  - path: files/own/
    when: region == "us"
  - path: files/need/
    when: api-auth.port == "80"
  - path: files/other/
    when: vault.mode == "fast"
  - path: files/deep/
    when: base.level == "high"
  - path: files/starter/
    when: starter.region == "eu"
  - path: files/readable/
    when: store && debug && api-auth.method == "oauth" && api-auth.scope && tokens.ttl && starter.module != "x" && starter.go-version
`,
			})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\noptions:\n  module:\n    default: example.com/service\n  go-version:\n    default: \"1.25\"\n")}
			for _, entry := range []string{"own", "need", "other", "deep", "starter", "readable"} {
				files["features/cli-auth/files/"+entry+"/go/"+entry+".go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature cli-auth: the when "region == \"us\"" of the files entry "files/own/" reads "region", which the feature cannot read: it can read api-auth.method, api-auth.scope, debug, starter.go-version, starter.module, store, tokens.ttl`,
				`the feature cli-auth: the when "api-auth.port == \"80\"" of the files entry "files/need/" reads "api-auth.port", which the feature cannot read: it can read api-auth.method, api-auth.scope, debug, starter.go-version, starter.module, store, tokens.ttl`,
				`the feature cli-auth: the when "vault.mode == \"fast\"" of the files entry "files/other/" reads "vault.mode", which the feature cannot read: it can read api-auth.method, api-auth.scope, debug, starter.go-version, starter.module, store, tokens.ttl`,
				`the feature cli-auth: the when "base.level == \"high\"" of the files entry "files/deep/" reads "base.level", which the feature cannot read: it can read api-auth.method, api-auth.scope, debug, starter.go-version, starter.module, store, tokens.ttl`,
				`the feature cli-auth: the when "starter.region == \"eu\"" of the files entry "files/starter/" reads "starter.region", which the starter service does not declare`,
			}, mistakes)
		})

		t.Run("a when of an option that reads a reference that it cannot read is a mistake, and the message names the feature, the option, the when, the reference and the references that the when of an option can read", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"base":     "targets: [service]\noptions:\n  level:\n    default: low\n",
				"api-auth": "targets: [service]\nneeds: [base]\noptions:\n  method:\n    default: api-key\n  scope:\n    default: read\n",
				"tokens":   "targets: [service]\noptions:\n  ttl:\n    default: 1h\n",
				"vault":    "targets: [service]\noptions:\n  mode:\n    default: slow\n",
				"cli-auth": `targets: [service]
needs: [api-auth, tokens]
options:
  debug:
    default: "false"
  store:
    default: "false"
  a-region:
    default: x
    when: region == "us"
  b-need:
    default: x
    when: api-auth.port == "80"
  c-other:
    default: x
    when: vault.mode == "fast"
  d-deep:
    default: x
    when: base.level == "high"
  e-starter:
    default: x
    when: starter.region == "eu"
  f-own:
    default: x
    when: a-region == "x"
  g-itself:
    default: x
    when: g-itself == "x"
  h-readable:
    default: x
    when: store && debug && api-auth.method == "oauth" && api-auth.scope && tokens.ttl && starter.module != "x" && starter.go-version
`,
			})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\noptions:\n  module:\n    default: example.com/service\n  go-version:\n    default: \"1.25\"\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			canRead := "which the when of an option cannot read: it can read api-auth.method, api-auth.scope, debug, starter.go-version, starter.module, store, tokens.ttl"
			require.Equal(t, []string{
				`the feature cli-auth: the when "region == \"us\"" of the option "a-region" reads "region", ` + canRead,
				`the feature cli-auth: the when "api-auth.port == \"80\"" of the option "b-need" reads "api-auth.port", ` + canRead,
				`the feature cli-auth: the when "vault.mode == \"fast\"" of the option "c-other" reads "vault.mode", ` + canRead,
				`the feature cli-auth: the when "base.level == \"high\"" of the option "d-deep" reads "base.level", ` + canRead,
				`the feature cli-auth: the when "starter.region == \"eu\"" of the option "e-starter" reads "starter.region", which the starter service does not declare`,
				`the feature cli-auth: the when "a-region == \"x\"" of the option "f-own" reads "a-region", which has a when of its own, so the when of an option cannot read it`,
				`the feature cli-auth: the when "g-itself == \"x\"" of the option "g-itself" reads "g-itself", which has a when of its own, so the when of an option cannot read it`,
			}, mistakes)
		})

		t.Run("a starter option that a when of an option reads and one target does not declare is one mistake, which names that target", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"cli": "targets: [service, worker]\noptions:\n  mode:\n    default: x\n    when: starter.region == \"eu\"\n",
			})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\noptions:\n  region:\n    default: us\n")}
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
			files["starters/worker/files/go/main.go"] = &fstest.MapFile{Data: []byte("package main\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature cli: the when "starter.region == \"eu\"" of the option "mode" reads "starter.region", which the starter worker does not declare`,
			}, mistakes)
		})

		t.Run("each reference of a when of an option is checked one time: on both sides of && and ||, under !, in parentheses, in a comparison and as a bare name", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"cli": "targets: [service]\noptions:\n  mode:\n    default: x\n    when: '!(a && b == \"x\") || c != \"y\" || a'\n",
			})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature cli: the when "!(a && b == \"x\") || c != \"y\" || a" of the option "mode" reads "a", which the when of an option cannot read: it can read no option`,
				`the feature cli: the when "!(a && b == \"x\") || c != \"y\" || a" of the option "mode" reads "b", which the when of an option cannot read: it can read no option`,
				`the feature cli: the when "!(a && b == \"x\") || c != \"y\" || a" of the option "mode" reads "c", which the when of an option cannot read: it can read no option`,
			}, mistakes)
		})

		t.Run("the reads of the when of the options come after the reads of the when of the files entries and before the reads of the templates", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"cli": "targets: [service]\noptions:\n  mode:\n    default: x\n    when: b == \"x\"\nfiles:\n  - path: files/\n    when: a == \"x\"\n",
			})
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte(`{{ option "c" }}`)}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature cli: the when "a == \"x\"" of the files entry "files/" reads "a", which the feature cannot read: it can read mode`,
				`the feature cli: the when "b == \"x\"" of the option "mode" reads "b", which the when of an option cannot read: it can read no option`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "c", which the feature cannot read: it can read mode`,
			}, mistakes)
		})

		t.Run("a starter option that a target does not declare is one mistake of the feature, which names each target that does not declare it and no other starter, and a feature with no target in the catalog can read no starter option", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"cli":    "targets: [cron, ghost, service, worker]\nfiles:\n  - path: files/\n    when: 'starter.region == \"eu\" || starter.module == \"x\" || starter.go-version == \"1.25\"'\n",
				"orphan": "targets: [ghost]\nfiles:\n  - path: files/\n    when: 'starter.module == \"x\"'\n",
			})
			for starterName, options := range map[string]string{
				"api":     "  module:\n    default: example.com/api\n  region:\n    default: us\n",
				"cron":    "  go-version:\n    default: \"1.25\"\n",
				"service": "  go-version:\n    default: \"1.25\"\n  module:\n    default: example.com/service\n  region:\n    default: us\n",
				"worker":  "  go-version:\n    default: \"1.25\"\n  module:\n    default: example.com/worker\n",
			} {
				files["starters/"+starterName+"/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\noptions:\n" + options)}
				files["starters/"+starterName+"/files/go/main.go"] = &fstest.MapFile{Data: []byte("package main\n")}
			}
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte("package cli\n")}
			files["features/orphan/files/go/orphan.go"] = &fstest.MapFile{Data: []byte("package orphan\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature cli: the when "starter.region == \"eu\" || starter.module == \"x\" || starter.go-version == \"1.25\"" of the files entry "files/" reads "starter.region", which the starters cron, worker do not declare`,
				`the feature cli: the when "starter.region == \"eu\" || starter.module == \"x\" || starter.go-version == \"1.25\"" of the files entry "files/" reads "starter.module", which the starter cron does not declare`,
				`the feature orphan: the when "starter.module == \"x\"" of the files entry "files/" reads "starter.module", which the feature cannot read: it can read no option`,
			}, mistakes)
		})

		t.Run("each reference of a when is checked one time, also for a files entry with two templates: on both sides of && and ||, under !, in parentheses, in a comparison and as a bare name", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"cli": "targets: [service]\nfiles:\n  - path: files/\n    when: '!(a && b == \"x\") || c != \"y\" || a'\n",
			})
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte("package cli\n")}
			files["features/cli/files/go/flags.go"] = &fstest.MapFile{Data: []byte("package cli\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature cli: the when "!(a && b == \"x\") || c != \"y\" || a" of the files entry "files/" reads "a", which the feature cannot read: it can read no option`,
				`the feature cli: the when "!(a && b == \"x\") || c != \"y\" || a" of the files entry "files/" reads "b", which the feature cannot read: it can read no option`,
				`the feature cli: the when "!(a && b == \"x\") || c != \"y\" || a" of the files entry "files/" reads "c", which the feature cannot read: it can read no option`,
			}, mistakes)
		})

		t.Run("the mistakes of the reads of a feature come after its whens that do not parse, and two files entries with one when give one mistake each", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": "targets: [service]\nfiles:\n  - path: files/first/\n    when: mode\n  - path: files/broken/\n    when: 'mode =='\n  - path: files/second/\n    when: mode\n",
			})
			for _, entry := range []string{"first", "broken", "second"} {
				files["features/auth/files/"+entry+"/go/"+entry+".go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature auth: the when "mode ==" of the files entry "files/broken/": mode == needs a quoted value after it`,
				`the feature auth: the when "mode" of the files entry "files/first/" reads "mode", which the feature cannot read: it can read no option`,
				`the feature auth: the when "mode" of the files entry "files/second/" reads "mode", which the feature cannot read: it can read no option`,
			}, mistakes)
		})

		t.Run("each option call in a template of a feature that reads a reference that the feature cannot read is one mistake, wherever the call is in the parse tree, also in each define in the order of their names, and the message names the template, its files entry, the reference and the feature", func(t *testing.T) {
			files := withFeatures(map[string]string{"cli": "targets: [service]\noptions:\n  method:\n    default: api-key\nfiles:\n  - path: files/\n"})
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte(`{{ option "a" }}
{{ if eq (option "b") "x" }}{{ option "c" }}{{ else }}{{ option "d" }}{{ end }}
{{ range items "api.middleware" }}{{ option "e" }}{{ end }}
{{ with option "f" }}{{ . }}{{ end }}
{{ define "t" }}{{ option "q" }}{{ end }}{{ define "part" }}{{ option "g" }}{{ end }}{{ define "s" }}{{ option "p" }}{{ end }}{{ define "r" }}{{ option "o" }}{{ end }}
{{ githubRepository (option "h") }}
{{ template "part" }}{{ template "part" option "i" }}
{{ .Name | printf "%s-%s" (option "k") }}{{ (option "l").Size }}
{{ option "a" }}{{ option "method" }}{{ "j" | printf "%s" }}
`)}
			files[`features/cli/files/go/{{ option "path" }}.go`] = &fstest.MapFile{Data: []byte("package cli\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature cli: the template go/cli.go of the files entry files/ reads "a", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "b", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "c", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "d", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "e", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "f", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "h", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "i", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "k", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "l", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "g", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "o", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "p", which the feature cannot read: it can read method`,
				`the feature cli: the template go/cli.go of the files entry files/ reads "q", which the feature cannot read: it can read method`,
				`the feature cli: the template go/{{ option "path" }}.go of the files entry files/ reads "path", which the feature cannot read: it can read method`,
			}, mistakes)
		})

		t.Run("an option call whose argument is not quoted text, and a call whose function is not a name, read nothing", func(t *testing.T) {
			files := withFeatures(map[string]string{"cli": "targets: [service]\nfiles:\n  - path: files/\n"})
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte(`{{ option .Name }}{{ .Name "m" }}{{ option "n" }}` + "\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{`the feature cli: the template go/cli.go of the files entry files/ reads "n", which the feature cannot read: it can read no option`}, mistakes)
		})

		t.Run("a template of a feature is parsed with the delimiters of its feature, and its path with {{ }}", func(t *testing.T) {
			files := withFeatures(map[string]string{"tasks": `targets: [service]
delimiters:
  - glob: go/Taskfile*.yml
    left: '[['
    right: ']]'
files:
  - path: files/
`})
			files["features/tasks/files/go/Taskfile.tasks.yml"] = &fstest.MapFile{Data: []byte(`run: [[ option "x" ]] {{ option "y" }} {{.CLI_ARGS}}` + "\n")}
			files[`features/tasks/files/go/Taskfile.{{ option "w" }}.yml`] = &fstest.MapFile{Data: []byte("tasks: {}\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature tasks: the template go/Taskfile.tasks.yml of the files entry files/ reads "x", which the feature cannot read: it can read no option`,
				`the feature tasks: the template go/Taskfile.{{ option "w" }}.yml of the files entry files/ reads "w", which the feature cannot read: it can read no option`,
			}, mistakes)
		})

		t.Run("a template of a feature is checked whatever the when of its files entry", func(t *testing.T) {
			files := withFeatures(map[string]string{"tasks": "targets: [service]\noptions:\n  mode:\n    values: [fast]\n    default: fast\nfiles:\n  - path: files/slow/\n    when: mode == \"slow\"\n"})
			files["features/tasks/files/slow/go/slow.go"] = &fstest.MapFile{Data: []byte(`{{ option "z" }}` + "\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{`the feature tasks: the template go/slow.go of the files entry files/slow/ reads "z", which the feature cannot read: it can read mode`}, mistakes)
		})

		t.Run("a template of the starter that reads a reference other than an option of the starter is a mistake of the starter after its cycles, parsed with the delimiters of the starter, and the message names the starter, the template and the reference", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"a": "targets: [service]\nneeds: [b]\n",
				"b": "targets: [service]\nneeds: [a]\n",
			})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\noptions:\n  module:\n    default: example.com/service\n  region:\n    default: us\ndelimiters:\n  - glob: go/*.yml\n    left: '[['\n    right: ']]'\n")}
			files["starters/service/files/go/main.go"] = &fstest.MapFile{Data: []byte(`{{ option "starter.module" }} {{ option "cli.mode" }} {{ option "starter.port" }} {{ option "starter.region" }}` + "\n")}
			files["starters/service/files/go/ci.yml"] = &fstest.MapFile{Data: []byte(`ref: ${{ github.ref }} [[ option "ci.token" ]]` + "\n")}
			files["starters/bare/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
			files["starters/bare/files/go/main.go"] = &fstest.MapFile{Data: []byte(`{{ option "starter.module" }}` + "\n")}
			files["starters/one/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\noptions:\n  module:\n    default: example.com/one\n")}
			files["starters/one/files/go/main.go"] = &fstest.MapFile{Data: []byte(`{{ option "starter.region" }}` + "\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter bare: the template go/main.go reads "starter.module", which the starter cannot read: it can read no option`,
				`the starter one: the template go/main.go reads "starter.region", which the starter cannot read: it can read starter.module`,
				"the starter service: the needs of the features form a cycle: a needs b needs a",
				`the starter service: the template go/ci.yml reads "ci.token", which the starter cannot read: it can read starter.module, starter.region`,
				`the starter service: the template go/main.go reads "cli.mode", which the starter cannot read: it can read starter.module, starter.region`,
				`the starter service: the template go/main.go reads "starter.port", which the starter cannot read: it can read starter.module, starter.region`,
			}, mistakes)
		})

		t.Run("a template that does not parse is a mistake of its owner that names the template and the error, after the reads of the templates of that owner", func(t *testing.T) {
			files := withFeatures(map[string]string{"cli": "targets: [service]\noptions:\n  method:\n    default: api-key\nfiles:\n  - path: files/\n"})
			files["starters/service/files/go/bad.go"] = &fstest.MapFile{Data: []byte("{{ if }}\n")}
			files["starters/service/files/go/reads.go"] = &fstest.MapFile{Data: []byte(`{{ option "x" }}` + "\n")}
			files["features/cli/files/go/a.go"] = &fstest.MapFile{Data: []byte("{{ nope }}\n")}
			files["features/cli/files/go/c.go"] = &fstest.MapFile{Data: []byte(`{{ option "mode" }}` + "\n")}
			files[`features/cli/files/go/{{ if }}.go`] = &fstest.MapFile{Data: []byte("package cli\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the template go/reads.go reads "x", which the starter cannot read: it can read no option`,
				`the starter service: the template go/bad.go does not parse: template: go/bad.go:1: missing value for if`,
				`the feature cli: the template go/c.go of the files entry files/ reads "mode", which the feature cannot read: it can read method`,
				`the feature cli: the template go/a.go of the files entry files/ does not parse: template: go/a.go:1: function "nope" not defined`,
				`the feature cli: the template go/{{ if }}.go of the files entry files/ does not parse: template: go/{{ if }}.go:1: missing value for if`,
			}, mistakes)
		})

		t.Run("a template that calls each template function of new parses", func(t *testing.T) {
			files := withFeatures(map[string]string{"cli": "targets: [service]\noptions:\n  method:\n    default: api-key\nfiles:\n  - path: files/\n"})
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte(`{{ option "method" }}{{ items "api.middleware" }}{{ unique "code" "api.middleware" }}{{ indent 2 "x" }}{{ githubRepository "github.com/a/b" }}{{ kebab "A" }}{{ snake "A" }}{{ camel "A" }}{{ pascal "A" }}{{ upper "a" }}` + "\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Empty(t, mistakes)
		})

		t.Run("the reads of the templates of a feature come after the reads of its when expressions", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": "targets: [service]\nfiles:\n  - path: files/a/\n  - path: files/b/\n    when: mode\n"})
			files["features/auth/files/a/go/a.go"] = &fstest.MapFile{Data: []byte(`{{ option "region" }}` + "\n")}
			files["features/auth/files/b/go/b.go"] = &fstest.MapFile{Data: []byte("package auth\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature auth: the when "mode" of the files entry "files/b/" reads "mode", which the feature cannot read: it can read no option`,
				`the feature auth: the template go/a.go of the files entry files/a/ reads "region", which the feature cannot read: it can read no option`,
			}, mistakes)
		})

		t.Run("an extension point of the starter whose file is not a template of the starter, also a file that only a feature has, or that names no file, is a mistake of the starter after its templates, and a template with .tmpl after the file matches", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": "targets: [service]\nfiles:\n  - path: files/\n"})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  a.missing:\n    file: go/missing.go\n    item: [code]\n  b.empty:\n    item: [code]\n  c.feature:\n    file: go/auth.go\n    item: [code]\n  d.module:\n    file: go/go.mod\n    item: [code]\n")}
			files["starters/service/files/go/go.mod.tmpl"] = rendersItems("d.module")
			files["starters/service/files/go/main.go"] = &fstest.MapFile{Data: []byte(`{{ option "auth.method" }}` + "\n")}
			files["features/auth/files/go/auth.go"] = rendersItems("c.feature")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the template go/main.go reads "auth.method", which the starter cannot read: it can read no option`,
				`the starter service: the file "go/missing.go" of the extension point "a.missing" does not exist`,
				`the starter service: the extension point "b.empty" names no file`,
				`the starter service: the file "go/auth.go" of the extension point "c.feature" does not exist`,
			}, mistakes)
		})

		t.Run("an extension point of a feature whose file is not a template of the feature in any files entry is one mistake of the feature, also with two targets", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"cli": "targets: [service, worker]\nextension_points:\n  cli.commands:\n    file: go/cli.go\n    item: [code]\n  cli.flags:\n    file: go/flags.go\n    item: [name]\nfiles:\n  - path: files/common/\n  - path: files/extra/\n",
			})
			files["starters/worker/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
			files["starters/worker/files/go/main.go"] = &fstest.MapFile{Data: []byte("package main\n")}
			files["features/cli/files/common/go/root.go"] = &fstest.MapFile{Data: []byte("package cli\n")}
			files["features/cli/files/extra/go/flags.go"] = rendersItems("cli.flags")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{`the feature cli: the file "go/cli.go" of the extension point "cli.commands" does not exist`}, mistakes)
		})

		t.Run("each copy of the file of an extension point in the files entries of a feature must call items, and the message names the files entry of each copy that does not", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": "targets: [service]\noptions:\n  method:\n    values: [api-key, oauth, sso]\n    default: api-key\nextension_points:\n  auth.checks:\n    file: go/checks.go\n    item: [code]\nfiles:\n  - path: files/api-key/\n    when: method == \"api-key\"\n  - path: files/oauth/\n    when: method == \"oauth\"\n  - path: files/sso/\n    when: method == \"sso\"\n",
			})
			files["features/auth/files/api-key/go/checks.go"] = rendersItems("auth.checks")
			files["features/auth/files/oauth/go/checks.go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			files["features/auth/files/sso/go/checks.go"] = &fstest.MapFile{Data: []byte("package auth\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature auth: the template go/checks.go of the files entry files/oauth/ does not call items "auth.checks"`,
				`the feature auth: the template go/checks.go of the files entry files/sso/ does not call items "auth.checks"`,
			}, mistakes)
		})

		t.Run("the mistakes of the files of the extension points of a feature come in the order of the names of the extension points", func(t *testing.T) {
			files := withFeatures(map[string]string{"cli": "targets: [service]\nextension_points:\n  cli.hooks:\n    file: go/hooks.go\n    item: [code]\n  cli.flags:\n    file: go/flags.go\n    item: [name]\n  cli.args:\n    file: go/args.go\n    item: [name]\n  cli.env:\n    file: go/env.go\n    item: [name]\n  cli.commands:\n    file: go/commands.go\n    item: [code]\n"})

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the feature cli: the file "go/args.go" of the extension point "cli.args" does not exist`,
				`the feature cli: the file "go/commands.go" of the extension point "cli.commands" does not exist`,
				`the feature cli: the file "go/env.go" of the extension point "cli.env" does not exist`,
				`the feature cli: the file "go/flags.go" of the extension point "cli.flags" does not exist`,
				`the feature cli: the file "go/hooks.go" of the extension point "cli.hooks" does not exist`,
			}, mistakes)
		})

		t.Run("a file whose parse tree does not call items with the name of its extension point is a mistake, also when it calls items for another extension point or only unique for it, and it is parsed with the delimiters of its declarer", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"cli": "targets: [service]\nextension_points:\n  cli.commands:\n    file: go/cli.go\n    item: [code]\n  cli.flags:\n    file: go/cli.go\n    item: [name]\nfiles:\n  - path: files/\n",
			})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\ndelimiters:\n  - glob: go/*.yml\n    left: '[['\n    right: ']]'\nextension_points:\n  api.config:\n    file: go/server.go\n    item: [code]\n  api.middleware:\n    file: go/server.go\n    item: [code]\n  ci.jobs:\n    file: go/ci.yml\n    item: [text]\n  ci.steps:\n    file: go/ci.yml\n    item: [text]\n")}
			files["starters/service/files/go/server.go"] = &fstest.MapFile{Data: []byte(`{{ range items "api.config" }}{{ .code }}{{ end }}{{ range unique "code" "api.middleware" }}{{ . }}{{ end }}` + "\n")}
			files["starters/service/files/go/ci.yml"] = &fstest.MapFile{Data: []byte(`[[ range items "ci.jobs" ]][[ .text ]][[ end ]] {{ range items "ci.steps" }}{{ .text }}{{ end }}` + "\n")}
			files["features/cli/files/go/cli.go"] = rendersItems("cli.flags")

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the template go/server.go does not call items "api.middleware"`,
				`the starter service: the template go/ci.yml does not call items "ci.steps"`,
				`the feature cli: the template go/cli.go of the files entry files/ does not call items "cli.commands"`,
			}, mistakes)
		})

		t.Run("a file of an extension point that does not parse gives only its parse mistake, and the mistakes of the files of a feature come after its parse mistakes", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"cli": "targets: [service]\nextension_points:\n  cli.commands:\n    file: go/cli.go\n    item: [code]\n  cli.flags:\n    file: go/flags.go\n    item: [name]\nfiles:\n  - path: files/\n",
			})
			files["starters/service/files/go/server.go"] = &fstest.MapFile{Data: []byte("{{ if }}\n")}
			files["features/cli/files/go/cli.go"] = &fstest.MapFile{Data: []byte("{{ end }}\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				`the starter service: the template go/server.go does not parse: template: go/server.go:1: missing value for if`,
				`the feature cli: the template go/cli.go of the files entry files/ does not parse: template: go/cli.go:1: unexpected {{end}}`,
				`the feature cli: the file "go/flags.go" of the extension point "cli.flags" does not exist`,
			}, mistakes)
		})

		t.Run("a path of the repo that the starter and a feature, or two features, write is one mistake for each pair of owners after the other mistakes of the starter, and the message names the starter, the path and both owners", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": "targets: [service]\nfiles:\n  - path: files/\n",
				"cli":  "targets: [service]\nfiles:\n  - path: files/\n",
				"x":    "targets: [service]\nneeds: [y]\n",
				"y":    "targets: [service]\nneeds: [x]\n",
			})
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code, import]\n  api.routes:\n    file: go/routes.go\n    item: [path]\n")}
			files["starters/service/files/go/main.go"] = &fstest.MapFile{Data: []byte(`{{ option "auth.method" }}` + "\n")}
			files["starters/service/files/go/go.mod.tmpl"] = &fstest.MapFile{Data: []byte("module example.com/service\n")}
			files["starters/service/files/go/README.md"] = &fstest.MapFile{Data: []byte("# Service\n")}
			files["features/auth/files/go/go.mod"] = &fstest.MapFile{Data: []byte("module example.com/auth\n")}
			files["features/auth/files/go/README.md"] = &fstest.MapFile{Data: []byte("# Auth\n")}
			files["features/auth/files/go/internal/shared.go"] = &fstest.MapFile{Data: []byte("package internal\n")}
			files["features/cli/files/go/README.md"] = &fstest.MapFile{Data: []byte("# CLI\n")}
			files["features/cli/files/go/internal/shared.go"] = &fstest.MapFile{Data: []byte("package internal\n")}
			files["features/cli/files/go/internal/cli.go"] = &fstest.MapFile{Data: []byte("package internal\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the needs of the features form a cycle: x needs y needs x",
				`the starter service: the template go/main.go reads "auth.method", which the starter cannot read: it can read no option`,
				`the starter service: the file "go/routes.go" of the extension point "api.routes" does not exist`,
				"the starter service: the starter and the feature auth both write README.md",
				"the starter service: the starter and the feature cli both write README.md",
				"the starter service: the feature auth and the feature cli both write README.md",
				"the starter service: the starter and the feature auth both write go.mod",
				"the starter service: the feature auth and the feature cli both write internal/shared.go",
			}, mistakes)
		})

		t.Run("two templates of one owner that write one path are not a mistake, and the templates of each files entry count whatever its when", func(t *testing.T) {
			files := withFeatures(map[string]string{
				"auth": "targets: [service]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\nfiles:\n  - path: files/api-key/\n    when: method == \"api-key\"\n  - path: files/oauth/\n    when: method == \"oauth\"\n",
				"cli":  "targets: [service]\noptions:\n  mode:\n    values: [fast]\n    default: fast\nfiles:\n  - path: files/slow/\n    when: mode == \"slow\"\n",
			})
			files["features/auth/files/api-key/go/config.go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			files["features/auth/files/oauth/go/config.go"] = &fstest.MapFile{Data: []byte("package auth\n")}
			files["features/cli/files/slow/go/config.go"] = &fstest.MapFile{Data: []byte("package cli\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{"the starter service: the feature auth and the feature cli both write config.go"}, mistakes)
		})

		t.Run("two folders whose paths lead to one path give one path of the repo, and two features with no target in common or a template in no folder are not compared", func(t *testing.T) {
			files := withWorker(withFeatures(map[string]string{
				"a":    "targets: [service]\nfiles:\n  - path: files/\n",
				"b":    "targets: [worker]\nfiles:\n  - path: files/\n",
				"c":    "targets: [service]\nfiles:\n  - path: files/\n",
				"d":    "targets: [service]\nfiles:\n  - path: files/\n",
				"docs": "targets: [service]\nfiles:\n  - path: files/\n",
			}))
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n  docs:\n    path: ./guide/..\n")}
			files["starters/service/files/go/README.md"] = &fstest.MapFile{Data: []byte("# Service\n")}
			files["features/a/files/go/x.go"] = &fstest.MapFile{Data: []byte("package x\n")}
			files["features/b/files/go/x.go"] = &fstest.MapFile{Data: []byte("package x\n")}
			files["features/c/files/NOTES.md"] = &fstest.MapFile{Data: []byte("c\n")}
			files["features/d/files/NOTES.md"] = &fstest.MapFile{Data: []byte("d\n")}
			files["features/docs/files/docs/README.md"] = &fstest.MapFile{Data: []byte("# Docs\n")}

			mistakes, err := catalog.New(files).Check()

			require.NoError(t, err)
			require.Equal(t, []string{
				"the starter service: the template NOTES.md of the files entry files/ of the feature c is not in a folder",
				"the starter service: the template NOTES.md of the files entry files/ of the feature d is not in a folder",
				"the starter service: the starter and the feature docs both write README.md",
			}, mistakes)
		})

		t.Run("the catalog of this repo has no mistake", func(t *testing.T) {
			mistakes, err := repoCatalog().Check()

			require.NoError(t, err)
			require.Empty(t, mistakes)
		})

		t.Run("two files entries of a feature with one path stop the check and the read of the starter, and the message names the feature.yml and the path", func(t *testing.T) {
			files := withFeatures(map[string]string{"auth": "targets: [service]\nfiles:\n  - path: files/\n    when: a\n  - path: files\n    when: b\n"})
			files["features/auth/files/go/auth.go"] = &fstest.MapFile{Data: []byte("package auth\n")}

			_, checkErr := catalog.New(files).Check()
			_, starterErr := catalog.New(files).Starter("service")

			for _, err := range []error{checkErr, starterErr} {
				require.EqualError(t, err, `read features/auth/feature.yml: files names the path "files" two times`)
			}
		})

		t.Run("a starter.yml that is not valid YAML stops the check, and the message names the file", func(t *testing.T) {
			files := withFeatures(nil)
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders: [go\n")}

			_, err := catalog.New(files).Check()

			require.ErrorContains(t, err, "starters/service/starter.yml")
		})

		for _, failing := range []string{"starters", "starters/service/starter.yml", "features", "features/auth/feature.yml"} {
			t.Run("a read of "+failing+" that fails stops the check with its cause", func(t *testing.T) {
				files := failingFS{failing: failing, files: withFeatures(map[string]string{"auth": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newAuthCommand()\n"})}

				_, err := catalog.New(files).Check()

				require.ErrorContains(t, err, "disk failure")
			})
		}
	})

	t.Run("embedded", func(t *testing.T) {
		writtenStarter := func(t *testing.T, given map[string]string) (string, map[string]string) {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", given, nil, catalog.Answers{})
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
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", map[string]string{"module": "example.com/acme ledger;rm -rf ~"}, nil, catalog.Answers{})

			require.ErrorContains(t, err, "module")
		})

		t.Run("a name with capitals and dots gives a kebab-case name in the Docker names of the repo", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("MyService.v2", nil, nil, catalog.Answers{})
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

		t.Run("the README of the repo tells how a web app on another origin calls the API: the routes under /api, ALLOWED_ORIGINS, the form of an origin, the default origins, and the local API with the setting", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			for _, part := range []string{
				"## Call the API From a Web App", "`POST /api/accounts`", "`ALLOWED_ORIGINS`", "`<scheme>://<host>[:<port>]`",
				"`http://localhost:8080` and `http://127.0.0.1:8080`", "ALLOWED_ORIGINS=https://app.example.com task api:run",
			} {
				require.Contains(t, files["README.md"], part)
			}
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

			processManager := regexp.MustCompile(`^(internal/use_cases/payment/|internal/api/use_cases/(payment|webhook)/|internal/use_cases/account/(holds/|hold(_test)?\.go$|event/hold_|command/[^/]*_hold\.go$)|cmd/fake_services/|Taskfile\.fake_services\.yml$|Dockerfile\.fake_services$|migrations/[^/]*_create-payment(s|-deadlines)-table\.|docs/process-manager-example\.md$|docs/event-model-payment\.)`)
			for path, contents := range files {
				require.False(t, processManager.MatchString(path), path)
				require.NotContains(t, contents, "fake_services", path)
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

		writtenWithGuard := func(t *testing.T) map[string]string {
			t.Helper()
			files := catalogOnDisk(t)
			files["features/guard/feature.yml"] = &fstest.MapFile{Data: []byte(`summary: Guard.
targets: [go-event-sourcing-service]
add_to:
  api.middleware:
    - code: guard.Middleware(cfg.Guard)
      import: internal/guard
    - code: guard.Audit(cfg.Guard)
      import: internal/guard
  api.config:
    - name: Guard
      type: guard.Config
      import: internal/guard
  catchup.config:
    - name: Guard
      type: guard.Config
      import: internal/guard
    - name: GuardAudit
      type: guard.AuditConfig
      import: internal/guard
  catchup.consumers:
    - code: guardWorkers(cfg.Guard, logger)
  api.workers:
    - name: guardWorker
      code: newGuardWorker(cfg.Guard, logger)
    - name: auditWorker
      code: newAuditWorker(pool, reg, bus)
  api.routes:
    - code: guardWorker.routables()
files:
  - path: files/
`)}
			files["features/guard/files/go/cmd/catchup/guard.go.tmpl"] = &fstest.MapFile{Data: []byte("package main\n")}
			starter, err := catalog.New(files).Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues("guard"), catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		t.Run("the server gives the items of api.middleware, in their order, to the protected routes, and imports their package one time", func(t *testing.T) {
			files := writtenWithGuard(t)

			require.Contains(t, files["internal/api/server/server.go"], "\t\tguard.Middleware(cfg.Guard),\n\t\tguard.Audit(cfg.Guard),\n\t}")
			require.Equal(t, 1, strings.Count(files["internal/api/server/server.go"], `"github.com/hpcsc/my-service/internal/guard"`))
		})

		t.Run("the config of the API has a field for each item of api.config, and imports its package one time", func(t *testing.T) {
			files := writtenWithGuard(t)

			require.Contains(t, files["internal/api/config.go"], "\tGuard guard.Config\n}")
			require.Equal(t, 1, strings.Count(files["internal/api/config.go"], `"github.com/hpcsc/my-service/internal/guard"`))
		})

		t.Run("the config of catchup has a field for each item of catchup.config, and imports their package one time", func(t *testing.T) {
			files := writtenWithGuard(t)

			require.Contains(t, files["cmd/catchup/main.go"], "type Config struct {\n\tDB config.DB\n\tGuard guard.Config\n\tGuardAudit guard.AuditConfig\n}")
			require.Equal(t, 1, strings.Count(files["cmd/catchup/main.go"], `"github.com/hpcsc/my-service/internal/guard"`))
		})

		t.Run("catchup appends the workers of each item of catchup.consumers to the workers of the starter, and runs them all", func(t *testing.T) {
			files := writtenWithGuard(t)

			require.Contains(t, files["cmd/catchup/guard.go"], "package main")
			require.Contains(t, files["cmd/catchup/main.go"], "\t}\n\tall = append(all, guardWorkers(cfg.Guard, logger)...)\n\treturn all\n}")
			require.Contains(t, files["cmd/catchup/main.go"], "run(ctx, workers(pool, cfg, logger), lock.NewPostgres(pool), logger)")
		})

		t.Run("Register declares a variable for each item of api.workers in their order, an item of api.routes can use it, and Register gives each worker to the server", func(t *testing.T) {
			files := writtenWithGuard(t)

			require.Contains(t, files["internal/api/use_cases/register.go"], "\tcoreUseCases.RegisterHandlers(bus, eventStream)\n\tguardWorker := newGuardWorker(cfg.Guard, logger)\n\tauditWorker := newAuditWorker(pool, reg, bus)\n\n\troutables := []route.Routable{")
			require.Contains(t, files["internal/api/use_cases/register.go"], "\t}\n\troutables = append(routables, guardWorker.routables()...)\n")
			require.Contains(t, files["internal/api/use_cases/register.go"], "\treturn []api.Worker{\n\t\tguardWorker,\n\t\tauditWorker,\n\t}\n}")
			require.Contains(t, files["internal/api/server/server.go"], "\tworkers := use_cases.Register(r, protectedMiddleware(cfg), cfg, pool, logger)\n")
		})

		t.Run("with no item, catchup has only the config and the workers of the starter", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.Contains(t, files["cmd/catchup/main.go"], "\t\"github.com/hpcsc/my-service/internal/use_cases/account/welcome_bonus\"\n\t\"github.com/jackc/pgx/v5/pgxpool\"\n)")
			require.Contains(t, files["cmd/catchup/main.go"], "type Config struct {\n\tDB config.DB\n}")
			require.Contains(t, files["cmd/catchup/main.go"], "\tall := []worker{\n\t\tprojection.NewProjector(balance.NewProjection(), pool, checkpointStore, eventSubscription, logger),\n\t\treaction.NewReactor(reaction.NewConflictRetry(welcome_bonus.NewReaction(bus), 3), checkpointStore, eventSubscription, logger),\n\t}\n\treturn all\n}")
		})

		t.Run("with no item, the server gives no middleware to the protected routes, and the config has only its own fields", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.Contains(t, files["internal/api/server/server.go"], "return []func(http.Handler) http.Handler{\n\t}")
			require.Contains(t, files["internal/api/server/server.go"], "\t\"github.com/hpcsc/my-service/internal/api/use_cases\"\n\t\"github.com/jackc/pgx/v5/pgxpool\"\n)")
			require.Contains(t, files["internal/api/config.go"], "import (\n\t\"github.com/hpcsc/my-service/internal/common/config\"\n)\n")
			require.Contains(t, files["internal/api/config.go"], "type Config struct {\n\tPort           string   `env:\"PORT\" envDefault:\"3333\"`\n\tAllowedOrigins []string `env:\"ALLOWED_ORIGINS\" envSeparator:\",\"`\n\tDB             config.DB\n}")
		})

		writtenWithProcessManager := func(t *testing.T) map[string]string {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues("process-manager"), catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		writtenWithPushOf := func(t *testing.T, values map[string]string) map[string]string {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"push": values}, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}
		writtenWithPush := func(t *testing.T) map[string]string {
			t.Helper()
			return writtenWithPushOf(t, nil)
		}

		t.Run("no written file holds the name of an extension point of the starter, with or without items", func(t *testing.T) {
			definitions, err := repoCatalog().Definitions()
			require.NoError(t, err)
			extensionPoints := slices.Sorted(maps.Keys(definitions.Starters["go-event-sourcing-service"].ExtensionPoints))
			require.Subset(t, extensionPoints, []string{"api.middleware", "catchup.consumers", "use_cases.handlers"})
			_, withoutItems := writtenStarter(t, nil)

			for _, files := range []map[string]string{withoutItems, writtenWithGuard(t), writtenWithProcessManager(t), writtenWithPush(t), writtenWithPushOf(t, map[string]string{"delivery": "nats"})} {
				for path, contents := range files {
					for _, extensionPoint := range extensionPoints {
						require.NotContains(t, contents, extensionPoint, path)
					}
				}
			}
		})

		t.Run("process-manager adds the files of its folder, and of the starter it changes only the file of each extension point that it adds items to", func(t *testing.T) {
			definitions, err := repoCatalog().Definitions()
			require.NoError(t, err)
			extensionPoints := definitions.Starters["go-event-sourcing-service"].ExtensionPoints
			var extended []string
			for name := range definitions.Features["process-manager"].AddTo {
				extended = append(extended, strings.TrimPrefix(extensionPoints[name].File, "go/"))
			}
			var featureFiles []string
			featureFolder := filepath.Join(repoCatalogDir, "features", "process-manager", "files", "go")
			require.NoError(t, filepath.WalkDir(featureFolder, func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				relative, err := filepath.Rel(featureFolder, path)
				featureFiles = append(featureFiles, strings.TrimSuffix(filepath.ToSlash(relative), ".tmpl"))
				return err
			}))
			_, withoutFeature := writtenStarter(t, nil)

			withFeature := writtenWithProcessManager(t)

			var added, changed []string
			for path, contents := range withFeature {
				before, ok := withoutFeature[path]
				switch {
				case !ok:
					added = append(added, path)
				case before != contents:
					changed = append(changed, path)
				}
			}
			require.Contains(t, featureFiles, "internal/use_cases/account/hold.go")
			require.ElementsMatch(t, featureFiles, added)
			require.ElementsMatch(t, slices.Compact(slices.Sorted(slices.Values(extended))), changed)
			require.Len(t, withFeature, len(withoutFeature)+len(added))
		})

		t.Run("with process-manager, the account keeps a hold for each payment, registers the hold events, takes a captured hold off the balance, the registry knows the payment events, and the bus takes the hold commands and the payment commands", func(t *testing.T) {
			files := writtenWithProcessManager(t)

			require.Contains(t, files["internal/use_cases/account/account.go"], "\tclosed              bool\n\tholds map[uuid.UUID]hold\n}")
			require.Contains(t, files["internal/use_cases/account/account.go"], "\t\ta.closed = true\n\t}\n\ta.applyHold(e)\n}")
			require.Contains(t, files["internal/use_cases/account/events.go"], "\treg.Register[event.AccountClosed]()\n\treg.Register[event.HoldPlaced]()\n\treg.Register[event.HoldRejected]()\n\treg.Register[event.HoldReleased]()\n\treg.Register[event.HoldCaptured]()\n}")
			require.Contains(t, files["internal/use_cases/account/balance/projection.go"], "\t\tevent.AccountClosed{}.EventType(),\n\t\tevent.HoldCaptured{}.EventType(),\n\t}")
			require.Contains(t, files["internal/use_cases/account/balance/projection.go"], "\t\treturn err\n\tcase *event.HoldCaptured:\n\t\treturn p.change(ctx, tx, payload.AccountID, -payload.Amount, evt)\n\tdefault:")
			require.Contains(t, files["internal/use_cases/register.go"], "\taccount.RegisterEvents(reg)\n\tregisterPaymentEvents(reg)\n}")
			require.Contains(t, files["internal/use_cases/register.go"], "\tclose_account.Register(bus, accounts)\n\tregisterPaymentHandlers(bus, stream, accounts)\n}")
		})

		t.Run("with process-manager, catchup reads the config of the outside systems and runs the workers of the payment", func(t *testing.T) {
			files := writtenWithProcessManager(t)

			require.Contains(t, files["cmd/catchup/main.go"], "\tDB config.DB\n\tOutside outside.Config\n}")
			require.Equal(t, 1, strings.Count(files["cmd/catchup/main.go"], `"github.com/hpcsc/my-service/internal/use_cases/payment/outside"`))
			require.Contains(t, files["cmd/catchup/main.go"], "\tall = append(all, paymentWorkers(pool, cfg, logger, bus, eventStream, checkpointStore, eventSubscription)...)\n")
			var reactors []string
			for _, line := range strings.Split(files["cmd/catchup/payment.go"], "\n") {
				if strings.Contains(line, "reaction.NewReactor(") {
					reactors = append(reactors, line)
				}
			}
			require.Len(t, reactors, 5)
			for _, reactor := range reactors {
				require.Contains(t, reactor, "reaction.NewReactor(reaction.NewConflictRetry(")
				require.True(t, strings.HasSuffix(reactor, ", 3), checkpointStore, eventSubscription, logger, reaction.StartAtBeginning()),"), reactor)
			}
		})

		t.Run("with process-manager, the migrations create the tables payments and payment_deadlines, and drop them again", func(t *testing.T) {
			files := writtenWithProcessManager(t)

			require.Contains(t, files["migrations/20261005124000_create-payments-table.up.sql"], "CREATE TABLE IF NOT EXISTS payments (")
			require.Contains(t, files["migrations/20261005124000_create-payments-table.down.sql"], "DROP TABLE IF EXISTS payments;")
			require.Contains(t, files["migrations/20261005124100_create-payment-deadlines-table.up.sql"], "CREATE TABLE IF NOT EXISTS payment_deadlines (")
			require.Contains(t, files["migrations/20261005124100_create-payment-deadlines-table.down.sql"], "DROP TABLE IF EXISTS payment_deadlines;")
		})

		t.Run("each migration of a feature that targets the starter has a version after each migration of the starter", func(t *testing.T) {
			definitions, err := repoCatalog().Definitions()
			require.NoError(t, err)
			versionsUnder := func(t *testing.T, folder string) []string {
				t.Helper()
				var versions []string
				require.NoError(t, fs.WalkDir(os.DirFS(repoCatalogDir), folder, func(path string, entry fs.DirEntry, err error) error {
					if err == nil && !entry.IsDir() && strings.Contains(path, "/migrations/") {
						versions = append(versions, strings.SplitN(entry.Name(), "_", 2)[0])
					}
					return err
				}))
				return versions
			}
			latestOfStarter := slices.Max(versionsUnder(t, "starters/go-event-sourcing-service"))
			featureVersions := map[string][]string{}
			for featureName, feature := range definitions.Features {
				if slices.Contains(feature.Targets, "go-event-sourcing-service") {
					featureVersions[featureName] = versionsUnder(t, "features/"+featureName)
				}
			}

			require.NotEmpty(t, featureVersions["process-manager"])
			for featureName, versions := range featureVersions {
				for _, version := range versions {
					require.Greater(t, version, latestOfStarter, featureName)
				}
			}
		})

		t.Run("with process-manager, the API serves the routes of the payment and of the webhooks after the routes of the starter", func(t *testing.T) {
			files := writtenWithProcessManager(t)

			require.Contains(t, files["internal/api/use_cases/register.go"], "\t\tapiAccount.NewHandler(bus, balance.NewQuery(pool)),\n\t}\n\troutables = append(routables, paymentRoutables(bus, pool)...)\n")
		})

		t.Run("with no item, the API serves only the routes of the starter and has no worker", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.Contains(t, files["internal/api/use_cases/register.go"], "\tcoreUseCases.RegisterHandlers(bus, eventStream)\n\n\troutables := []route.Routable{\n\t\troot.NewHandler(),\n\t\tapiAccount.NewHandler(bus, balance.NewQuery(pool)),\n\t}\n\n\tvar publicRoutes, protectedRoutes []*route.Route\n")
			require.Contains(t, files["internal/api/use_cases/register.go"], "\treturn []api.Worker{\n\t}\n}")
		})

		parsedYAML := func(t *testing.T, contents string) map[string]any {
			t.Helper()
			var parsed map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(contents), &parsed))
			return parsed
		}

		t.Run("with process-manager, task run starts the fakes beside the API and catchup, and Docker Compose builds the fakes and gives catchup their URLs", func(t *testing.T) {
			files := writtenWithProcessManager(t)

			require.Contains(t, files["Taskfile.yml"], "  catchup: Taskfile.catchup.yml\n  fake_services: Taskfile.fake_services.yml\n")
			require.Equal(t, "Taskfile.fake_services.yml", parsedYAML(t, files["Taskfile.yml"])["includes"].(map[string]any)["fake_services"])
			require.Equal(t, []any{"go run ./cmd/fake_services"}, parsedYAML(t, files["Taskfile.fake_services.yml"])["tasks"].(map[string]any)["run"].(map[string]any)["cmds"])
			require.Equal(t, []any{"task api:run", "task catchup:run", "task fake_services:run"}, parsedYAML(t, files[".tmuxp.yaml"])["windows"].([]any)[0].(map[string]any)["panes"])
			services := parsedYAML(t, files["docker-compose.local.yml"])["services"].(map[string]any)
			require.Equal(t, map[string]any{
				"DB_NAME": "${DB_NAME}", "DB_USERNAME": "${DB_USERNAME}", "DB_PASSWORD": "${DB_PASSWORD}", "DB_HOST_PORT": 5432, "DB_HOST": "db",
				"FRAUD_SERVICE_URL":     "http://fake_services:4444/fraud",
				"SANCTIONS_SERVICE_URL": "http://fake_services:4444/sanctions",
				"PAYMENT_NETWORK_URL":   "http://fake_services:4444/network",
				"WEBHOOK_URL":           "http://api:3333/api/webhooks",
			}, services["catchup"].(map[string]any)["environment"])
			require.Equal(t, map[string]any{"build": map[string]any{"dockerfile": "Dockerfile.fake_services"}}, services["fake_services"])
			require.Contains(t, files["Dockerfile.fake_services"], "FROM golang:1.27.1 AS build-env")
		})

		t.Run("with process-manager, no file of the fakes reads, holds or sends a token", func(t *testing.T) {
			files := writtenWithProcessManager(t)

			fakes := []string{"cmd/fake_services/main.go", "internal/use_cases/payment/outside/fake_system.go", "internal/use_cases/payment/checks/fake_services.go", "internal/use_cases/payment/submission/fake_network.go", "Taskfile.fake_services.yml", "Dockerfile.fake_services"}
			for _, path := range fakes {
				require.Contains(t, files, path)
				for _, word := range []string{"token", "bearer", "authorization"} {
					require.NotContains(t, strings.ToLower(files[path]), word, path)
				}
			}
		})

		t.Run("with no item, the Taskfile, the tmux window and Docker Compose have only the values of the starter", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.Equal(t, map[string]any{"migrate": "Taskfile.migrate.yml", "test": "Taskfile.test.yml", "api": "Taskfile.api.yml", "catchup": "Taskfile.catchup.yml"}, parsedYAML(t, files["Taskfile.yml"])["includes"])
			require.Contains(t, files["Taskfile.yml"], "  catchup: Taskfile.catchup.yml\n\ntasks:\n")
			require.Equal(t, []any{"task api:run", "task catchup:run"}, parsedYAML(t, files[".tmuxp.yaml"])["windows"].([]any)[0].(map[string]any)["panes"])
			require.True(t, strings.HasSuffix(files[".tmuxp.yaml"], "      - task catchup:run\n"))
			services := parsedYAML(t, files["docker-compose.local.yml"])["services"].(map[string]any)
			require.ElementsMatch(t, []string{"api", "catchup"}, slices.Collect(maps.Keys(services)))
			require.Len(t, services["catchup"].(map[string]any)["environment"], 5)
			require.True(t, strings.HasSuffix(files["docker-compose.local.yml"], "    environment:\n      <<: *x-db-config\n"))
		})

		namedPathsAreIn := func(t *testing.T, text string, files map[string]string) {
			t.Helper()
			for _, path := range regexp.MustCompile("`([A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+)`").FindAllStringSubmatch(text, -1) {
				named := strings.TrimSuffix(path[1], "/")
				require.True(t, slices.ContainsFunc(slices.Collect(maps.Keys(files)), func(written string) bool {
					return written == named || strings.HasPrefix(written, named+"/")
				}), path[1])
			}
		}

		t.Run("with process-manager, the tasks pay and get-payment of the API call the payment routes with no credential, beside the tasks of the starter", func(t *testing.T) {
			files := writtenWithProcessManager(t)
			_, starterAlone := writtenStarter(t, nil)

			tasks := parsedYAML(t, files["Taskfile.api.yml"])["tasks"].(map[string]any)

			require.Subset(t, slices.Collect(maps.Keys(tasks)), slices.Collect(maps.Keys(parsedYAML(t, starterAlone["Taskfile.api.yml"])["tasks"].(map[string]any))))
			pay := tasks["pay"].(map[string]any)["cmds"].([]any)[0].(string)
			require.Contains(t, pay, `curl -s -X POST "{{.API}}/payments"`)
			require.Contains(t, pay, `{{default "300" .CLI_ARGS}}`)
			require.Equal(t, []any{`curl -s "{{.API}}/payments/${PAYMENT}" | jq`}, tasks["get-payment"].(map[string]any)["cmds"])
			require.NotContains(t, files["Taskfile.api.yml"], "AUTH")
		})

		t.Run("with process-manager, the README ends with the section of the example, which tells how to run a payment, the paths of the fakes, that the fakes send no credential, and only files that the repo has", func(t *testing.T) {
			files := writtenWithProcessManager(t)
			readme := files["README.md"]
			section := readme[strings.Index(readme, "\n## Process Manager Example\n"):]

			require.NotContains(t, section[1:], "\n## ")
			for _, part := range []string{
				"task api:pay -- 300", "task api:get-payment", "| IBAN that ends in `4444` |", "| Amount above 10000 |",
				"`internal/use_cases/payment.go`, `cmd/catchup/payment.go` and `internal/api/use_cases/payment.go`",
				"The fakes send no credential", "`api-auth`", "The webhook routes check no signature of an outside system",
			} {
				require.Contains(t, section, part)
			}
			namedPathsAreIn(t, section, files)
		})

		t.Run("with process-manager, the docs of the example name the connector files and only files that the repo has, and do not say that an outside system sends a token", func(t *testing.T) {
			files := writtenWithProcessManager(t)
			docs := files["docs/process-manager-example.md"]

			for _, path := range []string{"docs/event-model-payment.emod", "docs/event-model-payment.md", "docs/event-model-payment.svg"} {
				require.Contains(t, files, path)
			}
			for _, part := range []string{"`internal/use_cases/payment.go`", "`cmd/catchup/payment.go`", "`internal/api/use_cases/payment.go`", "The fakes send no credential", "No webhook route checks that a result comes from the outside system"} {
				require.Contains(t, docs, part)
			}
			for _, path := range []string{"docs/process-manager-example.md", "docs/event-model-payment.md", "docs/event-model-payment.emod", "docs/event-model-payment.svg"} {
				require.NotContains(t, strings.ToLower(files[path]), "token", path)
			}
			namedPathsAreIn(t, docs, files)
		})

		t.Run("with no item, the Taskfile of the API and the README have only the lines of the starter", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.True(t, strings.HasSuffix(files["Taskfile.api.yml"], "          --build-arg=VERSION=$(git rev-parse HEAD 2>/dev/null || echo main) \\\n          .\n"))
			require.True(t, strings.HasSuffix(files["README.md"], "[docs/event-subscription.md](docs/event-subscription.md#one-process-for-each-consumer) tells how.\n"))
		})

		t.Run("the catalog has one copy of each file of the process manager, in the folder of the feature process-manager", func(t *testing.T) {
			processManager := regexp.MustCompile(`^(internal/use_cases/payment(/|(_test)?\.go$)|internal/api/use_cases/(payment/|webhook/|payment(_test)?\.go$)|cmd/catchup/payment(_test)?\.go$|internal/use_cases/account/(holds/|hold(_test)?\.go$|event/hold_|command/[^/]*_hold\.go$)|cmd/fake_services/|Taskfile\.fake_services\.yml$|Dockerfile\.fake_services$|migrations/[^/]*_create-payment(s|-deadlines)-table\.|docs/process-manager-example\.md$|docs/event-model-payment\.)`)
			pathInRepo := regexp.MustCompile(`^(?:starters|features)/[^/]+/files/(?:[^/]+/)*?go/(.+?)(?:\.tmpl)?$`)
			copies := map[string][]string{}

			require.NoError(t, fs.WalkDir(os.DirFS(repoCatalogDir), ".", func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				if match := pathInRepo.FindStringSubmatch(path); match != nil && processManager.MatchString(match[1]) {
					copies[match[1]] = append(copies[match[1]], path)
				}
				return nil
			}))

			require.Subset(t, slices.Collect(maps.Keys(copies)), []string{"internal/use_cases/account/hold.go", "cmd/catchup/payment.go", "internal/api/use_cases/payment.go", "internal/use_cases/payment.go", "internal/use_cases/payment/checks/fake_services.go", "Taskfile.fake_services.yml"})
			for path, catalogFiles := range copies {
				require.Len(t, catalogFiles, 1, path)
				require.True(t, strings.HasPrefix(catalogFiles[0], "features/process-manager/files/go/"), catalogFiles[0])
			}
		})

		t.Run("with no item, the account, its events, the balance projection and the registration of the handlers have only the lines of the starter", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.Contains(t, files["internal/use_cases/account/account.go"], "\tclosed              bool\n}")
			require.Contains(t, files["internal/use_cases/account/account.go"], "\t\ta.closed = true\n\t}\n}")
			require.Contains(t, files["internal/use_cases/account/events.go"], "\treg.Register[event.AccountClosed]()\n}")
			require.Contains(t, files["internal/use_cases/account/balance/projection.go"], "\t\tevent.AccountClosed{}.EventType(),\n\t}")
			require.Contains(t, files["internal/use_cases/account/balance/projection.go"], "\t\treturn err\n\tdefault:")
			require.Contains(t, files["internal/use_cases/register.go"], "\taccount.RegisterEvents(reg)\n}")
			require.Contains(t, files["internal/use_cases/register.go"], "\tclose_account.Register(bus, accounts)\n}")
		})

		targetsOf := func(t *testing.T, featureName string) []string {
			t.Helper()
			definitions, err := repoCatalog().Definitions()
			require.NoError(t, err)
			return definitions.Features[featureName].Targets
		}

		writtenTarget := func(t *testing.T, starterName string, given map[string]string, features map[string]map[string]string) map[string]string {
			t.Helper()
			starter, err := repoCatalog().Starter(starterName)
			require.NoError(t, err)
			selection, err := starter.Select("my-tool", given, features, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-tool")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		extensionPointFilesOf := func(t *testing.T, featureName string, starterName string) []string {
			t.Helper()
			definitions, err := repoCatalog().Definitions()
			require.NoError(t, err)
			var files []string
			for name := range definitions.Features[featureName].AddTo {
				files = append(files, strings.TrimPrefix(definitions.Starters[starterName].ExtensionPoints[name].File, "go/"))
			}
			return slices.Compact(slices.Sorted(slices.Values(files)))
		}

		changesBy := func(t *testing.T, starterName string, given map[string]string, features map[string]map[string]string) (added map[string]string, changed []string, removed []string) {
			t.Helper()
			without := writtenTarget(t, starterName, given, nil)
			with := writtenTarget(t, starterName, given, features)
			added = map[string]string{}
			for path, contents := range with {
				before, ok := without[path]
				switch {
				case !ok:
					added[path] = contents
				case before != contents:
					changed = append(changed, path)
				}
			}
			for path := range without {
				if _, ok := with[path]; !ok {
					removed = append(removed, path)
				}
			}
			return added, changed, removed
		}

		for delivery, expected := range map[string]struct {
			folders []string
			changed []string
		}{
			"pg-notify": {[]string{"common", "pg-notify"}, []string{"README.md", "go.mod", "internal/api/use_cases/register.go"}},
			"nats": {[]string{"common", "nats"}, []string{
				".env.integration", ".env.local", "cmd/catchup/main.go", "docker-compose.dependencies.yml", "go.mod", "internal/api/config.go",
				"internal/api/use_cases/register.go", "scripts/test/start-test-dependencies.sh",
			}},
		} {
			t.Run("push with the delivery "+delivery+" adds the files of its folders for that value, and of the starter it changes only the file of each extension point that an item of that value adds to", func(t *testing.T) {
				var featureFiles []string
				for _, folder := range expected.folders {
					featureFolder := filepath.Join(repoCatalogDir, "features", "push", "files", folder, "go")
					require.NoError(t, filepath.WalkDir(featureFolder, func(path string, entry fs.DirEntry, err error) error {
						if err != nil || entry.IsDir() {
							return err
						}
						relative, err := filepath.Rel(featureFolder, path)
						featureFiles = append(featureFiles, strings.TrimSuffix(filepath.ToSlash(relative), ".tmpl"))
						return err
					}))
				}

				added, changed, removed := changesBy(t, "go-event-sourcing-service", nil, map[string]map[string]string{"push": {"delivery": delivery}})

				require.Subset(t, featureFiles, []string{"internal/api/push/clients.go", "internal/api/use_cases/push.go"})
				require.ElementsMatch(t, featureFiles, slices.Collect(maps.Keys(added)))
				require.ElementsMatch(t, expected.changed, changed)
				require.Subset(t, extensionPointFilesOf(t, "push", "go-event-sourcing-service"), changed)
				require.Empty(t, removed)
			})
		}

		t.Run("push has the option delivery with the values pg-notify and nats and the default pg-notify, and with no value it writes the repo of pg-notify", func(t *testing.T) {
			definitions, err := repoCatalog().Definitions()
			require.NoError(t, err)

			require.Equal(t, catalog.OptionDefinition{Default: "pg-notify", Values: []string{"pg-notify", "nats"}}, definitions.Features["push"].Options["delivery"])
			require.Equal(t, writtenWithPushOf(t, map[string]string{"delivery": "pg-notify"}), writtenWithPush(t))
		})

		natsPlacesIn := func(t *testing.T, files map[string]string) []string {
			t.Helper()
			var found []string
			for path, contents := range files {
				switch {
				case regexp.MustCompile(`^docker-compose[^/]*\.yml$`).MatchString(path):
					services, _ := parsedYAML(t, contents)["services"].(map[string]any)
					if _, ok := services["nats"]; ok {
						found = append(found, path+": the service nats")
					}
				case strings.HasPrefix(path, ".env") || regexp.MustCompile(`^Taskfile[^/]*\.yml$`).MatchString(path):
					if strings.Contains(contents, "NATS_") {
						found = append(found, path+": a NATS variable")
					}
				case path == "go.mod" || strings.HasSuffix(path, ".go"):
					if strings.Contains(contents, "github.com/nats-io/") {
						found = append(found, path+": a nats-io module")
					}
				}
			}
			slices.Sort(found)
			return found
		}

		t.Run("with pg-notify, the repo has no NATS: no service of a Docker Compose file, no NATS variable of an .env file or a Taskfile, no nats-io module in go.mod, and no Go file that imports one", func(t *testing.T) {
			require.Empty(t, natsPlacesIn(t, writtenWithPushOf(t, map[string]string{"delivery": "pg-notify"})))
			require.Equal(t, []string{
				".env.integration: a NATS variable", ".env.local: a NATS variable", "cmd/catchup/push_test.go: a nats-io module",
				"docker-compose.dependencies.yml: the service nats", "go.mod: a nats-io module",
				"internal/api/use_cases/push_test.go: a nats-io module", "internal/common/nats_push/connection.go: a nats-io module",
			}, natsPlacesIn(t, writtenWithPushOf(t, map[string]string{"delivery": "nats"})))
		})

		t.Run("with nats, docker-compose.dependencies.yml has the service nats with JetStream, a health check and the port of NATS_HOST_PORT, the .env files set that port and the URL of the integration tests on another port, and the test script starts nats", func(t *testing.T) {
			files := writtenWithPushOf(t, map[string]string{"delivery": "nats"})

			services := parsedYAML(t, files["docker-compose.dependencies.yml"])["services"].(map[string]any)
			require.ElementsMatch(t, []string{"db", "migrate", "nats"}, slices.Collect(maps.Keys(services)))
			nats := services["nats"].(map[string]any)
			require.Equal(t, []any{"-js", "-m", "8222"}, nats["command"])
			require.Equal(t, []any{"CMD", "wget", "--spider", "-q", "http://localhost:8222/healthz"}, nats["healthcheck"].(map[string]any)["test"])
			require.Equal(t, []any{"${NATS_HOST_PORT}:4222"}, nats["ports"])
			require.True(t, strings.HasSuffix(files[".env.local"], "DB_HOST_PORT=45432\nNATS_HOST_PORT=4222\n"))
			require.True(t, strings.HasSuffix(files[".env.integration"], "DB_HOST_PORT=45433\nNATS_HOST_PORT=44222\nNATS_URL=nats://127.0.0.1:44222\n"))
			require.Contains(t, files["scripts/test/start-test-dependencies.sh"], "\ndocker compose up --wait db nats\n")
		})

		t.Run("with no item, docker-compose.dependencies.yml, the .env files and the test script have only the lines of the starter", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			require.ElementsMatch(t, []string{"db", "migrate"}, slices.Collect(maps.Keys(parsedYAML(t, files["docker-compose.dependencies.yml"])["services"].(map[string]any))))
			require.True(t, strings.HasSuffix(files["docker-compose.dependencies.yml"], "      - ./migrations:/app/migrations\n"))
			require.True(t, strings.HasSuffix(files[".env.local"], "DB_PASSWORD=local-password\nDB_HOST_PORT=45432\n"))
			require.True(t, strings.HasSuffix(files[".env.integration"], "DB_PASSWORD=local-password\nDB_HOST_PORT=45433\n"))
			require.Contains(t, files["scripts/test/start-test-dependencies.sh"], "\ndocker compose up --wait db\n")
		})

		t.Run("with nats, the config of the API and of catchup has the field Push of nats_push, which each imports one time, catchup runs the publish worker, Register makes the push worker of NATS, and go.mod requires nats.go", func(t *testing.T) {
			files := writtenWithPushOf(t, map[string]string{"delivery": "nats"})

			require.Contains(t, files["internal/api/config.go"], "\tDB             config.DB\n\tPush nats_push.Config\n}")
			require.Contains(t, files["cmd/catchup/main.go"], "\tDB config.DB\n\tPush nats_push.Config\n}")
			for _, path := range []string{"internal/api/config.go", "cmd/catchup/main.go"} {
				require.Equal(t, 1, strings.Count(files[path], `"github.com/hpcsc/my-service/internal/common/nats_push"`), path)
			}
			require.Contains(t, files["cmd/catchup/main.go"], "\tall = append(all, pushWorkers(cfg.Push, checkpointStore, eventSubscription, logger)...)\n")
			require.Contains(t, files["internal/api/use_cases/register.go"], "\tcoreUseCases.RegisterHandlers(bus, eventStream)\n\twebSocketPush := newPushWorker(cfg.Push, logger)\n")
			require.Contains(t, files["go.mod"], "\tgithub.com/nats-io/nats.go v1.55.0\n")
		})

		t.Run("with push, the API runs the push worker, serves its route, and go.mod requires the module of the WebSocket", func(t *testing.T) {
			files := writtenWithPush(t)

			require.Contains(t, files["internal/api/use_cases/register.go"], "\tcoreUseCases.RegisterHandlers(bus, eventStream)\n\twebSocketPush := newPushWorker(pool, reg, logger)\n")
			require.Contains(t, files["internal/api/use_cases/register.go"], "\troutables = append(routables, webSocketPush.routables(cfg)...)\n")
			require.Contains(t, files["internal/api/use_cases/register.go"], "\treturn []api.Worker{\n\t\twebSocketPush,\n\t}\n}")
			require.Contains(t, files["internal/api/push/handler.go"], `route.Public("GET", "/ws", h.serve)`)
			require.Contains(t, files["go.mod"], "\tgithub.com/coder/websocket v1.8.15\n")
		})

		t.Run("with push, the README ends with the section of push, which names the route, the form of a message, the checkpoint, one instance, the route with no credential, and only files that the repo has", func(t *testing.T) {
			files := writtenWithPush(t)
			readme := files["README.md"]
			section := readme[strings.Index(readme, "\n## Push the Events to WebSocket Clients\n"):]

			require.NotContains(t, section[1:], "\n## ")
			for _, part := range []string{
				"`GET /api/ws`", `{"type": "<name>.account-opened", "data": {"accountId": "<account ID>", "owner": "alice"}}`,
				"ws://127.0.0.1:3333/api/ws", "the checkpoint `push`", "does not send an old event again",
				"Run one instance of the API", "The route asks for no credential, also in a repository with `api-auth`", "an origin of `ALLOWED_ORIGINS`",
				"the status 1001, going away", "the exit status 1", "`--feature push:delivery=nats`",
			} {
				require.Contains(t, strings.Join(strings.Fields(section), " "), part)
			}
			namedPathsAreIn(t, section, files)
		})

		writtenWithAPIAuth := func(t *testing.T, given map[string]string, apiAuthValues map[string]string) map[string]string {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", given, map[string]map[string]string{"api-auth": apiAuthValues}, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		for method, files := range map[string][]string{
			"api-key": {
				"internal/auth/bearer.go", "internal/auth/refuse.go", "internal/auth/config.go", "internal/auth/key_hashes.go", "internal/auth/key_hashes_test.go",
				"internal/auth/middleware.go", "internal/auth/middleware_test.go", "docs/api-auth.md",
			},
			"oauth": {
				"internal/auth/bearer.go", "internal/auth/refuse.go", "internal/auth/config.go", "internal/auth/issuer.go", "internal/auth/issuer_url.go", "internal/auth/issuer_url_test.go",
				"internal/auth/signing_keys.go", "internal/auth/signing_keys_test.go", "internal/auth/middleware.go",
				"internal/auth/middleware_test.go", "internal/auth/test_issuer_test.go", "docs/api-auth.md",
			},
		} {
			t.Run("with the method "+method+", api-auth adds the files of the method to each of its targets, chi-api among them, and of the files of the target it changes only the file of each extension point that it adds items to", func(t *testing.T) {
				targets := targetsOf(t, "api-auth")

				require.Contains(t, targets, "chi-api")
				for _, starterName := range targets {
					added, changed, removed := changesBy(t, starterName, nil, map[string]map[string]string{"api-auth": {"method": method}})

					require.ElementsMatch(t, files, slices.Collect(maps.Keys(added)), starterName)
					require.ElementsMatch(t, extensionPointFilesOf(t, "api-auth", starterName), changed, starterName)
					require.Empty(t, removed, starterName)
				}
			})

			t.Run("with the method "+method+", api-auth adds the same files with the same contents to each two of its targets", func(t *testing.T) {
				targets := targetsOf(t, "api-auth")
				require.GreaterOrEqual(t, len(targets), 2)
				first, _, _ := changesBy(t, targets[0], nil, map[string]map[string]string{"api-auth": {"method": method}})

				for _, starterName := range targets[1:] {
					added, _, _ := changesBy(t, starterName, nil, map[string]map[string]string{"api-auth": {"method": method}})
					require.Equal(t, first, added, targets[0]+" and "+starterName)
				}
			})
		}

		t.Run("with the method api-key, no written file names an issuer, an audience, discovery or a JWT, and the Go files import no JWT module", func(t *testing.T) {
			files := writtenWithAPIAuth(t, nil, map[string]string{"method": "api-key"})

			for path, contents := range files {
				for _, word := range []string{"issuer", "audience", "openid", "jwt", "jwks"} {
					require.NotContains(t, strings.ToLower(contents), word, path)
				}
			}
		})

		t.Run("with the method oauth, no written file holds a value of the issuer URL or the audience", func(t *testing.T) {
			files := writtenWithAPIAuth(t, nil, map[string]string{"method": "oauth"})

			for path, contents := range files {
				for _, setting := range []string{"OAUTH_ISSUER_URL=", "OAUTH_AUDIENCE="} {
					for _, line := range strings.Split(contents, "\n") {
						if strings.Contains(line, setting) {
							require.Contains(t, line, setting+"<", path)
						}
					}
				}
			}
		})

		for module, auth := range map[string]string{
			"":                       "github.com/hpcsc/my-service/internal/auth",
			"github.com/acme/ledger": "github.com/acme/ledger/internal/auth",
		} {
			t.Run("the server and the config of the API hold the items of api-auth and import "+auth, func(t *testing.T) {
				given := map[string]string{}
				if module != "" {
					given["module"] = module
				}

				files := writtenWithAPIAuth(t, given, nil)

				require.Contains(t, files["internal/api/server/server.go"], "\t\tauth.Middleware(cfg.Auth),\n\t}")
				require.Contains(t, files["internal/api/config.go"], "\tAuth auth.Config\n}")
				for _, path := range []string{"internal/api/server/server.go", "internal/api/config.go"} {
					require.Equal(t, 1, strings.Count(files[path], strconv.Quote(auth)), path)
				}
				if module != "" {
					for path, contents := range files {
						require.NotContains(t, contents, "github.com/hpcsc/my-service", path)
					}
				}
			})
		}

		t.Run("with the value api-key for method, api-auth writes the same files as with no value", func(t *testing.T) {
			require.Equal(t, writtenWithAPIAuth(t, nil, nil), writtenWithAPIAuth(t, nil, map[string]string{"method": "api-key"}))
		})

		t.Run("a value that method does not have stops, and the message names method and its values", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, map[string]map[string]string{"api-auth": {"method": "basic"}}, catalog.Answers{})

			require.ErrorContains(t, err, `the feature api-auth: the value "basic" of the option method is not one of its values: api-key, oauth`)
		})

		t.Run("cli-auth with the method oauth of api-auth selects cli, and the preview names the command line as the source of the method and the files of oauth of cli-auth", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": nil, "api-auth": {"method": "oauth"}}, catalog.Answers{})
			require.NoError(t, err)

			preview, err := selection.Preview()

			require.NoError(t, err)
			for _, part := range []string{
				"  api-auth: command line, needed by cli-auth\n", "  cli: needed by cli-auth\n", "  cli-auth: command line\n",
				"  api-auth.method: oauth (command line)\n", "  internal/credential/login.go: the feature cli-auth\n", "  internal/credential/store.go: the feature cli-auth\n",
			} {
				require.Contains(t, preview, part)
			}
		})

		for method, expected := range map[string][]string{
			"api-key": {"api-auth.method=api-key", "cli-auth.store-key=false"},
			"oauth":   {"api-auth.method=oauth"},
		} {
			t.Run("with the method "+method+", the questions of a selection with cli-auth hold method of api-auth, and store-key of cli-auth only with api-key", func(t *testing.T) {
				starter, err := repoCatalog().Starter("go-event-sourcing-service")
				require.NoError(t, err)
				selection, err := starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": nil, "api-auth": {"method": method}}, catalog.Answers{})
				require.NoError(t, err)

				var featureQuestions []string
				for _, q := range selection.Questions() {
					if q.Feature != "" {
						featureQuestions = append(featureQuestions, q.Feature+"."+q.Option+"="+q.Value)
					}
				}

				require.Equal(t, expected, featureQuestions)
			})
		}

		t.Run("with cli-auth and the answer oauth for method of api-auth, the selection writes the files of oauth of cli-auth, and the preview names the answer as the source", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": nil}, catalog.Answers{}.WithValue("api-auth", "method", "oauth"))
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "  api-auth.method: oauth (answered)\n")
			files := filesIn(t, output)
			require.Contains(t, files, "internal/credential/login.go")
			require.Contains(t, files["internal/credential/credential.go"], "_TOKEN")
		})

		for _, method := range []string{"api-key", "oauth"} {
			t.Run("with the method "+method+", no Go file of api-auth imports a package of the module other than the packages under internal/auth", func(t *testing.T) {
				files := writtenWithAPIAuth(t, nil, map[string]string{"method": method})

				for path, contents := range files {
					if !strings.HasPrefix(path, "internal/auth/") {
						continue
					}
					parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.ImportsOnly)
					require.NoError(t, err, path)
					for _, imported := range parsed.Imports {
						importPath, err := strconv.Unquote(imported.Path.Value)
						require.NoError(t, err)
						if strings.HasPrefix(importPath, "github.com/hpcsc/my-service/") {
							require.True(t, strings.HasPrefix(importPath, "github.com/hpcsc/my-service/internal/auth"), path+" imports "+importPath)
						}
					}
				}
			})
		}

		docsNameOnlyWhatTheRepoHas := func(t *testing.T, docs string, files map[string]string, starterName string) {
			t.Helper()
			for _, path := range regexp.MustCompile("`([A-Za-z0-9_./-]+\\.(?:ya?ml|go|md|local))`").FindAllStringSubmatch(docs, -1) {
				require.Contains(t, files, path[1], starterName)
			}
			namedPathsAreIn(t, docs, files)
			var named []string
			for _, match := range regexp.MustCompile(`\btask ([a-z0-9:_-]+)`).FindAllStringSubmatch(docs, -1) {
				named = append(named, match[1])
			}
			require.Contains(t, named, "api:run", starterName)
			require.Subset(t, tasksOf(t, files), named, starterName)
		}

		t.Run("for each target of api-auth, the docs of the method api-key name the setting of the key hashes, the run of the API with it, a request under /api and the answers of the API, and only paths and tasks that the repo has", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "api-auth") {
				files := writtenTarget(t, starterName, nil, map[string]map[string]string{"api-auth": nil})
				docs := files["docs/api-auth.md"]

				for _, part := range []string{
					"API_KEY_HASHES", "Authorization: Bearer", "401", "WWW-Authenticate: Bearer", "403",
					"`GET /api/` is public", "curl http://localhost:3333/api/<route>",
					`{"successful": false, "messages": ["sign in to continue"]}`,
					"With the method `api-key`", "API_KEY_HASHES=<hash of your key> task api:run", "`README.md`",
				} {
					require.Contains(t, docs, part, starterName)
				}
				docsNameOnlyWhatTheRepoHas(t, docs, files, starterName)
			}
		})

		t.Run("for each target of api-auth, the docs of the method oauth name the issuer, the two settings, the run of the API with them, the allowlist and the checks, a request under /api, and the answers of the API, and only paths and tasks that the repo has", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "api-auth") {
				files := writtenTarget(t, starterName, nil, map[string]map[string]string{"api-auth": {"method": "oauth"}})
				docs := files["docs/api-auth.md"]

				for _, part := range []string{
					"JWT access tokens", "/.well-known/openid-configuration", "Google gives no JWT access token for your own API",
					"`GET /api/` is public", "curl http://localhost:3333/api/<route>",
					"OAUTH_ISSUER_URL", "OAUTH_AUDIENCE", "OAUTH_ISSUER_URL=<url> OAUTH_AUDIENCE=<audience> task api:run", "the repo needs no new render",
					"`RS256`", "`iss`", "`aud`", "`exp`", "Authorization: Bearer", "401", "WWW-Authenticate: Bearer", "403", "With the method `oauth`",
					`{"successful": false, "messages": ["sign in to continue"]}`, "`internal/auth/middleware.go`", "`README.md`",
				} {
					require.Contains(t, docs, part, starterName)
				}
				docsNameOnlyWhatTheRepoHas(t, docs, files, starterName)
			}
		})

		writtenWithCLI := func(t *testing.T, name string, given map[string]string) (string, map[string]string) {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select(name, given, withoutValues("cli"), catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), name)
			require.NoError(t, selection.Write(output))
			return output, filesIn(t, output)
		}

		t.Run("cli adds each of its files at its path to each of its targets, chi-api and go among them, and of the files of the target it changes only the file of each extension point that it adds items to", func(t *testing.T) {
			targets := targetsOf(t, "cli")

			require.Subset(t, targets, []string{"chi-api", "go"})
			for _, starterName := range targets {
				added, changed, removed := changesBy(t, starterName, nil, withoutValues("cli"))

				require.ElementsMatch(t, []string{
					".dockerignore", ".goreleaser.yaml", "scripts/install.sh", "Taskfile.cli.yml",
					"cmd/my-tool/main.go",
					"internal/cli/adapter.go", "internal/cli/adapter_test.go", "internal/cli/root.go", "internal/cli/update.go",
					"internal/cli/progress/line.go", "internal/cli/progress/line_test.go",
					"internal/cli/release/client.go", "internal/cli/release/updater.go", "internal/cli/release/updater_test.go",
					"internal/cli/version/version.go", "internal/cli/version/version_test.go",
					"docs/cli.md", "docs/e2e-tests.md",
					"e2e/Dockerfile", "e2e/package.json", "e2e/testUtils.ts", "e2e/tsconfig.json", "e2e/vitest.config.ts",
					"e2e/tests/cli.test.ts", "e2e/tests/update.test.ts",
				}, slices.Collect(maps.Keys(added)), starterName)
				require.Contains(t, changed, "Taskfile.yml", starterName)
				require.ElementsMatch(t, extensionPointFilesOf(t, "cli", starterName), changed, starterName)
				require.Empty(t, removed, starterName)
			}
		})

		for name, given := range map[string]map[string]string{
			"with no given values":                            nil,
			"with given values for the module and Go version": {"module": "github.com/acme/ledger", "go-version": "1.28.0"},
		} {
			t.Run("cli adds the same files with the same contents to each two of its targets, "+name, func(t *testing.T) {
				targets := targetsOf(t, "cli")
				require.GreaterOrEqual(t, len(targets), 2)
				first, _, _ := changesBy(t, targets[0], given, withoutValues("cli"))

				for _, starterName := range targets[1:] {
					added, _, _ := changesBy(t, starterName, given, withoutValues("cli"))
					require.Equal(t, first, added, targets[0]+" and "+starterName)
				}
			})
		}

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
			require.Contains(t, files["Taskfile.cli.yml"], "-X github.com/acme/ledger/internal/cli/version.releaseTag={{.E2E_TAG}}")
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

		t.Run("the root command of the CLI lists each command that a selected feature adds to cli.commands after version and update, and root.go parses as Go", func(t *testing.T) {
			files := catalogOnDisk(t)
			files["features/accounts/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Accounts.\ntargets: [go-event-sourcing-service]\nadd_to:\n  cli.commands:\n    - code: newOpenCommand()\n    - code: newCloseCommand()\n")}
			starter, err := catalog.New(files).Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, withoutValues("cli", "accounts"), catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")

			require.NoError(t, selection.Write(output))

			root, err := os.ReadFile(filepath.Join(output, "internal", "cli", "root.go"))
			require.NoError(t, err)
			require.Contains(t, string(root), "\t\t\tnewVersionCommand(),\n\t\t\tnewUpdateCommand(),\n\t\t\tnewOpenCommand(),\n\t\t\tnewCloseCommand(),\n\t\t},")
			_, err = parser.ParseFile(token.NewFileSet(), "root.go", root, parser.AllErrors)
			require.NoError(t, err)
		})

		writtenWithFeatures := func(t *testing.T, givenByFeature map[string]map[string]string) map[string]string {
			t.Helper()
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, givenByFeature, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		cliAuthVariants := map[string]map[string]map[string]string{
			"api-key":        {"cli-auth": nil, "api-auth": {"method": "api-key"}},
			"oauth":          {"cli-auth": nil, "api-auth": {"method": "oauth"}},
			"stored api-key": {"cli-auth": {"store-key": "true"}, "api-auth": {"method": "api-key"}},
		}
		writtenWithCLIAuthVariant := func(t *testing.T, variant string) map[string]string {
			t.Helper()
			return writtenWithFeatures(t, cliAuthVariants[variant])
		}

		addedAndChangedByCLIAuth := func(t *testing.T, starterName string, variant string) (map[string]string, []string) {
			t.Helper()
			withoutCLIAuth := writtenTarget(t, starterName, nil, map[string]map[string]string{"cli": nil, "api-auth": cliAuthVariants[variant]["api-auth"]})
			withCLIAuth := writtenTarget(t, starterName, nil, cliAuthVariants[variant])
			for path := range withoutCLIAuth {
				require.Contains(t, withCLIAuth, path, "cli-auth removes "+path)
			}
			added := map[string]string{}
			var changed []string
			for path, contents := range withCLIAuth {
				before, ok := withoutCLIAuth[path]
				switch {
				case !ok:
					added[path] = contents
				case before != contents:
					changed = append(changed, path)
				}
			}
			return added, changed
		}

		t.Run("with each variant, cli-auth changes only the root command of the CLI, which lists auth after version and update, and go.mod of the repo with cli and api-auth of each of its targets, chi-api among them, and adds the same files with the same contents to each two of its targets", func(t *testing.T) {
			targets := targetsOf(t, "cli-auth")
			require.Contains(t, targets, "chi-api")
			require.GreaterOrEqual(t, len(targets), 2)

			for _, variant := range []string{"api-key", "oauth", "stored api-key"} {
				first, _ := addedAndChangedByCLIAuth(t, targets[0], variant)
				for _, starterName := range targets {
					added, changed := addedAndChangedByCLIAuth(t, starterName, variant)

					require.ElementsMatch(t, []string{"internal/cli/root.go", "go.mod"}, changed, variant+": "+starterName)
					require.Equal(t, first, added, variant+": "+targets[0]+" and "+starterName)
					require.Contains(t, writtenTarget(t, starterName, nil, cliAuthVariants[variant])["internal/cli/root.go"], "\t\t\tnewVersionCommand(),\n\t\t\tnewUpdateCommand(),\n\t\t\tnewAuthCommand(),\n\t\t},", variant+": "+starterName)
				}
			}
		})

		t.Run("with each variant and each target of cli-auth, go.mod of the selection requires a version of each module that a Go file of cli-auth imports, other than the standard library and the module of the repo", func(t *testing.T) {
			majorVersion := regexp.MustCompile(`^/v([2-9]|[1-9][0-9]+)(/|$)`)
			for _, starterName := range targetsOf(t, "cli-auth") {
				for _, variant := range []string{"api-key", "oauth", "stored api-key"} {
					added, _ := addedAndChangedByCLIAuth(t, starterName, variant)
					required := slices.Collect(maps.Keys(requiredModulesOf(t, writtenTarget(t, starterName, nil, cliAuthVariants[variant])["go.mod"])))

					var imported []string
					for path, contents := range added {
						if !strings.HasSuffix(path, ".go") {
							continue
						}
						parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.ImportsOnly)
						require.NoError(t, err, path)
						for _, spec := range parsed.Imports {
							importPath, err := strconv.Unquote(spec.Path.Value)
							require.NoError(t, err)
							if strings.Contains(strings.Split(importPath, "/")[0], ".") && !strings.HasPrefix(importPath, "github.com/hpcsc/my-tool/") {
								imported = append(imported, importPath)
							}
						}
					}
					require.Contains(t, imported, "github.com/urfave/cli/v3", starterName+": "+variant)
					if variant == "oauth" {
						require.Subset(t, imported, []string{"golang.org/x/oauth2", "github.com/zalando/go-keyring", "github.com/cli/browser"}, starterName+": "+variant)
					}
					if variant == "stored api-key" {
						require.Subset(t, imported, []string{"github.com/zalando/go-keyring", "golang.org/x/term"}, starterName+": "+variant)
					}
					for _, importPath := range imported {
						require.True(t, slices.ContainsFunc(required, func(module string) bool {
							rest, ok := strings.CutPrefix(importPath, module)
							return ok && (rest == "" || strings.HasPrefix(rest, "/")) && !majorVersion.MatchString(rest)
						}), starterName+": "+variant+": "+importPath)
					}
				}
			}
		})

		t.Run("with each variant and each target of cli-auth, the docs of cli-auth name only paths that the repo has, and no route that the API of a target serves", func(t *testing.T) {
			routePattern := regexp.MustCompile(`route\.(?:Public|Protected)\("[A-Z]+", "([^"]+)"`)
			module := regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9-]+)+/`)
			targets := targetsOf(t, "cli-auth")
			var routes []string
			for _, starterName := range targets {
				for path, contents := range writtenTarget(t, starterName, nil, nil) {
					if regexp.MustCompile(`^internal/(?:api/)?use_cases/.*\.go$`).MatchString(path) {
						for _, match := range routePattern.FindAllStringSubmatch(contents, -1) {
							if match[1] != "/" {
								routes = append(routes, match[1])
							}
						}
					}
				}
			}
			require.Subset(t, routes, []string{"/accounts", "/users"})

			for _, starterName := range targets {
				for _, variant := range []string{"api-key", "oauth", "stored api-key"} {
					files := writtenTarget(t, starterName, nil, cliAuthVariants[variant])
					docs := files["docs/cli-auth.md"]

					require.Contains(t, docs, `client.Do(ctx, http.MethodPost, "/<route>", body)`, starterName+": "+variant)
					for _, route := range routes {
						require.NotContains(t, docs, `"`+route+`"`, starterName+": "+variant)
					}
					for _, path := range regexp.MustCompile("`([A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+)`").FindAllStringSubmatch(docs, -1) {
						if module.MatchString(path[1]) {
							continue
						}
						require.True(t, slices.ContainsFunc(slices.Collect(maps.Keys(files)), func(written string) bool {
							return written == path[1] || strings.HasPrefix(written, path[1]+"/")
						}), starterName+": "+variant+": "+path[1])
					}
				}
			}
		})

		t.Run("with the method api-key of api-auth, cli-auth adds the files of api-key", func(t *testing.T) {
			added, _ := addedAndChangedByCLIAuth(t, "go-event-sourcing-service", "api-key")

			require.ElementsMatch(t, []string{
				"internal/credential/credential.go", "internal/credential/command.go", "internal/credential/config.go",
				"internal/credential/credential_test.go", "internal/cli/auth.go", "e2e/tests/auth.test.ts", "docs/cli-auth.md",
				"internal/apiclient/client.go", "internal/apiclient/client_test.go",
			}, slices.Collect(maps.Keys(added)))
		})

		t.Run("cli-auth alone, and cli-auth with cli and api-auth, give the same files", func(t *testing.T) {
			require.Equal(t, writtenWithFeatures(t, withoutValues("cli-auth")), writtenWithFeatures(t, withoutValues("cli-auth", "cli", "api-auth")))
		})

		t.Run("cli-auth with no method of api-auth writes the files of the default method api-key", func(t *testing.T) {
			require.Equal(t, writtenWithFeatures(t, withoutValues("cli-auth")), writtenWithCLIAuthVariant(t, "api-key"))
		})

		t.Run("with the method oauth of api-auth, cli-auth adds the files of oauth and of the store", func(t *testing.T) {
			added, _ := addedAndChangedByCLIAuth(t, "go-event-sourcing-service", "oauth")

			require.ElementsMatch(t, []string{
				"internal/credential/credential.go", "internal/credential/command.go", "internal/credential/config.go", "internal/credential/credential_test.go",
				"internal/credential/issuer.go", "internal/credential/oauth_setting.go", "internal/credential/login.go", "internal/credential/login_test.go",
				"internal/credential/tokens.go", "internal/credential/tokens_test.go", "internal/credential/test_issuer_test.go", "internal/credential/test_api_test.go",
				"internal/credential/store.go", "internal/credential/store_test.go", "internal/credential/api_url.go", "internal/credential/api_url_test.go",
				"internal/cli/auth.go", "e2e/tests/auth.test.ts", "docs/cli-auth.md", "internal/apiclient/client.go", "internal/apiclient/client_test.go",
			}, slices.Collect(maps.Keys(added)))
		})

		t.Run("with the method api-key, no file that cli-auth adds names an issuer, an audience, OAuth, a refresh, a keyring, the credential store or auth login, and no Go file imports a module of the method oauth", func(t *testing.T) {
			added, _ := addedAndChangedByCLIAuth(t, "go-event-sourcing-service", "api-key")
			oauthAdded, _ := addedAndChangedByCLIAuth(t, "go-event-sourcing-service", "oauth")
			words := []string{"issuer", "audience", "oauth", "refresh", "keyring", "credential store", "auth login"}
			for _, word := range words {
				require.True(t, slices.ContainsFunc(slices.Collect(maps.Values(oauthAdded)), func(contents string) bool {
					return strings.Contains(strings.ToLower(contents), word)
				}), word)
			}

			for path, contents := range added {
				for _, word := range words {
					require.NotContains(t, strings.ToLower(contents), word, path)
				}
			}
			for path, contents := range writtenWithCLIAuthVariant(t, "api-key") {
				if !strings.HasSuffix(path, ".go") {
					continue
				}
				parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.ImportsOnly)
				require.NoError(t, err, path)
				for _, imported := range parsed.Imports {
					for _, module := range []string{"golang.org/x/oauth2", "github.com/zalando/go-keyring", "github.com/cli/browser", "golang.org/x/term"} {
						require.NotContains(t, imported.Path.Value, module, path)
					}
				}
			}
		})

		t.Run("with the method oauth, the issuer URL, the client ID, the audience and the API URL are empty constants", func(t *testing.T) {
			files := writtenWithCLIAuthVariant(t, "oauth")

			for path, constants := range map[string][]string{
				"internal/credential/oauth_setting.go": {"issuerURL", "clientID", "audience"},
				"internal/credential/api_url.go":       {"apiURL"},
			} {
				for _, constant := range constants {
					require.Regexp(t, `(?m)^\t`+constant+` += ""$`, files[path], path)
				}
			}
		})

		t.Run("a value of method for cli-auth stops, and the message lists store-key as the option of cli-auth", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": {"method": "api-key"}}, catalog.Answers{})

			require.ErrorContains(t, err, `unknown option "method": the options of the feature cli-auth are store-key`)
		})

		t.Run("store-key=true of cli-auth with the method oauth of api-auth stops, and the message says that store-key applies only to the method api-key", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)

			_, err = starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": {"store-key": "true"}, "api-auth": {"method": "oauth"}}, catalog.Answers{})

			require.EqualError(t, err, `the option store-key of the feature cli-auth applies only when api-auth.method == "api-key", and api-auth.method is "oauth", but the command line gives it "true"`)
		})

		t.Run("the files of a repo with cli-auth hold no value of scaffold, with each variant", func(t *testing.T) {
			for _, variant := range []string{"api-key", "oauth", "stored api-key"} {
				files := writtenWithCLIAuthVariant(t, variant)

				for path, contents := range files {
					for _, value := range []string{".Scaffold.", ".Computed.", "go-cli-project", "GO_CLI_PROJECT", "zzsentinel", "<no value>"} {
						require.NotContains(t, contents, value, variant+": "+path)
					}
				}
			}
		})

		t.Run("with each variant, the Go files of cli-auth import only its own packages and cli, and only a test imports internal/auth of api-auth", func(t *testing.T) {
			for _, variant := range []string{"api-key", "oauth", "stored api-key"} {
				added, _ := addedAndChangedByCLIAuth(t, "go-event-sourcing-service", variant)
				var goFiles []string
				for path := range added {
					if strings.HasSuffix(path, ".go") {
						goFiles = append(goFiles, path)
					}
				}
				require.Subset(t, goFiles, []string{"internal/credential/credential.go", "internal/cli/auth.go", "internal/apiclient/client.go", "internal/apiclient/client_test.go"}, variant)

				for _, path := range goFiles {
					allowed := `^internal/(credential|cli|apiclient)$`
					if strings.HasSuffix(path, "_test.go") {
						allowed = `^internal/(credential|cli|apiclient|auth)$`
					}
					parsed, err := parser.ParseFile(token.NewFileSet(), path, added[path], parser.ImportsOnly)
					require.NoError(t, err, path)
					for _, imported := range parsed.Imports {
						importPath, err := strconv.Unquote(imported.Path.Value)
						require.NoError(t, err)
						if inModule, ok := strings.CutPrefix(importPath, "github.com/hpcsc/my-tool/"); ok {
							require.Regexp(t, allowed, inModule, variant+": "+path+" imports "+importPath)
						}
					}
				}
			}
		})

		t.Run("the docs of cli-auth name the variable, credentialCommand, auth status and the method of api-auth", func(t *testing.T) {
			files := writtenWithFeatures(t, withoutValues("cli-auth"))
			docs := files["docs/cli-auth.md"]

			for _, part := range []string{
				"MY_SERVICE_API_KEY", "credentialCommand", "my-service auth status", "Authorization: Bearer <key>",
				"the method `api-key`", "the CLI follows the method of the API", "[api-auth.md](api-auth.md)",
				`apiclient.New("http://localhost:3333/api")`, `client.Do(ctx, http.MethodPost, "/<route>", body)`, "ends with `/api`",
			} {
				require.Contains(t, docs, part)
			}
		})

		t.Run("with store-key=true, cli-auth adds the files of api-key and of the store and none of oauth", func(t *testing.T) {
			added, _ := addedAndChangedByCLIAuth(t, "go-event-sourcing-service", "stored api-key")

			require.ElementsMatch(t, []string{
				"internal/credential/credential.go", "internal/credential/command.go", "internal/credential/config.go", "internal/credential/credential_test.go",
				"internal/credential/store.go", "internal/credential/store_test.go", "internal/credential/api_url.go", "internal/credential/api_url_test.go",
				"internal/cli/auth.go", "e2e/tests/auth.test.ts", "docs/cli-auth.md", "internal/apiclient/client.go", "internal/apiclient/client_test.go",
			}, slices.Collect(maps.Keys(added)))
		})

		t.Run("with store-key=true, no file that cli-auth adds names an issuer, an audience, OAuth or a refresh, and no Go file imports a module of the login of oauth", func(t *testing.T) {
			added, _ := addedAndChangedByCLIAuth(t, "go-event-sourcing-service", "stored api-key")

			for path, contents := range added {
				for _, word := range []string{"issuer", "audience", "oauth", "refresh"} {
					require.NotContains(t, strings.ToLower(contents), word, path)
				}
				if strings.HasSuffix(path, ".go") {
					parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.ImportsOnly)
					require.NoError(t, err, path)
					for _, imported := range parsed.Imports {
						require.NotContains(t, []string{`"golang.org/x/oauth2"`, `"github.com/cli/browser"`}, imported.Path.Value, path)
					}
				}
			}
		})

		t.Run("with store-key=true, the docs of cli-auth name the method api-key, the commands of auth, where the key is, the API URL and the limits of the store", func(t *testing.T) {
			files := writtenWithCLIAuthVariant(t, "stored api-key")
			docs := files["docs/cli-auth.md"]

			for _, part := range []string{
				"the method `api-key`", "MY_SERVICE_API_KEY", "credentialCommand", "2. The API key that `my-service auth login` stores",
				"my-service auth login", "my-service auth status", "my-service auth logout", "the OS credential store", "credentials.json",
				"| `apiURL` | `internal/credential/api_url.go` | `MY_SERVICE_API_URL` |", "apiclient.New(apiURL.String())",
				"MY_SERVICE_CREDENTIAL_STORE", "fails each call or none", "[api-auth.md](api-auth.md)",
			} {
				require.Contains(t, docs, part)
			}
		})

		t.Run("with the method oauth, the docs of cli-auth name the token, the commands of auth, the settings with their constants and variables, the audience of the API and the API URL of the client", func(t *testing.T) {
			files := writtenWithCLIAuthVariant(t, "oauth")
			docs := files["docs/cli-auth.md"]

			for _, part := range []string{
				"MY_SERVICE_TOKEN", "credentialCommand", "my-service auth login", "my-service auth login --device", "my-service auth status",
				"my-service auth token", "my-service auth logout", "Authorization: Bearer <token>", "the method `oauth`",
				"the CLI follows the method of the API", "[api-auth.md](api-auth.md)",
				"| `issuerURL` | `internal/credential/oauth_setting.go` | `MY_SERVICE_OAUTH_ISSUER_URL` |",
				"| `clientID` | `internal/credential/oauth_setting.go` | `MY_SERVICE_OAUTH_CLIENT_ID` |",
				"| `audience` | `internal/credential/oauth_setting.go` | `MY_SERVICE_OAUTH_AUDIENCE` |",
				"| `apiURL` | `internal/credential/api_url.go` | `MY_SERVICE_API_URL` |",
				"`OAUTH_AUDIENCE` of the API", "the parameter `audience`", "http://127.0.0.1/callback", "apiclient.New(apiURL.String())",
				"MY_SERVICE_CREDENTIAL_STORE=file", "credentials.json",
			} {
				require.Contains(t, docs, part)
			}
		})

		t.Run("the docs, the README and the Taskfiles of a repo name each route of its API under /api, and do not name GET / alone", func(t *testing.T) {
			_, starterAlone := writtenStarter(t, nil)
			routePattern := regexp.MustCompile(`route\.(?:Public|Protected)\("[A-Z]+", "([^"]+)"`)
			apiVariable := regexp.MustCompile(`(?m)^\s+API: (\S+)$`)
			rootAlone := regexp.MustCompile("GET /(?:[\\s`'\")|,.]|$)")

			type repo struct {
				files        map[string]string
				exampleRoute string
			}
			for name, written := range map[string]repo{
				"the starter alone":                {starterAlone, "/accounts"},
				"api-auth with the method api-key": {writtenWithAPIAuth(t, nil, map[string]string{"method": "api-key"}), "/accounts"},
				"api-auth with the method oauth":   {writtenWithAPIAuth(t, nil, map[string]string{"method": "oauth"}), "/accounts"},
				"cli-auth":                         {writtenWithFeatures(t, withoutValues("cli-auth")), "/accounts"},
				"cli-auth with the method oauth":   {writtenWithCLIAuthVariant(t, "oauth"), "/accounts"},
				"process-manager":                  {writtenWithProcessManager(t), "/accounts"},
				"push":                             {writtenWithPush(t), "/ws"},
				"push with the delivery nats":      {writtenWithPushOf(t, map[string]string{"delivery": "nats"}), "/ws"},
				"chi-api alone":                    {writtenTarget(t, "chi-api", nil, nil), "/users"},
				"chi-api with api-auth of api-key": {writtenTarget(t, "chi-api", nil, map[string]map[string]string{"api-auth": {"method": "api-key"}}), "/users"},
				"chi-api with api-auth of oauth":   {writtenTarget(t, "chi-api", nil, map[string]map[string]string{"api-auth": {"method": "oauth"}}), "/users"},
				"chi-api with cli-auth":            {writtenTarget(t, "chi-api", nil, cliAuthVariants["api-key"]), "/users"},
				"chi-api with cli-auth of oauth":   {writtenTarget(t, "chi-api", nil, cliAuthVariants["oauth"]), "/users"},
			} {
				files := written.files
				var routes []string
				for path, contents := range files {
					if regexp.MustCompile(`^internal/.*\.go$`).MatchString(path) {
						for _, match := range routePattern.FindAllStringSubmatch(contents, -1) {
							if match[1] != "/" {
								routes = append(routes, match[1])
							}
						}
					}
				}
				var withoutPrefix []string
				for path, contents := range files {
					if !strings.HasPrefix(path, "docs/") && path != "README.md" && !regexp.MustCompile(`^Taskfile[^/]*\.yml$`).MatchString(path) {
						continue
					}
					if api := apiVariable.FindStringSubmatch(contents); api != nil {
						contents = strings.ReplaceAll(contents, "{{.API}}", api[1])
					}
					for _, route := range routes {
						for _, match := range regexp.MustCompile("(\"?)([^\\s\"'`(|]*)"+regexp.QuoteMeta(route)+`(\S* of the (?:payment )?network)?`).FindAllStringSubmatch(contents, -1) {
							pathAfterBaseURL := match[1] == `"` && match[2] == ""
							pathOfAnotherSystem := match[3] != ""
							if !pathAfterBaseURL && !pathOfAnotherSystem && !strings.HasSuffix(match[2], "/api") {
								withoutPrefix = append(withoutPrefix, path+": "+match[0])
							}
						}
					}
					for _, match := range rootAlone.FindAllString(contents, -1) {
						withoutPrefix = append(withoutPrefix, path+": "+match)
					}
				}

				require.Contains(t, routes, written.exampleRoute, name)
				require.Empty(t, withoutPrefix, name)
			}
		})

		t.Run("the need of cli-auth sets no method of api-auth, so the preview names the default as the source of the method", func(t *testing.T) {
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"cli-auth": nil}, catalog.Answers{})
			require.NoError(t, err)

			preview, err := selection.Preview()

			require.NoError(t, err)
			require.Contains(t, preview, "  api-auth.method: api-key (default)\n")
		})

		t.Run("the files of cli hold no value of scaffold", func(t *testing.T) {
			_, files := writtenWithCLI(t, "my-service", nil)

			for path, contents := range files {
				for _, value := range []string{".Scaffold.", ".ProjectKebab", "<no value>", "zzsentinel"} {
					require.NotContains(t, contents, value, path)
				}
			}
		})

		t.Run("the docs of cli name the tasks build, cli, release:local, test:e2e and test:e2e:local, and only tasks and paths that the repo of each target of cli has", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				files := writtenTarget(t, starterName, nil, withoutValues("cli"))
				tasks := tasksOf(t, files)
				filesAfterNpmInstall := maps.Clone(files)
				filesAfterNpmInstall["e2e/package-lock.json"] = ""

				for path, expected := range map[string][]string{
					"docs/cli.md":       {"build", "cli", "release:local", "test:e2e", "test:e2e:local"},
					"docs/e2e-tests.md": {"test:e2e", "test:e2e:local"},
				} {
					var named []string
					for _, match := range regexp.MustCompile("(?m)(?:^|`)task ([a-z0-9:_-]+)").FindAllStringSubmatch(files[path], -1) {
						named = append(named, match[1])
					}
					require.Subset(t, named, expected, starterName+": "+path)
					require.Subset(t, tasks, named, starterName+": "+path)
					namedPathsAreIn(t, files[path], filesAfterNpmInstall)
				}
			}
		})

		t.Run("with cli, each target has the tasks of Taskfile.cli.yml beside test:unit, build, cli, release:local, test:e2e and test:e2e:local among them, and without cli it has none of them", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				filesWithCLI := writtenTarget(t, starterName, nil, withoutValues("cli"))
				tasksWithoutCLI := tasksOf(t, writtenTarget(t, starterName, nil, nil))

				cliTasks := slices.Collect(maps.Keys(parsedYAML(t, filesWithCLI["Taskfile.cli.yml"])["tasks"].(map[string]any)))
				require.Subset(t, cliTasks, []string{"build", "cli", "release:local", "test:e2e", "test:e2e:local"}, starterName)
				require.Contains(t, tasksWithoutCLI, "test:unit", starterName)
				for _, name := range cliTasks {
					require.NotContains(t, tasksWithoutCLI, name, starterName)
				}
				require.ElementsMatch(t, append(tasksWithoutCLI, cliTasks...), tasksOf(t, filesWithCLI), starterName)
			}
		})

		t.Run("with cli, mise.toml of each target has the tools of the starter at their versions, GoReleaser 2.18.2 and Node 26.10.0, and without cli it has neither of the two", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				withoutCLI := toolsOf(t, writtenTarget(t, starterName, nil, nil)["mise.toml"])
				withCLI := toolsOf(t, writtenTarget(t, starterName, nil, withoutValues("cli"))["mise.toml"])

				require.Equal(t, "3.42.1", withoutCLI["aqua:go-task/task"], starterName)
				require.NotContains(t, withoutCLI, "aqua:goreleaser/goreleaser", starterName)
				require.NotContains(t, withoutCLI, "node", starterName)
				expected := maps.Clone(withoutCLI)
				expected["aqua:goreleaser/goreleaser"] = "2.18.2"
				expected["node"] = "26.10.0"
				require.Equal(t, expected, withCLI, starterName)
			}
		})

		t.Run("with cli, .gitignore of each target has dist/ and e2e/node_modules/ after the lines of the starter, and without cli it has neither", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				withoutCLI := writtenTarget(t, starterName, nil, nil)[".gitignore"]
				withCLI := writtenTarget(t, starterName, nil, withoutValues("cli"))[".gitignore"]

				require.Contains(t, strings.Split(withoutCLI, "\n"), "bin", starterName)
				require.NotContains(t, strings.Split(withoutCLI, "\n"), "dist/", starterName)
				require.NotContains(t, strings.Split(withoutCLI, "\n"), "e2e/node_modules/", starterName)
				require.Equal(t, withoutCLI+"dist/\ne2e/node_modules/\n", withCLI, starterName)
			}
		})

		t.Run("with no item, mise.toml and .gitignore of each target of cli end with the last line of the starter", func(t *testing.T) {
			for starterName, lastLines := range map[string][2]string{
				"chi-api":                   {`"go:golang.org/x/vuln/cmd/govulncheck" = "1.8.0"` + "\n", "go.work\nbin\n.idea\n"},
				"go":                        {`"go:golang.org/x/vuln/cmd/govulncheck" = "1.8.0"` + "\n", "go.work\nbin\n.idea\n"},
				"go-event-sourcing-service": {`"go:golang.org/x/vuln/cmd/govulncheck" = "1.8.0"` + "\n", "tmp\ntags\nbin\n"},
			} {
				files := writtenTarget(t, starterName, nil, nil)

				require.True(t, strings.HasSuffix(files["mise.toml"], lastLines[0]), starterName+": "+files["mise.toml"])
				require.True(t, strings.HasSuffix(files[".gitignore"], lastLines[1]), starterName+": "+files[".gitignore"])
			}
		})

		t.Run("with cli, go.mod of each target requires a version of each module that a Go file of the repo imports, other than the standard library and the module of the repo", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				files := writtenTarget(t, starterName, nil, withoutValues("cli"))
				required := slices.Collect(maps.Keys(requiredModulesOf(t, files["go.mod"])))

				var imported []string
				for path, contents := range files {
					if !strings.HasSuffix(path, ".go") {
						continue
					}
					parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.ImportsOnly)
					require.NoError(t, err, path)
					for _, spec := range parsed.Imports {
						importPath, err := strconv.Unquote(spec.Path.Value)
						require.NoError(t, err)
						host := strings.Split(importPath, "/")[0]
						if strings.Contains(host, ".") && !strings.HasPrefix(importPath, "github.com/hpcsc/my-tool/") {
							imported = append(imported, importPath)
						}
					}
				}
				require.Contains(t, imported, "github.com/urfave/cli/v3", starterName)
				majorVersion := regexp.MustCompile(`^/v([2-9]|[1-9][0-9]+)(/|$)`)
				for _, importPath := range imported {
					require.True(t, slices.ContainsFunc(required, func(module string) bool {
						rest, ok := strings.CutPrefix(importPath, module)
						return ok && (rest == "" || strings.HasPrefix(rest, "/")) && !majorVersion.MatchString(rest)
					}), starterName+": "+importPath)
				}
			}
		})

		t.Run("with no item, go.mod of each target of cli holds only the lines of the starter", func(t *testing.T) {
			require.Equal(t, "module github.com/hpcsc/my-tool\n\ngo 1.27.1\n\nrequire github.com/stretchr/testify v1.12.1\n", writtenTarget(t, "go", nil, nil)["go.mod"])
			require.Contains(t, writtenTarget(t, "go-event-sourcing-service", nil, nil)["go.mod"], "\tgithub.com/unrolled/render v1.8.1\n)\n\nrequire (\n")
			require.Contains(t, writtenTarget(t, "chi-api", nil, nil)["go.mod"], "\tgithub.com/unrolled/render v1.8.1\n)\n\nrequire (\n")
		})

		t.Run("with no item, ci.yml of each target of cli has only the jobs of the starter and ends with the last line of the starter", func(t *testing.T) {
			for starterName, expected := range map[string]struct {
				jobs     []string
				lastLine string
			}{
				"chi-api":                   {[]string{"test"}, "        run: task test:vulnerabilities\n"},
				"go":                        {[]string{"test"}, "        run: task test:vulnerabilities\n"},
				"go-event-sourcing-service": {[]string{"test", "publish"}, "          labels: ${{ steps.meta.outputs.labels }}\n"},
			} {
				workflow := writtenTarget(t, starterName, nil, nil)[".github/workflows/ci.yml"]

				require.ElementsMatch(t, expected.jobs, slices.Collect(maps.Keys(parsedYAML(t, workflow)["jobs"].(map[string]any))), starterName)
				require.True(t, strings.HasSuffix(workflow, expected.lastLine), starterName+": "+workflow)
			}
		})

		t.Run("with cli, ci.yml of each target keeps the triggers, the permissions and the jobs of the starter, and has the job e2e, which runs task test:e2e", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				withoutCLI := parsedYAML(t, writtenTarget(t, starterName, nil, nil)[".github/workflows/ci.yml"])
				withCLI := parsedYAML(t, writtenTarget(t, starterName, nil, withoutValues("cli"))[".github/workflows/ci.yml"])

				require.Equal(t, withoutCLI["on"], withCLI["on"], starterName)
				require.Equal(t, withoutCLI["permissions"], withCLI["permissions"], starterName)
				jobs := withCLI["jobs"].(map[string]any)
				require.Contains(t, withoutCLI["jobs"], "test", starterName)
				for name, job := range withoutCLI["jobs"].(map[string]any) {
					require.Equal(t, job, jobs[name], starterName+": "+name)
				}
				var runs []any
				for _, step := range jobs["e2e"].(map[string]any)["steps"].([]any) {
					runs = append(runs, step.(map[string]any)["run"])
				}
				require.Contains(t, runs, "task test:e2e", starterName)
			}
		})

		t.Run("the jobs that cli adds to ci.yml of each target keep the template syntax of GitHub Actions", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				workflow := writtenTarget(t, starterName, nil, withoutValues("cli"))[".github/workflows/ci.yml"]

				require.Contains(t, workflow, "repo-token: ${{ secrets.GITHUB_TOKEN }}", starterName)
			}
		})

		ciJobsWithCLI := func(t *testing.T, starterName string) map[string]any {
			t.Helper()
			workflow := writtenTarget(t, starterName, nil, withoutValues("cli"))[".github/workflows/ci.yml"]
			return parsedYAML(t, workflow)["jobs"].(map[string]any)
		}

		t.Run("with cli, ci.yml of each target runs on each push, tags among them, and publishes a release only for a tag that starts with v and a prerelease only for main, each after the jobs test and e2e", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				workflow := parsedYAML(t, writtenTarget(t, starterName, nil, withoutValues("cli"))[".github/workflows/ci.yml"])
				jobs := workflow["jobs"].(map[string]any)

				require.Equal(t, map[string]any{"push": map[string]any{}}, workflow["on"], starterName)

				release := jobs["release"].(map[string]any)
				prerelease := jobs["prerelease"].(map[string]any)
				require.Equal(t, "startsWith(github.ref, 'refs/tags/v')", release["if"], starterName)
				require.Equal(t, "github.ref == 'refs/heads/main'", prerelease["if"], starterName)
				require.ElementsMatch(t, []any{"test", "e2e"}, release["needs"], starterName)
				require.ElementsMatch(t, []any{"test", "e2e"}, prerelease["needs"], starterName)
			}
		})

		t.Run("with cli, each needs of each job of ci.yml of each target names a job of the workflow", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				jobs := ciJobsWithCLI(t, starterName)

				var needed []any
				for _, job := range jobs {
					switch needs := job.(map[string]any)["needs"].(type) {
					case string:
						needed = append(needed, needs)
					case []any:
						needed = append(needed, needs...)
					}
				}
				require.Contains(t, needed, "e2e", starterName)
				for _, name := range needed {
					require.Contains(t, jobs, name, starterName)
				}
			}
		})

		t.Run("with cli, only the jobs release and prerelease of ci.yml of each target can write the contents of the repo", func(t *testing.T) {
			for _, starterName := range targetsOf(t, "cli") {
				workflow := parsedYAML(t, writtenTarget(t, starterName, nil, withoutValues("cli"))[".github/workflows/ci.yml"])
				jobs := workflow["jobs"].(map[string]any)

				require.Equal(t, map[string]any{"contents": "read"}, workflow["permissions"], starterName)
				for name, job := range jobs {
					permissions := job.(map[string]any)["permissions"]
					if name == "release" || name == "prerelease" {
						require.Equal(t, map[string]any{"contents": "write"}, permissions, starterName+": "+name)
						continue
					}
					if permissions != nil {
						require.IsType(t, map[string]any{}, permissions, starterName+": "+name)
						require.NotEqual(t, "write", permissions.(map[string]any)["contents"], starterName+": "+name)
					}
				}
			}
		})

		t.Run("with cli, the jobs release and prerelease of ci.yml of each target have the steps of the release workflows of this repo, prerelease publishes only the newest commit of main, and prereleases run one at a time", func(t *testing.T) {
			newestCommitOnly := "head=$(git ls-remote --exit-code origin refs/heads/main)\n" +
				"if [ \"${head%%[[:space:]]*}\" != \"$GITHUB_SHA\" ]; then\n" +
				"  echo \"main has a newer commit, so this run publishes no prerelease\"\n" +
				"  exit 0\n" +
				"fi\n"
			stepsOfThisRepo := func(t *testing.T, workflow, job string) any {
				t.Helper()
				contents, err := os.ReadFile(filepath.Join(repoCatalogDir, "..", ".github", "workflows", workflow))
				require.NoError(t, err)
				return parsedYAML(t, string(contents))["jobs"].(map[string]any)[job].(map[string]any)["steps"]
			}
			releaseSteps := stepsOfThisRepo(t, "release.yml", "release")
			prereleaseSteps := stepsOfThisRepo(t, "prerelease.yml", "prerelease")
			require.NotEmpty(t, releaseSteps)
			require.NotEmpty(t, prereleaseSteps)

			for _, starterName := range targetsOf(t, "cli") {
				jobs := ciJobsWithCLI(t, starterName)

				require.Equal(t, releaseSteps, jobs["release"].(map[string]any)["steps"], starterName)
				var prerelease []any
				for _, written := range jobs["prerelease"].(map[string]any)["steps"].([]any) {
					step := maps.Clone(written.(map[string]any))
					switch step["name"] {
					case "Tag the commit":
						run, found := strings.CutPrefix(step["run"].(string), newestCommitOnly)
						require.True(t, found, starterName+": "+step["run"].(string))
						step["run"] = run
					case "Release", "Keep the 5 newest prereleases of main":
						require.Equal(t, "steps.tag.outputs.tag != ''", step["if"], starterName)
						delete(step, "if")
					}
					prerelease = append(prerelease, step)
				}
				require.Equal(t, prereleaseSteps, prerelease, starterName)
				require.Equal(t, map[string]any{"group": "prerelease", "cancel-in-progress": false}, jobs["prerelease"].(map[string]any)["concurrency"], starterName)
			}
		})

		publishJob := func(t *testing.T, jobs map[string]any) map[string]any {
			t.Helper()
			publish, ok := jobs["publish"].(map[string]any)
			require.True(t, ok, jobs)
			return publish
		}

		publishStep := func(t *testing.T, publish map[string]any, name string) map[string]any {
			t.Helper()
			for _, step := range publish["steps"].([]any) {
				if step.(map[string]any)["name"] == name {
					return step.(map[string]any)
				}
			}
			require.Fail(t, "the job publish has no step "+name, publish)
			return nil
		}

		t.Run("ci.yml of go-event-sourcing-service runs the job test on each push, and the job publish after test only for main and for a tag, with the tags of the short SHA and of the version, and main and latest only for the head of main, one run of a ref at a time", func(t *testing.T) {
			workflow := parsedYAML(t, writtenTarget(t, "go-event-sourcing-service", nil, nil)[".github/workflows/ci.yml"])
			jobs := workflow["jobs"].(map[string]any)
			publish := publishJob(t, jobs)
			meta := publishStep(t, publish, "Docker meta")["with"]

			require.Equal(t, map[string]any{"push": map[string]any{}}, workflow["on"])
			require.NotContains(t, jobs["test"], "if")
			require.Equal(t, []any{"test"}, publish["needs"])
			require.Equal(t, "github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/tags/')", publish["if"])
			require.Equal(t, map[string]any{
				"images": "ghcr.io/${{ github.repository }}-${{ matrix.app }}",
				"flavor": "latest=false",
				"tags":   "type=sha,prefix=\ntype=ref,event=branch,enable=${{ steps.head.outputs.newest == 'true' }}\ntype=semver,pattern={{version}}\ntype=raw,value=latest,enable=${{ steps.head.outputs.newest == 'true' }}\n",
			}, meta)
			headStep := maps.Clone(publishStep(t, publish, "Find whether the commit is the head of main"))
			delete(headStep, "run")
			require.Equal(t, map[string]any{"name": "Find whether the commit is the head of main", "id": "head", "if": "github.ref == 'refs/heads/main'", "shell": "bash"}, headStep)
			require.Equal(t, map[string]any{"group": "${{ github.workflow }}-${{ github.ref }}", "cancel-in-progress": false}, workflow["concurrency"])
		})

		for name, head := range map[string]struct {
			lsRemote string
			fails    bool
			outputs  string
		}{
			"when the commit of the run is the head of main, the step that finds the head of main writes newest=true to its outputs": {"printf '%s\\t%s\\n' 4829f92e0c5d1b7a3f6e8d9c2b1a0f9e8d7c6b5a refs/heads/main", false, "newest=true\n"},
			"when main has a newer commit, the step that finds the head of main writes no output":                                    {"printf '%s\\t%s\\n' 9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b refs/heads/main", false, ""},
			"when git ls-remote fails, the step that finds the head of main fails and writes no output":                              {"echo 'fatal: unable to access the remote' >&2; exit 128", true, ""},
		} {
			t.Run("in the job publish of go-event-sourcing-service, run with shell bash as GitHub Actions runs it, "+name, func(t *testing.T) {
				publish := publishJob(t, parsedYAML(t, writtenTarget(t, "go-event-sourcing-service", nil, nil)[".github/workflows/ci.yml"])["jobs"].(map[string]any))
				script := publishStep(t, publish, "Find whether the commit is the head of main")["run"].(string)
				stubs := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(stubs, "git"), []byte("#!/bin/sh\ncase \"$*\" in\n*ls-remote*main*) "+head.lsRemote+" ;;\n*) exit 2 ;;\nesac\n"), 0o755))
				githubOutput := filepath.Join(t.TempDir(), "github-output")
				require.NoError(t, os.WriteFile(githubOutput, nil, 0o644))
				shell := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
				shell.Env = append(os.Environ(),
					"PATH="+stubs+string(os.PathListSeparator)+os.Getenv("PATH"),
					"GITHUB_SHA=4829f92e0c5d1b7a3f6e8d9c2b1a0f9e8d7c6b5a", "GITHUB_OUTPUT="+githubOutput,
				)

				printed, err := shell.CombinedOutput()

				require.Equal(t, head.fails, err != nil, string(printed))
				written, readErr := os.ReadFile(githubOutput)
				require.NoError(t, readErr)
				require.Equal(t, head.outputs, string(written))
			})
		}

		t.Run("the job publish of go-event-sourcing-service gives the build of each image the SHA of its commit as VERSION", func(t *testing.T) {
			publish := publishJob(t, parsedYAML(t, writtenTarget(t, "go-event-sourcing-service", nil, nil)[".github/workflows/ci.yml"])["jobs"].(map[string]any))

			require.Equal(t, "VERSION=${{ github.sha }}", publishStep(t, publish, "Build and publish")["with"].(map[string]any)["build-args"])
		})

		t.Run("the jobs test and publish of ci.yml of go-event-sourcing-service have the steps of the go-event-sourcing-service of repo-templates, apart from the step that finds the head of main, the flavor of Docker meta, the head of main in the tags main and latest, and the VERSION of the build", func(t *testing.T) {
			upstream := parsedYAML(t, `name: CI

on:
  push: {}

permissions:
  contents: read

jobs:

  test:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@v7

      - name: Set up Go
        uses: actions/setup-go@v7
        with:
          go-version-file: 'go.mod'

      - name: Mise install
        uses: jdx/mise-action@v5
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}

      - name: Run shellcheck
        run: task test:shellcheck

      - name: Run Unit Test
        run: task test:unit

      - name: Run Integration Test
        run: task test:integration

      - name: Run vulnerabilities check
        run: task test:vulnerabilities

  publish:
    runs-on: ubuntu-latest
    needs:
      - test
    if: github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/tags/')
    permissions:
      contents: read
      packages: write
    strategy:
      matrix:
        app: [api, catchup]
    steps:
      - uses: actions/checkout@v7

      - name: Docker meta
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ghcr.io/${{ github.repository }}-${{ matrix.app }}
          tags: |
            type=sha,prefix=
            type=ref,event=branch
            type=semver,pattern={{version}}
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.repository_owner }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and publish
        uses: docker/build-push-action@v6
        with:
          context: .
          file: Dockerfile.${{ matrix.app }}
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
`)["jobs"].(map[string]any)
			jobs := parsedYAML(t, writtenTarget(t, "go-event-sourcing-service", nil, nil)[".github/workflows/ci.yml"])["jobs"].(map[string]any)

			for _, name := range []string{"test", "publish"} {
				var steps []any
				for _, written := range jobs[name].(map[string]any)["steps"].([]any) {
					step := maps.Clone(written.(map[string]any))
					switch step["name"] {
					case "Find whether the commit is the head of main":
						continue
					case "Docker meta":
						with := maps.Clone(step["with"].(map[string]any))
						delete(with, "flavor")
						with["tags"] = strings.NewReplacer(
							"type=ref,event=branch,enable=${{ steps.head.outputs.newest == 'true' }}", "type=ref,event=branch",
							"type=raw,value=latest,enable=${{ steps.head.outputs.newest == 'true' }}", "type=raw,value=latest,enable={{is_default_branch}}",
						).Replace(with["tags"].(string))
						step["with"] = with
					case "Build and publish":
						with := maps.Clone(step["with"].(map[string]any))
						delete(with, "build-args")
						step["with"] = with
					}
					steps = append(steps, step)
				}
				require.Equal(t, upstream[name].(map[string]any)["steps"], steps, name)
			}
		})

		t.Run("for each app of the matrix of publish, a repo of go-event-sourcing-service has the Dockerfile of the app", func(t *testing.T) {
			files := writtenTarget(t, "go-event-sourcing-service", nil, nil)
			publish := publishJob(t, parsedYAML(t, files[".github/workflows/ci.yml"])["jobs"].(map[string]any))
			apps := publish["strategy"].(map[string]any)["matrix"].(map[string]any)["app"].([]any)

			require.Equal(t, []any{"api", "catchup"}, apps)
			for _, app := range apps {
				require.Contains(t, files, "Dockerfile."+app.(string))
			}
		})

		t.Run("for each app of the matrix of publish, a repo of go-event-sourcing-service has an ignore file beside the Dockerfile of the app with the lines of the .dockerignore of the go-event-sourcing-service of repo-templates", func(t *testing.T) {
			files := writtenTarget(t, "go-event-sourcing-service", nil, nil)
			publish := publishJob(t, parsedYAML(t, files[".github/workflows/ci.yml"])["jobs"].(map[string]any))

			for _, app := range publish["strategy"].(map[string]any)["matrix"].(map[string]any)["app"].([]any) {
				require.Equal(t, ".git\n.DS_Store\n**/node_modules\n**/dist\n**/bin\n**/.env.*\n**/*.out\n", files["Dockerfile."+app.(string)+".dockerignore"], app)
			}
		})

		t.Run("for each app of the matrix of publish, the last stage of the Dockerfile of the app copies only the binary of the app from the build stage", func(t *testing.T) {
			files := writtenTarget(t, "go-event-sourcing-service", nil, nil)
			publish := publishJob(t, parsedYAML(t, files[".github/workflows/ci.yml"])["jobs"].(map[string]any))

			for _, app := range publish["strategy"].(map[string]any)["matrix"].(map[string]any)["app"].([]any) {
				dockerfile := files["Dockerfile."+app.(string)]
				lastStage := dockerfile[strings.LastIndex(dockerfile, "\nFROM "):]
				var copies []string
				for _, line := range strings.Split(lastStage, "\n") {
					instruction := strings.ToUpper(strings.SplitN(strings.TrimSpace(line), " ", 2)[0])
					if instruction == "COPY" || instruction == "ADD" {
						copies = append(copies, strings.TrimSpace(line))
					}
				}
				require.Equal(t, []string{"COPY --from=build-env /build/" + app.(string) + " ."}, copies, app)
			}
		})

		t.Run("with cli, a repo of go-event-sourcing-service has the .dockerignore of cli and the ignore file of each image", func(t *testing.T) {
			files := writtenTarget(t, "go-event-sourcing-service", nil, withoutValues("cli"))

			require.Contains(t, files, ".dockerignore")
			require.Contains(t, files, "Dockerfile.api.dockerignore")
			require.Contains(t, files, "Dockerfile.catchup.dockerignore")
		})

		t.Run("with each feature that targets go-event-sourcing-service, the workflow and the job test of ci.yml read the contents, and only the job publish writes the packages", func(t *testing.T) {
			definitions, err := repoCatalog().Definitions()
			require.NoError(t, err)
			var featureNames []string
			for name, feature := range definitions.Features {
				if slices.Contains(feature.Targets, "go-event-sourcing-service") {
					featureNames = append(featureNames, name)
				}
			}
			workflow := parsedYAML(t, writtenTarget(t, "go-event-sourcing-service", nil, withoutValues(featureNames...))[".github/workflows/ci.yml"])
			jobs := workflow["jobs"].(map[string]any)

			require.Subset(t, slices.Collect(maps.Keys(jobs)), []string{"test", "publish", "e2e", "release", "prerelease"})
			require.Equal(t, map[string]any{"contents": "read"}, workflow["permissions"])
			require.Equal(t, map[string]any{"contents": "read"}, jobs["test"].(map[string]any)["permissions"])
			require.Equal(t, map[string]any{"contents": "read", "packages": "write"}, publishJob(t, jobs)["permissions"])
			for name, job := range jobs {
				if name != "publish" {
					permissions, _ := job.(map[string]any)["permissions"].(map[string]any)
					require.NotContains(t, permissions, "packages", name)
				}
			}
		})

		t.Run("the README of go-event-sourcing-service tells that a push to main and a tag publish the images of the API and of catchup to the GitHub container registry after the tests, with their names and their tags", func(t *testing.T) {
			_, files := writtenStarter(t, nil)
			readme := files["README.md"]
			require.Contains(t, readme, "\n## Publish the Images\n")
			section := readme[strings.Index(readme, "\n## Publish the Images\n"):strings.Index(readme, "\n## Replace the Example\n")]

			for _, part := range []string{
				"a push to `main`", "a push of a tag", "after the tests pass", "A push to another branch runs only the tests",
				"GitHub container registry", "`ghcr.io/<owner>/<repository>-api`", "`ghcr.io/<owner>/<repository>-catchup`",
				"short SHA", "`main` and `latest`", "`v1.2.3`", "`1.2.3`", "Only an image from `main` gets `latest`",
				"a re-run of an older run", "publishes its image with the short SHA only", "When the tests of the newest commit of `main` fail",
			} {
				require.Contains(t, section, part)
			}
		})

		t.Run("Taskfile.cli.yml builds the CLI into bin with the kebab-case name of the repo, runs it from the source with the arguments of task, makes a GoReleaser snapshot, and runs the end-to-end tests in Docker and with Node against a binary of the tag v0.1.0", func(t *testing.T) {
			_, files := writtenWithCLI(t, "MyService.v2", nil)

			taskfile := parsedYAML(t, files["Taskfile.cli.yml"])
			require.Equal(t, map[string]any{"E2E_TAG": "v0.1.0"}, taskfile["vars"])
			tasks := taskfile["tasks"].(map[string]any)
			require.Equal(t, map[string]any{
				"cmds": []any{"mkdir -p ./bin", "go build -o ./bin/my-service-v2 ./cmd/my-service-v2"},
				"env":  map[string]any{"CGO_ENABLED": "0"},
			}, tasks["build"])
			require.Equal(t, map[string]any{"cmds": []any{"go run ./cmd/my-service-v2 {{.CLI_ARGS}}"}}, tasks["cli"])
			require.Equal(t, map[string]any{"cmds": []any{"goreleaser release --snapshot --clean"}}, tasks["release:local"])
			require.Equal(t, []any{
				"docker build -f e2e/Dockerfile --build-arg BUILD_TAG={{.E2E_TAG}} -t my-service-v2-e2e .",
				"docker container run --rm my-service-v2-e2e",
			}, tasks["test:e2e"].(map[string]any)["cmds"])
			require.Equal(t, "e2e", tasks["test:e2e:local"].(map[string]any)["dir"])
			require.Equal(t, []any{
				`CGO_ENABLED=0 go build -ldflags "-X github.com/hpcsc/MyService.v2/internal/cli/version.releaseTag={{.E2E_TAG}}" -o ../bin/my-service-v2-e2e ../cmd/my-service-v2`,
				"npm install --no-audit --no-fund",
				"EXECUTABLE=../bin/my-service-v2-e2e BUILD_TAG={{.E2E_TAG}} npx vitest run",
			}, tasks["test:e2e:local"].(map[string]any)["cmds"])
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
			starter, err := repoCatalog().Starter("go-event-sourcing-service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", map[string]string{"module": "example.com/acme/ledger"}, withoutValues("cli"), catalog.Answers{})
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

		for _, starterName := range []string{"chi-api", "go-event-sourcing-service"} {
			t.Run("a repo of "+starterName+" named api with cli stops the write, because the starter and cli both write cmd/api/main.go", func(t *testing.T) {
				starter, err := repoCatalog().Starter(starterName)
				require.NoError(t, err)
				selection, err := starter.Select("api", nil, withoutValues("cli"), catalog.Answers{})
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "api")

				err = selection.Write(output)

				require.ErrorContains(t, err, "of the starter")
				require.ErrorContains(t, err, "of the feature cli")
				require.ErrorContains(t, err, "both write cmd/api/main.go")
				require.NoDirExists(t, output)
			})
		}

		for name, migration := range map[string]struct {
			env     []string
			sslMode string
		}{
			"DB_SSLMODE=require":     {[]string{"DB_SSLMODE=require"}, "require"},
			"DB_SSLMODE=verify-full": {[]string{"DB_SSLMODE=verify-full"}, "verify-full"},
			"an empty DB_SSLMODE":    {[]string{"DB_SSLMODE="}, "disable"},
			"no DB_SSLMODE":          {nil, "disable"},
		} {
			t.Run("with "+name+", the entrypoint of the migrations gives migrate the database of the DB_ variables with the SSL mode "+migration.sslMode, func(t *testing.T) {
				output, _ := writtenStarter(t, nil)
				stubs := t.TempDir()
				recorded := filepath.Join(stubs, "migrate-arguments")
				require.NoError(t, os.WriteFile(filepath.Join(stubs, "nc"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(stubs, "migrate"), []byte("#!/bin/sh\nfor argument in \"$@\"; do printf '%s\\n' \"$argument\" >> \""+recorded+"\"; done\n"), 0o755))
				var env []string
				for _, variable := range os.Environ() {
					if !strings.HasPrefix(variable, "DB_") && !strings.HasPrefix(variable, "PATH=") {
						env = append(env, variable)
					}
				}
				entrypoint := exec.Command("sh", filepath.Join(output, "scripts", "local", "migrate", "entrypoint.sh"))
				entrypoint.Env = append(env, append([]string{
					"PATH=" + stubs + string(os.PathListSeparator) + os.Getenv("PATH"),
					"DB_USERNAME=ledger-user", "DB_PASSWORD=ledger-password", "DB_HOST=db.example.test", "DB_PORT=6543", "DB_NAME=ledger",
				}, migration.env...)...)

				printed, err := entrypoint.CombinedOutput()

				require.NoError(t, err, string(printed))
				arguments, readErr := os.ReadFile(recorded)
				require.NoError(t, readErr, string(printed))
				require.Equal(t, []string{
					"-database", "postgres://ledger-user:ledger-password@db.example.test:6543/ledger?sslmode=" + migration.sslMode,
					"-path", "/app/migrations",
					"up",
				}, strings.Split(strings.TrimSuffix(string(arguments), "\n"), "\n"))
			})
		}

		t.Run("the README of go-event-sourcing-service tells that the migrate container connects with the SSL mode of DB_SSLMODE, and with disable when it has no value", func(t *testing.T) {
			_, files := writtenStarter(t, nil)

			for _, part := range []string{
				"The `migrate` container applies them with\n`scripts/local/migrate/entrypoint.sh`", "the SSL mode of `DB_SSLMODE`",
				"`require` for a database that needs SSL", "With no value, the mode is `disable`.",
			} {
				require.Contains(t, files["README.md"], part)
			}
		})

		shellCheckWith := func(t *testing.T, rootScripts map[string]string) (checked []string, failed bool) {
			t.Helper()
			output, files := writtenStarter(t, nil)
			for path, contents := range rootScripts {
				require.NoError(t, os.WriteFile(filepath.Join(output, path), []byte(contents), 0o755))
			}
			stubs := t.TempDir()
			recorded := filepath.Join(stubs, "shellcheck-arguments")
			require.NoError(t, os.WriteFile(filepath.Join(stubs, "shellcheck"), []byte("#!/bin/sh\nstatus=3\nfor argument in \"$@\"; do\n  printf '%s\\n' \"$argument\" >> \""+recorded+"\"\n  case \"$argument\" in\n  -S|error) ;;\n  *)\n    [ -f \"$argument\" ] || exit 2\n    [ \"$status\" = 3 ] && status=0\n    if grep -q 'an error for shellcheck' \"$argument\"; then status=1; fi\n    ;;\n  esac\ndone\nexit $status\n"), 0o755))
			for _, command := range parsedYAML(t, files["Taskfile.yml"])["tasks"].(map[string]any)["test:shellcheck"].(map[string]any)["cmds"].([]any) {
				shell := exec.Command("sh", "-c", command.(string))
				shell.Dir = output
				shell.Env = append(os.Environ(), "PATH="+stubs+string(os.PathListSeparator)+os.Getenv("PATH"))
				if shell.Run() != nil {
					failed = true
					break
				}
			}
			arguments, err := os.ReadFile(recorded)
			require.NoError(t, err)
			return strings.Split(strings.TrimSuffix(string(arguments), "\n"), "\n"), failed
		}

		t.Run("test:shellcheck of go-event-sourcing-service gives shellcheck each .sh file in the root folder of the repo, and fails when shellcheck finds an error in one of them", func(t *testing.T) {
			checked, failed := shellCheckWith(t, map[string]string{
				"deploy.sh":  "#!/bin/sh\n# an error for shellcheck\n",
				"release.sh": "#!/bin/sh\necho release\n",
			})

			require.True(t, failed)
			require.Subset(t, checked, []string{"./deploy.sh", "./release.sh"})
		})

		for name, rootScripts := range map[string]map[string]string{
			"a root script with no error": {"release.sh": "#!/bin/sh\necho release\n"},
			"no root script":              nil,
		} {
			t.Run("with "+name+", test:shellcheck of go-event-sourcing-service passes and still gives shellcheck each .sh file under scripts and each file of hooks", func(t *testing.T) {
				checked, failed := shellCheckWith(t, rootScripts)

				require.False(t, failed)
				require.Subset(t, checked, []string{
					"./scripts/libs/git-hook-prepush.sh", "./scripts/local/migrate/entrypoint.sh", "./scripts/test/start-test-dependencies.sh", "./hooks/pre-push",
				})
				for path := range rootScripts {
					require.Contains(t, checked, "./"+path)
				}
			})
		}

		t.Run("hooks/pre-push and the scripts can be run", func(t *testing.T) {
			output, _ := writtenStarter(t, nil)

			for _, path := range []string{"hooks/pre-push", "scripts/libs/git-hook-prepush.sh", "scripts/local/migrate/entrypoint.sh", "scripts/test/start-test-dependencies.sh"} {
				require.Equal(t, 0o755&^umask(), modeOf(t, filepath.Join(output, path)), path)
			}
		})

		writtenGo := func(t *testing.T, given map[string]string) (string, map[string]string) {
			t.Helper()
			starter, err := repoCatalog().Starter("go")
			require.NoError(t, err)
			selection, err := starter.Select("my-tool", given, nil, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-tool")
			require.NoError(t, selection.Write(output))
			return output, filesIn(t, output)
		}

		t.Run("the starter go writes a Go module with an example package, the tasks, the tools, the CI workflow and the pre-push hook, and no other file", func(t *testing.T) {
			_, files := writtenGo(t, nil)

			require.ElementsMatch(t, []string{
				"go.mod", ".gitignore", "mise.toml", "Taskfile.yml", "Taskfile.test.yml", ".github/workflows/ci.yml", "hooks/pre-push",
				"scripts/libs/git-hook-prepush.sh", "README.md", "internal/greeting/greeting.go", "internal/greeting/greeting_test.go",
			}, slices.Collect(maps.Keys(files)))
			require.Contains(t, files["go.mod"], "module github.com/hpcsc/my-tool\n")
			require.Contains(t, files["go.mod"], "go 1.27.1\n")
			require.Contains(t, files["mise.toml"], `go = "1.27.1"`)
			require.Contains(t, files["internal/greeting/greeting_test.go"], `"github.com/hpcsc/my-tool/internal/greeting"`)
		})

		t.Run("given values for the module and the Go version replace the defaults in each file of go", func(t *testing.T) {
			_, files := writtenGo(t, map[string]string{"module": "github.com/acme/ledger", "go-version": "1.28.0"})

			require.Contains(t, files["go.mod"], "module github.com/acme/ledger\n")
			require.Contains(t, files["go.mod"], "go 1.28.0\n")
			require.Contains(t, files["mise.toml"], `go = "1.28.0"`)
			require.Contains(t, files["internal/greeting/greeting_test.go"], `"github.com/acme/ledger/internal/greeting"`)
			require.Contains(t, files["README.md"], "`github.com/acme/ledger`")
			for path, contents := range files {
				require.NotContains(t, contents, "github.com/hpcsc/my-tool", path)
				require.NotContains(t, contents, "1.27.1", path)
			}
		})

		t.Run("the Taskfiles and the workflow of go keep the template syntax of Task and of GitHub Actions", func(t *testing.T) {
			_, files := writtenGo(t, nil)

			require.Contains(t, files["Taskfile.test.yml"], "{{.CLI_ARGS}}")
			require.Contains(t, files[".github/workflows/ci.yml"], "${{ secrets.GITHUB_TOKEN }}")
		})

		t.Run("with no item, the Taskfile of go includes only the tests of the starter", func(t *testing.T) {
			_, files := writtenGo(t, nil)

			require.Contains(t, files["Taskfile.yml"], "includes:\n  test: Taskfile.test.yml\n\ntasks:\n")
		})

		t.Run("no file of go holds a value of scaffold", func(t *testing.T) {
			_, files := writtenGo(t, nil)

			for path, contents := range files {
				for _, value := range []string{".Scaffold.", ".ProjectKebab", "<no value>", "zzsentinel"} {
					require.NotContains(t, contents, value, path)
				}
			}
		})

		t.Run("hooks/pre-push and the script of the hook of go can be run", func(t *testing.T) {
			output, _ := writtenGo(t, nil)

			for _, path := range []string{"hooks/pre-push", "scripts/libs/git-hook-prepush.sh"} {
				require.Equal(t, 0o755&^umask(), modeOf(t, filepath.Join(output, path)), path)
			}
		})

		t.Run("the README of go names the module, the install of the tools, the unit tests and the hook, and only paths and tasks that the repo has", func(t *testing.T) {
			_, files := writtenGo(t, nil)
			readme := files["README.md"]

			require.True(t, strings.HasPrefix(readme, "# my-tool\n"))
			for _, part := range []string{"`github.com/hpcsc/my-tool`", "mise install", "task test:unit", "task hook:setup"} {
				require.Contains(t, readme, part)
			}
			for _, path := range regexp.MustCompile("`([A-Za-z0-9_-]*[./][A-Za-z0-9_./-]*)`").FindAllStringSubmatch(readme, -1) {
				if path[1] == "github.com/hpcsc/my-tool" {
					continue
				}
				require.True(t, slices.ContainsFunc(slices.Collect(maps.Keys(files)), func(file string) bool {
					return file == path[1] || strings.HasPrefix(file, path[1]+"/")
				}), path[1])
			}
			tasks := tasksOf(t, files)
			for _, task := range regexp.MustCompile(`task ([a-z0-9:-]+)`).FindAllStringSubmatch(readme, -1) {
				require.Contains(t, tasks, task[1])
			}
		})

		writtenChiAPI := func(t *testing.T, given map[string]string) (string, map[string]string) {
			t.Helper()
			starter, err := repoCatalog().Starter("chi-api")
			require.NoError(t, err)
			selection, err := starter.Select("my-api", given, nil, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-api")
			require.NoError(t, selection.Write(output))
			return output, filesIn(t, output)
		}

		t.Run("the starter chi-api writes an HTTP API in cmd/api with its config, server, middleware, routes and example use cases, the tasks, the tools, the Dockerfile, the CI workflow and the pre-push hook, and no other file", func(t *testing.T) {
			_, files := writtenChiAPI(t, nil)

			require.ElementsMatch(t, []string{
				"go.mod", ".gitignore", "mise.toml", "Taskfile.yml", "Taskfile.api.yml", "Taskfile.test.yml", "Dockerfile", ".github/workflows/ci.yml",
				"hooks/pre-push", "scripts/libs/git-hook-prepush.sh", "README.md",
				"cmd/api/main.go", "internal/config/config.go", "internal/server/server.go",
				"internal/middleware/request_logger.go", "internal/middleware/request_logger_test.go",
				"internal/middleware/request/clock.go", "internal/middleware/request/clock_fake.go", "internal/middleware/request/clock_real.go",
				"internal/middleware/request/log_entry.go", "internal/middleware/request/route_matcher.go", "internal/middleware/request/route_matcher_test.go",
				"internal/response/response.go", "internal/response/response_test.go", "internal/route/routable.go", "internal/route/route.go",
				"internal/test/common.go", "internal/test/fake_log_handler.go",
				"internal/use_cases/register.go", "internal/use_cases/register_test.go", "internal/use_cases/root/handler.go",
				"internal/use_cases/user/handler.go", "internal/use_cases/user/handler_test.go", "internal/use_cases/user/post_request.go",
			}, slices.Collect(maps.Keys(files)))
			require.Contains(t, files["go.mod"], "module github.com/hpcsc/my-api\n")
			require.Contains(t, files["go.mod"], "go 1.27.1\n")
			require.Contains(t, files["mise.toml"], `go = "1.27.1"`)
			require.Contains(t, files["Dockerfile"], "FROM golang:1.27.1 ")
		})

		t.Run("given values for the module and the Go version replace the defaults in each file of chi-api", func(t *testing.T) {
			_, files := writtenChiAPI(t, map[string]string{"module": "github.com/acme/ledger", "go-version": "1.28.0"})

			require.Contains(t, files["go.mod"], "module github.com/acme/ledger\n")
			require.Contains(t, files["go.mod"], "go 1.28.0\n")
			require.Contains(t, files["mise.toml"], `go = "1.28.0"`)
			require.Contains(t, files["Dockerfile"], "FROM golang:1.28.0 ")
			require.Contains(t, files["Dockerfile"], "-X 'github.com/acme/ledger/internal/use_cases/root.Version=${VERSION}'")
			require.Contains(t, files["Taskfile.api.yml"], "-X 'github.com/acme/ledger/internal/use_cases/root.Version=${VERSION}'")
			require.Contains(t, files["cmd/api/main.go"], `"github.com/acme/ledger/internal/server"`)
			require.Contains(t, files["README.md"], "`github.com/acme/ledger`")
			for path, contents := range files {
				require.NotContains(t, contents, "github.com/hpcsc/my-api", path)
				require.NotContains(t, contents, "1.27.1", path)
			}
		})

		t.Run("with no item, the Taskfile of chi-api includes only the tests and the tasks of the API", func(t *testing.T) {
			_, files := writtenChiAPI(t, nil)

			require.Contains(t, files["Taskfile.yml"], "includes:\n  test: Taskfile.test.yml\n  api: Taskfile.api.yml\n\ntasks:\n")
		})

		t.Run("the Taskfiles and the workflow of chi-api keep the template syntax of Task and of GitHub Actions", func(t *testing.T) {
			_, files := writtenChiAPI(t, nil)

			require.Contains(t, files["Taskfile.test.yml"], "{{.CLI_ARGS}}")
			require.Contains(t, files["Taskfile.api.yml"], `-t my-api-api:{{ default "latest" .TAG }}`)
			require.Contains(t, files[".github/workflows/ci.yml"], "${{ secrets.GITHUB_TOKEN }}")
		})

		t.Run("no file of chi-api holds a value of scaffold", func(t *testing.T) {
			_, files := writtenChiAPI(t, nil)

			for path, contents := range files {
				for _, value := range []string{".Scaffold.", ".ProjectKebab", "<no value>", "zzsentinel"} {
					require.NotContains(t, contents, value, path)
				}
			}
		})

		t.Run("hooks/pre-push and the script of the hook of chi-api can be run", func(t *testing.T) {
			output, _ := writtenChiAPI(t, nil)

			for _, path := range []string{"hooks/pre-push", "scripts/libs/git-hook-prepush.sh"} {
				require.Equal(t, 0o755&^umask(), modeOf(t, filepath.Join(output, path)), path)
			}
		})

		t.Run("chi-api has no shared-token check: no token file, and no file reads, sets, sends or describes a token", func(t *testing.T) {
			_, files := writtenChiAPI(t, nil)

			for _, path := range []string{"internal/middleware/auth.go", "internal/middleware/auth_test.go", "internal/middleware/testdata/valid-token", "local/token"} {
				require.NotContains(t, files, path)
			}
			for path, contents := range files {
				withoutGitHubToken := strings.NewReplacer("github_token", "", "GITHUB_TOKEN", "").Replace(contents)
				for _, word := range []string{"token", "bearer", "authorization"} {
					require.NotContains(t, strings.ToLower(withoutGitHubToken), word, path)
				}
			}
		})

		writtenChiAPIWithGuard := func(t *testing.T) map[string]string {
			t.Helper()
			files := catalogOnDisk(t)
			files["features/guard/feature.yml"] = &fstest.MapFile{Data: []byte(`summary: Guard.
targets: [chi-api]
add_to:
  api.middleware:
    - code: guard.Middleware(cfg.Guard)
      import: internal/guard
    - code: guard.Audit(cfg.Guard)
      import: internal/guard
  api.config:
    - name: Guard
      type: guard.Config
      import: internal/guard
`)}
			starter, err := catalog.New(files).Starter("chi-api")
			require.NoError(t, err)
			selection, err := starter.Select("my-api", nil, withoutValues("guard"), catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-api")
			require.NoError(t, selection.Write(output))
			return filesIn(t, output)
		}

		t.Run("the server of chi-api gives the items of api.middleware, in their order, to the protected routes, and imports their package one time", func(t *testing.T) {
			files := writtenChiAPIWithGuard(t)

			require.Contains(t, files["internal/server/server.go"], "\tuse_cases.Register(r, protectedMiddleware(cfg), logger)\n")
			require.Contains(t, files["internal/server/server.go"], "return []func(http.Handler) http.Handler{\n\t\tguard.Middleware(cfg.Guard),\n\t\tguard.Audit(cfg.Guard),\n\t}")
			require.Equal(t, 1, strings.Count(files["internal/server/server.go"], `"github.com/hpcsc/my-api/internal/guard"`))
		})

		t.Run("the config of chi-api has a field for each item of api.config, and imports its package one time", func(t *testing.T) {
			files := writtenChiAPIWithGuard(t)

			require.Contains(t, files["internal/config/config.go"], "\tGuard guard.Config\n}")
			require.Equal(t, 1, strings.Count(files["internal/config/config.go"], `"github.com/hpcsc/my-api/internal/guard"`))
		})

		t.Run("with no item, the server of chi-api gives no middleware to the protected routes, and the config has only its own fields", func(t *testing.T) {
			_, files := writtenChiAPI(t, nil)

			require.Contains(t, files["internal/server/server.go"], "return []func(http.Handler) http.Handler{\n\t}")
			require.Contains(t, files["internal/server/server.go"], "\t\"github.com/hpcsc/my-api/internal/use_cases\"\n)")
			require.Contains(t, files["internal/config/config.go"], "import (\n\t\"fmt\"\n\n\t\"github.com/caarlos0/env/v6\"\n)\n")
			require.Contains(t, files["internal/config/config.go"], "type Config struct {\n\tPort string `env:\"PORT\" envDefault:\"3333\"`\n}")
		})

		t.Run("no written file of chi-api holds the name of an extension point of the starter, with or without items", func(t *testing.T) {
			definitions, err := repoCatalog().Definitions()
			require.NoError(t, err)
			extensionPoints := slices.Sorted(maps.Keys(definitions.Starters["chi-api"].ExtensionPoints))
			require.Subset(t, extensionPoints, []string{"api.middleware", "api.config"})
			_, withoutItems := writtenChiAPI(t, nil)

			for _, files := range []map[string]string{withoutItems, writtenChiAPIWithGuard(t)} {
				for path, contents := range files {
					for _, extensionPoint := range extensionPoints {
						require.NotContains(t, contents, extensionPoint, path)
					}
				}
			}
		})

		for _, method := range []string{"api-key", "oauth"} {
			t.Run("with api-auth of the method "+method+", the server and the config of chi-api hold the items of api-auth and import internal/auth one time", func(t *testing.T) {
				starter, err := repoCatalog().Starter("chi-api")
				require.NoError(t, err)
				selection, err := starter.Select("my-api", nil, map[string]map[string]string{"api-auth": {"method": method}}, catalog.Answers{})
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "my-api")
				require.NoError(t, selection.Write(output))
				files := filesIn(t, output)

				require.Contains(t, files["internal/server/server.go"], "return []func(http.Handler) http.Handler{\n\t\tauth.Middleware(cfg.Auth),\n\t}")
				require.Contains(t, files["internal/config/config.go"], "\tAuth auth.Config\n}")
				for _, path := range []string{"internal/server/server.go", "internal/config/config.go"} {
					require.Equal(t, 1, strings.Count(files[path], `"github.com/hpcsc/my-api/internal/auth"`), path)
				}
			})
		}

		t.Run("chi-api has the tasks of the API, the tests and the hook, and no task of cli", func(t *testing.T) {
			_, files := writtenChiAPI(t, nil)
			tasks := tasksOf(t, files)

			require.Subset(t, tasks, []string{"api:run", "api:build", "api:docker:build", "api:docker:run", "test:unit", "test:shellcheck", "test:vulnerabilities", "hook:setup", "hook:validate-go-changes"})
			for _, name := range []string{"build", "cli", "release:local", "test:e2e", "test:e2e:local"} {
				require.NotContains(t, tasks, name)
			}
		})

		t.Run("the README of chi-api names the module, the install of the tools, the run of the API with a setting, the unit tests and the hook, and only paths and tasks that the repo has", func(t *testing.T) {
			_, files := writtenChiAPI(t, nil)
			readme := files["README.md"]

			require.True(t, strings.HasPrefix(readme, "# my-api\n"))
			for _, part := range []string{"`github.com/hpcsc/my-api`", "mise install", "task api:run", "PORT=8080 task api:run", "task test:unit", "task hook:setup"} {
				require.Contains(t, readme, part)
			}
			for _, path := range regexp.MustCompile("`([A-Za-z0-9_-]*[./][A-Za-z0-9_./-]*)`").FindAllStringSubmatch(readme, -1) {
				if path[1] == "github.com/hpcsc/my-api" || strings.HasPrefix(path[1], "route.") {
					continue
				}
				require.True(t, slices.ContainsFunc(slices.Collect(maps.Keys(files)), func(file string) bool {
					return file == path[1] || strings.HasPrefix(file, path[1]+"/")
				}), path[1])
			}
			tasks := tasksOf(t, files)
			for _, task := range regexp.MustCompile(`task ([a-z0-9:-]+)`).FindAllStringSubmatch(readme, -1) {
				require.Contains(t, tasks, task[1])
			}
		})
	})

	t.Run("combinations", func(t *testing.T) {
		namesOf := func(t *testing.T, files fs.FS) []string {
			t.Helper()
			combinations, err := catalog.New(files).Combinations()
			require.NoError(t, err)
			names := make([]string, 0, len(combinations))
			for _, combination := range combinations {
				names = append(names, combination.String())
			}
			return names
		}
		oneFolder := &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\n")}
		mainGo := &fstest.MapFile{Data: []byte("package main\n")}

		t.Run("each starter, in the order of the names, gives a combination of the starter alone and one of each feature that targets it alone", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/worker/starter.yml":         oneFolder,
				"starters/worker/files/go/main.go":    mainGo,
				"starters/api/starter.yml":            oneFolder,
				"starters/api/files/go/main.go":       mainGo,
				"starters/static/starter.yml":         oneFolder,
				"starters/static/files/go/index.html": &fstest.MapFile{Data: []byte("<html></html>\n")},
				"features/docs/feature.yml":           &fstest.MapFile{Data: []byte("summary: Docs.\ntargets: [api]\n")},
				"features/cli/feature.yml":            &fstest.MapFile{Data: []byte("summary: CLI.\ntargets: [worker, api]\n")},
			}

			names := namesOf(t, files)

			require.Equal(t, []string{
				"api",
				"api --feature cli",
				"api --feature docs",
				"static",
				"worker",
				"worker --feature cli",
			}, names)
		})

		withValues := fstest.MapFS{
			"starters/api/starter.yml":         &fstest.MapFile{Data: []byte("options:\n  region:\n    values: [eu, us]\n    default: us\n  module:\n    default: example.com/{{ .Name }}\n  tier:\n    values: [standard]\n    default: standard\n  database:\n    values: [postgres, sqlite, mysql]\n    default: postgres\nfolders:\n  go:\n    path: .\n")},
			"starters/api/files/go/values.txt": &fstest.MapFile{Data: []byte(`{{ option "starter.database" }} {{ option "starter.region" }} {{ option "starter.tier" }}`)},
			"features/plain/feature.yml":       &fstest.MapFile{Data: []byte("summary: Plain.\ntargets: [api]\n")},
			"features/auth/feature.yml":        &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\nfiles:\n  - path: files/\noptions:\n  store:\n    values: [file, keychain]\n    default: file\n  realm:\n    default: \"{{ .Name }}-api\"\n  method:\n    values: [api-key, oauth, none]\n    default: oauth\n")},
			"features/auth/files/go/auth.txt":  &fstest.MapFile{Data: []byte(`{{ option "method" }} {{ option "store" }} {{ option "realm" }}`)},
		}
		defaults := "api --option database=postgres --option region=us --option tier=standard"

		t.Run("each other value of an option with values gives one combination, with the other options at their defaults", func(t *testing.T) {
			names := namesOf(t, withValues)

			require.Equal(t, []string{
				defaults,
				"api --option database=sqlite --option region=us --option tier=standard",
				"api --option database=mysql --option region=us --option tier=standard",
				"api --option database=postgres --option region=eu --option tier=standard",
				defaults + " --feature auth:method=oauth,store=file",
				defaults + " --feature auth:method=api-key,store=file",
				defaults + " --feature auth:method=none,store=file",
				defaults + " --feature auth:method=oauth,store=keychain",
				defaults + " --feature plain",
			}, names)
		})

		t.Run("the repo of each combination holds the values that its name gives", func(t *testing.T) {
			combinations, err := catalog.New(withValues).Combinations()
			require.NoError(t, err)
			written := map[string]map[string]string{}

			for _, combination := range combinations {
				starter, err := catalog.New(withValues).Starter(combination.StarterName)
				require.NoError(t, err)
				selection, err := starter.Select("my-service", combination.StarterValues, combination.FeatureValues, catalog.Answers{})
				require.NoError(t, err)
				output := filepath.Join(t.TempDir(), "my-service")
				require.NoError(t, selection.Write(output))
				written[combination.String()] = filesIn(t, output)
			}

			require.Equal(t, map[string]map[string]string{
				defaults: {"values.txt": "postgres us standard"},
				"api --option database=sqlite --option region=us --option tier=standard":   {"values.txt": "sqlite us standard"},
				"api --option database=mysql --option region=us --option tier=standard":    {"values.txt": "mysql us standard"},
				"api --option database=postgres --option region=eu --option tier=standard": {"values.txt": "postgres eu standard"},
				defaults + " --feature auth:method=oauth,store=file":                       {"values.txt": "postgres us standard", "auth.txt": "oauth file my-service-api"},
				defaults + " --feature auth:method=api-key,store=file":                     {"values.txt": "postgres us standard", "auth.txt": "api-key file my-service-api"},
				defaults + " --feature auth:method=none,store=file":                        {"values.txt": "postgres us standard", "auth.txt": "none file my-service-api"},
				defaults + " --feature auth:method=oauth,store=keychain":                   {"values.txt": "postgres us standard", "auth.txt": "oauth keychain my-service-api"},
				defaults + " --feature plain":                                              {"values.txt": "postgres us standard"},
			}, written)
		})

		t.Run("each pair of features that add items to one extension point gives one combination with both", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/files/go/main.go": mainGo,
				"starters/api/starter.yml":      &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n  api.config:\n    file: go/config.go\n    item: [name]\n")},
				"features/auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n  api.config:\n    - name: Auth\n")},
				"features/audit/feature.yml":    &fstest.MapFile{Data: []byte("summary: Audit.\ntargets: [api]\nadd_to:\n  api.config:\n    - name: Audit\n  api.middleware:\n    - code: audit.Middleware\n")},
				"features/trace/feature.yml":    &fstest.MapFile{Data: []byte("summary: Trace.\ntargets: [api]\nadd_to:\n  api.middleware:\n    - code: trace.Middleware\n")},
				"features/metrics/feature.yml":  &fstest.MapFile{Data: []byte("summary: Metrics.\ntargets: [api]\nadd_to:\n  metrics.collectors:\n    - code: runtime.Collector\n")},
				"features/cli/feature.yml":      &fstest.MapFile{Data: []byte("summary: CLI.\ntargets: [api]\nextension_points:\n  cli.commands:\n    file: go/root.go\n    item: [code]\n")},
				"features/greet/feature.yml":    &fstest.MapFile{Data: []byte("summary: Greet.\ntargets: [api]\nadd_to:\n  cli.commands:\n    - code: greet.Command\n")},
				"features/hello/feature.yml":    &fstest.MapFile{Data: []byte("summary: Hello.\ntargets: [api]\nadd_to:\n  cli.commands:\n    - code: hello.Command\n")},
				"features/other/feature.yml":    &fstest.MapFile{Data: []byte("summary: Other.\ntargets: [worker]\nadd_to:\n  api.middleware:\n    - code: other.Middleware\n")},
			}

			names := namesOf(t, files)

			require.Equal(t, []string{
				"api",
				"api --feature audit",
				"api --feature auth:method=api-key",
				"api --feature auth:method=oauth",
				"api --feature cli",
				"api --feature greet",
				"api --feature hello",
				"api --feature metrics",
				"api --feature trace",
				"api --feature audit --feature auth:method=api-key",
				"api --feature audit --feature trace",
				"api --feature auth:method=api-key --feature trace",
				"api --feature greet --feature hello",
			}, names)
		})

		t.Run("a pair in which one feature needs the other, also through a need of a need, gives no combination, and the repo of a feature holds its needs", func(t *testing.T) {
			adds := func(featureName, needs string) *fstest.MapFile {
				return &fstest.MapFile{Data: []byte("summary: A feature.\ntargets: [api]\n" + needs + "add_to:\n  api.middleware:\n    - code: " + featureName + "\n")}
			}
			files := fstest.MapFS{
				"starters/api/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code]\n")},
				"starters/api/files/go/server.go": &fstest.MapFile{Data: []byte(`{{ range items "api.middleware" }}{{ .code }};{{ end }}`)},
				"features/app/feature.yml":        adds("app", "needs: [side, mid]\n"),
				"features/side/feature.yml":       &fstest.MapFile{Data: []byte("summary: Side.\ntargets: [api]\n")},
				"features/mid/feature.yml":        &fstest.MapFile{Data: []byte("summary: Mid.\ntargets: [api]\nneeds: [base]\n")},
				"features/base/feature.yml":       adds("base", ""),
				"features/other/feature.yml":      adds("other", ""),
				"features/zone/feature.yml":       adds("zone", "needs: [base]\n"),
			}

			names := namesOf(t, files)
			starter, err := catalog.New(files).Starter("api")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, map[string]map[string]string{"app": {}}, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-service")
			require.NoError(t, selection.Write(output))

			require.Equal(t, []string{
				"api",
				"api --feature app",
				"api --feature base",
				"api --feature mid",
				"api --feature other",
				"api --feature side",
				"api --feature zone",
				"api --feature app --feature other",
				"api --feature app --feature zone",
				"api --feature base --feature other",
				"api --feature other --feature zone",
			}, names)
			require.Equal(t, map[string]string{"server.go": "base;app;"}, filesIn(t, output))
		})

		withNeed := func(clientNeeds string) fstest.MapFS {
			return fstest.MapFS{
				"starters/api/starter.yml":            oneFolder,
				"starters/api/files/go/main.go":       mainGo,
				"features/auth/feature.yml":           &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\nfiles:\n  - path: files/\noptions:\n  method:\n    values: [api-key, oauth, none]\n    default: api-key\n  store:\n    values: [file, keychain]\n    default: file\n")},
				"features/auth/files/go/auth.txt":     &fstest.MapFile{Data: []byte(`{{ option "method" }} {{ option "store" }}`)},
				"features/client/feature.yml":         &fstest.MapFile{Data: []byte("summary: Client.\ntargets: [api]\nfiles:\n  - path: files/\n" + clientNeeds)},
				"features/client/files/go/client.txt": &fstest.MapFile{Data: []byte(`{{ option "auth.method" }} {{ option "auth.store" }}`)},
			}
		}

		t.Run("each other value of an option with values of a feature that a feature needs gives one combination with both features, with the other options of the need at their defaults", func(t *testing.T) {
			names := namesOf(t, withNeed("needs: [auth]\n"))

			require.Equal(t, []string{
				"api",
				"api --feature auth:method=api-key,store=file",
				"api --feature auth:method=oauth,store=file",
				"api --feature auth:method=none,store=file",
				"api --feature auth:method=api-key,store=keychain",
				"api --feature client",
				"api --feature auth:method=oauth,store=file --feature client",
				"api --feature auth:method=none,store=file --feature client",
				"api --feature auth:method=api-key,store=keychain --feature client",
			}, names)
		})

		t.Run("an option that the need of a feature sets gives no combination with another value, and the other options of the need keep the value that the need sets", func(t *testing.T) {
			names := namesOf(t, withNeed("needs:\n  auth:\n    method: oauth\n"))

			require.Equal(t, []string{
				"api",
				"api --feature auth:method=api-key,store=file",
				"api --feature auth:method=oauth,store=file",
				"api --feature auth:method=none,store=file",
				"api --feature auth:method=api-key,store=keychain",
				"api --feature client",
				"api --feature auth:method=oauth,store=keychain --feature client",
			}, names)
		})

		t.Run("a need with no option with values, a feature with no need, and a need of a need give no combination with a value of a need", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/starter.yml":      oneFolder,
				"starters/api/files/go/main.go": mainGo,
				"features/auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  realm:\n    default: the-realm\n")},
				"features/mid/feature.yml":      &fstest.MapFile{Data: []byte("summary: Mid.\ntargets: [api]\nneeds: [auth]\n")},
				"features/top/feature.yml":      &fstest.MapFile{Data: []byte("summary: Top.\ntargets: [api]\nneeds: [mid]\n")},
				"features/plain/feature.yml":    &fstest.MapFile{Data: []byte("summary: Plain.\ntargets: [api]\noptions:\n  label:\n    default: plain\n")},
				"features/side/feature.yml":     &fstest.MapFile{Data: []byte("summary: Side.\ntargets: [api]\nneeds: [plain]\n")},
			}

			names := namesOf(t, files)

			require.Equal(t, []string{
				"api",
				"api --feature auth:method=api-key",
				"api --feature auth:method=oauth",
				"api --feature mid",
				"api --feature auth:method=oauth --feature mid",
				"api --feature plain",
				"api --feature side",
				"api --feature top",
			}, names)
		})

		t.Run("an option that another need of the feature sets in its own need gives no combination with another value, and Select takes each combination", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/starter.yml":      oneFolder,
				"starters/api/files/go/main.go": mainGo,
				"features/auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  store:\n    values: [file, keychain]\n    default: file\n")},
				"features/side/feature.yml":     &fstest.MapFile{Data: []byte("summary: Side.\ntargets: [api]\nneeds:\n  auth:\n    method: api-key\n")},
				"features/app/feature.yml":      &fstest.MapFile{Data: []byte("summary: App.\ntargets: [api]\nneeds: [auth, side]\n")},
			}

			combinations, err := catalog.New(files).Combinations()
			require.NoError(t, err)
			var names []string
			for _, combination := range combinations {
				starter, err := catalog.New(files).Starter(combination.StarterName)
				require.NoError(t, err)
				_, err = starter.Select("my-service", combination.StarterValues, combination.FeatureValues, catalog.Answers{})
				require.NoError(t, err, combination.String())
				names = append(names, combination.String())
			}

			require.Equal(t, []string{
				"api",
				"api --feature app",
				"api --feature app --feature auth:method=api-key,store=keychain",
				"api --feature auth:method=api-key,store=file",
				"api --feature auth:method=oauth,store=file",
				"api --feature auth:method=api-key,store=keychain",
				"api --feature side",
				"api --feature auth:method=api-key,store=keychain --feature side",
			}, names)
		})

		t.Run("two needs that set two values of one option of a need of the feature give no combination with a value of that need", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/starter.yml":      oneFolder,
				"starters/api/files/go/main.go": mainGo,
				"features/auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n  store:\n    values: [file, keychain]\n    default: file\n")},
				"features/keys/feature.yml":     &fstest.MapFile{Data: []byte("summary: Keys.\ntargets: [api]\nneeds:\n  auth:\n    method: api-key\n")},
				"features/tokens/feature.yml":   &fstest.MapFile{Data: []byte("summary: Tokens.\ntargets: [api]\nneeds:\n  auth:\n    method: oauth\n")},
				"features/app/feature.yml":      &fstest.MapFile{Data: []byte("summary: App.\ntargets: [api]\nneeds: [auth, keys, tokens]\n")},
			}

			names := namesOf(t, files)

			require.NotContains(t, strings.Join(names, "\n"), "--feature app --feature auth")
			require.Contains(t, names, "api --feature auth:method=oauth,store=keychain --feature tokens")
		})

		t.Run("an option with a when gives a combination for each of its values, the need of the feature gives its other values with the option at its default, and Select takes each combination", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/starter.yml":      oneFolder,
				"starters/api/files/go/main.go": mainGo,
				"features/auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n")},
				"features/client/feature.yml":   &fstest.MapFile{Data: []byte("summary: Client.\ntargets: [api]\nneeds: [auth]\noptions:\n  store-key:\n    values: [\"true\", \"false\"]\n    default: \"false\"\n    when: auth.method == \"api-key\"\n")},
			}

			combinations, err := catalog.New(files).Combinations()
			require.NoError(t, err)
			var names []string
			for _, combination := range combinations {
				starter, err := catalog.New(files).Starter(combination.StarterName)
				require.NoError(t, err)
				_, err = starter.Select("my-service", combination.StarterValues, combination.FeatureValues, catalog.Answers{})
				require.NoError(t, err, combination.String())
				names = append(names, combination.String())
			}

			require.Equal(t, []string{
				"api",
				"api --feature auth:method=api-key",
				"api --feature auth:method=oauth",
				"api --feature client:store-key=false",
				"api --feature client:store-key=true",
				"api --feature auth:method=oauth --feature client:store-key=false",
			}, names)
		})

		t.Run("a feature with two needs gives the combinations of the values of each need, in the order of the names of the needs", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/api/starter.yml":      oneFolder,
				"starters/api/files/go/main.go": mainGo,
				"features/auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [api]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n")},
				"features/cache/feature.yml":    &fstest.MapFile{Data: []byte("summary: Cache.\ntargets: [api]\noptions:\n  size:\n    values: [small, large]\n    default: small\n")},
				"features/app/feature.yml":      &fstest.MapFile{Data: []byte("summary: App.\ntargets: [api]\nneeds: [cache, auth]\n")},
			}

			names := namesOf(t, files)

			require.Equal(t, []string{
				"api",
				"api --feature app",
				"api --feature app --feature auth:method=oauth",
				"api --feature app --feature cache:size=large",
				"api --feature auth:method=api-key",
				"api --feature auth:method=oauth",
				"api --feature cache:size=small",
				"api --feature cache:size=large",
			}, names)
		})

		t.Run("the repo of each combination with a value of a need holds the values that its name gives", func(t *testing.T) {
			for clientNeeds, expected := range map[string]map[string]map[string]string{
				"needs: [auth]\n": {
					"api --feature auth:method=oauth,store=file --feature client":       {"main.go": "package main\n", "auth.txt": "oauth file", "client.txt": "oauth file"},
					"api --feature auth:method=none,store=file --feature client":        {"main.go": "package main\n", "auth.txt": "none file", "client.txt": "none file"},
					"api --feature auth:method=api-key,store=keychain --feature client": {"main.go": "package main\n", "auth.txt": "api-key keychain", "client.txt": "api-key keychain"},
				},
				"needs:\n  auth:\n    method: oauth\n": {
					"api --feature auth:method=oauth,store=keychain --feature client": {"main.go": "package main\n", "auth.txt": "oauth keychain", "client.txt": "oauth keychain"},
				},
			} {
				files := withNeed(clientNeeds)
				combinations, err := catalog.New(files).Combinations()
				require.NoError(t, err)
				written := map[string]map[string]string{}

				for _, combination := range combinations {
					if len(combination.FeatureValues) != 2 {
						continue
					}
					starter, err := catalog.New(files).Starter(combination.StarterName)
					require.NoError(t, err)
					selection, err := starter.Select("my-service", combination.StarterValues, combination.FeatureValues, catalog.Answers{})
					require.NoError(t, err, combination.String())
					output := filepath.Join(t.TempDir(), "my-service")
					require.NoError(t, selection.Write(output))
					written[combination.String()] = filesIn(t, output)
				}

				require.Equal(t, expected, written)
			}
		})

		t.Run("the catalog of this repo gives cli-auth with each value of store-key and with the method oauth for each target of cli-auth, no combination of store-key=true with oauth, and Select takes each combination", func(t *testing.T) {
			combinations, err := repoCatalog().Combinations()
			require.NoError(t, err)
			var names []string
			for _, combination := range combinations {
				starter, err := repoCatalog().Starter(combination.StarterName)
				require.NoError(t, err)
				_, err = starter.Select("my-service", combination.StarterValues, combination.FeatureValues, catalog.Answers{})
				require.NoError(t, err, combination.String())
				names = append(names, combination.String())
			}

			require.Subset(t, names, []string{
				"chi-api --feature cli-auth:store-key=false",
				"chi-api --feature cli-auth:store-key=true",
				"chi-api --feature api-auth:method=oauth --feature cli-auth:store-key=false",
				"go-event-sourcing-service --feature cli-auth:store-key=false",
				"go-event-sourcing-service --feature cli-auth:store-key=true",
				"go-event-sourcing-service --feature api-auth:method=oauth --feature cli-auth:store-key=false",
			})
			for _, name := range names {
				require.False(t, strings.Contains(name, "api-auth:method=oauth") && strings.Contains(name, "store-key=true"), name)
			}
		})

		t.Run("the catalog of this repo gives push with each value of delivery", func(t *testing.T) {
			combinations, err := repoCatalog().Combinations()
			require.NoError(t, err)
			var names []string
			for _, combination := range combinations {
				names = append(names, combination.String())
			}

			require.Subset(t, names, []string{
				"go-event-sourcing-service --feature push:delivery=pg-notify",
				"go-event-sourcing-service --feature push:delivery=nats",
			})
		})

		for _, failing := range []string{"starters", "starters/api/starter.yml", "features", "features/cli/feature.yml"} {
			t.Run("a read of "+failing+" that fails stops the list with its cause", func(t *testing.T) {
				files := failingFS{failing: failing, files: fstest.MapFS{
					"starters/api/starter.yml":      oneFolder,
					"starters/api/files/go/main.go": mainGo,
					"features/cli/feature.yml":      &fstest.MapFile{Data: []byte("summary: CLI.\ntargets: [api]\n")},
				}}

				_, err := catalog.New(files).Combinations()

				require.ErrorContains(t, err, "disk failure")
			})
		}
	})
}

func tasksOf(t *testing.T, files map[string]string) []string {
	t.Helper()
	type include struct {
		Taskfile string `yaml:"taskfile"`
		Flatten  bool   `yaml:"flatten"`
	}
	type taskfile struct {
		Includes map[string]yaml.Node `yaml:"includes"`
		Tasks    map[string]yaml.Node `yaml:"tasks"`
	}
	read := func(path string) taskfile {
		var parsed taskfile
		require.Contains(t, files, path)
		require.NoError(t, yaml.Unmarshal([]byte(files[path]), &parsed), path)
		return parsed
	}
	root := read("Taskfile.yml")
	tasks := slices.Collect(maps.Keys(root.Tasks))
	for namespace, node := range root.Includes {
		included := include{Taskfile: node.Value}
		if node.Kind == yaml.MappingNode {
			require.NoError(t, node.Decode(&included), namespace)
		}
		prefix := namespace + ":"
		if included.Flatten {
			prefix = ""
		}
		for name := range read(included.Taskfile).Tasks {
			tasks = append(tasks, prefix+name)
		}
	}
	return tasks
}

func requiredModulesOf(t *testing.T, goMod string) map[string]string {
	t.Helper()
	required := map[string]string{}
	inBlock := false
	for _, line := range strings.Split(goMod, "\n") {
		fields := strings.Fields(line)
		switch {
		case strings.TrimSpace(line) == "require (":
			inBlock = true
			continue
		case inBlock && strings.TrimSpace(line) == ")":
			inBlock = false
			continue
		case !inBlock && len(fields) == 3 && fields[0] == "require":
			fields = fields[1:]
		case !inBlock:
			continue
		}
		require.GreaterOrEqual(t, len(fields), 2, line)
		require.Regexp(t, `^v[0-9]+\.[0-9]+\.[0-9]+`, fields[1], line)
		required[fields[0]] = fields[1]
	}
	return required
}

func toolsOf(t *testing.T, miseConfig string) map[string]string {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(miseConfig, "\n"), "\n")
	require.Equal(t, "[tools]", lines[0])
	tool := regexp.MustCompile(`^(?:"([^"]+)"|([A-Za-z0-9_-]+)) = "([^"]+)"$`)
	tools := map[string]string{}
	for _, line := range lines[1:] {
		match := tool.FindStringSubmatch(line)
		require.NotNil(t, match, line)
		name := match[1] + match[2]
		require.NotContains(t, tools, name, line)
		tools[name] = match[3]
	}
	return tools
}

func catalogOnDisk(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	err := fs.WalkDir(os.DirFS(repoCatalogDir), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !strings.HasPrefix(path, "starters/") && !strings.HasPrefix(path, "features/") {
			return nil
		}
		contents, err := fs.ReadFile(os.DirFS(repoCatalogDir), path)
		files[path] = &fstest.MapFile{Data: contents}
		return err
	})
	require.NoError(t, err)
	return files
}
