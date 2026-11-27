//go:build unit

package browser_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/browser"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestModel(t *testing.T) {
	definitions := func() catalog.Definitions {
		return catalog.Definitions{
			Starters: map[string]catalog.StarterDefinition{
				"worker": {Summary: "A worker for jobs."},
				"api": {
					Summary: "An API that a worker calls.",
					Folders: map[string]catalog.FolderDefinition{
						"go":   {Path: ".", AfterRender: []string{"go mod tidy", "gofmt -w ."}, Test: []string{"task test:unit"}},
						"web":  {Path: "web", AfterRender: []string{"npm install"}},
						"docs": {Path: "docs"},
					},
					Options: map[string]catalog.OptionDefinition{
						"region": {Default: "us", Values: []string{"us", "eu"}},
						"module": {Default: "example.com/{{ .Name }}", Pattern: "^[a-z./]+$"},
						"tier":   {Default: "standard", Values: []string{"standard"}},
					},
					ExtensionPoints: map[string]catalog.ExtensionPointDefinition{
						"api.middleware": {Summary: "A middleware.", File: "go/server.go", Item: []string{"code", "import"}},
						"api.config":     {Summary: "A config field.", File: "go/config.go", Item: []string{"name"}},
					},
				},
				"zone": {Summary: "A zone with one feature."},
			},
			Features: map[string]catalog.FeatureDefinition{
				"cli": {Summary: "A command line program.", Targets: []string{"api"}, ExtensionPoints: map[string]catalog.ExtensionPointDefinition{
					"cli.commands": {Summary: "A command.", File: "go/root.go", Item: []string{"code"}},
					"cli.flags":    {Summary: "A flag.", File: "go/flags.go", Item: []string{"name"}},
				}},
				"auth": {
					Summary: "API keys.", Targets: []string{"api"}, Needs: map[string]map[string]string{"cli": {}},
					Options: map[string]catalog.OptionDefinition{"method": {Default: "api-key", Values: []string{"api-key", "oauth"}}},
					AddTo:   map[string][]map[string]string{"api.middleware": {{"code": "auth.Middleware"}}, "api.config": {{"name": "Auth"}}},
				},
				"docs":       {Summary: "Docs.", Targets: []string{"api"}},
				"zone-tools": {Summary: "Tools of the zone.", Targets: []string{"zone"}},
				"other":      {Summary: "Only for another starter.", Targets: []string{"web"}},
			},
		}
	}
	manyStarters := func(count int) catalog.Definitions {
		many := catalog.Definitions{Starters: map[string]catalog.StarterDefinition{}}
		for i := range count {
			many.Starters[fmt.Sprintf("starter-%02d", i)] = catalog.StarterDefinition{Summary: fmt.Sprintf("Summary %02d.", i)}
		}
		return many
	}

	named := func() catalog.Definitions {
		return catalog.Definitions{
			Catalogs: map[string]catalog.Origin{
				"hpcsc": {Name: "hpcsc", Source: "https://github.com/hpcsc/feature-compose", Path: "catalog", Commit: "2868ee8c1f0e4b6a9d3e2f7a5b1c0d9e8f7a6b5c"},
				"team":  {Name: "team", Source: "/code/team-catalog"},
			},
			Starters: map[string]catalog.StarterDefinition{
				"hpcsc/api": {Summary: "An API.", ExtensionPoints: map[string]catalog.ExtensionPointDefinition{
					"api.middleware": {Summary: "A middleware.", File: "go/main.go", Item: []string{"code"}},
				}},
				"hpcsc/worker": {Summary: "A worker."},
				"team/worker":  {Summary: "The worker of the team."},
			},
			Features: map[string]catalog.FeatureDefinition{
				"hpcsc/auth": {Summary: "Auth.", Targets: []string{"hpcsc/api"}, Needs: map[string]map[string]string{"hpcsc/log": {}}, AddTo: map[string][]map[string]string{"api.middleware": {{"code": "auth()"}}}},
				"hpcsc/log":  {Summary: "Logs.", Targets: []string{"hpcsc/api"}},
				"team/cli":   {Summary: "A CLI.", Targets: []string{"team/worker"}},
			},
		}
	}

	t.Run("layout", func(t *testing.T) {
		t.Run("the screen holds a title line with the counts, the tree pane, the details pane of the selected row, and the keys", func(t *testing.T) {
			screen := screenOf(sized(browser.New(definitions()), 80, 10))

			require.Equal(t, ` feature-compose show                                   3 starters · 5 features
╭─ Catalog ────────────────╮╭─ api ────────────────────────────────────────────╮
│ ▌▾ api                   ││ An API that a worker calls.                      │
│    ├─ ▾ features         ││                                                  │
│    │  ├─ auth            ││ Folders                                          │
│    │  ├─ cli             ││   docs  path          docs                       │
│    │  └─ docs            ││   go    path          .                          │
│    └─ ▾ extension points ││         after_render  go mod tidy                │
╰──────────────────────────╯╰───────────────────────────────────────────── 0% ─╯
 j/k move • l open • h close • / filter • ctrl+d/u scroll details • q quit`, screen)
		})

		t.Run("the title line counts one starter and one feature in the singular", func(t *testing.T) {
			one := catalog.Definitions{
				Starters: map[string]catalog.StarterDefinition{"api": {}},
				Features: map[string]catalog.FeatureDefinition{"cli": {Targets: []string{"api"}}},
			}

			require.True(t, strings.HasSuffix(linesOf(opened(browser.New(one)))[0], " 1 starter · 1 feature"))
		})

		for name, size := range map[string][2]int{
			"39 columns": {39, 20},
			"7 rows":     {80, 7},
		} {
			t.Run("a terminal of "+name+" shows only a message that asks for a larger terminal", func(t *testing.T) {
				require.Equal(t, "The terminal is too small for show. Make it at least 40 columns wide and 8 rows high.", strings.Join(strings.Fields(screenOf(sized(browser.New(definitions()), size[0], size[1]))), " "))
			})
		}

		t.Run("the tree pane takes at most two fifths of the width, and a longer name ends with …", func(t *testing.T) {
			long := catalog.Definitions{Starters: map[string]catalog.StarterDefinition{"a-very-long-starter-name": {}}}

			require.Equal(t, []string{"▌  a-very-long-…"}, treeOf(sized(browser.New(long), 50, 10)))
		})

		t.Run("a terminal of 40 columns and 8 rows shows the panes", func(t *testing.T) {
			treeTitle, _ := paneTitlesOf(sized(browser.New(definitions()), 40, 8))

			require.Equal(t, "Catalog", treeTitle)
		})

		t.Run("a title line too narrow for the counts shows only the title", func(t *testing.T) {
			require.Equal(t, " feature-compose show", linesOf(sized(browser.New(definitions()), 40, 10))[0])
		})

		for name, test := range map[string]struct {
			definitions catalog.Definitions
			width       int
		}{
			"a starter":                     {catalog.Definitions{Starters: map[string]catalog.StarterDefinition{"a-starter-with-a-long-name-xyz": {}}}, 37},
			"a feature of the last starter": {catalog.Definitions{Starters: map[string]catalog.StarterDefinition{"api": {}, "web": {}}, Features: map[string]catalog.FeatureDefinition{"aa": {Targets: []string{"web"}}, "zz-a-feature-with-a-long-name": {Targets: []string{"web"}}}}, 42},
			"an empty catalog":              {catalog.Definitions{}, 20},
		} {
			t.Run("the tree pane is five columns wider than its widest row, and at least 20 columns wide: "+name, func(t *testing.T) {
				require.Equal(t, test.width, treePaneWidthOf(sized(browser.New(test.definitions), 120, 10)))
			})
		}
	})

	t.Run("tree", func(t *testing.T) {
		t.Run("shows each starter open, in the order of the names, with its features and its extension points in two open groups, and the first starter selected", func(t *testing.T) {
			require.Equal(t, []string{"▌▾ api", "   ├─ ▾ features", "   │  ├─ auth", "   │  ├─ cli", "   │  └─ docs", "   └─ ▾ extension points", "      ├─ api.config", "      ├─ api.middleware", "      ├─ cli.commands", "      └─ cli.flags", "   worker", " ▾ zone", "   └─ ▾ features", "      └─ zone-tools"}, treeOf(opened(browser.New(definitions()))))
		})

		t.Run("the extension points of a starter include those that its features declare", func(t *testing.T) {
			require.Contains(t, treeOf(opened(browser.New(definitions()))), "      ├─ cli.commands")
		})

		t.Run("a starter shows only the groups that have an entry, and a starter with no group has no ▾", func(t *testing.T) {
			require.Equal(t, []string{"   worker", " ▾ zone", "   └─ ▾ features", "      └─ zone-tools"}, treeOf(opened(browser.New(definitions())))[10:])
		})

		for keys, selected := range map[string]string{
			"l":   "▌  ├─ ▾ features",
			"l l": "▌  │  ├─ auth",
		} {
			t.Run("l at an open row selects its first child: "+keys, func(t *testing.T) {
				require.Equal(t, selected, selectedRowOf(pressed(opened(browser.New(definitions())), strings.Fields(keys)...)))
			})
		}

		for keys, selected := range map[string]string{
			"j":                                "▌  ├─ ▾ features",
			"j j":                              "▌  │  ├─ auth",
			"j j k":                            "▌  ├─ ▾ features",
			"down down up":                     "▌  ├─ ▾ features",
			"up k":                             "▌▾ api",
			"j j j j j j":                      "▌     ├─ api.config",
			"j j j j j j j j j j":              "▌  worker",
			"j j j j j j j j j j j j j j down": "▌     └─ zone-tools",
		} {
			t.Run("j and k move the selection by one row, as down and up do, and stop at the ends: "+keys, func(t *testing.T) {
				require.Equal(t, selected, selectedRowOf(pressed(opened(browser.New(definitions())), strings.Fields(keys)...)))
			})
		}

		for _, back := range []string{"h", "left", "esc"} {
			t.Run(back+" at a feature or an extension point selects its group, which stays open", func(t *testing.T) {
				atFeature := pressed(opened(browser.New(definitions())), "l", "l", "j", back)
				atExtensionPoint := pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j", "j", back)

				require.Equal(t, []string{"▌  ├─ ▾ features", "▌  └─ ▾ extension points"}, []string{selectedRowOf(atFeature), selectedRowOf(atExtensionPoint)})
				require.Equal(t, withoutSelection(treeOf(opened(browser.New(definitions())))), withoutSelection(treeOf(atFeature)))
			})

			t.Run(back+" at an open group closes it, and a second "+back+" selects its starter", func(t *testing.T) {
				closed := pressed(opened(browser.New(definitions())), "l", back)

				require.Equal(t, []string{" ▾ api", "▌  ├─ ▸ features", "   └─ ▾ extension points", "      ├─ api.config", "      ├─ api.middleware", "      ├─ cli.commands", "      └─ cli.flags", "   worker", " ▾ zone", "   └─ ▾ features", "      └─ zone-tools"}, treeOf(closed))
				require.Equal(t, "▌▾ api", selectedRowOf(pressed(closed, back)))
			})

			t.Run(back+" at an open starter closes it", func(t *testing.T) {
				require.Equal(t, []string{"▌▸ api", "   worker", " ▾ zone", "   └─ ▾ features", "      └─ zone-tools"}, treeOf(pressed(opened(browser.New(definitions())), back)))
			})
		}

		for _, open := range []string{"l", "enter", "right"} {
			t.Run(open+" at a closed starter opens it again", func(t *testing.T) {
				require.Equal(t, treeOf(opened(browser.New(definitions()))), treeOf(pressed(opened(browser.New(definitions())), "h", open)))
			})
		}

		t.Run("a closed group stays closed when its starter closes and opens again", func(t *testing.T) {
			m := pressed(opened(browser.New(definitions())), "l", "h", "h", "h", "l")

			require.Equal(t, []string{"▌▾ api", "   ├─ ▸ features", "   └─ ▾ extension points"}, treeOf(m)[:3])
		})

		t.Run("h at a closed starter changes nothing", func(t *testing.T) {
			closed := pressed(opened(browser.New(definitions())), "h")

			require.Equal(t, screenOf(closed), screenOf(pressed(closed, "h")))
		})

		t.Run("l and h at a starter with no feature and no extension point change nothing", func(t *testing.T) {
			atWorker := pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j", "j", "j", "j", "j", "j")

			require.Equal(t, screenOf(atWorker), screenOf(pressed(atWorker, "l", "h")))
		})

		t.Run("l at a feature changes nothing", func(t *testing.T) {
			atFeature := pressed(opened(browser.New(definitions())), "l", "l")

			require.Equal(t, screenOf(atFeature), screenOf(pressed(atFeature, "l")))
		})

		t.Run("l at a closed starter with one group opens it again", func(t *testing.T) {
			atZone := pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j")

			require.Equal(t, treeOf(atZone), treeOf(pressed(atZone, "h", "l")))
		})

		t.Run("l at an open starter with one group selects the group", func(t *testing.T) {
			atZone := pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j")

			require.Equal(t, "▌  └─ ▾ features", selectedRowOf(pressed(atZone, "l")))
		})

		t.Run("l at a closed group opens it again", func(t *testing.T) {
			atFeatures := pressed(opened(browser.New(definitions())), "l")

			require.Equal(t, treeOf(atFeatures), treeOf(pressed(atFeatures, "h", "l")))
		})

		t.Run("a tree taller than its pane moves with the selection, so the selected row stays in view", func(t *testing.T) {
			screen := pressed(sized(browser.New(manyStarters(12)), 80, 8), "j", "j", "j", "j", "j", "j")

			require.Equal(t, []string{"   starter-03", "   starter-04", "   starter-05", "▌  starter-06"}, treeOf(screen))
		})

		t.Run("a tree scrolled down shows its first row again when the terminal grows tall enough for all rows", func(t *testing.T) {
			scrolled := pressed(sized(browser.New(manyStarters(12)), 80, 8), "j", "j", "j", "j", "j", "j")

			require.Equal(t, "   starter-00", treeOf(sized(scrolled, 80, 20))[0])
		})

		t.Run("an empty catalog says that it has no starter, and the keys change nothing", func(t *testing.T) {
			empty := opened(browser.New(catalog.Definitions{}))

			require.Equal(t, "The catalog has no starter.", strings.Join(treeOf(empty), " "))
			require.Equal(t, screenOf(empty), screenOf(pressed(empty, "j", "k", "l", "h", "ctrl+d")))
		})

		t.Run("the tree groups the starters under the name of their catalog, with the names that the catalog gives them, and the header counts the catalogs", func(t *testing.T) {
			m := opened(browser.New(named()))

			require.Equal(t, []string{
				" ▾ hpcsc",
				"▌  ▾ api",
				"     ├─ ▾ features",
				"     │  ├─ auth",
				"     │  └─ log",
				"     └─ ▾ extension points",
				"        └─ api.middleware",
				"     worker",
				" ▾ team",
				"   ▾ worker",
				"     └─ ▾ features",
				"        └─ cli",
			}, treeOf(m))
			require.Contains(t, linesOf(m)[0], "2 catalogs · 3 starters · 3 features")
		})

		t.Run("h at a starter selects its catalog, and h again closes the catalog", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), "h", "h", "h")

			require.Equal(t, []string{"▌▸ hpcsc", " ▾ team", "   ▾ worker", "     └─ ▾ features", "        └─ cli"}, treeOf(m))
		})

		t.Run("l at a closed catalog opens it again", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), "h", "h", "h", "l")

			require.Equal(t, []string{" ▾ team", "   ▾ worker", "     └─ ▾ features", "        └─ cli"}, treeOf(m)[len(treeOf(m))-4:])
			require.Equal(t, "▌▾ hpcsc", selectedRowOf(m))
		})

		t.Run("h at a closed group selects its starter", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), "j", "h", "h")

			require.Equal(t, "▌  ▾ api", selectedRowOf(m))
		})

		t.Run("l at an open catalog selects its first starter", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), "k", "l")

			require.Equal(t, "▌  ▾ api", selectedRowOf(m))
		})

		t.Run("the tree is wide enough for a catalog name that is wider than the rows under it", func(t *testing.T) {
			definitions := catalog.Definitions{
				Starters: map[string]catalog.StarterDefinition{"a-catalog-with-a-long-name/x": {Summary: "X."}},
				Features: map[string]catalog.FeatureDefinition{},
			}

			require.Equal(t, " ▾ a-catalog-with-a-long-name", treeOf(opened(browser.New(definitions)))[0])
		})
	})

	t.Run("details", func(t *testing.T) {
		t.Run("a starter shows its summary, its folders with their commands, and its options", func(t *testing.T) {
			require.Equal(t, `An API that a worker calls.

Folders
  docs  path          docs
  go    path          .
        after_render  go mod tidy
                      gofmt -w .
        test          task test:unit
  web   path          web
        after_render  npm install

Options
  module  default  example.com/{{ .Name }}
          pattern  ^[a-z./]+$
  region  default  us
          values   us, eu
  tier    default  standard
          values   standard`, detailsOf(opened(browser.New(definitions()))))
		})

		t.Run("a starter with no folder and no option says none for each", func(t *testing.T) {
			require.Equal(t, `A worker for jobs.

Folders
  none

Options
  none`, detailsOf(pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j", "j", "j", "j", "j", "j")))
		})

		t.Run("the group of the features lists each feature that targets the starter, with its summary", func(t *testing.T) {
			require.Equal(t, `Features
  auth  API keys.
  cli   A command line program.
  docs  Docs.`, detailsOf(pressed(opened(browser.New(definitions())), "j")))
		})

		t.Run("the group of the extension points lists each extension point of the starter and of its features, with its item fields and its declarer", func(t *testing.T) {
			require.Equal(t, `Extension points
  api.config      item         name
                  declared by  the starter
  api.middleware  item         code, import
                  declared by  the starter
  cli.commands    item         code
                  declared by  the feature cli
  cli.flags       item         name
                  declared by  the feature cli`, detailsOf(pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j")))
		})

		t.Run("an extension point shows its summary, its file, its item fields, its declarer, and the features that add items to it", func(t *testing.T) {
			require.Equal(t, `A middleware.

file         go/server.go
item         code, import
declared by  the starter
added to by  auth`, detailsOf(pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j", "j", "j")))
		})

		t.Run("an extension point that no feature adds items to says none", func(t *testing.T) {
			require.Equal(t, `A flag.

file         go/flags.go
item         name
declared by  the feature cli
added to by  none`, detailsOf(pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j", "j", "j", "j", "j")))
		})

		withGenerator := func() catalog.Definitions {
			return catalog.Definitions{Starters: map[string]catalog.StarterDefinition{"phoenix": {
				Summary: "A Phoenix app.",
				Folders: map[string]catalog.FolderDefinition{"app": {Path: ".", Generate: "mix phx.new {{ .Folder }}", AfterRender: []string{"mix format"}}},
				ExtensionPoints: map[string]catalog.ExtensionPointDefinition{"mix.deps": {
					Summary: "A dependency.", File: "app/mix.exs", Item: []string{"dep"}, Language: "elixir", Pattern: "defp deps do [$$$DEPS] end", After: "DEPS",
				}},
				Remove: []catalog.RemovalDefinition{{File: "app/lib/page.ex"}, {File: "app/router.ex", Language: "elixir", Pattern: `get "/", PageController, :home`}},
				AddTo:  map[string][]map[string]string{"mix.deps": {{"dep": `{:jason, "~> 1.2"}`}}},
			}}}
		}

		t.Run("a starter with a generator shows its generate command, and each removal with its pattern", func(t *testing.T) {
			require.Equal(t, `A Phoenix app.

Folders
  app  path          .
       generate      mix phx.new {{ .Folder }}
       after_render  mix format

Options
  none

Removals
  app/lib/page.ex
  app/router.ex    pattern  get "/", PageController, :home`, detailsOf(opened(browser.New(withGenerator()))))
		})

		t.Run("an extension point in a generated file shows its language, its pattern and its position, and the starter when the starter adds items to it", func(t *testing.T) {
			require.Equal(t, `A dependency.

file         app/mix.exs
language     elixir
pattern      defp deps do [$$$DEPS] end
after        DEPS
item         dep
declared by  the starter
added to by  the starter`, detailsOf(pressed(opened(browser.New(withGenerator())), "j", "j")))
		})

		t.Run("an extension point in a generated file with a rule shows the rule as JSON, and its position before a node", func(t *testing.T) {
			definitions := catalog.Definitions{Starters: map[string]catalog.StarterDefinition{"phoenix": {
				Folders: map[string]catalog.FolderDefinition{"app": {Path: ".", Generate: "mix phx.new {{ .Folder }}"}},
				ExtensionPoints: map[string]catalog.ExtensionPointDefinition{"agents.guidelines": {
					Summary: "A guideline.", File: "app/AGENTS.md", Item: []string{"line"},
					QueryDefinition: catalog.QueryDefinition{Language: "markdown", Rule: map[string]any{"kind": "list_item"}},
					Before:          "match",
				}},
			}}}

			require.Equal(t, `A guideline.

file         app/AGENTS.md
language     markdown
rule         {"kind":"list_item"}
before       match
item         line
declared by  the starter
added to by  none`, detailsOf(pressed(opened(browser.New(definitions)), "j", "j")))
		})

		t.Run("a feature shows its summary, its needs, its options, and the extension points that it declares or adds items to", func(t *testing.T) {
			require.Equal(t, `API keys.

Needs
  cli

Options
  method  default  api-key
          values   api-key, oauth

Extension points that it declares
  none

Extension points that it adds items to
  api.config
  api.middleware`, detailsOf(pressed(opened(browser.New(definitions())), "j", "j")))
		})

		t.Run("an option with a when shows the when after its default and its values, and an option with no when shows no when", func(t *testing.T) {
			withWhen := definitions()
			auth := withWhen.Features["auth"]
			auth.Options = map[string]catalog.OptionDefinition{
				"method":    {Default: "api-key", Values: []string{"api-key", "oauth"}},
				"store-key": {Default: "false", Values: []string{"true", "false"}, When: `method == "api-key"`},
			}
			withWhen.Features["auth"] = auth

			details := detailsOf(pressed(opened(browser.New(withWhen)), "j", "j"))

			require.Contains(t, details, `Options
  method     default  api-key
             values   api-key, oauth
  store-key  default  false
             values   true, false
             when     method == "api-key"
`)
		})

		t.Run("a feature with no need, option or added item says none for each, and shows the summary and the item fields of the extension points that it declares", func(t *testing.T) {
			require.Equal(t, `A command line program.

Needs
  none

Options
  none

Extension points that it declares
  cli.commands  A command.
                item  code
  cli.flags     A flag.
                item  name

Extension points that it adds items to
  none`, detailsOf(pressed(opened(browser.New(definitions())), "j", "j", "j")))
		})

		for keys, title := range map[string]string{
			"j j j j j j j j j j j":     "zone",
			"j j j j j j j j j j j j":   "zone › features",
			"j j j j j j j j j j j j j": "zone-tools",
			"j j j j j":                 "api › extension points",
			"j j j j j j j":             "api.middleware",
		} {
			t.Run("the title of the details pane names the selected row: "+keys, func(t *testing.T) {
				_, details := paneTitlesOf(pressed(opened(browser.New(definitions())), strings.Fields(keys)...))

				require.Equal(t, title, details)
			})
		}

		t.Run("a summary longer than the details pane wraps at a space", func(t *testing.T) {
			long := catalog.Definitions{Starters: map[string]catalog.StarterDefinition{"api": {Summary: "An API with a summary that is longer than the pane is wide."}}}

			require.Equal(t, []string{"An API with a summary that", "is longer than the pane is", "wide."}, strings.Split(detailsOf(sized(browser.New(long), 50, 12)), "\n")[:3])
		})

		t.Run("a details pane too narrow for a name and its value puts the value under the name", func(t *testing.T) {
			screen := pressed(sized(browser.New(definitions()), 56, 60), "j", "j", "j", "j", "j")

			require.True(t, strings.HasPrefix(detailsOf(screen), `Extension points
  api.config
    item         name
    declared by  the starter
  api.middleware`), detailsOf(screen))
		})

		t.Run("ctrl+d scrolls the details down by half the pane, ctrl+u scrolls them up again, and the bottom border shows how far they are scrolled", func(t *testing.T) {
			top := sized(browser.New(definitions()), 80, 12)
			scrolled := pressed(top, "ctrl+d")

			require.Equal(t, []string{"An API that a worker calls.", "  go    path          ."}, []string{firstDetailsLineOf(top), firstDetailsLineOf(scrolled)})
			require.Equal(t, []string{"0%", "40%"}, []string{scrollNoteOf(top), scrollNoteOf(scrolled)})
			require.Equal(t, screenOf(top), screenOf(pressed(scrolled, "ctrl+u")))
		})

		t.Run("a new selection shows its details from the top", func(t *testing.T) {
			screen := pressed(sized(browser.New(definitions()), 80, 12), "ctrl+d", "j", "j")

			require.Equal(t, "API keys.", firstDetailsLineOf(screen))
		})

		t.Run("details scrolled to their end show 100%", func(t *testing.T) {
			require.Equal(t, "100%", scrollNoteOf(pressed(sized(browser.New(definitions()), 80, 12), "ctrl+d", "ctrl+d", "ctrl+d")))
		})

		t.Run("a feature with two needs lists each need", func(t *testing.T) {
			twoNeeds := catalog.Definitions{
				Starters: map[string]catalog.StarterDefinition{"api": {}},
				Features: map[string]catalog.FeatureDefinition{
					"a": {Targets: []string{"api"}},
					"b": {Targets: []string{"api"}},
					"c": {Summary: "Needs two.", Targets: []string{"api"}, Needs: map[string]map[string]string{"b": {}, "a": {}}},
				},
			}

			require.True(t, strings.HasPrefix(detailsOf(pressed(opened(browser.New(twoNeeds)), "j", "j", "j", "j")), "Needs two.\n\nNeeds\n  a\n  b\n"), detailsOf(pressed(opened(browser.New(twoNeeds)), "j", "j", "j", "j")))
		})

		t.Run("a need that sets values shows each option that it sets with its value beside the need", func(t *testing.T) {
			withValues := catalog.Definitions{
				Starters: map[string]catalog.StarterDefinition{"api": {}},
				Features: map[string]catalog.FeatureDefinition{
					"a": {Targets: []string{"api"}},
					"b": {Targets: []string{"api"}},
					"c": {Summary: "Needs two.", Targets: []string{"api"}, Needs: map[string]map[string]string{"a": {}, "b": {"mode": "fast", "level": "high"}}},
				},
			}

			require.True(t, strings.HasPrefix(detailsOf(pressed(opened(browser.New(withValues)), "j", "j", "j", "j")), "Needs two.\n\nNeeds\n  a\n  b  level  high\n     mode   fast\n"), detailsOf(pressed(opened(browser.New(withValues)), "j", "j", "j", "j")))
		})

		t.Run("details that fit the pane show no scroll position", func(t *testing.T) {
			require.Equal(t, "", scrollNoteOf(pressed(opened(browser.New(definitions())), "j", "j", "j", "j", "j", "j", "j", "j", "j", "j")))
		})

		t.Run("the details of a catalog list its starters and its features, with the names that the catalog gives them", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), "k")

			_, title := paneTitlesOf(m)
			require.Equal(t, "the catalog hpcsc", title)
			require.Equal(t, "source  https://github.com/hpcsc/feature-compose\npath    catalog\nref     the default branch\ncommit  2868ee8c1f0e4b6a9d3e2f7a5b1c0d9e8f7a6b5c\n\nStarters\n  api     An API.\n  worker  A worker.\n\nFeatures\n  auth  Auth.\n  log   Logs.", detailsOf(m))
		})

		t.Run("the details of a catalog from a local folder say that it has no ref and no commit", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), "j", "j", "j", "j", "j", "j", "j")

			_, title := paneTitlesOf(m)
			require.Equal(t, "the catalog team", title)
			require.Equal(t, []string{"source  /code/team-catalog", "path    the root of the source", "ref     none: a local folder", "commit  none: a local folder"}, strings.Split(detailsOf(m), "\n")[:4])
		})

		t.Run("the details of a feature name its needs with the names that the catalog gives them", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), "j", "j")

			_, title := paneTitlesOf(m)
			require.Equal(t, "auth", title)
			require.Contains(t, detailsOf(m), "Needs\n  log")
		})

		t.Run("an extension point names the features that add items to it with the names that the catalog gives them", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), "j", "j", "j", "j", "j")

			require.Contains(t, detailsOf(m), "added to by  auth")
		})
	})

	t.Run("filter", func(t *testing.T) {
		t.Run("/ starts a filter on the names, and a starter whose summary holds the text but whose name does not is hidden", func(t *testing.T) {
			require.Equal(t, []string{"▌  worker"}, treeOf(pressed(opened(browser.New(definitions())), "/", "w", "o", "r")))
		})

		t.Run("a feature whose name matches shows in its group under its starter, and both open", func(t *testing.T) {
			require.Equal(t, []string{"▌▾ zone", "   └─ ▾ features", "      └─ zone-tools"}, treeOf(pressed(opened(browser.New(definitions())), "/", "t", "o", "o", "l")))
		})

		t.Run("an extension point whose name matches shows in its group under its starter, and both open", func(t *testing.T) {
			require.Equal(t, []string{"▌▾ api", "   └─ ▾ extension points", "      └─ cli.flags"}, treeOf(pressed(opened(browser.New(definitions())), "/", "f", "l", "a", "g")))
		})

		t.Run("a starter whose name matches keeps each of its features and extension points", func(t *testing.T) {
			require.Equal(t, []string{"▌▾ api", "   ├─ ▾ features", "   │  ├─ auth", "   │  ├─ cli", "   │  └─ docs", "   └─ ▾ extension points", "      ├─ api.config", "      ├─ api.middleware", "      ├─ cli.commands", "      └─ cli.flags"}, treeOf(pressed(opened(browser.New(definitions())), "/", "a", "p", "i", "enter")))
		})

		t.Run("the filter ignores case", func(t *testing.T) {
			require.Equal(t, []string{"▌  worker"}, treeOf(pressed(opened(browser.New(definitions())), "/", "W", "O")))
		})

		t.Run("while the filter is typed, h, j, k, l and q go to the filter and do not stop the TUI", func(t *testing.T) {
			m, quit := pressedAndQuit(opened(browser.New(definitions())), "/", "h", "j", "k", "l", "q")

			require.False(t, quit)
			require.Equal(t, "No name matches the filter.", strings.Join(treeOf(m), " "))
			require.True(t, strings.HasPrefix(footerOf(m), " /hjklq"))
		})

		t.Run("enter keeps the filter, the title of the tree pane shows it, and the keys of the tree work again", func(t *testing.T) {
			m := pressed(opened(browser.New(definitions())), "/", "s", "enter", "j")

			treeTitle, _ := paneTitlesOf(m)
			require.Equal(t, "Catalog /s", treeTitle)
			require.Equal(t, []string{" ▾ api", "▌  ├─ ▾ features", "   │  └─ docs", "   └─ ▾ extension points", "      ├─ cli.commands", "      └─ cli.flags", " ▾ zone", "   └─ ▾ features", "      └─ zone-tools"}, treeOf(m))
			require.Equal(t, " esc clear filter • j/k move • l open • h close • / filter • ctrl+d/u scroll details • q quit", footerOf(m))
		})

		for name, keys := range map[string][]string{
			"esc while the filter is typed": {"/", "w", "o", "esc"},
			"esc after enter":               {"/", "w", "o", "enter", "esc"},
		} {
			t.Run(name+" clears the filter, and the tree holds each starter again with the same row selected", func(t *testing.T) {
				m := pressed(opened(browser.New(definitions())), keys...)

				require.Equal(t, withoutSelection(treeOf(opened(browser.New(definitions())))), withoutSelection(treeOf(m)))
				require.Equal(t, "▌  worker", selectedRowOf(m))
			})
		}

		t.Run("the rows that the filter opens close again when the filter clears, and a hidden selection moves to its starter", func(t *testing.T) {
			m := pressed(opened(browser.New(definitions())), "h", "/", "f", "l", "a", "g", "enter", "j", "j", "esc")

			require.Equal(t, []string{"▌▸ api", "   worker", " ▾ zone", "   └─ ▾ features", "      └─ zone-tools"}, treeOf(m))
		})

		t.Run("a row that the filter opens is closed by h while the filter is kept", func(t *testing.T) {
			m := pressed(opened(browser.New(definitions())), "/", "f", "l", "a", "g", "enter", "h")

			require.Equal(t, []string{"▌▸ api"}, treeOf(m))
		})

		t.Run("the footer shows the keys of the filter while it is typed", func(t *testing.T) {
			require.Equal(t, " /w   enter keep filter • esc clear filter", footerOf(pressed(opened(browser.New(definitions())), "/", "w")))
		})

		t.Run("a filter with the name of a feature shows that feature under its starter and its catalog", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), append([]string{"/"}, strings.Split("log", "")...)...)

			require.Equal(t, []string{" ▾ hpcsc", "   ▾ api", "     └─ ▾ features", "        └─ log"}, withoutSelection(treeOf(m)))
		})

		t.Run("a filter opens a closed catalog that holds a match", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), append([]string{"h", "h", "h", "/"}, strings.Split("log", "")...)...)

			require.Equal(t, []string{" ▾ hpcsc", "   ▾ api", "     └─ ▾ features", "        └─ log"}, withoutSelection(treeOf(m)))
		})

		t.Run("a filter that hides the selected feature selects its group", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), append([]string{"j", "j", "j", "/"}, strings.Split("auth", "")...)...)

			require.Equal(t, "▌    └─ ▾ features", selectedRowOf(m))
		})

		t.Run("a filter does not match the name of a catalog in the full name of a starter or a feature", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), append([]string{"/"}, strings.Split("pcs/", "")...)...)

			require.Equal(t, []string{"No name matches the", "filter."}, treeOf(m))
		})

		t.Run("a filter with the name of a catalog shows only that catalog", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), append([]string{"/"}, strings.Split("team", "")...)...)

			require.Equal(t, []string{" ▾ team", "   ▾ worker", "     └─ ▾ features", "        └─ cli"}, withoutSelection(treeOf(m)))
		})

		t.Run("a filter that hides the selected extension point and its group selects its starter", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), append([]string{"j", "j", "j", "j", "j", "/"}, strings.Split("auth", "")...)...)

			require.Equal(t, "▌  ▾ api", selectedRowOf(m))
		})

		t.Run("a filter opens a closed group of a starter that is not the first, when the group holds a match", func(t *testing.T) {
			m := pressed(opened(browser.New(named())), append([]string{"j", "j", "j", "j", "j", "j", "j", "j", "j", "h", "/"}, strings.Split("cli", "")...)...)

			require.Equal(t, []string{" ▾ team", "   ▾ worker", "     └─ ▾ features", "        └─ cli"}, withoutSelection(treeOf(m)))
		})
	})

	t.Run("stop", func(t *testing.T) {
		for name, keys := range map[string][]string{
			"q":                                {"q"},
			"q at a feature":                   {"j", "j", "q"},
			"ctrl+c":                           {"ctrl+c"},
			"ctrl+c while the filter is typed": {"/", "w", "ctrl+c"},
		} {
			t.Run(name+" stops the TUI", func(t *testing.T) {
				_, quit := pressedAndQuit(opened(browser.New(definitions())), keys...)

				require.True(t, quit)
			})
		}

		t.Run("esc at a closed starter with no filter does not stop the TUI", func(t *testing.T) {
			_, quit := pressedAndQuit(opened(browser.New(definitions())), "esc")

			require.False(t, quit)
		})
	})

	t.Run("at a starter", func(t *testing.T) {
		t.Run("opens the TUI with that starter selected", func(t *testing.T) {
			atStarter, err := browser.NewAtStarter(definitions(), "zone")
			require.NoError(t, err)

			require.Equal(t, []string{" ▾ api", "   ├─ ▾ features", "   │  ├─ auth", "   │  ├─ cli", "   │  └─ docs", "   └─ ▾ extension points", "      ├─ api.config", "      ├─ api.middleware", "      ├─ cli.commands", "      └─ cli.flags", "   worker", "▌▾ zone", "   └─ ▾ features", "      └─ zone-tools"}, treeOf(opened(atStarter)))
		})

		t.Run("an unknown starter is an error that lists the starters", func(t *testing.T) {
			_, err := browser.NewAtStarter(definitions(), "nope")

			require.EqualError(t, err, `unknown starter "nope": the catalog has api, worker, zone`)
		})

		t.Run("a short name that one catalog has opens the TUI with the starter of that full name selected", func(t *testing.T) {
			named := catalog.Definitions{
				Starters: map[string]catalog.StarterDefinition{"hpcsc/api": {Summary: "An API."}, "team/worker": {Summary: "A worker."}},
				Features: map[string]catalog.FeatureDefinition{},
			}

			atStarter, err := browser.NewAtStarter(named, "worker")
			require.NoError(t, err)

			require.Equal(t, "▌    worker", selectedRowOf(opened(atStarter)))
		})
	})
}

