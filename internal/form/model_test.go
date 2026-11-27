//go:build unit

package form_test

import (
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/form"
	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/muesli/termenv"
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
	withWhen := func() fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"features/auth/feature.yml":         &fstest.MapFile{Data: []byte("summary: Auth.\ntargets: [service]\noptions:\n  method:\n    values: [api-key, oauth]\n    default: api-key\n")},
			"features/client/feature.yml":       &fstest.MapFile{Data: []byte("summary: Client.\ntargets: [service]\nneeds: [auth]\noptions:\n  store-key:\n    values: [\"true\", \"false\"]\n    default: \"false\"\n    when: auth.method == \"api-key\"\n")},
		}
	}
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
	withCycle := func() fstest.MapFS {
		files := withNeeds()
		files["features/beta/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Feature beta.\ntargets: [service]\nneeds: [gamma]\n")}
		files["features/gamma/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Feature gamma.\ntargets: [service]\nneeds: [beta]\n")}
		return files
	}
	withSameExtensionPoint := func() fstest.MapFS {
		files := withNeeds()
		files["features/p/feature.yml"] = &fstest.MapFile{Data: []byte("summary: P.\ntargets: [service]\nextension_points:\n  shared.point:\n    summary: A point.\n    file: go/main.go\n    item: [code]\n")}
		files["features/q/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Q.\ntargets: [service]\nextension_points:\n  shared.point:\n    summary: A point.\n    file: go/main.go\n    item: [code]\n")}
		files["features/r/feature.yml"] = &fstest.MapFile{Data: []byte("summary: R.\ntargets: [service]\nneeds: [missing]\n")}
		return files
	}
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
	withSummaries := func(firstSummary, secondSummary string) fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":                    &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go":               &fstest.MapFile{Data: []byte("package main\n")},
			"features/a-feature-with-a-long-name/feature.yml": &fstest.MapFile{Data: []byte("summary: " + firstSummary + "\ntargets: [service]\n")},
			"features/b/feature.yml":                          &fstest.MapFile{Data: []byte("summary: " + secondSummary + "\ntargets: [service]\n")},
		}
	}
	noStarter := func() fstest.MapFS {
		return fstest.MapFS{"starters": &fstest.MapFile{Mode: fs.ModeDir | 0o755}}
	}
	withAlpha := func() map[string]map[string]string { return map[string]map[string]string{"alpha": {}} }
	openedWith := func(t *testing.T, files fstest.MapFS, starterName, name string, given map[string]string, givenByFeature map[string]map[string]string) form.Model {
		t.Helper()
		m, err := form.New(catalog.New(files), starterName, name, given, givenByFeature)
		require.NoError(t, err)
		return sized(m, 100, 40)
	}
	opened := func(t *testing.T, files fstest.MapFS, starterName, name string, givenByFeature map[string]map[string]string) form.Model {
		t.Helper()
		return openedWith(t, files, starterName, name, nil, givenByFeature)
	}
	twoCatalogs := func() catalog.Catalogs {
		return catalog.NewCatalogs(
			catalog.FromOrigin(fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/alpha/feature.yml":        &fstest.MapFile{Data: []byte("summary: Feature alpha.\ntargets: [service]\nneeds: [zed]\n")},
				"features/zed/feature.yml":          &fstest.MapFile{Data: []byte("summary: Feature zed.\ntargets: [service]\n")},
			}, catalog.Origin{Name: "hpcsc"}),
			catalog.FromOrigin(fstest.MapFS{
				"starters/worker/starter.yml":      &fstest.MapFile{Data: []byte("summary: A worker.\nfolders:\n  go:\n    path: .\n")},
				"starters/worker/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/alpha/feature.yml":       &fstest.MapFile{Data: []byte("summary: The alpha of the team.\ntargets: [worker]\n")},
			}, catalog.Origin{Name: "team"}),
		)
	}
	openedFrom := func(t *testing.T, catalogs catalog.Catalogs, starterName string, givenByFeature map[string]map[string]string) form.Model {
		t.Helper()
		m, err := form.New(catalogs, starterName, "my-service", nil, givenByFeature)
		require.NoError(t, err)
		return sized(m, 100, 40)
	}
	const writeButton = "Write the repo"

	t.Run("layout", func(t *testing.T) {
		t.Run("the screen holds a title line with the count of the selected features, a panel for each section, the button, and the keys of the field with the focus", func(t *testing.T) {
			require.Equal(t, []string{
				" feature-compose new" + strings.Repeat(" ", 49) + "2 features",
				"╭─ Repo " + strings.Repeat("─", 71) + "╮",
				inPanel(80, "  name     my-service  from the command line"),
				inPanel(80, "  starter  service  from the command line"),
				"╰" + strings.Repeat("─", 78) + "╯",
				"╭─ Features " + strings.Repeat("─", 58) + " 2 of 4 ─╮",
				inPanel(80, "▌ [ ] beta   Feature beta."),
				inPanel(80, "  [x] zed    needed by alpha"),
				inPanel(80, "  [x] alpha  from the command line"),
				inPanel(80, "  [ ] gamma  Feature gamma."),
				"╰" + strings.Repeat("─", 78) + "╯",
				strings.Repeat(" ", 63) + "Write the repo",
				"", "", "", "", "", "", "",
				" j/k move • space select • tab next • ctrl+c quit",
			}, linesOf(sized(opened(t, withNeeds(), "service", "my-service", withAlpha()), 80, 20)))
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

		t.Run("a terminal of 40 columns and 8 rows shows the form, and a row wider than its panel ends with …", func(t *testing.T) {
			m := sized(opened(t, withNeeds(), "service", "my-service", nil), 40, 8)

			require.Equal(t, []any{"  name     my-service  from the com…", 8}, []any{lineOf(m, "name"), len(linesOf(m))})
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

		for width, footer := range map[int]string{
			40: " j/k move • space select • tab next …",
			48: " j/k move • space select • tab next …",
			49: " j/k move • space select • tab next • ctrl+c quit",
		} {
			t.Run(fmt.Sprintf("the footer shows each key that fits in %d columns", width), func(t *testing.T) {
				require.Equal(t, footer, footerOf(sized(opened(t, withNeeds(), "service", "my-service", nil), width, 10)))
			})
		}

		for name, test := range map[string]struct {
			m      func(t *testing.T) form.Model
			footer string
		}{
			"a text": {func(t *testing.T) form.Model { return opened(t, withNeeds(), "service", "", nil) }, " type to edit • ↑/↓ move • enter next • ctrl+c quit"},
			"a list of values": {func(t *testing.T) form.Model { return opened(t, withNeeds(), "", "my-service", nil) },
				" j/k move • enter choose • tab next • ctrl+c quit"},
			"the features": {func(t *testing.T) form.Model { return opened(t, withNeeds(), "service", "my-service", nil) }, " j/k move • space select • tab next • ctrl+c quit"},
			"the button":   {func(t *testing.T) form.Model { return opened(t, manyFeatures(0), "service", "my-service", nil) }, " enter write • ↑/↓ move • ctrl+c quit"},
		} {
			t.Run("the footer shows the keys of "+name+" while it has the focus", func(t *testing.T) {
				require.Equal(t, test.footer, footerOf(test.m(t)))
			})
		}

		t.Run("a form taller than the screen follows the cursor down", func(t *testing.T) {
			m := pressed(sized(opened(t, manyFeatures(12), "service", "my-service", nil), 90, 10), slices.Repeat([]string{"down"}, 7)...)

			require.Equal(t, []string{
				"  [ ] feature-00  A feature.",
				"  [ ] feature-01  A feature.",
				"  [ ] feature-02  A feature.",
				"  [ ] feature-03  A feature.",
				"  [ ] feature-04  A feature.",
				"  [ ] feature-05  A feature.",
				"  [ ] feature-06  A feature.",
				"▌ [ ] feature-07  A feature.",
			}, rowsOf(m))
		})

		t.Run("a form scrolled down shows the cursor at its top when the cursor moves up past it", func(t *testing.T) {
			keys := append(slices.Repeat([]string{"down"}, 9), slices.Repeat([]string{"up"}, 8)...)

			require.Equal(t, "▌ [ ] feature-01  A feature.", rowsOf(pressed(sized(opened(t, manyFeatures(12), "service", "my-service", nil), 90, 10), keys...))[0])
		})

		t.Run("a form scrolled down shows its top again when the cursor comes back to the first row of the first field", func(t *testing.T) {
			keys := append(slices.Repeat([]string{"down"}, 9), slices.Repeat([]string{"up"}, 9)...)

			require.Equal(t, "  name     my-service  from the command line", rowsOf(pressed(sized(opened(t, manyFeatures(12), "service", "my-service", nil), 90, 10), keys...))[1])
		})
	})

	t.Run("show", func(t *testing.T) {
		t.Run("with no argument, the form shows only the panel Repo, where the name and the starter are missing, and the button", func(t *testing.T) {
			require.Equal(t, []string{
				"╭─ Repo " + strings.Repeat("─", 91) + "╮",
				"▌ name        missing",
				"  starter    missing",
				"╰" + strings.Repeat("─", 98) + "╯",
				strings.Repeat(" ", 83) + "Write the repo",
			}, rowsOf(opened(t, withNeeds(), "", "", nil)))
		})

		t.Run("a starter with no option and no feature shows only the panel Repo and the button with the focus", func(t *testing.T) {
			require.Equal(t, []string{
				"  name     my-service  from the command line",
				"  starter  service  from the command line",
			}, panelOf(opened(t, manyFeatures(0), "service", "my-service", nil), "Repo"))
			require.Equal(t, strings.Repeat(" ", 81)+"▐ Write the repo ▌", focusedOf(opened(t, manyFeatures(0), "service", "my-service", nil)))
		})

		t.Run("a catalog with no starter says so beside the starter", func(t *testing.T) {
			require.Equal(t, "  starter    the catalog has no starter", lineOf(opened(t, noStarter(), "", "", nil), "starter"))
		})

		t.Run("a feature of --feature that a selected feature needs shows both sources", func(t *testing.T) {
			require.Equal(t, "  [x] zed    from the command line, needed by alpha", lineOf(opened(t, withNeeds(), "service", "my-service", map[string]map[string]string{"alpha": {}, "zed": {}}), "[x] zed"))
		})

		t.Run("the options of the starter are in their own panel, in the order of their names, and an option with no default and no value is marked missing", func(t *testing.T) {
			m := at(t, openedWith(t, withOptions(), "service", "my-service", nil, nil), writeButton)

			require.Equal(t, []string{"  module  example.com/my-service", "  region  us", "  zone      missing"}, panelOf(m, "Options of service"))
		})

		t.Run("a selected feature has its options in a panel of its own, and so does each feature that it needs", func(t *testing.T) {
			m := pressed(at(t, openedWith(t, withOptions(), "service", "my-service", nil, nil), "[ ] alpha"), "space")

			require.Equal(t, [][]string{
				{"  [ ] beta   Feature beta.", "  [x] zed    needed by alpha", "▌ [x] alpha  Feature alpha.", "             needs zed", "  [ ] gamma  Feature gamma."},
				{"  level  low"},
				{"  mode     fast", "  timeout  30"},
			}, [][]string{panelOf(m, "Features"), panelOf(m, "Options of zed"), panelOf(m, "Options of alpha")})
		})

		t.Run("an option that --option or --feature gives shows that the command line gives it", func(t *testing.T) {
			m := openedWith(t, withOptions(), "service", "my-service", map[string]string{"region": "eu"}, map[string]map[string]string{"zed": {"level": "high"}})

			require.Equal(t, []string{"  region  eu  from the command line", "  level  high  from the command line"}, []string{lineOf(m, "region"), lineOf(m, "level")})
		})

		t.Run("an option with no default that a flag gives the empty text shows that the command line gives it, and that it is missing", func(t *testing.T) {
			require.Equal(t, "  zone      from the command line, missing", lineOf(openedWith(t, withOptions(), "service", "my-service", map[string]string{"zone": ""}, nil), "zone"))
		})

		t.Run("an option whose value the need of a selected feature sets shows the need, and loses the mark when no selected need sets it", func(t *testing.T) {
			needed := pressed(at(t, pressed(at(t, opened(t, withNeedValue(), "service", "my-service", nil), "[ ] auth"), "space"), "[ ] cli-auth"), "space")
			cleared := pressed(needed, "space")

			require.Equal(t, [][]string{{"  method  api-key  from the need of cli-auth"}, {"  method  api-key"}}, [][]string{panelOf(needed, "Options of auth"), panelOf(cleared, "Options of auth")})
		})

		t.Run("an option whose value a flag gives with the value that a selected need sets shows both sources", func(t *testing.T) {
			m := pressed(at(t, opened(t, withNeedValue(), "service", "my-service", map[string]map[string]string{"auth": {"method": "api-key"}}), "[ ] cli-auth"), "space")

			require.Equal(t, "  method  api-key  from the command line and the need of cli-auth", lineOf(m, "method"))
		})

		t.Run("the default that reads the name has the empty text while the name is missing", func(t *testing.T) {
			require.Equal(t, "  module  example.com/", lineOf(openedWith(t, withOptions(), "service", "", nil, nil), "module"))
		})
	})

	t.Run("move", func(t *testing.T) {
		t.Run("the focus starts at the name when no argument gives it", func(t *testing.T) {
			require.Equal(t, "▌ name        missing", focusedOf(opened(t, withNeeds(), "", "", nil)))
		})

		t.Run("the focus starts at the first field that the command line does not fix", func(t *testing.T) {
			require.Equal(t, "▌ [ ] beta   Feature beta.", focusedOf(opened(t, withNeeds(), "service", "my-service", withAlpha())))
		})

		for keys, focused := range map[string]string{
			"down":          "▌ [x] zed    needed by alpha",
			"j":             "▌ [x] zed    needed by alpha",
			"j k":           "▌ [ ] beta   Feature beta.",
			"j j j j j":     "▌ [ ] gamma  Feature gamma.",
			"down x4":       strings.Repeat(" ", 81) + "▐ Write the repo ▌",
			"down x4 up":    "▌ [ ] gamma  Feature gamma.",
			"tab":           strings.Repeat(" ", 81) + "▐ Write the repo ▌",
			"tab shift+tab": "▌ [ ] gamma  Feature gamma.",
			"tab tab":       strings.Repeat(" ", 81) + "▐ Write the repo ▌",
			"up":            "▌ [ ] beta   Feature beta.",
			"k":             "▌ [ ] beta   Feature beta.",
			"enter":         strings.Repeat(" ", 81) + "▐ Write the repo ▌",
		} {
			t.Run("j and k move the cursor in a list and stop at its ends, down and up also leave the list at its ends, tab and shift+tab move to the next and the previous field: "+keys, func(t *testing.T) {
				require.Equal(t, focused, focusedOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), keysOf(keys)...)))
			})
		}

		t.Run("the focus skips an option that the command line fixes", func(t *testing.T) {
			require.Equal(t, "▌ zone       missing", focusedOf(pressed(openedWith(t, withOptions(), "service", "my-service", map[string]string{"region": "eu"}, nil), "down")))
		})

		t.Run("enter at a text moves the focus to the next field", func(t *testing.T) {
			require.Equal(t, "▌ region  us", focusedOf(pressed(openedWith(t, withOptions(), "service", "my-service", nil, nil), "enter")))
		})

		for name, test := range map[string]struct {
			field string
			lines []string
		}{
			"an option of the starter with a default": {"module", []string{"▌ module  example.com/my-service", "          default example.com/my-service"}},
			"an option of the starter with a pattern": {"region", []string{"▌ region  us", "          default us, pattern ^[a-z]+$"}},
			"an option with no default":               {"zone", []string{"▌ zone       missing", "          pattern ^[a-z]*$"}},
			"a feature with a need":                   {"[ ] alpha", []string{"▌ [ ] alpha  Feature alpha.", "             needs zed"}},
			"an option with values and a default":     {"level", []string{"▌ level  › ● low", "           ○ mid", "           ○ high", "         default low"}},
			"an option with values and no value":      {"color", []string{"▌ color  › ○ red", "           ○ blue"}},
		} {
			t.Run("the field with the focus shows its hint under it: "+name, func(t *testing.T) {
				m := openedWith(t, withOptions(), "service", "my-service", nil, map[string]map[string]string{"zed": {}, "beta": {}})

				require.Equal(t, test.lines, focusedLinesOf(at(t, m, test.field), len(test.lines)))
			})
		}

		t.Run("the feature under the cursor shows its whole summary under its marks", func(t *testing.T) {
			require.Equal(t, []string{"▌ [x] zed    needed by alpha", "             Feature zed."}, focusedLinesOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j"), 2))
		})

		t.Run("the feature under the cursor wraps its whole summary at the column of the summaries, and another feature cuts its summary at the edge", func(t *testing.T) {
			m := sized(opened(t, withSummaries("This feature does one thing and then another thing for the service.", "Feature b does many things for the service and more."), "service", "my-service", nil), 60, 14)

			require.Equal(t, []string{
				"▌ [ ] a-feature-with-a-long-name  This feature does one",
				strings.Repeat(" ", 34) + "thing and then another",
				strings.Repeat(" ", 34) + "thing for the service.",
				"  [ ] b" + strings.Repeat(" ", 27) + "Feature b does many t…",
			}, focusedLinesOf(m, 4))
		})

		t.Run("in a narrow terminal, a list cuts the summaries at the edge, and shows the whole summary of the value under the cursor under the list", func(t *testing.T) {
			m := sized(opened(t, manyStarters(2), "", "my-service", nil), 40, 10)

			require.Equal(t, []string{"▌ starter  › ○ starter-00  Summary …", strings.Repeat(" ", 13) + "○ starter-01  Summary …", strings.Repeat(" ", 15) + "Summary 00."}, focusedLinesOf(m, 3))
		})
	})

	t.Run("select", func(t *testing.T) {
		t.Run("space at a feature that is not selected selects it, and the counts follow", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "space")

			require.Equal(t, "▌ [x] beta   Feature beta.", focusedOf(m))
			require.True(t, strings.HasSuffix(linesOf(m)[0], " 1 feature"), linesOf(m)[0])
			require.Contains(t, screenOf(m), " 1 of 4 ─╮")
		})

		t.Run("space at a feature also selects the features that it needs, each marked needed by", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "j", "j", "space")

			require.Equal(t, []string{"  [ ] beta   Feature beta.", "  [x] zed    needed by alpha", "▌ [x] alpha  Feature alpha.", "             needs zed", "  [ ] gamma  Feature gamma."}, panelOf(m, "Features"))
		})

		t.Run("space at a feature that the developer selected clears it and each feature that only its needs selected", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "j", "j", "space", "space")

			require.Equal(t, []string{"  [ ] beta   Feature beta.", "  [ ] zed    Feature zed.", "▌ [ ] alpha  Feature alpha.", "             needs zed", "  [ ] gamma  Feature gamma."}, panelOf(m, "Features"))
		})

		t.Run("a clear keeps a need that the developer selected before", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", nil), "j", "space", "j", "space", "space", "k")

			require.Equal(t, []string{"  [ ] beta   Feature beta.", "▌ [x] zed    Feature zed.", "  [ ] alpha  Feature alpha.", "  [ ] gamma  Feature gamma."}, panelOf(m, "Features"))
		})

		for name, keys := range map[string][]string{
			"selected only by the need":      {"j", "j", "space", "k", "space"},
			"also selected by the developer": {"j", "space", "j", "space", "k", "space"},
		} {
			t.Run("space at a feature that a selected feature needs changes nothing, and the message under it says which features need it: "+name, func(t *testing.T) {
				m := pressed(opened(t, withNeeds(), "service", "my-service", nil), keys...)

				require.Equal(t, []string{"▌ [x] zed    needed by alpha", "             Feature zed.", "  zed stays selected: it is needed by alpha."}, focusedLinesOf(m, 3))
			})
		}

		t.Run("space at a feature of --feature changes nothing, and the message says that --feature selects it", func(t *testing.T) {
			m := pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "space")

			require.Equal(t, []string{"▌ [x] alpha  from the command line", "             Feature alpha.", "             needs zed", "  alpha stays selected: --feature selects it."}, focusedLinesOf(m, 4))
		})

		t.Run("the developer can select more features next to a feature of --feature", func(t *testing.T) {
			require.Equal(t, "▌ [x] gamma  Feature gamma.", focusedOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "j", "space")))
		})

		for name, test := range map[string]struct {
			selected []string
			feature  string
			message  string
		}{
			"two features that declare one extension point": {[]string{"[ ] p"}, "[ ] q", `both declare the extension point "shared.point"`},
			"a need that does not target the starter":       {nil, "[ ] r", `the feature r needs "missing", which is not a feature that targets the starter service`},
		} {
			t.Run("space at a feature whose selection Select refuses leaves it as it was, and the message is the message of Select: "+name, func(t *testing.T) {
				m := opened(t, withSameExtensionPoint(), "service", "my-service", nil)
				for _, feature := range test.selected {
					m = pressed(at(t, m, feature), "space")
				}

				m = pressed(at(t, m, test.feature), "space")

				require.Contains(t, focusedOf(m), test.feature)
				require.Contains(t, wordsOf(m), test.message)
			})
		}

		t.Run("space at a feature whose need sets another value than a flag leaves it as it was, and the message names both sources", func(t *testing.T) {
			m := pressed(at(t, sized(opened(t, withNeedValue(), "service", "my-service", map[string]map[string]string{"auth": {"method": "oauth"}}), 160, 40), "[ ] cli-auth"), "space")

			require.Contains(t, focusedOf(m), "[ ] cli-auth")
			require.Contains(t, wordsOf(m), `the option method of the feature auth has two values: "oauth" from the command line and "api-key" from the need of cli-auth`)
		})

		t.Run("the message of a space goes away at the next key", func(t *testing.T) {
			require.NotContains(t, wordsOf(pressed(opened(t, withNeeds(), "service", "my-service", withAlpha()), "j", "j", "space", "k", "j")), "stays selected")
		})

		t.Run("a space that clears a feature removes the panels of its options and of the options of its needs", func(t *testing.T) {
			atAlpha := at(t, openedWith(t, withOptions(), "service", "my-service", nil, nil), "[ ] alpha")

			require.Equal(t, screenOf(atAlpha), screenOf(pressed(atAlpha, "space", "space")))
		})

		t.Run("each option is in the form one time, also when two selected features need its feature", func(t *testing.T) {
			m := pressed(at(t, pressed(at(t, openedWith(t, withOptions(), "service", "my-service", nil, nil), "[ ] alpha"), "space"), "[ ] gamma"), "space")

			require.Equal(t, 1, strings.Count(screenOf(m), "level"))
		})
	})

	t.Run("type", func(t *testing.T) {
		unnamed := func(t *testing.T) form.Model {
			t.Helper()
			return openedWith(t, withOptions(), "service", "", nil, nil)
		}

		t.Run("each key goes to the text with the focus, also q, j, k, h, l, w and space, and none of them stops the form", func(t *testing.T) {
			m, quit := pressedAndQuit(unnamed(t), "m", "y", "q", "j", "k", "h", "l", "w", "space", "x")

			require.False(t, quit)
			require.Equal(t, "▌ name     myqjkhlw x", focusedOf(m))
		})

		t.Run("the form takes a text that passes its check at each key, and the default that reads the name follows it", func(t *testing.T) {
			m := pressed(unnamed(t), "m", "y")

			require.Equal(t, []string{"▌ name     my", "  module  example.com/my"}, []string{focusedOf(m), lineOf(m, "module")})
		})

		t.Run("a text that fails its check stays in the field with the message of the check under it, and the form keeps the value before it", func(t *testing.T) {
			m := pressed(unnamed(t), "-", "x")

			require.Equal(t, []string{"▌ name     -x", "  module  example.com/"}, []string{focusedOf(m), lineOf(m, "module")})
			require.Contains(t, wordsOf(m), `the name "-x" of the repo must start with a letter or a digit`)
		})

		t.Run("a text that fails its check keeps its message when the focus moves away", func(t *testing.T) {
			m := pressed(unnamed(t), "-", "x", "down")

			require.Equal(t, "  name     -x", lineOf(m, "name"))
			require.Contains(t, wordsOf(m), `the name "-x" of the repo must start with a letter or a digit`)
		})

		t.Run("an option without values takes the typed text when it passes its check, and the message of the check goes away", func(t *testing.T) {
			refused := pressed(at(t, sized(unnamed(t), 140, 40), "region"), "backspace", "backspace", "E", "U")
			kept := pressed(refused, "backspace", "backspace", "e", "u", "down")

			require.Contains(t, wordsOf(refused), `the value "EU" of the option region does not match its pattern ^[a-z]+$`)
			require.Equal(t, "  region  eu", lineOf(kept, "region"))
			require.NotContains(t, wordsOf(kept), "does not match")
		})

		t.Run("an empty text of an option with no default shows that the option needs a value", func(t *testing.T) {
			require.Contains(t, wordsOf(pressed(at(t, unnamed(t), "zone"), "a", "backspace")), "the option zone of the starter service has no default, so it needs a value")
		})

		t.Run("an option with no default loses the mark missing once it has a text", func(t *testing.T) {
			require.Equal(t, "▌ zone    abc", focusedOf(pressed(at(t, unnamed(t), "zone"), "a", "b", "c")))
		})

		t.Run("the default that reads the name follows the name until the developer types a value for the option", func(t *testing.T) {
			named := pressed(unnamed(t), typed("my-service")...)
			renamed := pressed(named, append(slices.Repeat([]string{"backspace"}, 10), typed("other")...)...)
			moduleTyped := at(t, pressed(at(t, renamed, "module"), "x", "backspace"), "name")
			renamedAgain := pressed(moduleTyped, append(slices.Repeat([]string{"backspace"}, 5), typed("third")...)...)

			require.Equal(t, []string{"  module  example.com/my-service", "  module  example.com/other", "  module  example.com/other"}, []string{lineOf(named, "module"), lineOf(renamed, "module"), lineOf(renamedAgain, "module")})
		})

		t.Run("with no starter yet, the form takes the typed name, also when flags select features", func(t *testing.T) {
			require.Equal(t, "▌ name     my", focusedOf(pressed(opened(t, withNeeds(), "", "", withAlpha()), "m", "y")))
		})

		t.Run("a text that gets the focus again starts with its text, and the typed keys go after it", func(t *testing.T) {
			require.Equal(t, "▌ name     myx", focusedOf(pressed(unnamed(t), "m", "y", "down", "up", "x")))
		})

		t.Run("left and right move the cursor in the text", func(t *testing.T) {
			require.Equal(t, "▌ name     mxy", focusedOf(pressed(unnamed(t), "m", "y", "left", "x")))
		})

		t.Run("a typed value has the source answered in the preview, also when it equals the default", func(t *testing.T) {
			m := pressed(at(t, pressed(at(t, openedWith(t, withOptions(), "service", "my-service", map[string]string{"zone": "abc"}, nil), "region"), "backspace", "s"), writeButton), "enter")

			require.Contains(t, wordsOf(m), "starter.region: us (answered)")
		})
	})

	t.Run("choose", func(t *testing.T) {
		withZed := func(t *testing.T) form.Model {
			t.Helper()
			return openedWith(t, withOptions(), "service", "my-service", map[string]string{"zone": "abc"}, map[string]map[string]string{"zed": {}})
		}

		for keys, rows := range map[string][]string{
			"":      {"▌ level  › ● low", "           ○ mid", "           ○ high", "         default low"},
			"j":     {"  level    ● low", "▌        › ○ mid", "           ○ high", "         default low"},
			"j j j": {"  level    ● low", "           ○ mid", "▌        › ○ high", "         default low"},
			"j k":   {"▌ level  › ● low", "           ○ mid", "           ○ high", "         default low"},
			"k":     {"▌ level  › ● low", "           ○ mid", "           ○ high", "         default low"},
		} {
			t.Run("a list of values opens with the cursor at the chosen value, and j and k move the cursor in the order of the file and stop at the ends: "+keys, func(t *testing.T) {
				require.Equal(t, rows, panelOf(pressed(at(t, withZed(t), "level"), keysOf(keys)...), "Options of zed"))
			})
		}

		t.Run("space chooses the value under the cursor, and the list stays open", func(t *testing.T) {
			require.Equal(t, []string{"  level    ○ low", "▌        › ● mid", "           ○ high", "         default low"}, panelOf(pressed(at(t, withZed(t), "level"), "j", "space"), "Options of zed"))
		})

		t.Run("enter chooses the value under the cursor, and moves the focus to the next field, where the list shows only the chosen value", func(t *testing.T) {
			m := pressed(at(t, withZed(t), "level"), "j", "enter")

			require.Equal(t, []string{"  level  mid"}, panelOf(m, "Options of zed"))
			require.Equal(t, strings.Repeat(" ", 81)+"▐ Write the repo ▌", focusedOf(m))
		})

		t.Run("down at the last value leaves the list without a choice, and up from the next field opens the list at the chosen value", func(t *testing.T) {
			left := pressed(at(t, withZed(t), "level"), "down", "down", "down")

			require.Equal(t, []string{"  level  low"}, panelOf(left, "Options of zed"))
			require.Equal(t, "▌ level  › ● low", focusedOf(pressed(left, "up")))
		})

		t.Run("up at the first value moves the focus to the field before, with the cursor at its last row", func(t *testing.T) {
			require.Equal(t, "▌ [ ] gamma  Feature gamma.", focusedOf(pressed(at(t, withZed(t), "level"), "up")))
		})

		t.Run("a chosen value has the source answered in the preview", func(t *testing.T) {
			require.Contains(t, wordsOf(pressed(at(t, withZed(t), "level"), "j", "enter", "enter")), "zed.level: mid (answered)")
		})

		t.Run("an option with values and no value is marked missing, its list has no value chosen, and space chooses the value under the cursor", func(t *testing.T) {
			selected := pressed(at(t, withZed(t), "[ ] beta"), "space")
			listed := at(t, selected, "color")

			require.Equal(t, "  color    missing", lineOf(selected, "color"))
			require.Equal(t, []string{"▌ color  › ○ red", "           ○ blue"}, panelOf(listed, "Options of beta"))
			require.Equal(t, []string{"▌ color  › ● red", "           ○ blue"}, panelOf(pressed(listed, "space"), "Options of beta"))
		})

		t.Run("an option with one value has that value chosen", func(t *testing.T) {
			require.Equal(t, []string{"  tone  soft"}, panelOf(openedWith(t, withOptions(), "service", "my-service", nil, map[string]map[string]string{"gamma": {}}), "Options of gamma"))
		})

		t.Run("the form shows an option only while its when is true, and the option has its chosen value again when its when is true again", func(t *testing.T) {
			chosen := pressed(at(t, opened(t, withWhen(), "service", "my-service", map[string]map[string]string{"client": {}}), "● false"), "k", "space")
			withOAuth := pressed(at(t, chosen, "method"), "j", "space")
			withAPIKey := pressed(withOAuth, "k", "space")

			require.Equal(t, []string{"  store-key  true"}, panelOf(withAPIKey, "Options of client"))
			require.NotContains(t, screenOf(withOAuth), "store-key")
			require.Contains(t, wordsOf(pressed(at(t, withOAuth, writeButton), "enter")), "client.store-key: false (default)")
		})

		t.Run("a choice that makes the when of an option false while a flag gives the option another value than its default leaves the value as it was, and the message says why", func(t *testing.T) {
			m := pressed(at(t, opened(t, withWhen(), "service", "my-service", map[string]map[string]string{"client": {"store-key": "true"}}), "method"), "j", "space")

			require.Equal(t, "  method    ● api-key", lineOf(m, "method"))
			require.Contains(t, wordsOf(m), `the option store-key of the feature client applies only when auth.method == "api-key", and auth.method is "oauth", but the command line gives it "true"`)
		})

		t.Run("a chosen value gives way to the value of a need, and comes back when no selected need sets it", func(t *testing.T) {
			chosen := pressed(at(t, pressed(at(t, opened(t, withNeedValue(), "service", "my-service", nil), "[ ] auth"), "space"), "method"), "j", "space")
			needed := pressed(at(t, chosen, "[ ] cli-auth"), "space")
			cleared := pressed(needed, "space")

			require.Equal(t, [][]string{{"  method  api-key  from the need of cli-auth"}, {"  method  oauth"}}, [][]string{panelOf(needed, "Options of auth"), panelOf(cleared, "Options of auth")})
		})

		t.Run("a feature that a space clears loses its values, also a need that the clear removes", func(t *testing.T) {
			selected := pressed(at(t, openedWith(t, withOptions(), "service", "my-service", map[string]string{"zone": "abc"}, nil), "[ ] alpha"), "space")
			changed := at(t, pressed(at(t, pressed(at(t, selected, "level"), "j", "space"), "mode"), append(slices.Repeat([]string{"backspace"}, 4), typed("slow")...)...), "[x] alpha")
			again := pressed(changed, "space", "space")

			require.Equal(t, [][]string{{"  level  mid"}, {"  mode     slow", "  timeout  30"}}, [][]string{panelOf(changed, "Options of zed"), panelOf(changed, "Options of alpha")})
			require.Equal(t, [][]string{{"  level  low"}, {"  mode     fast", "  timeout  30"}}, [][]string{panelOf(again, "Options of zed"), panelOf(again, "Options of alpha")})
		})

		t.Run("a space at another feature keeps the values of the selected features", func(t *testing.T) {
			chosen := pressed(at(t, pressed(at(t, withZed(t), "[ ] beta"), "space"), "color"), "space")

			require.Equal(t, []string{"  color  red"}, panelOf(pressed(at(t, chosen, "[ ] gamma"), "space"), "Options of beta"))
		})
	})

	t.Run("starter", func(t *testing.T) {
		unchosen := func(t *testing.T, given map[string]string, givenByFeature map[string]map[string]string) form.Model {
			t.Helper()
			return openedWith(t, withStarters(), "", "my-service", given, givenByFeature)
		}

		t.Run("the starters come in the order of their names with their summaries, the cursor starts at the first, and no other panel shows", func(t *testing.T) {
			m := unchosen(t, nil, nil)

			require.Equal(t, []string{
				"  name     my-service  from the command line",
				"▌ starter  › ○ broken   Broken.",
				"             ○ service  A service.",
				"             ○ worker   A worker.",
			}, panelOf(m, "Repo"))
			require.NotContains(t, screenOf(m), "Features")
		})

		t.Run("j moves the cursor to the next starter and chooses nothing", func(t *testing.T) {
			require.Equal(t, []string{
				"  name     my-service  from the command line",
				"  starter    ○ broken   Broken.",
				"▌          › ○ service  A service.",
				"             ○ worker   A worker.",
			}, panelOf(pressed(unchosen(t, nil, nil), "j"), "Repo"))
		})

		t.Run("space chooses the starter under the cursor, and the form shows its options and its features", func(t *testing.T) {
			m := pressed(unchosen(t, nil, nil), "j", "space")

			require.Equal(t, "▌          › ● service  A service.", focusedOf(m))
			require.Equal(t, [][]string{{"  region  us"}, {"  [ ] only    Only.", "  [ ] shared  Shared."}}, [][]string{panelOf(m, "Options of service"), panelOf(m, "Features")})
		})

		t.Run("enter chooses the starter under the cursor, and moves the focus to the first field after it", func(t *testing.T) {
			m := pressed(unchosen(t, nil, nil), "j", "enter")

			require.Equal(t, []string{"  name     my-service  from the command line", "  starter  service  A service."}, panelOf(m, "Repo"))
			require.Equal(t, []string{"▌ region  us", "          default us"}, focusedLinesOf(m, 2))
		})

		t.Run("a change of the starter keeps each selected feature that targets the new starter with its values, and clears the other features and the values of the old starter", func(t *testing.T) {
			m := pressed(unchosen(t, nil, nil), "j", "enter", "backspace", "backspace", "e", "u")
			m = pressed(at(t, pressed(at(t, m, "[ ] only"), "space"), "mode"), append(slices.Repeat([]string{"backspace"}, 4), typed("slow")...)...)
			m = pressed(at(t, pressed(at(t, m, "[ ] shared"), "space"), "tone"), append(slices.Repeat([]string{"backspace"}, 4), typed("loud")...)...)
			changed := pressed(at(t, m, "○ worker"), "space")

			require.Equal(t, [][]string{{"  region  eu"}, {"  [x] only    Only.", "  [x] shared  Shared."}, {"  mode  slow"}, {"▌ tone  loud", "        default soft"}},
				[][]string{panelOf(m, "Options of service"), panelOf(m, "Features"), panelOf(m, "Options of only"), panelOf(m, "Options of shared")})
			require.Equal(t, [][]string{{"  queue  jobs"}, {"  [x] shared  Shared."}, {"  tone  loud"}},
				[][]string{panelOf(changed, "Options of worker"), panelOf(changed, "Features"), panelOf(changed, "Options of shared")})
		})

		t.Run("a space at the chosen starter keeps the values of its options", func(t *testing.T) {
			m := pressed(unchosen(t, nil, nil), "j", "enter", "backspace", "backspace", "e", "u")

			require.Equal(t, "  region  eu", lineOf(pressed(at(t, m, "● service"), "space"), "region"))
		})

		for name, test := range map[string]struct {
			given          map[string]string
			givenByFeature map[string]map[string]string
			starter        string
			message        string
		}{
			"a feature of --feature that does not target it": {nil, map[string]map[string]string{"only": {}}, "○ worker", `the feature only does not target the starter worker: the features that target the starter worker are shared`},
			"an option of --option that it does not declare": {map[string]string{"region": "eu"}, nil, "○ worker", `unknown option "region": the options of the starter worker are queue`},
			"features whose needs form a cycle":              {nil, nil, "○ broken", "the needs of the features form a cycle: x needs y needs x"},
		} {
			t.Run("a starter that does not fit is not chosen, and the message says why: "+name, func(t *testing.T) {
				m := pressed(at(t, unchosen(t, test.given, test.givenByFeature), test.starter), "space")

				require.Contains(t, focusedOf(m), test.starter)
				require.Contains(t, wordsOf(m), test.message)
			})
		}

		t.Run("the message of a starter that does not fit goes away at the next key", func(t *testing.T) {
			require.NotContains(t, wordsOf(pressed(unchosen(t, nil, nil), "space", "j")), "cycle")
		})

		t.Run("a change of the starter drops the values of a feature that only a cleared feature needed", func(t *testing.T) {
			files := fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"starters/worker/starter.yml":       &fstest.MapFile{Data: []byte("summary: A worker.\nfolders:\n  go:\n    path: .\n")},
				"starters/worker/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
				"features/p/feature.yml":            &fstest.MapFile{Data: []byte("summary: P.\ntargets: [service]\nneeds: [q]\n")},
				"features/q/feature.yml":            &fstest.MapFile{Data: []byte("summary: Q.\ntargets: [service, worker]\noptions:\n  level:\n    values: [low, high]\n    default: low\n")},
			}
			m := pressed(at(t, opened(t, files, "", "my-service", nil), "○ service"), "space")
			m = pressed(at(t, pressed(at(t, m, "[ ] p"), "space"), "level"), "j", "space")
			changed := pressed(at(t, m, "○ worker"), "space")

			require.Equal(t, "▌        › ● high", focusedOf(m))
			require.Equal(t, []string{"  [ ] q  Q."}, panelOf(changed, "Features"))
			require.Equal(t, []string{"  level  low"}, panelOf(pressed(at(t, changed, "[ ] q"), "space"), "Options of q"))
		})

		t.Run("a list of starters taller than the screen follows the cursor", func(t *testing.T) {
			m := pressed(sized(opened(t, manyStarters(12), "", "my-service", nil), 90, 10), slices.Repeat([]string{"j"}, 8)...)

			require.Equal(t, []string{
				"             ○ starter-01  Summary 01.",
				"             ○ starter-02  Summary 02.",
				"             ○ starter-03  Summary 03.",
				"             ○ starter-04  Summary 04.",
				"             ○ starter-05  Summary 05.",
				"             ○ starter-06  Summary 06.",
				"             ○ starter-07  Summary 07.",
				"▌          › ○ starter-08  Summary 08.",
			}, rowsOf(m))
		})

		t.Run("the starters come with their full names", func(t *testing.T) {
			require.Equal(t, []string{
				"  name     my-service  from the command line",
				"▌ starter  › ○ hpcsc/service  A service.",
				"             ○ team/worker    A worker.",
			}, panelOf(openedFrom(t, twoCatalogs(), "", nil), "Repo"))
		})

		t.Run("a short name from the command line shows the full name of the starter, and the features of its catalog with their summaries and needs", func(t *testing.T) {
			m := at(t, openedFrom(t, twoCatalogs(), "service", map[string]map[string]string{"alpha": {}}), "[x] alpha")

			require.Equal(t, [][]string{
				{"  name     my-service  from the command line", "  starter  hpcsc/service  from the command line"},
				{"  [x] zed    needed by alpha", "▌ [x] alpha  from the command line", "             Feature alpha.", "             needs zed"},
			}, [][]string{panelOf(m, "Repo"), panelOf(m, "Features")})
		})

		t.Run("a change to a starter of another catalog clears each feature of the old catalog", func(t *testing.T) {
			m := pressed(openedFrom(t, twoCatalogs(), "", nil), "enter")
			m = pressed(at(t, m, "[ ] alpha"), "space")
			changed := pressed(at(t, m, "○ team/worker"), "space")

			require.Equal(t, []string{"  [ ] alpha  The alpha of the team."}, panelOf(changed, "Features"))
		})

		t.Run("a change to another starter of the same catalog keeps each selected feature that also targets it", func(t *testing.T) {
			catalogs := catalog.NewCatalogs(catalog.FromOrigin(fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"starters/worker/starter.yml":       &fstest.MapFile{Data: []byte("summary: A worker.\nfolders:\n  go:\n    path: .\n")},
				"starters/worker/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
				"features/shared/feature.yml":       &fstest.MapFile{Data: []byte("summary: Shared.\ntargets: [service, worker]\n")},
			}, catalog.Origin{Name: "hpcsc"}))
			m := pressed(openedFrom(t, catalogs, "", nil), "enter")
			m = pressed(at(t, m, "[ ] shared"), "space")
			changed := pressed(at(t, m, "○ hpcsc/worker"), "space")

			require.Equal(t, []string{"  [x] shared  Shared."}, panelOf(changed, "Features"))
		})
	})

	t.Run("write", func(t *testing.T) {
		t.Run("enter at the button with no answer missing shows the preview in a panel, with the source of each feature and option, and asks whether to write the repo", func(t *testing.T) {
			m := pressed(at(t, opened(t, withNeeds(), "service", "my-service", withAlpha()), writeButton), "enter")

			require.Equal(t, []string{
				"Write the repo?",
				"",
				"Repo: my-service",
				"Starter: service",
				"",
				"Features:",
				"  zed: needed by alpha",
				"  alpha: command line",
				"",
				"Options: none",
				"",
				"Files:",
				"  .feature-compose.yml: feature-compose",
				"  main.go: the starter",
			}, panelOf(m, "Preview"))
		})

		t.Run("space at the button also shows the preview", func(t *testing.T) {
			require.Equal(t, "Write the repo?", panelOf(pressed(at(t, opened(t, withNeeds(), "service", "my-service", nil), writeButton), "space"), "Preview")[0])
		})

		for _, scroll := range [][2]string{{"down", "up"}, {"j", "k"}, {"ctrl+d", "ctrl+u"}, {"pgdown", "pgup"}} {
			t.Run(scroll[0]+" scrolls the preview down, and "+scroll[1]+" scrolls it back", func(t *testing.T) {
				top := pressed(at(t, sized(opened(t, withManyFiles(), "service", "my-service", nil), 100, 14), writeButton), "enter")
				scrolled := pressed(top, scroll[0])

				require.Equal(t, "Write the repo?", panelOf(top, "Preview")[0])
				require.NotEqual(t, screenOf(top), screenOf(scrolled))
				require.Equal(t, screenOf(top), screenOf(pressed(scrolled, scroll[1])))
			})
		}

		t.Run("the bottom border of a preview taller than its panel shows how far it is scrolled", func(t *testing.T) {
			top := pressed(at(t, sized(opened(t, withManyFiles(), "service", "my-service", nil), 100, 14), writeButton), "enter")

			require.Equal(t, []string{"0%", "100%"}, []string{noteOf(top), noteOf(pressed(top, slices.Repeat([]string{"pgdown"}, 10)...))})
		})

		t.Run("enter at the preview stops the form, which then gives the selection of the preview with the answers of the developer", func(t *testing.T) {
			typedZone := pressed(at(t, openedWith(t, withOptions(), "service", "my-service", nil, nil), "zone"), "a", "b", "c")
			m, quit := pressedAndQuit(at(t, typedZone, writeButton), "enter", "enter")

			selection, err := m.Confirmed()
			require.NoError(t, err)
			require.True(t, quit)
			preview, err := selection.Preview()
			require.NoError(t, err)
			require.Contains(t, preview, "\n  starter.zone: abc (answered)\n")
		})

		t.Run("esc at the preview goes back to the form with the focus at the button", func(t *testing.T) {
			atButton := at(t, opened(t, withNeeds(), "service", "my-service", withAlpha()), writeButton)
			m, quit := pressedAndQuit(atButton, "enter", "esc")

			require.False(t, quit)
			require.Equal(t, screenOf(atButton), screenOf(m))
		})

		for name, keys := range map[string][]string{
			"ctrl+c at the preview": {"enter", "ctrl+c"},
			"esc and then ctrl+c":   {"enter", "esc", "ctrl+c"},
			"ctrl+c at the button":  {"ctrl+c"},
		} {
			t.Run(name+" stops the form with the cancel", func(t *testing.T) {
				m, quit := pressedAndQuit(at(t, opened(t, withNeeds(), "service", "my-service", nil), writeButton), keys...)

				_, err := m.Confirmed()
				require.True(t, quit)
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		t.Run("the form gives the selection only after enter at the preview", func(t *testing.T) {
			m, quit := pressedAndQuit(at(t, opened(t, withNeeds(), "service", "my-service", nil), writeButton), "enter", "down", "up", "space", "y")

			_, err := m.Confirmed()
			require.False(t, quit)
			require.EqualError(t, err, "you canceled new, so it wrote no file")
		})

		t.Run("with no name, enter at the button moves the focus to the name, and the message is the message of the check of the name", func(t *testing.T) {
			m, quit := pressedAndQuit(at(t, opened(t, withNeeds(), "service", "", nil), writeButton), "enter")

			require.False(t, quit)
			require.Equal(t, "▌ name", focusedOf(m))
			require.Contains(t, wordsOf(m), `the name "" of the repo must start with a letter or a digit`)
		})

		t.Run("with more than one missing answer, enter at the button moves the focus to the first in the order of the form", func(t *testing.T) {
			require.Equal(t, "▌ name", focusedOf(pressed(at(t, openedWith(t, withOptions(), "service", "", nil, nil), writeButton), "enter")))
		})

		t.Run("with no starter, enter at the button opens the list of starters, and the message says that no starter is chosen", func(t *testing.T) {
			m := pressed(at(t, opened(t, withNeeds(), "", "my-service", nil), writeButton), "enter")

			require.Equal(t, []string{"▌ starter  › ○ other    Another service.", "             ○ service  A service.", "           No starter is chosen."}, focusedLinesOf(m, 3))
		})

		t.Run("with a catalog with no starter, enter at the button keeps the focus, and the message says that no starter is chosen", func(t *testing.T) {
			m := pressed(at(t, opened(t, noStarter(), "", "my-service", nil), writeButton), "enter")

			require.Equal(t, []string{strings.Repeat(" ", 81) + "▐ Write the repo ▌", "  No starter is chosen."}, focusedLinesOf(m, 2))
		})

		t.Run("with an option that has no default and no value, enter at the button opens its list, and the message is the message of its check", func(t *testing.T) {
			selected := pressed(at(t, openedWith(t, withOptions(), "service", "my-service", map[string]string{"zone": "abc"}, nil), "[ ] beta"), "space")
			m := pressed(at(t, selected, writeButton), "enter")

			require.Equal(t, "▌ color  › ○ red", focusedOf(m))
			require.Contains(t, wordsOf(m), `the feature beta: the value "" of the option color is not one of its values: red, blue`)
		})

		t.Run("with a text that fails its check, enter at the button moves the focus to it", func(t *testing.T) {
			require.Equal(t, "▌ name     -x", focusedOf(pressed(at(t, pressed(opened(t, withNeeds(), "service", "", nil), "-", "x"), writeButton), "enter")))
		})

		t.Run("a preview that stops shows its message with no question, enter does not write, and esc goes back to the form", func(t *testing.T) {
			stopped := pressed(at(t, opened(t, withBadTemplate(), "service", "my-service", nil), writeButton), "enter")
			m, quit := pressedAndQuit(stopped, "enter")

			_, err := m.Confirmed()
			require.False(t, quit)
			require.EqualError(t, err, "you canceled new, so it wrote no file")
			require.Contains(t, wordsOf(stopped), "bad.go")
			require.NotContains(t, wordsOf(stopped), "Write the repo?")
			require.Equal(t, strings.Repeat(" ", 81)+"▐ Write the repo ▌", focusedOf(pressed(stopped, "esc")))
		})

		t.Run("the footer shows the keys of the preview while it shows", func(t *testing.T) {
			require.Equal(t, []string{" enter write • esc back • ↑/↓ scroll • ctrl+c quit", " esc back • ↑/↓ scroll • ctrl+c quit"}, []string{
				footerOf(pressed(at(t, opened(t, withNeeds(), "service", "my-service", nil), writeButton), "enter")),
				footerOf(pressed(at(t, opened(t, withBadTemplate(), "service", "my-service", nil), writeButton), "enter")),
			})
		})
	})

	t.Run("stop", func(t *testing.T) {
		for name, test := range map[string]struct {
			m    func(t *testing.T) form.Model
			keys []string
		}{
			"at a text":           {func(t *testing.T) form.Model { return opened(t, withNeeds(), "service", "", nil) }, []string{"m", "ctrl+c"}},
			"at the features":     {func(t *testing.T) form.Model { return opened(t, withNeeds(), "service", "my-service", nil) }, []string{"ctrl+c"}},
			"at a list of values": {func(t *testing.T) form.Model { return opened(t, withNeeds(), "", "my-service", nil) }, []string{"ctrl+c"}},
		} {
			t.Run("ctrl+c "+name+" stops the form, which then gives the cancel", func(t *testing.T) {
				m, quit := pressedAndQuit(test.m(t), test.keys...)

				_, err := m.Confirmed()
				require.True(t, quit)
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		t.Run("q and esc do not stop the form", func(t *testing.T) {
			_, quit := pressedAndQuit(opened(t, withNeeds(), "service", "my-service", nil), "q", "esc")

			require.False(t, quit)
		})

		t.Run("a message that is not a key or a size changes nothing", func(t *testing.T) {
			m := opened(t, withNeeds(), "service", "my-service", nil)

			next, cmd := m.Update(tea.FocusMsg{})

			require.Nil(t, cmd)
			require.Equal(t, screenOf(m), screenOf(next.(form.Model)))
		})
	})

	t.Run("style", func(t *testing.T) {
		colored := func(t *testing.T) {
			t.Helper()
			profile := lipgloss.ColorProfile()
			lipgloss.SetColorProfile(termenv.TrueColor)
			t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
		}

		t.Run("a selected feature is bold, and a feature that is not selected is faint", func(t *testing.T) {
			colored(t)
			view := opened(t, withNeeds(), "service", "my-service", withAlpha()).View()

			require.Contains(t, view, screen.NameStyle.Render("[x] alpha"))
			require.Contains(t, view, screen.NotSelectedStyle.Render("[ ] gamma"))
		})

		t.Run("the mark missing has the color of a problem, and the other marks are faint", func(t *testing.T) {
			colored(t)

			require.Contains(t, openedWith(t, withOptions(), "service", "my-service", map[string]string{"zone": ""}, nil).View(), screen.MarkStyle.Render("from the command line")+screen.MarkStyle.Render(", ")+screen.ProblemStyle.Render("missing"))
		})

		t.Run("the text with the focus shows its label in the style of the focus, and the label of another field is faint", func(t *testing.T) {
			colored(t)
			view := openedWith(t, withOptions(), "service", "my-service", nil, nil).View()

			require.Contains(t, view, "▌ "+screen.FocusStyle.Render("module  "))
			require.Contains(t, view, screen.LabelStyle.Render("region  ")+"us")
		})

		t.Run("the panel with the focus has the border of the focus, and the other panels have the plain border", func(t *testing.T) {
			colored(t)
			lines := strings.Split(openedWith(t, withOptions(), "service", "my-service", nil, nil).View(), "\n")

			require.Equal(t, []bool{true, true}, []bool{strings.HasPrefix(lines[1], screen.PlainBorder.Render("╭─")), strings.HasPrefix(lines[5], screen.FocusedBorder.Render("╭─"))})
		})

		t.Run("the button is filled with the color of the focus while it has the focus", func(t *testing.T) {
			colored(t)
			m := opened(t, manyFeatures(0), "service", "my-service", nil)

			require.Contains(t, m.View(), screen.FocusButtonStyle.Render(" Write the repo "))
			require.Contains(t, opened(t, manyFeatures(1), "service", "my-service", nil).View(), screen.ButtonStyle.Render(" Write the repo "))
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

func footerOf(m form.Model) string {
	lines := linesOf(m)
	return lines[len(lines)-1]
}

func rowsOf(m form.Model) []string {
	lines := linesOf(m)
	rows := make([]string, 0, len(lines))
	for _, line := range lines[1 : len(lines)-1] {
		if inner, isRow := strings.CutPrefix(line, "│ "); isRow {
			line = strings.TrimRight(strings.TrimSuffix(inner, "│"), " ")
		}
		rows = append(rows, line)
	}
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return rows
}

func panelOf(m form.Model, title string) []string {
	rows := rowsOf(m)
	start := slices.IndexFunc(rows, func(row string) bool { return strings.HasPrefix(row, "╭─ "+title+" ") })
	if start < 0 {
		return nil
	}
	end := start + 1 + slices.IndexFunc(rows[start+1:], func(row string) bool { return strings.HasPrefix(row, "╰") })
	panel := rows[start+1 : end]
	for len(panel) > 0 && panel[len(panel)-1] == "" {
		panel = panel[:len(panel)-1]
	}
	return panel
}

func noteOf(m form.Model) string {
	rows := rowsOf(m)
	if found := regexp.MustCompile(`─ (\d+%) ─╯$`).FindStringSubmatch(rows[len(rows)-1]); found != nil {
		return found[1]
	}
	return ""
}

func inPanel(width int, row string) string {
	return "│ " + row + strings.Repeat(" ", width-4-lipgloss.Width(row)) + " │"
}

func wordsOf(m form.Model) string {
	return strings.Join(strings.Fields(strings.Join(rowsOf(m), " ")), " ")
}

func focusedOf(m form.Model) string {
	return focusedLinesOf(m, 1)[0]
}

func focusedLinesOf(m form.Model, count int) []string {
	rows := rowsOf(m)
	for i, row := range rows {
		if strings.HasPrefix(row, "▌") || strings.Contains(row, "▐ Write the repo ▌") {
			return rows[i:min(i+count, len(rows))]
		}
	}
	return []string{""}
}

func lineOf(m form.Model, text string) string {
	for _, row := range rowsOf(m) {
		if strings.HasPrefix(strings.TrimLeft(strings.TrimPrefix(row, "▌"), " "), text) {
			return row
		}
	}
	return ""
}

func at(t *testing.T, m form.Model, text string) form.Model {
	t.Helper()
	m = pressed(m, slices.Repeat([]string{"up"}, 80)...)
	for range 80 {
		if strings.Contains(" "+strings.Join(strings.Fields(focusedOf(m)), " ")+" ", " "+text+" ") {
			return m
		}
		m = pressed(m, "down")
	}
	require.FailNow(t, "no row gets the focus with the text "+text)
	return m
}

func keysOf(text string) []string {
	var keys []string
	for _, word := range strings.Fields(text) {
		if name, count, ok := strings.Cut(word, "x"); ok && name == "" {
			var times int
			fmt.Sscan(count, &times)
			keys = append(keys, slices.Repeat(keys[len(keys)-1:], times-1)...)
			continue
		}
		keys = append(keys, word)
	}
	return keys
}

func typed(text string) []string {
	keys := make([]string, 0, len(text))
	for _, letter := range text {
		keys = append(keys, string(letter))
	}
	return keys
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
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
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
