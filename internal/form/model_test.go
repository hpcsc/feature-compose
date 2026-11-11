//go:build unit

package form_test

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/form"
	"github.com/stretchr/testify/require"
)

func TestModel(t *testing.T) {
	withNeeds := func() fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"starters/other/starter.yml":        &fstest.MapFile{Data: []byte("summary: Another service.\nfolders:\n  go:\n    path: .\n")},
			"starters/other/files/go/main.go":   &fstest.MapFile{Data: []byte("package main\n")},
			"features/alpha/feature.yml":        &fstest.MapFile{Data: []byte("summary: Feature alpha.\ntargets: [service]\nneeds: [zed]\n")},
			"features/beta/feature.yml":         &fstest.MapFile{Data: []byte("summary: Feature beta.\ntargets: [service]\n")},
			"features/gamma/feature.yml":        &fstest.MapFile{Data: []byte("summary: Feature gamma.\ntargets: [service]\nneeds: [zed]\n")},
			"features/zed/feature.yml":          &fstest.MapFile{Data: []byte("summary: Feature zed.\ntargets: [service]\n")},
			"features/elsewhere/feature.yml":    &fstest.MapFile{Data: []byte("summary: Elsewhere.\ntargets: [other]\n")},
		}
	}
	withOptions := func() fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\noptions:\n  region:\n    default: us\n    pattern: '^[a-z]+$'\n  module:\n    default: example.com/{{ .Name }}\n  zone:\n    pattern: '^[a-z]*$'\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"features/alpha/feature.yml":        &fstest.MapFile{Data: []byte("summary: Feature alpha.\ntargets: [service]\nneeds: [zed]\noptions:\n  mode:\n    default: fast\n  timeout:\n    default: \"30\"\n")},
			"features/beta/feature.yml":         &fstest.MapFile{Data: []byte("summary: Feature beta.\ntargets: [service]\noptions:\n  color:\n    values: [red, blue]\n")},
			"features/gamma/feature.yml":        &fstest.MapFile{Data: []byte("summary: Feature gamma.\ntargets: [service]\nneeds: [zed]\noptions:\n  tone:\n    values: [soft]\n    default: soft\n")},
			"features/zed/feature.yml":          &fstest.MapFile{Data: []byte("summary: Feature zed.\ntargets: [service]\noptions:\n  level:\n    values: [low, mid, high]\n    default: low\n")},
		}
	}
	withNeedValue := func() fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"features/auth/feature.yml":         &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [service]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n")},
			"features/cli-auth/feature.yml":     &fstest.MapFile{Data: []byte("summary: CLI auth.\ntargets: [service]\nneeds:\n  auth:\n    method: api-key\n")},
		}
	}
	withCycle := func() fstest.MapFS {
		files := withNeeds()
		files["features/beta/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Feature beta.\ntargets: [service]\nneeds: [gamma]\n")}
		files["features/gamma/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Feature gamma.\ntargets: [service]\nneeds: [beta]\n")}
		return files
	}
	manyStarters := func(count int) fstest.MapFS {
		files := fstest.MapFS{}
		for i := range count {
			files[fmt.Sprintf("starters/starter-%02d/starter.yml", i)] = &fstest.MapFile{Data: []byte(fmt.Sprintf("summary: Summary %02d.\nfolders:\n  go:\n    path: .\n", i))}
			files[fmt.Sprintf("starters/starter-%02d/files/go/main.go", i)] = &fstest.MapFile{Data: []byte("package main\n")}
		}
		return files
	}
	manyFeatures := func(count int) fstest.MapFS {
		files := fstest.MapFS{
			"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
		}
		for i := range count {
			files[fmt.Sprintf("features/feature-%02d/feature.yml", i)] = &fstest.MapFile{Data: []byte("summary: A feature.\ntargets: [service]\n")}
		}
		return files
	}
	withSummaries := func(starterSummary, featureSummary string) fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":                    &fstest.MapFile{Data: []byte("summary: " + starterSummary + "\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go":               &fstest.MapFile{Data: []byte("package main\n")},
			"features/a-feature-with-a-long-name/feature.yml": &fstest.MapFile{Data: []byte("summary: " + featureSummary + "\ntargets: [service]\n")},
		}
	}
	withTwoNeeds := func() fstest.MapFS {
		files := withNeeds()
		files["features/delta/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Feature delta.\ntargets: [service]\nneeds: [beta, zed]\n")}
		return files
	}
	withAlpha := func() map[string]map[string]string { return map[string]map[string]string{"alpha": {}} }
	opened := func(t *testing.T, files fstest.MapFS, starterName, name string, givenByFeature map[string]map[string]string) form.Model {
		t.Helper()
		m, err := form.New(catalog.New(files), starterName, name, nil, givenByFeature)
		require.NoError(t, err)
		return sized(m, 90, 14)
	}

	t.Run("layout", func(t *testing.T) {
		t.Run("the screen holds a title line with the count of the selected features, the tree pane, the details pane of the selected row, and the keys", func(t *testing.T) {
			require.Equal(t, ` feature-compose new                                                           2 features
╭─ Repo ───────────────────────────╮╭─ name ─────────────────────────────────────────────╮
│ ▌  name     my-service  fixed    ││ value   my-service                                 │
│  ▾ starter  service  fixed       ││ source  command line                               │
│    ├─   [ ] beta                 ││                                                    │
│    ├─   [x] zed  needed by alpha ││                                                    │
│    ├─   [x] alpha  fixed         ││                                                    │
│    └─   [ ] gamma                ││                                                    │
│                                  ││                                                    │
│                                  ││                                                    │
│                                  ││                                                    │
│                                  ││                                                    │
╰──────────────────────────────────╯╰────────────────────────────────────────────────────╯
 j/k move • l open • h close • space select • w write • ctrl+d/u scroll details • q quit`, screenOf(opened(t, withNeeds(), "service", "my-service", withAlpha())))
		})

		t.Run("the title line counts one selected feature in the singular", func(t *testing.T) {
			require.True(t, strings.HasSuffix(linesOf(opened(t, withNeeds(), "service", "my-service", map[string]map[string]string{"beta": {}}))[0], " 1 feature"))
		})

		for name, size := range map[string][2]int{
			"39 columns": {39, 20},
			"7 rows":     {80, 7},
		} {
			t.Run("a terminal of "+name+" shows only a message that asks for a larger terminal, and names new", func(t *testing.T) {
				require.Equal(t, "The terminal is too small for new. Make it at least 40 columns wide and 8 rows high.", strings.Join(strings.Fields(screenOf(sized(opened(t, withNeeds(), "service", "my-service", nil), size[0], size[1]))), " "))
			})
		}

		t.Run("a terminal of 40 columns and 8 rows shows the panes", func(t *testing.T) {
			treeTitle, _ := paneTitlesOf(sized(opened(t, withNeeds(), "service", "my-service", nil), 40, 8))

			require.Equal(t, "Repo", treeTitle)
		})

		t.Run("the tree pane takes at most two fifths of the width, and a longer row ends with …", func(t *testing.T) {
			require.Equal(t, "▌  name     my-…", treeOf(sized(opened(t, withNeeds(), "service", "my-service", nil), 50, 14))[0])
		})

		for name, size := range map[string][2]int{
			"before the first size": {0, 0},
			"with a width of 0":     {0, 10},
			"with a height of 0":    {10, 0},
		} {
			t.Run("the screen is empty "+name, func(t *testing.T) {
				m, err := form.New(catalog.New(withNeeds()), "service", "my-service", nil, nil)
				require.NoError(t, err)

				require.Equal(t, "", sized(m, size[0], size[1]).View())
			})
		}

		t.Run("the footer of a narrow terminal cuts the keys at its width", func(t *testing.T) {
			require.Equal(t, " j/k move • l open • h close …", linesOf(sized(opened(t, withNeeds(), "service", "my-service", nil), 40, 8))[7])
		})

		for width, footer := range map[int]string{
			87: " j/k move • l open • h close • space select • w write • ctrl+d/u scroll details …",
			88: " j/k move • l open • h close • space select • w write • ctrl+d/u scroll details • q quit",
		} {
			t.Run(fmt.Sprintf("the footer shows each key that fits in %d columns", width), func(t *testing.T) {
				require.Equal(t, footer, linesOf(sized(opened(t, withNeeds(), "service", "my-service", nil), width, 10))[9])
			})
		}

		t.Run("the tree pane is five columns wider than its widest row", func(t *testing.T) {
			require.Equal(t, 42, treePaneWidthOf(sized(opened(t, withSummaries("A service.", "A feature."), "service", "my-service", nil), 120, 14)))
		})

		t.Run("a tree taller than its pane follows the selected row", func(t *testing.T) {
			m := pressed(sized(opened(t, manyFeatures(12), "service", "my-service", nil), 90, 8), "j", "j", "j", "j", "j", "j")

			require.Equal(t, []string{"   ├─   [ ] feature-01", "   ├─   [ ] feature-02", "   ├─   [ ] feature-03", "▌  ├─   [ ] feature-04"}, treeOf(m))
		})
	})

	t.Run("tree", func(t *testing.T) {
		t.Run("a starter with one feature opens to show it", func(t *testing.T) {
			require.Equal(t, []string{"▌  name     my-service  fixed", " ▾ starter  service  fixed", "   └─   [ ] a-feature-with-a-long-name"}, treeOf(sized(opened(t, withSummaries("A service.", "A feature."), "service", "my-service", nil), 120, 14)))
		})

		t.Run("a tree scrolled down shows the selected row at its top when the selected row moves up past it", func(t *testing.T) {
			m := pressed(sized(opened(t, manyFeatures(12), "service", "my-service", nil), 90, 8), "j", "j", "j", "j", "j", "j", "k", "k", "k", "k")

			require.Equal(t, []string{"▌  ├─   [ ] feature-00", "   ├─   [ ] feature-01", "   ├─   [ ] feature-02", "   ├─   [ ] feature-03"}, treeOf(m))
		})

		t.Run("a tree scrolled down shows its first row again when it is selected again", func(t *testing.T) {
			m := pressed(sized(opened(t, manyFeatures(12), "service", "my-service", nil), 90, 8), "j", "j", "j", "j", "j", "j", "k", "k", "k", "k", "k", "k")

			require.Equal(t, []string{"▌  name     my-service  fixed", " ▾ starter  service  fixed", "   ├─   [ ] feature-00", "   ├─   [ ] feature-01"}, treeOf(m))
		})

		t.Run("a tree scrolled down shows the name row again when the starter closes", func(t *testing.T) {
			m := pressed(sized(opened(t, manyFeatures(12), "service", "my-service", nil), 90, 8), "j", "j", "j", "j", "j", "j", "h", "h")

			require.Equal(t, []string{"   name     my-service  fixed", "▌▸ starter  service  fixed"}, treeOf(m))
		})

		t.Run("holds the name, then the open starter with each feature that targets it, each after the features that it needs, with a box that shows whether it is selected", func(t *testing.T) {
			require.Equal(t, []string{"▌  name     my-service  fixed", " ▾ starter  service  fixed", "   ├─   [ ] beta", "   ├─   [x] zed  needed by alpha", "   ├─   [x] alpha  fixed", "   └─   [ ] gamma"}, treeOf(opened(t, withNeeds(), "service", "my-service", withAlpha())))
		})

		t.Run("a feature of --feature that a selected feature needs shows both marks", func(t *testing.T) {
			require.Equal(t, "   ├─   [x] zed  fixed, needed by alpha", treeOf(sized(opened(t, withNeeds(), "service", "my-service", map[string]map[string]string{"alpha": {}, "zed": {}}), 120, 14))[3])
		})

		t.Run("with no argument, the name row and the starter row show that their answers are missing, and the tree has no feature", func(t *testing.T) {
			require.Equal(t, []string{"▌  name       missing", "   starter    missing"}, treeOf(opened(t, withNeeds(), "", "", nil)))
		})

		for keys, selected := range map[string]string{
			"j":                  "▌▾ starter  service  fixed",
			"j j":                "▌  ├─   [ ] beta",
			"j j k":              "▌▾ starter  service  fixed",
			"down down up":       "▌▾ starter  service  fixed",
			"up k":               "▌  name     my-service  fixed",
			"j j j j j":          "▌  └─   [ ] gamma",
			"j j j j j j j down": "▌  └─   [ ] gamma",
		} {
			t.Run("j and k move the selected row by one row, as down and up do, and stop at the ends: "+keys, func(t *testing.T) {
				require.Equal(t, selected, selectedRowOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), strings.Fields(keys)...)))
			})
		}

		for _, open := range []string{"l", "right"} {
			t.Run(open+" at the open starter selects its first feature", func(t *testing.T) {
				require.Equal(t, "▌  ├─   [ ] beta", selectedRowOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", open)))
			})
		}

		for _, back := range []string{"h", "left", "esc"} {
			t.Run(back+" at a feature selects the starter, which stays open", func(t *testing.T) {
				m := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "j", back)

				require.Equal(t, "▌▾ starter  service  fixed", selectedRowOf(m))
				require.Equal(t, withoutSelection(treeOf(opened(t, withNeeds(), "service", "my-service", withAlpha()))), withoutSelection(treeOf(m)))
			})

			t.Run(back+" at the open starter closes it, and l opens it again", func(t *testing.T) {
				closed := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", back)

				require.Equal(t, []string{"   name     my-service  fixed", "▌▸ starter  service  fixed"}, treeOf(closed))
				require.Equal(t, treeOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j")), treeOf(pressed(closed, "l")))
			})
		}

		t.Run("h at the closed starter and at the name row changes nothing", func(t *testing.T) {
			closed := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "h")

			require.Equal(t, screenOf(closed), screenOf(pressed(closed, "h")))
			require.Equal(t, screenOf(pressed(closed, "k")), screenOf(pressed(closed, "k", "h")))
		})

		for _, open := range []string{"l", "enter"} {
			t.Run(open+" at a feature with no option changes nothing", func(t *testing.T) {
				atFeature := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j")

				require.Equal(t, screenOf(atFeature), screenOf(pressed(atFeature, open)))
			})
		}
	})

	t.Run("features", func(t *testing.T) {
		withSameExtensionPoint := func() fstest.MapFS {
			files := withNeeds()
			files["features/p/feature.yml"] = &fstest.MapFile{Data: []byte("summary: P.\ntargets: [service]\nextension_points:\n  shared.point:\n    summary: A point.\n    file: go/main.go\n    item: [code]\n")}
			files["features/q/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Q.\ntargets: [service]\nextension_points:\n  shared.point:\n    summary: A point.\n    file: go/main.go\n    item: [code]\n")}
			files["features/r/feature.yml"] = &fstest.MapFile{Data: []byte("summary: R.\ntargets: [service]\nneeds: [missing]\n")}
			return files
		}

		t.Run("space at a feature that is not selected selects it with the source answered, and the count follows", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "j", "j", "space")

			require.Equal(t, "▌  ├─   [x] beta", selectedRowOf(m))
			require.Equal(t, "source  answered", lastLineOf(detailsOf(m)))
			require.True(t, strings.HasSuffix(linesOf(m)[0], " 1 feature"), linesOf(m)[0])
		})

		t.Run("space at a feature also selects the features that it needs, each marked needed by", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "j", "j", "j", "j", "space")

			require.Equal(t, []string{"   ├─   [ ] beta", "   ├─   [x] zed  needed by alpha", "▌  ├─   [x] alpha", "   └─   [ ] gamma"}, treeOf(m)[2:])
			require.True(t, strings.HasSuffix(linesOf(m)[0], " 2 features"), linesOf(m)[0])
		})

		t.Run("space at a feature that the developer selected clears it and each feature that only its needs selected", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "j", "j", "j", "j", "space", "space")

			require.Equal(t, []string{"   ├─   [ ] beta", "   ├─   [ ] zed", "▌  ├─   [ ] alpha", "   └─   [ ] gamma"}, treeOf(m)[2:])
		})

		t.Run("a clear keeps a need that the developer selected before, with the source answered", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "j", "j", "j", "space", "j", "space", "space", "k")

			require.Equal(t, []string{"   ├─   [ ] beta", "▌  ├─   [x] zed", "   ├─   [ ] alpha", "   └─   [ ] gamma"}, treeOf(m)[2:])
			require.Equal(t, "source  answered", lastLineOf(detailsOf(m)))
		})

		t.Run("a feature that the developer selected and that a selected feature needs shows both sources", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "j", "j", "j", "space", "j", "space", "k")

			require.Equal(t, "source  answered, needed by alpha", lastLineOf(detailsOf(m)))
		})

		for name, keys := range map[string][]string{
			"selected only by the need":      {"j", "j", "j", "j", "space", "k", "space"},
			"also selected by the developer": {"j", "j", "j", "space", "j", "space", "k", "space"},
		} {
			t.Run("space at a feature that a selected feature needs changes nothing, and the details say which features need it: "+name, func(t *testing.T) {
				m := pressed(opened(t, withNeeds(), "service", "my-service", nil), keys...)

				require.Equal(t, "▌  ├─   [x] zed  needed by alpha", selectedRowOf(m))
				require.Equal(t, "zed stays selected: it is needed by alpha.", firstDetailsLineOf(m))
			})
		}

		t.Run("space at a feature of --feature changes nothing, and the details say that the command line selects it", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "j", "j", "space")

			require.Equal(t, "▌  ├─   [x] alpha  fixed", selectedRowOf(m))
			require.Equal(t, "alpha stays selected: --feature selects it.", firstDetailsLineOf(m))
		})

		t.Run("the developer can select more features next to a feature of --feature", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "j", "j", "j", "space")

			require.Equal(t, "▌  └─   [x] gamma", selectedRowOf(m))
		})

		for name, test := range map[string]struct {
			keys    []string
			message string
		}{
			"two features that declare one extension point": {[]string{"j", "j", "j", "space", "j", "space"}, `both declare the extension point "shared.point"`},
			"a need that does not target the starter":       {[]string{"j", "j", "j", "j", "j", "space"}, `the feature r needs "missing", which is not a feature that targets the starter service`},
		} {
			t.Run("space at a feature whose selection Select refuses leaves the row as it was, and the details show the message of Select: "+name, func(t *testing.T) {
				m := pressed(opened(t, withSameExtensionPoint(), "service", "my-service", nil), test.keys...)

				require.Contains(t, selectedRowOf(m), "[ ]")
				require.Contains(t, strings.Join(strings.Fields(detailsOf(m)), " "), test.message)
			})
		}

		t.Run("space at a feature whose need sets another value than a flag leaves the row as it was, and the details show the message that names both sources", func(t *testing.T) {
			m := pressed(opened(t, withNeedValue(), "service", "my-service", map[string]map[string]string{"auth": {"method": "oauth"}}), "j", "j", "j", "j", "space")

			require.Contains(t, selectedRowOf(m), "[ ] cli-auth")
			require.Contains(t, strings.Join(strings.Fields(detailsOf(m)), " "), `the option method of the feature auth has two values: "oauth" from the command line and "api-key" from the need of cli-auth`)
		})

		t.Run("space at the name row or at the starter row changes nothing", func(t *testing.T) {
			m := opened(t, withNeeds(), "service", "my-service", nil)

			require.Equal(t, []string{screenOf(m), screenOf(pressed(m, "j"))}, []string{screenOf(pressed(m, "space")), screenOf(pressed(m, "j", "space"))})
		})

		t.Run("the message of a space goes away at the next key", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "j", "j", "space", "k", "j")

			require.Equal(t, "Feature alpha.", firstDetailsLineOf(m))
		})
	})

	t.Run("options", func(t *testing.T) {
		openedWith := func(t *testing.T, given map[string]string, givenByFeature map[string]map[string]string) form.Model {
			t.Helper()
			m, err := form.New(catalog.New(withOptions()), "service", "my-service", given, givenByFeature)
			require.NoError(t, err)
			return sized(m, 120, 20)
		}
		atAlpha := []string{"j", "j", "j", "j", "j", "j", "j"}

		t.Run("under the starter, before the features, the tree shows each option of the starter in the order of the names with its value, and a feature that is not selected shows no option", func(t *testing.T) {
			require.Equal(t, []string{"   ├─   module  example.com/my-service", "   ├─   region  us", "   ├─   zone      missing", "   ├─   [ ] beta", "   ├─   [ ] zed", "   ├─   [ ] alpha", "   └─   [ ] gamma"}, treeOf(openedWith(t, nil, nil))[2:])
		})

		t.Run("a space that selects a feature shows its options under it and those of its needs, the options of a need before those of the feature that needs it", func(t *testing.T) {
			m := pressed(openedWith(t, nil, nil), append(atAlpha, "space")...)

			require.Equal(t, []string{"   ├─   [ ] beta", "   ├─ ▾ [x] zed  needed by alpha", "   │  └─   level  low", "▌  ├─ ▾ [x] alpha", "   │  ├─   mode     fast", "   │  └─   timeout  30", "   └─   [ ] gamma"}, treeOf(m)[5:])
		})

		t.Run("a space that clears a feature removes its options and those of its needs", func(t *testing.T) {
			m := pressed(openedWith(t, nil, nil), append(atAlpha, "space", "space")...)

			require.Equal(t, withoutSelection(treeOf(pressed(openedWith(t, nil, nil), atAlpha...))), withoutSelection(treeOf(m)))
		})

		t.Run("each option of the selection is in the tree one time, also when two selected features need its feature", func(t *testing.T) {
			m := pressed(openedWith(t, nil, nil), append(atAlpha, "space", "j", "j", "j", "space")...)

			require.Equal(t, 1, strings.Count(strings.Join(treeOf(m), "\n"), "level  low"))
		})

		t.Run("the options of the last feature have no line beside them", func(t *testing.T) {
			m := pressed(openedWith(t, nil, nil), "j", "j", "j", "j", "j", "j", "j", "j", "space")

			require.Equal(t, []string{"▌  └─ ▾ [x] gamma", "      └─   tone  soft"}, treeOf(m)[9:])
		})

		t.Run("enter at the starter row keeps it selected", func(t *testing.T) {
			require.Equal(t, "▌▾ starter  service  fixed", selectedRowOf(pressed(openedWith(t, nil, nil), "j", "enter")))
		})

		t.Run("h closes a selected feature with options, and l and enter open it again", func(t *testing.T) {
			atZed := pressed(openedWith(t, nil, nil), append(atAlpha, "space", "k", "k")...)
			closed := pressed(atZed, "h")

			require.Equal(t, "▌  ├─ ▸ [x] zed  needed by alpha", selectedRowOf(closed))
			require.NotContains(t, strings.Join(treeOf(closed), "\n"), "level")
			require.Equal(t, []string{screenOf(atZed), screenOf(atZed)}, []string{screenOf(pressed(closed, "l")), screenOf(pressed(closed, "enter"))})
		})

		t.Run("l at an open feature selects its first option, and h at an option selects its feature", func(t *testing.T) {
			atZed := pressed(openedWith(t, nil, nil), append(atAlpha, "space", "k", "k")...)

			require.Equal(t, []string{"▌  │  └─   level  low", "▌  ├─ ▾ [x] zed  needed by alpha"}, []string{selectedRowOf(pressed(atZed, "l")), selectedRowOf(pressed(atZed, "l", "h"))})
		})

		t.Run("h at an option of the starter selects the starter", func(t *testing.T) {
			require.Equal(t, "▌▾ starter  service  fixed", selectedRowOf(pressed(openedWith(t, nil, nil), "j", "j", "j", "h")))
		})

		t.Run("an option with no default that a flag gives the empty text is marked fixed and missing", func(t *testing.T) {
			require.Equal(t, "   ├─   zone      fixed, missing", treeOf(openedWith(t, map[string]string{"zone": ""}, nil))[4])
		})

		t.Run("an option whose value the need of a selected feature sets is marked fixed, and the mark goes when no selected need sets it", func(t *testing.T) {
			needed := pressed(sized(opened(t, withNeedValue(), "service", "my-service", nil), 120, 20), "j", "j", "space", "j", "j", "space")
			cleared := pressed(needed, "space")

			require.Equal(t, []string{"   │  └─   method  api-key  fixed", "   │  └─   method  api-key"}, []string{treeOf(needed)[3], treeOf(cleared)[3]})
		})

		for name, test := range map[string]struct {
			givenByFeature map[string]map[string]string
			source         string
		}{
			"the need of a selected feature sets":                   {nil, "need of cli-auth"},
			"a flag gives with the value that a selected need sets": {map[string]map[string]string{"auth": {"method": "api-key"}}, "command line, need of cli-auth"},
		} {
			t.Run("the details of an option whose value "+name+" show the source "+test.source, func(t *testing.T) {
				m := pressed(opened(t, withNeedValue(), "service", "my-service", test.givenByFeature), "j", "j", "j", "j", "space", "k")

				require.Equal(t, "value    api-key\ndefault  api-key\nvalues   api-key, oauth\nsource   "+test.source, detailsOf(m))
			})
		}

		t.Run("an option that --option or --feature gives is marked fixed", func(t *testing.T) {
			m := openedWith(t, map[string]string{"region": "eu"}, map[string]map[string]string{"zed": {"level": "high"}})

			require.Equal(t, []string{"   ├─   region  eu  fixed", "   │  └─   level  high  fixed"}, []string{treeOf(m)[3], treeOf(m)[7]})
		})

		for name, test := range map[string]struct {
			given          map[string]string
			givenByFeature map[string]map[string]string
			keys           []string
			title, details string
		}{
			"an option of the starter with a pattern":    {nil, nil, []string{"j", "j", "j"}, "starter › region", "value    us\ndefault  us\npattern  ^[a-z]+$\nsource   default"},
			"an option of the starter that a flag gives": {map[string]string{"region": "eu"}, nil, []string{"j", "j", "j"}, "starter › region", "value    eu\ndefault  us\npattern  ^[a-z]+$\nsource   command line"},
			"an option whose default reads the name":     {nil, nil, []string{"j", "j"}, "starter › module", "value    example.com/my-service\ndefault  example.com/my-service\nsource   default"},
			"an option of a feature with values":         {nil, map[string]map[string]string{"zed": {}}, []string{"j", "j", "j", "j", "j", "j", "j"}, "zed › level", "value    low\ndefault  low\nvalues   low, mid, high\nsource   default"},
			"an option of a feature with one value":      {nil, map[string]map[string]string{"gamma": {}}, []string{"j", "j", "j", "j", "j", "j", "j", "j", "j", "j"}, "gamma › tone", "value    soft\ndefault  soft\nvalues   soft\nsource   default"},
		} {
			t.Run("the details of "+name+" show its value, its default, its values or its pattern, and its source", func(t *testing.T) {
				m := pressed(openedWith(t, test.given, test.givenByFeature), test.keys...)

				_, title := paneTitlesOf(m)
				require.Equal(t, []string{test.title, test.details}, []string{title, detailsOf(m)})
			})
		}

		t.Run("the default that reads the name renders with the empty text while the name is missing", func(t *testing.T) {
			m, err := form.New(catalog.New(withOptions()), "service", "", nil, nil)
			require.NoError(t, err)

			require.Equal(t, "   ├─   module  example.com/", treeOf(sized(m, 120, 20))[2])
		})
	})

	t.Run("edit", func(t *testing.T) {
		editable := func(t *testing.T, given map[string]string, givenByFeature map[string]map[string]string) form.Model {
			t.Helper()
			m, err := form.New(catalog.New(withOptions()), "service", "", given, givenByFeature)
			require.NoError(t, err)
			return sized(m, 120, 20)
		}
		typed := func(text string) []string {
			keys := make([]string, 0, len(text))
			for _, letter := range text {
				keys = append(keys, string(letter))
			}
			return keys
		}
		keysOf := func(groups ...[]string) []string {
			var keys []string
			for _, group := range groups {
				keys = append(keys, group...)
			}
			return keys
		}

		t.Run("enter at the name row starts an edit, and each key goes to the text, also q, j, k, h, l, w and space", func(t *testing.T) {
			m, quit := pressedAndQuit(editable(t, nil, nil), keysOf([]string{"enter"}, typed("my"), []string{"q", "j", "k", "h", "l", "w", "space", "x"})...)

			require.False(t, quit)
			require.Equal(t, "> myqjkhlw x", firstDetailsLineOf(m))
		})

		t.Run("while the check of the name refuses the text, the details show its message, and enter keeps nothing", func(t *testing.T) {
			m := pressed(editable(t, nil, nil), keysOf([]string{"enter"}, typed("-x"), []string{"enter"})...)

			require.Equal(t, "> -x", firstDetailsLineOf(m))
			require.Contains(t, strings.Join(strings.Fields(detailsOf(m)), " "), `the name "-x" of the repo must start with a letter or a digit`)
			require.Equal(t, "▌  name       missing", treeOf(m)[0])
		})

		t.Run("enter keeps a name that the check accepts, and the name has the source answered", func(t *testing.T) {
			m := pressed(editable(t, nil, nil), keysOf([]string{"enter"}, typed("my"), []string{"enter"})...)

			require.Equal(t, []string{"▌  name     my", "value   my\nsource  answered"}, []string{treeOf(m)[0], detailsOf(m)})
		})

		t.Run("esc drops the change, and the name stays as it was before the edit", func(t *testing.T) {
			m := pressed(editable(t, nil, nil), keysOf([]string{"enter"}, typed("my"), []string{"enter", "enter"}, typed("x"), []string{"esc"})...)

			require.Equal(t, []string{"▌  name     my", "value   my\nsource  answered"}, []string{treeOf(m)[0], detailsOf(m)})
		})

		t.Run("an option without values takes typed text: a text that fails its check shows the message and is not kept, and a text that passes is kept with the source answered", func(t *testing.T) {
			atRegion := pressed(editable(t, nil, nil), "j", "j", "j", "enter")
			refused := pressed(atRegion, keysOf([]string{"backspace", "backspace"}, typed("EU"), []string{"enter"})...)
			kept := pressed(refused, keysOf([]string{"backspace", "backspace"}, typed("eu"), []string{"enter"})...)

			require.Equal(t, "> us", firstDetailsLineOf(atRegion))
			require.Contains(t, strings.Join(strings.Fields(detailsOf(refused)), " "), `the value "EU" of the option region does not match its pattern ^[a-z]+$`)
			require.Equal(t, []string{"▌  ├─   region  eu", "source   answered"}, []string{selectedRowOf(kept), lastLineOf(detailsOf(kept))})
		})

		t.Run("a kept value that equals the default has the source answered", func(t *testing.T) {
			require.Equal(t, "source   answered", lastLineOf(detailsOf(pressed(editable(t, nil, nil), "j", "j", "j", "enter", "enter"))))
		})

		t.Run("an option with no default shows no missing mark once the developer keeps a value", func(t *testing.T) {
			require.Equal(t, "▌  ├─   zone    abc", selectedRowOf(pressed(editable(t, nil, nil), keysOf([]string{"j", "j", "j", "j", "enter"}, typed("abc"), []string{"enter"})...)))
		})

		t.Run("enter at an option with values lists them in the order of the file with the current value chosen, j and k move, and enter keeps the chosen value", func(t *testing.T) {
			listed := pressed(editable(t, nil, map[string]map[string]string{"zed": {}}), "j", "j", "j", "j", "j", "j", "j", "enter")
			moved := pressed(listed, "j", "j", "k", "j")
			kept := pressed(moved, "enter")

			require.Equal(t, []string{"▌low\n mid\n high", " low\n mid\n▌high"}, []string{detailsOf(listed), detailsOf(moved)})
			require.Equal(t, []string{"▌  │  └─   level  high", "source   answered"}, []string{selectedRowOf(kept), lastLineOf(detailsOf(kept))})
		})

		t.Run("esc drops the chosen value of a list", func(t *testing.T) {
			m := pressed(editable(t, nil, map[string]map[string]string{"zed": {}}), "j", "j", "j", "j", "j", "j", "j", "enter", "j", "esc")

			require.Equal(t, "▌  │  └─   level  low", selectedRowOf(m))
		})

		t.Run("enter at the name of an argument starts no edit, and the details say that the command line fixes it", func(t *testing.T) {
			require.Equal(t, "The name is fixed: the command line gives it.", firstDetailsLineOf(pressed(opened(t, withOptions(), "service", "my-service", nil), "enter")))
		})

		for name, test := range map[string]struct {
			givenByFeature map[string]map[string]string
			message        string
		}{
			"a need sets":                          {nil, "The option method is fixed: the need of cli-auth sets it."},
			"a need sets and a flag gives with it": {map[string]map[string]string{"auth": {"method": "api-key"}}, "The option method is fixed: the command line and the need of cli-auth give it."},
		} {
			t.Run("enter at an option that "+name+" starts no edit, and the details say what fixes it", func(t *testing.T) {
				m := pressed(opened(t, withNeedValue(), "service", "my-service", test.givenByFeature), "j", "j", "j", "j", "space", "k", "enter")

				require.Equal(t, test.message, strings.Join(strings.Fields(strings.SplitN(detailsOf(m), "\n\n", 2)[0]), " "))
			})
		}

		t.Run("a value that the developer kept gives way to the value of a need, and the option has the kept value again when no selected need sets it", func(t *testing.T) {
			kept := pressed(sized(opened(t, withNeedValue(), "service", "my-service", nil), 120, 20), "j", "j", "space", "j", "enter", "j", "enter")
			needed := pressed(kept, "j", "space")
			cleared := pressed(needed, "space")

			require.Equal(t, []string{"▌  │  └─   method  oauth", "   │  └─   method  api-key  fixed", "   │  └─   method  oauth"}, []string{treeOf(kept)[3], treeOf(needed)[3], treeOf(cleared)[3]})
		})

		t.Run("enter at an option of a flag starts no edit, and the details say that the command line fixes it", func(t *testing.T) {
			require.Equal(t, "The option region is fixed: the command line gives it.", firstDetailsLineOf(pressed(editable(t, map[string]string{"region": "eu"}, nil), "j", "j", "j", "enter")))
		})

		t.Run("an option whose default reads the name follows the name until the developer keeps a value for it", func(t *testing.T) {
			named := pressed(editable(t, nil, nil), keysOf([]string{"enter"}, typed("my-service"), []string{"enter"})...)
			renamed := pressed(named, keysOf([]string{"enter"}, []string{"backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "backspace"}, typed("other"), []string{"enter"})...)
			moduleKept := pressed(renamed, "j", "j", "enter", "enter", "k", "k")
			renamedAgain := pressed(moduleKept, keysOf([]string{"enter"}, []string{"backspace", "backspace", "backspace", "backspace", "backspace"}, typed("third"), []string{"enter"})...)

			require.Equal(t, []string{"   ├─   module  example.com/", "   ├─   module  example.com/my-service", "   ├─   module  example.com/other", "   ├─   module  example.com/other"}, []string{treeOf(editable(t, nil, nil))[2], treeOf(named)[2], treeOf(renamed)[2], treeOf(renamedAgain)[2]})
		})

		t.Run("a feature that a space clears loses the values kept for it, also a need that the clear removes", func(t *testing.T) {
			selected := pressed(editable(t, nil, nil), "j", "j", "j", "j", "j", "j", "j", "space")
			edited := pressed(selected, keysOf([]string{"j", "enter", "backspace", "backspace", "backspace", "backspace"}, typed("slow"), []string{"enter", "k", "k", "enter", "j", "enter", "j"})...)
			again := pressed(edited, "space", "space")

			require.Equal(t, []string{"   │  └─   level  mid", "   │  ├─   mode     slow"}, []string{treeOf(edited)[7], treeOf(edited)[9]})
			require.Equal(t, []string{"   │  └─   level  low", "   │  ├─   mode     fast"}, []string{treeOf(again)[7], treeOf(again)[9]})
		})

		t.Run("an empty name is not kept, and the details show the message of the check of the name", func(t *testing.T) {
			m := pressed(editable(t, nil, nil), "enter", "enter")

			require.Equal(t, ">", firstDetailsLineOf(m))
			require.Contains(t, strings.Join(strings.Fields(detailsOf(m)), " "), `the name "" of the repo must start with a letter or a digit`)
		})

		t.Run("an empty value of an option with no default is not kept, and the details say that it needs a value", func(t *testing.T) {
			m := pressed(editable(t, nil, nil), "j", "j", "j", "j", "enter", "enter")

			require.Equal(t, ">", firstDetailsLineOf(m))
			require.Contains(t, strings.Join(strings.Fields(detailsOf(m)), " "), "the option zone of the starter service has no default, so it needs a value")
		})

		t.Run("a second edit of the name starts with the kept name", func(t *testing.T) {
			require.Equal(t, "> my", firstDetailsLineOf(pressed(editable(t, nil, nil), keysOf([]string{"enter"}, typed("my"), []string{"enter", "enter"})...)))
		})

		for keys, details := range map[string]string{
			"j":     " low\n▌mid\n high",
			"j k":   "▌low\n mid\n high",
			"k":     "▌low\n mid\n high",
			"j j j": " low\n mid\n▌high",
			"down":  " low\n▌mid\n high",
			"j up":  "▌low\n mid\n high",
		} {
			t.Run("j and k move the chosen value by one, as down and up do, and stop at the ends: "+keys, func(t *testing.T) {
				m := pressed(editable(t, nil, map[string]map[string]string{"zed": {}}), keysOf([]string{"j", "j", "j", "j", "j", "j", "j", "enter"}, strings.Fields(keys))...)

				require.Equal(t, details, detailsOf(m))
			})
		}

		t.Run("an option with values and no value yet lists its values with the first chosen", func(t *testing.T) {
			require.Equal(t, "▌red\n blue", detailsOf(pressed(editable(t, nil, nil), "j", "j", "j", "j", "j", "space", "j", "enter")))
		})

		t.Run("an option with one value lists that value", func(t *testing.T) {
			require.Equal(t, "▌soft", detailsOf(pressed(editable(t, nil, map[string]map[string]string{"gamma": {}}), "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "enter")))
		})

		t.Run("a second edit of an option with values chooses the kept value", func(t *testing.T) {
			m := pressed(editable(t, nil, map[string]map[string]string{"zed": {}}), "j", "j", "j", "j", "j", "j", "j", "enter", "j", "j", "enter", "enter")

			require.Equal(t, " low\n mid\n▌high", detailsOf(m))
		})

		t.Run("h and l at a list change nothing", func(t *testing.T) {
			listed := pressed(editable(t, nil, map[string]map[string]string{"zed": {}}), "j", "j", "j", "j", "j", "j", "j", "enter")
			m, quit := pressedAndQuit(listed, "h", "l")

			require.False(t, quit)
			require.Equal(t, screenOf(listed), screenOf(m))
		})

		t.Run("after a value of a list is kept, the footer shows the keys of the tree again", func(t *testing.T) {
			m := pressed(editable(t, nil, map[string]map[string]string{"zed": {}}), "j", "j", "j", "j", "j", "j", "j", "enter", "enter")

			require.Equal(t, " j/k move • l open • h close • space select • w write • ctrl+d/u scroll details • q quit", linesOf(m)[19])
		})

		t.Run("a space at another feature keeps the values kept for the selected features", func(t *testing.T) {
			edited := pressed(editable(t, nil, nil), keysOf([]string{"j", "j", "j", "j", "j", "j", "j", "space", "j", "enter", "backspace", "backspace", "backspace", "backspace"}, typed("slow"), []string{"enter"})...)
			m := pressed(edited, "k", "k", "k", "k", "space")

			require.Equal(t, []string{"▌  ├─ ▾ [x] beta", "   │  ├─   mode     slow"}, []string{treeOf(m)[5], treeOf(m)[10]})
		})

		t.Run("with no starter yet, the name of an edit is kept also when flags are given", func(t *testing.T) {
			m, err := form.New(catalog.New(withNeeds()), "", "", nil, withAlpha())
			require.NoError(t, err)

			require.Equal(t, "▌  name     my", treeOf(pressed(sized(m, 120, 20), "enter", "m", "y", "enter"))[0])
		})

		t.Run("ctrl+c stops the form during an edit, and the form then gives the cancel", func(t *testing.T) {
			m, quit := pressedAndQuit(editable(t, nil, nil), "enter", "m", "ctrl+c")

			_, err := m.Confirmed()
			require.True(t, quit)
			require.EqualError(t, err, "you canceled new, so it wrote no file")
		})

		t.Run("q at a list of values stops the form", func(t *testing.T) {
			_, quit := pressedAndQuit(editable(t, nil, map[string]map[string]string{"zed": {}}), "j", "j", "j", "j", "j", "j", "j", "enter", "q")

			require.True(t, quit)
		})

		t.Run("during an edit, the footer shows the keys of the edit", func(t *testing.T) {
			text := pressed(editable(t, nil, nil), "enter")
			list := pressed(editable(t, nil, map[string]map[string]string{"zed": {}}), "j", "j", "j", "j", "j", "j", "j", "enter")

			require.Equal(t, []string{" enter keep • esc drop • ctrl+c quit", " j/k move • enter keep • esc drop • q quit"}, []string{linesOf(text)[19], linesOf(list)[19]})
		})
	})

	t.Run("starter", func(t *testing.T) {
		withStarters := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\noptions:\n  region:\n    default: us\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"starters/worker/starter.yml":       &fstest.MapFile{Data: []byte("summary: A worker.\noptions:\n  queue:\n    default: jobs\nfolders:\n  go:\n    path: .\n")},
				"starters/worker/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
				"starters/broken/starter.yml":       &fstest.MapFile{Data: []byte("summary: Broken.\nfolders:\n  go:\n    path: .\n")},
				"starters/broken/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
				"features/shared/feature.yml":       &fstest.MapFile{Data: []byte("summary: Shared.\ntargets: [service, worker]\noptions:\n  tone:\n    default: soft\n")},
				"features/only/feature.yml":         &fstest.MapFile{Data: []byte("summary: Only.\ntargets: [service]\noptions:\n  mode:\n    default: fast\n")},
				"features/x/feature.yml":            &fstest.MapFile{Data: []byte("summary: X.\ntargets: [broken]\nneeds: [y]\n")},
				"features/y/feature.yml":            &fstest.MapFile{Data: []byte("summary: Y.\ntargets: [broken]\nneeds: [x]\n")},
			}
		}
		unchosen := func(t *testing.T, given map[string]string, givenByFeature map[string]map[string]string) form.Model {
			t.Helper()
			m, err := form.New(catalog.New(withStarters()), "", "my-service", given, givenByFeature)
			require.NoError(t, err)
			return sized(m, 120, 20)
		}
		chosen := func(t *testing.T, keys ...string) form.Model {
			t.Helper()
			return pressed(unchosen(t, nil, nil), append([]string{"j", "enter", "j", "enter"}, keys...)...)
		}

		t.Run("enter at the starter row lists each starter in the order of the names with its summary, the first chosen when none is", func(t *testing.T) {
			require.Equal(t, "▌broken   Broken.\n service  A service.\n worker   A worker.", detailsOf(pressed(unchosen(t, nil, nil), "j", "enter")))
		})

		t.Run("enter keeps the chosen starter, and the tree shows its options and its features with the source answered", func(t *testing.T) {
			m := chosen(t)

			require.Equal(t, []string{"▌▾ starter  service", "   ├─   region  us", "   ├─   [ ] only", "   └─   [ ] shared"}, treeOf(m)[1:])
			require.Equal(t, "A service.\n\nsource  answered", detailsOf(m))
		})

		t.Run("a second edit of the starter chooses the kept starter", func(t *testing.T) {
			require.Equal(t, " broken   Broken.\n▌service  A service.\n worker   A worker.", detailsOf(chosen(t, "enter")))
		})

		t.Run("esc drops the chosen starter", func(t *testing.T) {
			require.Equal(t, "▌  starter    missing", treeOf(pressed(unchosen(t, nil, nil), "j", "enter", "j", "esc"))[1])
		})

		t.Run("a change of the starter keeps each selected feature that targets the new starter with its kept values, and clears the others and the kept values of the starter", func(t *testing.T) {
			edited := chosen(t,
				"j", "enter", "backspace", "backspace", "e", "u", "enter",
				"j", "space", "j", "enter", "backspace", "backspace", "backspace", "backspace", "s", "l", "o", "w", "enter",
				"j", "space", "j", "enter", "backspace", "backspace", "backspace", "backspace", "l", "o", "u", "d", "enter")
			changed := pressed(edited, "k", "k", "k", "k", "k", "enter", "j", "enter")

			require.Equal(t, []string{"   ├─   region  eu", "   ├─ ▾ [x] only", "   │  └─   mode  slow", "   └─ ▾ [x] shared", "▌     └─   tone  loud"}, treeOf(edited)[2:])
			require.Equal(t, []string{"▌▾ starter  worker", "   ├─   queue  jobs", "   └─ ▾ [x] shared", "      └─   tone  loud"}, treeOf(changed)[1:])
		})

		t.Run("keeping the current starter keeps the kept values of the starter", func(t *testing.T) {
			m := chosen(t, "j", "enter", "backspace", "backspace", "e", "u", "enter", "k", "enter", "enter")

			require.Equal(t, "   ├─   region  eu", treeOf(m)[2])
		})

		t.Run("keeping the current starter changes nothing", func(t *testing.T) {
			selected := chosen(t, "j", "j", "space", "k", "k")

			require.Equal(t, screenOf(selected), screenOf(pressed(selected, "enter", "enter")))
		})

		for name, test := range map[string]struct {
			given          map[string]string
			givenByFeature map[string]map[string]string
			keys           []string
			message        string
		}{
			"a feature of --feature that does not target it": {nil, map[string]map[string]string{"only": {}}, []string{"j", "j"}, `unknown feature "only": the features that target the starter worker are shared`},
			"an option of --option that it does not declare": {map[string]string{"region": "eu"}, nil, []string{"j", "j"}, `unknown option "region": the options of the starter worker are queue`},
			"features whose needs form a cycle":              {nil, nil, nil, "the needs of the features form a cycle: x needs y needs x"},
		} {
			t.Run("a starter that does not fit is not kept, the tree stays as it was, and the details show the message: "+name, func(t *testing.T) {
				m := pressed(unchosen(t, test.given, test.givenByFeature), append([]string{"j", "enter"}, append(test.keys, "enter")...)...)

				require.Equal(t, "▌  starter    missing", treeOf(m)[1])
				require.Equal(t, test.message, strings.Join(strings.Fields(strings.SplitN(detailsOf(m), "\n\n", 2)[0]), " "))
			})
		}

		t.Run("the message of a starter that does not fit goes away at the next key", func(t *testing.T) {
			m := pressed(unchosen(t, nil, nil), "j", "enter", "enter", "j")

			require.Equal(t, " broken   Broken.\n▌service  A service.\n worker   A worker.", detailsOf(m))
		})

		t.Run("a change of the starter drops the kept values of a feature that only a cleared feature needed", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"starters/worker/starter.yml":       &fstest.MapFile{Data: []byte("summary: A worker.\nfolders:\n  go:\n    path: .\n")},
				"starters/worker/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
				"features/p/feature.yml":            &fstest.MapFile{Data: []byte("summary: P.\ntargets: [service]\nneeds: [q]\n")},
				"features/q/feature.yml":            &fstest.MapFile{Data: []byte("summary: Q.\ntargets: [service, worker]\noptions:\n  level:\n    values: [low, high]\n    default: low\n")},
			}
			m, err := form.New(catalog.New(files), "", "my-service", nil, nil)
			require.NoError(t, err)
			kept := pressed(sized(m, 120, 20), "j", "enter", "enter", "j", "j", "space", "k", "enter", "j", "enter")

			changed := pressed(kept, "k", "k", "enter", "j", "enter")

			require.Equal(t, "▌  │  └─   level  high", treeOf(kept)[3])
			require.Equal(t, []string{"▌▾ starter  worker", "   └─   [ ] q"}, treeOf(changed)[1:])
		})

		t.Run("a list longer than the details pane shows the chosen starter", func(t *testing.T) {
			m, err := form.New(catalog.New(manyStarters(12)), "", "my-service", nil, nil)
			require.NoError(t, err)

			down := pressed(sized(m, 90, 10), "j", "enter", "j", "j", "j", "j", "j", "j", "j", "j")
			up := pressed(down, "k", "k", "k", "k", "k", "k")

			require.Equal(t, []string{" starter-03  Summary 03.", "▌starter-08  Summary 08."}, []string{firstDetailsLineOf(down), strings.Split(detailsOf(down), "\n")[5]})
			require.Equal(t, "▌starter-02  Summary 02.", firstDetailsLineOf(up))
		})

		t.Run("a list under a message shows the chosen starter, and the message while the chosen starter fits under it", func(t *testing.T) {
			m, err := form.New(catalog.New(manyStarters(12)), "", "my-service", map[string]string{"x": "1"}, nil)
			require.NoError(t, err)
			keys := []string{"j", "enter", "j", "j", "j", "j", "j", "j", "j", "j", "enter"}

			low := pressed(sized(m, 90, 10), keys...)
			high := pressed(sized(m, 90, 16), keys...)

			require.Equal(t, "▌starter-08  Summary 08.", lastLineOf(detailsOf(low)))
			require.Equal(t, `unknown option "x": the starter starter-08 has no option`, strings.Join(strings.Fields(strings.SplitN(detailsOf(high), "\n\n", 2)[0]), " "))
		})

		t.Run("enter at the starter row of a catalog with no starter starts no list, and the details say that the catalog has no starter", func(t *testing.T) {
			require.Equal(t, "The catalog has no starter.", firstDetailsLineOf(pressed(opened(t, fstest.MapFS{"starters": &fstest.MapFile{Mode: fs.ModeDir | 0o755}}, "", "", nil), "j", "enter")))
		})

		t.Run("enter at the name row of a catalog with no starter starts an edit of the name", func(t *testing.T) {
			require.Equal(t, "> my", firstDetailsLineOf(pressed(opened(t, fstest.MapFS{"starters": &fstest.MapFile{Mode: fs.ModeDir | 0o755}}, "", "", nil), "enter", "m", "y")))
		})

		t.Run("enter at the starter of an argument starts no list, and the details say that the command line fixes it", func(t *testing.T) {
			require.Equal(t, "The starter is fixed: the command line gives it.", firstDetailsLineOf(pressed(opened(t, withStarters(), "service", "my-service", nil), "j", "enter")))
		})
	})

	t.Run("write", func(t *testing.T) {
		withBadTemplate := func() fstest.MapFS {
			files := withNeeds()
			files["starters/service/files/go/bad.go"] = &fstest.MapFile{Data: []byte("package {{ .Nmae }}\n")}
			return files
		}
		withManyFiles := func() fstest.MapFS {
			files := withNeeds()
			for i := range 30 {
				files[fmt.Sprintf("starters/service/files/go/f%02d.txt", i)] = &fstest.MapFile{Data: []byte("x\n")}
			}
			return files
		}
		t.Run("w with no answer missing shows the preview with the source of each feature and option, and asks whether to write the repo", func(t *testing.T) {
			m := pressed(sized(opened(t, withNeeds(), "service", "my-service", withAlpha()), 120, 20), "w")

			_, title := paneTitlesOf(m)
			require.Equal(t, "Preview", title)
			require.Equal(t, "Write the repo?\n\nRepo: my-service\nStarter: service\n\nFeatures:\n  zed: needed by alpha\n  alpha: command line\n\nOptions: none\n\nFiles:\n  main.go: the starter", detailsOf(m))
		})

		t.Run("the preview shows the answers of the developer with the source answered", func(t *testing.T) {
			m := pressed(sized(opened(t, withOptions(), "service", "my-service", map[string]map[string]string{"zed": {}}), 120, 30), "j", "j", "j", "j", "enter", "a", "b", "c", "enter", "w")

			require.Contains(t, detailsOf(m), "\n  starter.zone: abc (answered)\n")
		})

		t.Run("ctrl+d and ctrl+u scroll the preview, and the bottom border shows how far it is scrolled", func(t *testing.T) {
			top := pressed(sized(opened(t, withManyFiles(), "service", "my-service", nil), 120, 14), "w")
			scrolled := pressed(top, "ctrl+d")

			require.Equal(t, []string{"Write the repo?", "0%"}, []string{firstDetailsLineOf(top), scrollNoteOf(top)})
			require.NotEqual(t, "Write the repo?", firstDetailsLineOf(scrolled))
			require.Equal(t, screenOf(top), screenOf(pressed(scrolled, "ctrl+u")))
		})

		t.Run("the preview starts at its top after a scroll of the details, and the details start at their top after a scroll of the preview", func(t *testing.T) {
			var words []string
			for i := range 200 {
				words = append(words, fmt.Sprintf("w%03d", i))
			}
			files := withManyFiles()
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("summary: " + strings.Join(words, " ") + "\nfolders:\n  go:\n    path: .\n")}
			preview := pressed(sized(opened(t, files, "service", "my-service", nil), 120, 14), "j", "ctrl+d", "w")
			back := pressed(preview, "ctrl+d", "n")

			require.Equal(t, "Write the repo?", firstDetailsLineOf(preview))
			require.True(t, strings.HasPrefix(firstDetailsLineOf(back), "w000 w001 "), firstDetailsLineOf(back))
		})

		t.Run("y stops the form, which then gives the selection of the preview with the answers of the developer", func(t *testing.T) {
			m, quit := pressedAndQuit(sized(opened(t, withOptions(), "service", "my-service", map[string]map[string]string{"zed": {}}), 120, 30), "j", "j", "j", "j", "enter", "a", "b", "c", "enter", "w", "y")

			selection, err := m.Confirmed()
			require.NoError(t, err)
			require.True(t, quit)
			preview, err := selection.Preview()
			require.NoError(t, err)
			require.Contains(t, preview, "\n  starter.zone: abc (answered)\n")
		})

		for _, back := range []string{"n", "esc"} {
			t.Run(back+" goes back to the form with the same row selected and its details", func(t *testing.T) {
				atAlpha := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "j", "j")
				m, quit := pressedAndQuit(atAlpha, "w", back)

				require.False(t, quit)
				require.Equal(t, screenOf(atAlpha), screenOf(m))
			})
		}

		for name, keys := range map[string][]string{
			"q at the preview":      {"w", "q"},
			"ctrl+c at the preview": {"w", "ctrl+c"},
			"n and then q":          {"w", "n", "q"},
		} {
			t.Run(name+" stops the form with the cancel", func(t *testing.T) {
				m, quit := pressedAndQuit(opened(t, withNeeds(), "service", "my-service", nil), keys...)

				_, err := m.Confirmed()
				require.True(t, quit)
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		t.Run("the form gives the selection only after y", func(t *testing.T) {
			m, quit := pressedAndQuit(opened(t, withNeeds(), "service", "my-service", nil), "w", "j", "k", "space")

			_, err := m.Confirmed()
			require.False(t, quit)
			require.EqualError(t, err, "you canceled new, so it wrote no file")
		})

		t.Run("w with no name selects the name row, and its details show the message of the check of the name", func(t *testing.T) {
			m, quit := pressedAndQuit(opened(t, withNeeds(), "service", "", nil), "j", "j", "w")

			require.False(t, quit)
			require.Equal(t, "▌  name       missing", selectedRowOf(m))
			require.Contains(t, strings.Join(strings.Fields(detailsOf(m)), " "), `the name "" of the repo must start with a letter or a digit`)
		})

		t.Run("w with more than one missing answer selects the first in the order of the tree", func(t *testing.T) {
			m, err := form.New(catalog.New(withOptions()), "service", "", nil, nil)
			require.NoError(t, err)

			require.Equal(t, "▌  name       missing", selectedRowOf(pressed(sized(m, 120, 30), "j", "j", "j", "j", "w")))
		})

		t.Run("w with no starter selects the starter row, and its details say that no starter is chosen", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "", "my-service", nil), "w")

			require.Equal(t, "▌  starter    missing", selectedRowOf(m))
			require.Equal(t, "No starter is chosen.", firstDetailsLineOf(m))
		})

		t.Run("w with an option that has no default and no value selects its row, opens the rows above it, and the details show the message of its check", func(t *testing.T) {
			closed := pressed(sized(opened(t, withOptions(), "service", "my-service", nil), 120, 30), "j", "j", "j", "j", "enter", "a", "b", "c", "enter", "j", "space", "h", "k", "k", "k", "k", "h")
			m := pressed(closed, "w")

			require.Equal(t, []string{"   name     my-service  fixed", "▌▸ starter  service  fixed"}, treeOf(closed))
			require.Equal(t, "▌  │  └─   color    missing", selectedRowOf(m))
			require.Equal(t, `the feature beta: the value "" of the option color is not one of its values: red, blue`, strings.Join(strings.Fields(strings.SplitN(detailsOf(m), "\n\n", 2)[0]), " "))
		})

		t.Run("a preview that stops shows its message with no question, y does not write, and esc goes back to the form", func(t *testing.T) {
			stopped := pressed(sized(opened(t, withBadTemplate(), "service", "my-service", nil), 120, 20), "w")
			m, quit := pressedAndQuit(stopped, "y")

			_, err := m.Confirmed()
			require.False(t, quit)
			require.EqualError(t, err, "you canceled new, so it wrote no file")
			require.Contains(t, detailsOf(stopped), "bad.go")
			require.NotContains(t, detailsOf(stopped), "Write the repo?")
			require.Equal(t, "▌  name     my-service  fixed", selectedRowOf(pressed(stopped, "esc")))
		})

		t.Run("the footer shows the keys of the preview while it shows", func(t *testing.T) {
			require.Equal(t, []string{" y write • esc back • ctrl+d/u scroll details • q quit", " esc back • ctrl+d/u scroll details • q quit"}, []string{
				linesOf(pressed(sized(opened(t, withNeeds(), "service", "my-service", nil), 120, 20), "w"))[19],
				linesOf(pressed(sized(opened(t, withBadTemplate(), "service", "my-service", nil), 120, 20), "w"))[19],
			})
		})
	})

	t.Run("details", func(t *testing.T) {
		t.Run("the name of an argument shows its value and the source command line", func(t *testing.T) {
			require.Equal(t, "value   my-service\nsource  command line", detailsOf(opened(t, withNeeds(), "service", "my-service", nil)))
		})

		t.Run("a missing name shows the message of the check of the name", func(t *testing.T) {
			require.Equal(t, `the name "" of the repo must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'`, strings.Join(strings.Fields(detailsOf(opened(t, withNeeds(), "", "", nil))), " "))
		})

		t.Run("the starter of an argument shows its summary and the source command line", func(t *testing.T) {
			require.Equal(t, "A service.\n\nsource  command line", detailsOf(pressed(opened(t, withNeeds(), "service", "my-service", nil), "j")))
		})

		t.Run("a missing starter lists each starter of the catalog with its summary", func(t *testing.T) {
			require.Equal(t, "No starter is chosen.\n\nStarters\n  other    Another service.\n  service  A service.", detailsOf(pressed(opened(t, withNeeds(), "", "", nil), "j")))
		})

		t.Run("a starter with no summary shows only its source", func(t *testing.T) {
			require.Equal(t, "source  command line", detailsOf(pressed(opened(t, withSummaries("", "A feature."), "service", "my-service", nil), "j")))
		})

		t.Run("a catalog with no starter lists none", func(t *testing.T) {
			require.Equal(t, "No starter is chosen.\n\nStarters\n  none", detailsOf(pressed(opened(t, fstest.MapFS{"starters": &fstest.MapFile{Mode: fs.ModeDir | 0o755}}, "", "", nil), "j")))
		})

		t.Run("a need that sets values shows each option that it sets with its value beside the need, and the needs come in the order of their names", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/a/feature.yml":            &fstest.MapFile{Data: []byte("summary: A.\ntargets: [service]\noptions:\n  level:\n    default: low\n  mode:\n    default: slow\n")},
				"features/b/feature.yml":            &fstest.MapFile{Data: []byte("summary: B.\ntargets: [service]\n")},
				"features/c/feature.yml":            &fstest.MapFile{Data: []byte("summary: Needs two.\ntargets: [service]\nneeds:\n  b:\n  a:\n    mode: fast\n    level: high\n")},
			}

			require.Equal(t, "Needs two.\n\nNeeds\n  a  level  high\n     mode   fast\n  b\n\nsource  not selected", detailsOf(pressed(opened(t, files, "service", "my-service", nil), "j", "j", "j", "j")))
		})

		t.Run("a feature with no need says none, and a feature with two needs lists each", func(t *testing.T) {
			atBeta := pressed(opened(t, withTwoNeeds(), "service", "my-service", nil), "j", "j")
			atDelta := pressed(atBeta, "j", "j", "j")

			require.Equal(t, []string{"Feature beta.\n\nNeeds\n  none\n\nsource  not selected", "Feature delta.\n\nNeeds\n  beta\n  zed\n\nsource  not selected"}, []string{detailsOf(atBeta), detailsOf(atDelta)})
		})

		t.Run("a summary longer than the details pane wraps at a space at the width of the pane", func(t *testing.T) {
			m := pressed(sized(opened(t, withSummaries(strings.Repeat("x", 45)+" abcd rest", "A feature."), "service", "my-service", nil), 90, 14), "j")

			require.Equal(t, []string{strings.Repeat("x", 45) + " abcd", "rest"}, strings.Split(detailsOf(m), "\n")[:2])
		})

		t.Run("a feature shows its summary, its needs and its source", func(t *testing.T) {
			require.Equal(t, "Feature alpha.\n\nNeeds\n  zed\n\nsource  command line", detailsOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "j", "j")))
		})

		for name, test := range map[string]struct {
			givenByFeature map[string]map[string]string
			keys           []string
			source         string
		}{
			"a feature that a selected feature needs":     {withAlpha(), []string{"j", "j", "j"}, "source  needed by alpha"},
			"a feature of --feature that a feature needs": {map[string]map[string]string{"alpha": {}, "zed": {}}, []string{"j", "j", "j"}, "source  command line, needed by alpha"},
			"a feature that is not selected":              {withAlpha(), []string{"j", "j"}, "source  not selected"},
		} {
			t.Run(name+" shows its source with the text of the preview", func(t *testing.T) {
				details := detailsOf(pressed(opened(t, withNeeds(), "service", "my-service", test.givenByFeature), test.keys...))

				require.Equal(t, test.source, lastLineOf(details))
			})
		}

		for keys, title := range map[string]string{
			"":      "name",
			"j":     "service",
			"j j j": "zed",
		} {
			t.Run("the title of the details pane names the selected row: "+keys, func(t *testing.T) {
				_, details := paneTitlesOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), strings.Fields(keys)...))

				require.Equal(t, title, details)
			})
		}

		t.Run("the title of the details pane of a missing starter is starter", func(t *testing.T) {
			_, details := paneTitlesOf(pressed(opened(t, withNeeds(), "", "", nil), "j"))

			require.Equal(t, "starter", details)
		})

		for _, scroll := range [][2]string{{"ctrl+d", "ctrl+u"}, {"pgdown", "pgup"}} {
			t.Run(scroll[0]+" scrolls the details down by half the pane, "+scroll[1]+" scrolls them up again, and the bottom border shows how far they are scrolled", func(t *testing.T) {
				top := pressed(sized(opened(t, manyStarters(12), "", "", nil), 90, 10), "j")
				scrolled := pressed(top, scroll[0])

				require.Equal(t, []string{"No starter is chosen.", "  starter-00  Summary 00."}, []string{firstDetailsLineOf(top), firstDetailsLineOf(scrolled)})
				require.Equal(t, []string{"0%", "33%"}, []string{scrollNoteOf(top), scrollNoteOf(scrolled)})
				require.Equal(t, screenOf(top), screenOf(pressed(scrolled, scroll[1])))
			})
		}

		t.Run("a newly selected row after a scroll shows its details from the top, also when they are long", func(t *testing.T) {
			long := strings.Repeat("word ", 60)
			scrolled := pressed(sized(opened(t, withSummaries("Starter "+long, "Feature "+long), "service", "my-service", nil), 90, 10), "j", "ctrl+d")

			require.Equal(t, "Feature word word word word word word word word", firstDetailsLineOf(pressed(scrolled, "j")))
		})

		t.Run("details that fill the pane exactly show no scroll position", func(t *testing.T) {
			require.Equal(t, "", scrollNoteOf(pressed(sized(opened(t, withNeeds(), "service", "my-service", withAlpha()), 90, 10), "j", "j", "j", "j")))
		})

		t.Run("details scrolled to their end show 100%", func(t *testing.T) {
			require.Equal(t, "100%", scrollNoteOf(pressed(sized(opened(t, manyStarters(12), "", "", nil), 90, 10), "j", "ctrl+d", "ctrl+d", "ctrl+d", "ctrl+d")))
		})

		t.Run("a newly selected row shows its details from the top", func(t *testing.T) {
			scrolled := pressed(sized(opened(t, manyStarters(12), "", "", nil), 90, 10), "j", "ctrl+d")

			require.Equal(t, "No starter is chosen.", firstDetailsLineOf(pressed(scrolled, "k", "j")))
		})
	})

	t.Run("stop", func(t *testing.T) {
		for name, keys := range map[string][]string{
			"q":              {"q"},
			"q at a feature": {"j", "j", "q"},
			"ctrl+c":         {"ctrl+c"},
		} {
			t.Run(name+" stops the form, which then gives the cancel", func(t *testing.T) {
				m, quit := pressedAndQuit(opened(t, withNeeds(), "service", "my-service", nil), keys...)

				_, err := m.Confirmed()
				require.True(t, quit)
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		t.Run("a message that is not a key or a size changes nothing", func(t *testing.T) {
			m := opened(t, withNeeds(), "service", "my-service", nil)

			next, cmd := m.Update(tea.FocusMsg{})

			require.Nil(t, cmd)
			require.Equal(t, screenOf(m), screenOf(next.(form.Model)))
		})

		t.Run("esc at the closed starter does not stop the form", func(t *testing.T) {
			_, quit := pressedAndQuit(opened(t, withNeeds(), "service", "my-service", nil), "j", "esc", "esc")

			require.False(t, quit)
		})
	})

	t.Run("start", func(t *testing.T) {
		for name, test := range map[string]struct {
			files          fstest.MapFS
			starterName    string
			name           string
			given          map[string]string
			givenByFeature map[string]map[string]string
			message        string
		}{
			"an unknown starter":                                    {withNeeds(), "nope", "my-service", nil, nil, `unknown starter "nope": the catalog has other, service`},
			"an unknown feature":                                    {withNeeds(), "service", "my-service", nil, map[string]map[string]string{"nope": {}}, `unknown feature "nope": the features that target the starter service are alpha, beta, gamma, zed`},
			"an unknown feature with no name":                       {withNeeds(), "service", "", nil, map[string]map[string]string{"nope": {}}, `unknown feature "nope": the features that target the starter service are alpha, beta, gamma, zed`},
			"an unknown option":                                     {withOptions(), "service", "my-service", map[string]string{"nope": "x"}, nil, `unknown option "nope": the options of the starter service are module, region, zone`},
			"a value that does not match its pattern":               {withOptions(), "service", "my-service", map[string]string{"region": "EU"}, nil, `the starter service: the value "EU" of the option region does not match its pattern ^[a-z]+$`},
			"features of the starter whose needs form a cycle":      {withCycle(), "service", "my-service", nil, nil, "the needs of the features form a cycle: beta needs gamma needs beta"},
			"a catalog that cannot load, with no starter":           {fstest.MapFS{"starters/service/starter.yml": &fstest.MapFile{Data: []byte("folders: [")}}, "", "", nil, nil, "starters/service/starter.yml"},
			"a flag that gives an option another value than a need": {withNeedValue(), "service", "my-service", nil, map[string]map[string]string{"auth": {"method": "oauth"}, "cli-auth": {}}, `"oauth" from the command line and "api-key" from the need of cli-auth`},
			"a name that the check refuses, with no starter":        {withNeeds(), "", "../x", nil, nil, `the name "../x" of the repo`},
		} {
			t.Run(name+" stops the form before it opens, with the message of the catalog", func(t *testing.T) {
				_, err := form.New(catalog.New(test.files), test.starterName, test.name, test.given, test.givenByFeature)

				require.ErrorContains(t, err, test.message)
			})
		}
	})
}

func sized(m form.Model, width, height int) form.Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return next.(form.Model)
}

var escapeCode = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func screenOf(m form.Model) string {
	return strings.Join(linesOf(m), "\n")
}

func linesOf(m form.Model) []string {
	lines := strings.Split(escapeCode.ReplaceAllString(m.View(), ""), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return lines
}

func panesOf(m form.Model) (tree, details []string) {
	lines := linesOf(m)
	for _, line := range lines[2 : len(lines)-2] {
		left, right, _ := strings.Cut(line, "││")
		tree = append(tree, strings.TrimRight(strings.TrimPrefix(left, "│ "), " "))
		details = append(details, strings.TrimRight(strings.TrimSuffix(strings.TrimPrefix(right, " "), "│"), " "))
	}
	return withoutEmptyEnd(tree), withoutEmptyEnd(details)
}

func treeOf(m form.Model) []string {
	tree, _ := panesOf(m)
	return tree
}

func detailsOf(m form.Model) string {
	_, details := panesOf(m)
	return strings.Join(details, "\n")
}

func lastLineOf(details string) string {
	return details[strings.LastIndex(details, "\n")+1:]
}

func firstDetailsLineOf(m form.Model) string {
	_, details := panesOf(m)
	return details[0]
}

func selectedRowOf(m form.Model) string {
	for _, row := range treeOf(m) {
		if strings.HasPrefix(row, "▌") {
			return row
		}
	}
	return ""
}

func withoutSelection(rows []string) []string {
	unselected := make([]string, 0, len(rows))
	for _, row := range rows {
		unselected = append(unselected, strings.Replace(row, "▌", " ", 1))
	}
	return unselected
}

var paneTitles = regexp.MustCompile(`^╭─ (.*?) ─*╮╭─ (.*?) ─*╮$`)

func paneTitlesOf(m form.Model) (tree, details string) {
	found := paneTitles.FindStringSubmatch(linesOf(m)[1])
	return found[1], found[2]
}

func treePaneWidthOf(m form.Model) int {
	top := linesOf(m)[1]
	return len([]rune(top[:strings.Index(top, "╮")+len("╮")]))
}

var scrollNote = regexp.MustCompile(`─ (\d+%) ─╯$`)

func scrollNoteOf(m form.Model) string {
	lines := linesOf(m)
	if found := scrollNote.FindStringSubmatch(lines[len(lines)-2]); found != nil {
		return found[1]
	}
	return ""
}

func withoutEmptyEnd(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func pressed(m form.Model, keys ...string) form.Model {
	pressedModel, _ := pressedAndQuit(m, keys...)
	return pressedModel
}

func pressedAndQuit(m form.Model, keys ...string) (form.Model, bool) {
	quit := false
	for _, name := range keys {
		next, cmd := m.Update(keyMsg(name))
		m = next.(form.Model)
		for _, msg := range messagesOf(cmd) {
			if _, ok := msg.(tea.QuitMsg); ok {
				quit = true
				continue
			}
			next, _ = m.Update(msg)
			m = next.(form.Model)
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
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
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