func opened(m browser.Model) browser.Model {
	return sized(m, 100, 40)
}

func sized(m browser.Model, width, height int) browser.Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return next.(browser.Model)
}

var escapeCode = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func screenOf(m browser.Model) string {
	return strings.Join(linesOf(m), "\n")
}

func linesOf(m browser.Model) []string {
	lines := strings.Split(escapeCode.ReplaceAllString(m.View(), ""), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return lines
}

func panesOf(m browser.Model) (tree, details []string) {
	lines := linesOf(m)
	for _, line := range lines[2 : len(lines)-2] {
		left, right, _ := strings.Cut(line, "││")
		tree = append(tree, strings.TrimRight(strings.TrimPrefix(left, "│ "), " "))
		details = append(details, strings.TrimRight(strings.TrimSuffix(strings.TrimPrefix(right, " "), "│"), " "))
	}
	return withoutEmptyEnd(tree), withoutEmptyEnd(details)
}

func treeOf(m browser.Model) []string {
	tree, _ := panesOf(m)
	return tree
}

func detailsOf(m browser.Model) string {
	_, details := panesOf(m)
	return strings.Join(details, "\n")
}

func firstDetailsLineOf(m browser.Model) string {
	_, details := panesOf(m)
	return details[0]
}

func selectedRowOf(m browser.Model) string {
	for _, row := range treeOf(m) {
		if strings.HasPrefix(row, "▌") {
			return row
		}
	}
	return ""
}

var panelTitles = regexp.MustCompile(`^╭─ (.*?) ─*╮╭─ (.*?) ─*╮$`)

func paneTitlesOf(m browser.Model) (tree, details string) {
	found := panelTitles.FindStringSubmatch(linesOf(m)[1])
	return found[1], found[2]
}

var scrollNote = regexp.MustCompile(`─ (\d+%) ─╯$`)

func treePaneWidthOf(m browser.Model) int {
	top := linesOf(m)[1]
	return len([]rune(top[:strings.Index(top, "╮")+len("╮")]))
}

func scrollNoteOf(m browser.Model) string {
	lines := linesOf(m)
	if found := scrollNote.FindStringSubmatch(lines[len(lines)-2]); found != nil {
		return found[1]
	}
	return ""
}

func footerOf(m browser.Model) string {
	lines := linesOf(m)
	return lines[len(lines)-1]
}

func withoutSelection(rows []string) []string {
	unselected := make([]string, 0, len(rows))
	for _, row := range rows {
		unselected = append(unselected, strings.Replace(row, "▌", " ", 1))
	}
	return unselected
}

func withoutEmptyEnd(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func pressed(m browser.Model, keys ...string) browser.Model {
	pressedModel, _ := pressedAndQuit(m, keys...)
	return pressedModel
}

func pressedAndQuit(m browser.Model, keys ...string) (browser.Model, bool) {
	quit := false
	for _, name := range keys {
		next, cmd := m.Update(keyMsg(name))
		m = next.(browser.Model)
		for _, msg := range messagesOf(cmd) {
			if _, ok := msg.(tea.QuitMsg); ok {
				quit = true
				continue
			}
			next, _ = m.Update(msg)
			m = next.(browser.Model)
		}
	}
	return m, quit
}

func keyMsg(name string) tea.KeyMsg {
	switch name {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

// messagesOf runs a command and its batch, and drops a command that waits, such as the blink of the cursor.
func messagesOf(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		batch, ok := msg.(tea.BatchMsg)
		if !ok {
			return []tea.Msg{msg}
		}
		var messages []tea.Msg
		for _, each := range batch {
			messages = append(messages, messagesOf(each)...)
		}
		return messages
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}
