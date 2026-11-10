//go:build unit

package prompt_test

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/prompt"
	"github.com/stretchr/testify/require"
)

func TestModel(t *testing.T) {
	withFeatures := func() fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\noptions:\n  region:\n    default: us\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"features/cli/feature.yml":          &fstest.MapFile{Data: []byte("summary: A CLI.\ntargets: [service]\nfiles:\n  - path: files/\n")},
			"features/cli/files/go/cli.go":      &fstest.MapFile{Data: []byte("package cli\n")},
		}
	}
	withManyFiles := func() fstest.MapFS {
		files := fstest.MapFS{"starters/service/starter.yml": &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")}}
		for i := range 30 {
			files[fmt.Sprintf("starters/service/files/go/f%02d.txt", i)] = &fstest.MapFile{Data: []byte("x\n")}
		}
		return files
	}
	withCLI := func() map[string]map[string]string { return map[string]map[string]string{"cli": {}} }
	withRegion := func() map[string]string { return map[string]string{"region": "eu"} }
	withNeeds := func() fstest.MapFS {
		return fstest.MapFS{
			"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\nfolders:\n  go:\n    path: .\n")},
			"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"starters/other/starter.yml":        &fstest.MapFile{Data: []byte("summary: Another service.\nfolders:\n  go:\n    path: .\n")},
			"starters/other/files/go/main.go":   &fstest.MapFile{Data: []byte("package main\n")},
			"starters/lonely/starter.yml":       &fstest.MapFile{Data: []byte("summary: A starter that no feature targets.\nfolders:\n  go:\n    path: .\n")},
			"starters/lonely/files/go/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
			"features/a/feature.yml":            &fstest.MapFile{Data: []byte("summary: Feature A.\ntargets: [service]\nneeds: [b]\n")},
			"features/b/feature.yml":            &fstest.MapFile{Data: []byte("summary: Feature B.\ntargets: [service]\n")},
			"features/c/feature.yml":            &fstest.MapFile{Data: []byte("summary: Feature C.\ntargets: [service]\n")},
			"features/w/feature.yml":            &fstest.MapFile{Data: []byte("summary: Feature W.\ntargets: [service]\nneeds: [b, c]\n")},
			"features/elsewhere/feature.yml":    &fstest.MapFile{Data: []byte("summary: Elsewhere.\ntargets: [other]\n")},
		}
	}
	withBadTemplate := func() fstest.MapFS {
		files := withFeatures()
		files["starters/service/files/go/bad.go"] = &fstest.MapFile{Data: []byte("package {{ .Nmae }}\n")}
		return files
	}
	previewErrOf := func(t *testing.T, files fstest.MapFS) error {
		t.Helper()
		starter, err := catalog.New(files).Starter("service")
		require.NoError(t, err)
		selection, err := starter.Select("my-service", nil, nil, catalog.Answers{})
		require.NoError(t, err)
		_, previewErr := selection.Preview()
		require.Error(t, previewErr)
		return previewErr
	}

	featureStep := func(t *testing.T) prompt.Model {
		t.Helper()
		m, err := prompt.New(catalog.New(withNeeds()), "service", "my-service", nil, nil)
		require.NoError(t, err)
		return started(m, 100, 40)
	}
	confirmStep := func(t *testing.T, files fstest.MapFS, given map[string]string, givenByFeature map[string]map[string]string, height int) prompt.Model {
		t.Helper()
		m, err := prompt.New(catalog.New(files), "service", "my-service", given, givenByFeature)
		require.NoError(t, err)
		return started(m, 100, height)
	}

	t.Run("start", func(t *testing.T) {
		for name, test := range map[string]struct {
			starterName    string
			givenByFeature map[string]map[string]string
			message        string
		}{
			"an unknown starter": {starterName: "nope", message: `unknown starter "nope": the catalog has service`},
			"an unknown feature": {starterName: "service", givenByFeature: map[string]map[string]string{"nope": {}}, message: `unknown feature "nope": the features that target the starter service are cli`},
		} {
			t.Run(name+" stops the TUI before it starts", func(t *testing.T) {
				_, err := prompt.New(catalog.New(withFeatures()), test.starterName, "my-service", nil, test.givenByFeature)

				require.EqualError(t, err, test.message)
			})
		}

		for name, test := range map[string]struct {
			path, contents, message string
		}{
			"a starter.yml of another starter that does not parse": {path: "starters/other/starter.yml", contents: "folders: [", message: "starters/other/starter.yml"},
			"a pattern of the starter that does not compile":       {path: "starters/service/starter.yml", contents: "options:\n  region:\n    pattern: '('\nfolders:\n  go:\n    path: .\n", message: "the pattern of the option region"},
		} {
			t.Run(name+" stops the TUI before it starts, with the message of the catalog", func(t *testing.T) {
				files := withFeatures()
				files[test.path] = &fstest.MapFile{Data: []byte(test.contents)}

				_, err := prompt.New(catalog.New(files), "service", "my-service", nil, withCLI())

				require.ErrorContains(t, err, test.message)
			})
		}

		t.Run("a template that fails to render stops the TUI before it starts when flags answer the steps before the confirm step, with the message of the preview", func(t *testing.T) {
			_, err := prompt.New(catalog.New(withBadTemplate()), "service", "my-service", withRegion(), withCLI())

			require.EqualError(t, err, previewErrOf(t, withBadTemplate()).Error())
		})
	})

	t.Run("confirm", func(t *testing.T) {
		t.Run("shows the preview of the repo, where each value of a flag shows command line, and asks whether to write the repo", func(t *testing.T) {
			m := confirmStep(t, withFeatures(), map[string]string{"region": "eu"}, map[string]map[string]string{"cli": {}}, 40)

			screen := screenOf(m)

			for _, part := range []string{"Repo: my-service", "Starter: service", "  cli: command line", "  starter.region: eu (command line)", "  cli.go: the feature cli", "  main.go: the starter", "┃ Write the repo?", "Yes", "No"} {
				require.Contains(t, screen, part)
			}
		})

		for name, keys := range map[string][2]string{
			"ctrl+d and ctrl+u": {"ctrl+d", "ctrl+u"},
			"pgdown and pgup":   {"pgdown", "pgup"},
		} {
			t.Run(name+" scroll a preview that is longer than the screen down and up", func(t *testing.T) {
				m, err := prompt.New(catalog.New(withManyFiles()), "service", "my-service", nil, nil)
				require.NoError(t, err)
				m = started(m, 100, 20)
				require.Contains(t, screenOf(m), "Repo: my-service")
				require.NotContains(t, screenOf(m), "f29.txt: the starter")

				scrolled := pressed(m, keys[0], keys[0], keys[0], keys[0], keys[0])

				require.Contains(t, screenOf(scrolled), "f29.txt: the starter")
				require.NotContains(t, screenOf(scrolled), "Repo: my-service")
				back := pressed(scrolled, keys[1], keys[1], keys[1], keys[1], keys[1])
				require.Contains(t, screenOf(back), "Repo: my-service")
			})
		}

		t.Run("shows the percent only while a line of the preview is out of the screen", func(t *testing.T) {
			fits := 0
			for height := 10; height < 40 && fits == 0; height++ {
				screen := screenOf(confirmStep(t, withFeatures(), withRegion(), withCLI(), height))
				if strings.Contains(screen, "Repo: my-service") && strings.Contains(screen, "  main.go: the starter") {
					fits = height
				}
			}
			require.NotZero(t, fits)

			require.NotContains(t, lastLineOf(confirmStep(t, withFeatures(), withRegion(), withCLI(), fits)), "%")
			require.Contains(t, lastLineOf(confirmStep(t, withFeatures(), withRegion(), withCLI(), fits-1)), "0% ·")
		})

		t.Run("fills the rows of the terminal, and cuts each line of the preview at its width", func(t *testing.T) {
			files := withManyFiles()
			files["starters/service/files/go/a-"+strings.Repeat("long-name-", 12)+".txt"] = &fstest.MapFile{Data: []byte("x\n")}
			m, err := prompt.New(catalog.New(files), "service", "my-service", nil, nil)
			require.NoError(t, err)
			m = started(m, 60, 20)

			lines := strings.Split(screenOf(m), "\n")

			require.Len(t, lines, 20)
			require.Contains(t, screenOf(m), "a-long-name-long-name-")
			for _, line := range lines {
				require.LessOrEqual(t, lipgloss.Width(line), 60, line)
			}
		})

		t.Run("shows nothing before it knows the size of the terminal", func(t *testing.T) {
			m, err := prompt.New(catalog.New(withFeatures()), "service", "my-service", nil, nil)
			require.NoError(t, err)

			require.Empty(t, m.View())
		})

		t.Run("shows how far a preview that is longer than the screen is scrolled, cuts the help that does not fit, and shows no percent for a preview that fits", func(t *testing.T) {
			m, err := prompt.New(catalog.New(withManyFiles()), "service", "my-service", nil, nil)
			require.NoError(t, err)
			m = started(m, 61, 20)

			require.Equal(t, " 0% · ctrl+d/u scroll • ←/→ move • enter confirm • esc cancel", lastLineOf(m))
			require.Equal(t, " 100% · ctrl+d/u scroll • ←/→ move • enter confirm …", lastLineOf(pressed(m, "ctrl+d", "ctrl+d", "ctrl+d", "ctrl+d", "ctrl+d")))
			require.Equal(t, " ctrl+d/u scroll • ←/→ move • enter confirm • esc cancel", lastLineOf(confirmStep(t, withFeatures(), withRegion(), withCLI(), 40)))
		})

		for name, keys := range map[string][]string{
			"y":                     {"y"},
			"enter on Yes":          {"enter"},
			"right, left and enter": {"right", "left", "enter"},
		} {
			t.Run(name+" gives the selection and stops the TUI", func(t *testing.T) {
				m := confirmStep(t, withFeatures(), map[string]string{"region": "eu"}, map[string]map[string]string{"cli": {}}, 40)

				confirmed, quit := pressedAndQuit(m, keys...)

				require.True(t, quit)
				selection, err := confirmed.Confirmed()
				require.NoError(t, err)
				require.Equal(t, "service my-service --option region=eu --feature cli", selection.Arguments("my-service"))
			})
		}

		t.Run("a template that fails to render ends the TUI after the feature step, with the message of the preview", func(t *testing.T) {
			m, err := prompt.New(catalog.New(withBadTemplate()), "service", "my-service", withRegion(), nil)
			require.NoError(t, err)

			ended, quit := pressedAndQuit(started(m, 100, 40), "enter")

			require.True(t, quit)
			_, err = ended.Confirmed()
			require.EqualError(t, err, previewErrOf(t, withBadTemplate()).Error())
		})
	})

	t.Run("features", func(t *testing.T) {
		t.Run("with no --feature flag, the TUI starts at the feature step, which lists each feature that targets the starter in the order of the names, with its summary", func(t *testing.T) {
			screen := screenOf(featureStep(t))

			require.Contains(t, screen, "Select the features")
			require.Regexp(t, `(?s)a — Feature A\..*b — Feature B\..*c — Feature C\.`, screen)
			require.NotContains(t, screen, "Elsewhere.")
			require.NotContains(t, screen, "Write the repo?")
		})

		t.Run("a starter that no feature targets has no feature step", func(t *testing.T) {
			m, err := prompt.New(catalog.New(withNeeds()), "lonely", "my-service", nil, nil)
			require.NoError(t, err)

			require.Contains(t, screenOf(started(m, 100, 40)), "Write the repo?")
		})

		for name, toggle := range map[string]string{"x": "x", "space": "space"} {
			t.Run(name+" selects a feature, and with no need enter goes on to the confirm step, which shows the feature as answered", func(t *testing.T) {
				screen := screenOf(pressed(featureStep(t), "down", "down", toggle, "enter"))

				require.Contains(t, screen, "Write the repo?")
				require.Contains(t, screen, "Features:\n  c: answered\n")
			})
		}

		t.Run("enter with no feature selected goes on to the confirm step with no feature", func(t *testing.T) {
			screen := screenOf(pressed(featureStep(t), "enter"))

			require.Contains(t, screen, "Write the repo?")
			require.Contains(t, screen, "Features: none")
		})

		t.Run("when a need adds a feature, the needs step shows the line of each feature that a need adds, and Continue goes on to the confirm step", func(t *testing.T) {
			needs := pressed(featureStep(t), "x", "enter")
			require.Contains(t, screenOf(needs), "b: needed by a")
			require.Contains(t, screenOf(needs), "Continue")
			require.Contains(t, screenOf(needs), "Change the features")

			screen := screenOf(pressed(needs, "enter"))

			require.Contains(t, screen, "Write the repo?")
			require.Contains(t, screen, "Features:\n  b: needed by a\n  a: answered\n")
		})

		t.Run("Change the features shows the feature step again with the selected features kept, and a new choice replaces them", func(t *testing.T) {
			again := pressed(featureStep(t), "x", "enter", "down", "enter")
			require.Contains(t, screenOf(again), "Select the features")
			require.Contains(t, screenOf(again), "✓ a — Feature A.")

			screen := screenOf(pressed(again, "x", "down", "down", "x", "enter"))

			require.Contains(t, screen, "Write the repo?")
			require.Contains(t, screen, "Features:\n  c: answered\n\n")
		})

		t.Run("a selected feature that another selected feature needs shows answered and the features that need it, with no needs step", func(t *testing.T) {
			screen := screenOf(pressed(featureStep(t), "x", "down", "x", "enter"))

			require.Contains(t, screen, "Write the repo?")
			require.Contains(t, screen, "Features:\n  b: answered, needed by a\n  a: answered\n")
		})

		t.Run("with a --feature flag, the TUI shows no feature step and no needs step, and the confirm step shows the features of the flags", func(t *testing.T) {
			m, err := prompt.New(catalog.New(withNeeds()), "service", "my-service", nil, map[string]map[string]string{"a": {}})
			require.NoError(t, err)

			screen := screenOf(started(m, 100, 40))

			require.Contains(t, screen, "Write the repo?")
			require.Contains(t, screen, "Features:\n  b: needed by a\n  a: command line\n")
		})

		t.Run("after yes, the selection holds the features that the developer selected", func(t *testing.T) {
			confirmed, quit := pressedAndQuit(featureStep(t), "down", "down", "x", "enter", "y")

			require.True(t, quit)
			selection, err := confirmed.Confirmed()
			require.NoError(t, err)
			require.Equal(t, "service my-service --feature c", selection.Arguments("my-service"))
		})

		t.Run("shows the title, the features with the cursor at the first one, and the keys of the step", func(t *testing.T) {
			require.Equal(t, `feature-compose new

┃ Select the features
┃ > • a — Feature A.
┃   • b — Feature B.
┃   • c — Feature C.
┃   • w — Feature W.
 ↑/↓ move • x select • enter next • esc cancel`, screenOf(featureStep(t)))
		})

		t.Run("the needs step shows one line for each feature that a need adds, and Continue with the cursor on it", func(t *testing.T) {
			require.Equal(t, `feature-compose new

┃ The features that you selected need these features
┃ b: needed by w
┃ c: needed by w
┃ > Continue
┃   Change the features
 ↑/↓ move • enter choose • esc cancel`, screenOf(pressed(featureStep(t), "down", "down", "down", "x", "enter")))
		})

		t.Run("each step wraps its text at the width of a narrow terminal", func(t *testing.T) {
			narrow := func() prompt.Model {
				m, err := prompt.New(catalog.New(withNeeds()), "service", "my-service", nil, nil)
				require.NoError(t, err)
				return started(m, 40, 30)
			}

			for _, step := range []prompt.Model{narrow(), pressed(narrow(), "x", "enter"), pressed(narrow(), "x", "enter", "enter")} {
				require.Contains(t, step.View(), "feature-compose new")
				for _, line := range strings.Split(escapeCode.ReplaceAllString(step.View(), ""), "\n") {
					require.LessOrEqual(t, lipgloss.Width(line), 40, line)
				}
			}
		})

		t.Run("ctrl+d and ctrl+u at the feature step move the cursor of the list", func(t *testing.T) {
			down := screenOf(pressed(featureStep(t), "ctrl+d", "x", "enter"))
			up := screenOf(pressed(featureStep(t), "down", "down", "down", "ctrl+u", "x", "enter"))

			require.Contains(t, down, "Features:\n  c: answered\n")
			require.Contains(t, up, "Features:\n  b: answered\n")
		})

		t.Run("a value of a flag stays after the feature step, and shows command line", func(t *testing.T) {
			files := withNeeds()
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("summary: A service.\noptions:\n  region:\n    default: us\nfolders:\n  go:\n    path: .\n")}
			m, err := prompt.New(catalog.New(files), "service", "my-service", map[string]string{"region": "eu"}, nil)
			require.NoError(t, err)

			screen := screenOf(pressed(started(m, 100, 40), "enter"))

			require.Contains(t, screen, "  starter.region: eu (command line)\n")
		})

		t.Run("a need that is not a feature of the starter ends the TUI after the feature step, with the message of the catalog", func(t *testing.T) {
			files := withNeeds()
			files["features/c/feature.yml"] = &fstest.MapFile{Data: []byte("summary: Feature C.\ntargets: [service]\nneeds: [missing]\n")}
			m, err := prompt.New(catalog.New(files), "service", "my-service", nil, nil)
			require.NoError(t, err)

			ended, quit := pressedAndQuit(started(m, 100, 40), "down", "down", "x", "enter")

			require.True(t, quit)
			_, err = ended.Confirmed()
			require.ErrorContains(t, err, `the feature c needs "missing"`)
		})
	})

	t.Run("options", func(t *testing.T) {
		withOptions := func() fstest.MapFS {
			return fstest.MapFS{
				"starters/service/starter.yml":      &fstest.MapFile{Data: []byte("summary: A service.\noptions:\n  module:\n    default: example.com/{{ .Name }}\n  go-version:\n    default: \"1.27.1\"\n    pattern: '^1\\.(2[7-9])(\\.[0-9]+)?$'\nfolders:\n  go:\n    path: .\n")},
				"starters/service/files/go/main.go": &fstest.MapFile{Data: []byte("package main\n")},
				"features/alpha/feature.yml":        &fstest.MapFile{Data: []byte("summary: Alpha.\ntargets: [service]\nneeds: [zed]\noptions:\n  level:\n    default: \"1\"\n")},
				"features/beta/feature.yml":         &fstest.MapFile{Data: []byte("summary: Beta.\ntargets: [service]\nneeds: [zed]\n")},
				"features/solo/feature.yml":         &fstest.MapFile{Data: []byte("summary: Solo.\ntargets: [service]\noptions:\n  size:\n    values: [s, m]\n")},
				"features/uno/feature.yml":          &fstest.MapFile{Data: []byte("summary: Uno.\ntargets: [service]\noptions:\n  kind:\n    values: [only]\n    default: only\n")},
				"features/zed/feature.yml":          &fstest.MapFile{Data: []byte("summary: Zed.\ntargets: [service]\noptions:\n  mode:\n    values: [slow, fast]\n    default: fast\n  token:\n    pattern: '^.*$'\n")},
			}
		}
		optionsStep := func(t *testing.T, given map[string]string, givenByFeature map[string]map[string]string) prompt.Model {
			t.Helper()
			m, err := prompt.New(catalog.New(withOptions()), "service", "my-service", given, givenByFeature)
			require.NoError(t, err)
			return started(m, 160, 40)
		}
		alphaAndBeta := func() map[string]map[string]string { return map[string]map[string]string{"alpha": {}, "beta": {}} }

		t.Run("shows a group for the options of the starter first, with each text option at its default rendered with the name", func(t *testing.T) {
			require.Equal(t, `feature-compose new

The options of the starter service
┃ go-version
┃ > 1.27.1

  module
  > example.com/my-service
 ↑/↓ move • enter next • shift+tab back • esc cancel`, screenOf(optionsStep(t, nil, alphaAndBeta())))
		})

		t.Run("shows one group for each feature with a question, each after the features that it needs, and each option one time, also when two selected features need its feature", func(t *testing.T) {
			var titles []string
			m := optionsStep(t, nil, alphaAndBeta())
			for _, keys := range [][]string{{"enter", "enter"}, {"enter", "t", "enter"}, {"enter"}} {
				titles = append(titles, groupTitleOf(m))
				m = pressed(m, keys...)
			}

			require.Equal(t, []string{"The options of the starter service", "The options of the feature zed", "The options of the feature alpha"}, titles)
			require.Contains(t, screenOf(m), "Write the repo?")
		})

		t.Run("an option with values shows each value with the default first and chosen, and the chosen value is the answer", func(t *testing.T) {
			zed := pressed(optionsStep(t, nil, map[string]map[string]string{"zed": {}}), "enter", "enter")
			require.Contains(t, screenOf(zed), "┃ mode\n┃ > fast\n┃   slow\n\n")

			screen := screenOf(pressed(zed, "down", "enter", "t", "enter"))

			require.Contains(t, screen, "  zed.mode: slow (answered)\n")
		})

		t.Run("while a text does not match the pattern of its option, the TUI shows the message and enter keeps it at the option, and a text that matches goes on", func(t *testing.T) {
			typed := pressed(optionsStep(t, nil, map[string]map[string]string{"alpha": {}}), "backspace", "backspace", "backspace", "backspace", "backspace", "backspace", "1.20")
			message := `the starter service: the value "1.20" of the option go-version does not match its pattern ^1\.(2[7-9])(\.[0-9]+)?$`
			require.Contains(t, screenOf(typed), message)

			stayed := pressed(typed, "enter")
			require.Contains(t, screenOf(stayed), "┃ > 1.20")
			require.Contains(t, screenOf(stayed), message)

			fixed := pressed(stayed, "backspace", "8")
			require.NotContains(t, screenOf(fixed), "does not match")
			screen := screenOf(pressed(fixed, "enter", "enter", "enter", "t", "enter", "enter"))
			require.Contains(t, screen, "  starter.go-version: 1.28 (answered)\n")
		})

		for name, test := range map[string]struct{ typed, message string }{
			"the empty text for an option with no default": {typed: "", message: "the option token of the feature zed has no default, so it needs a value"},
			"a comma in a value of an option of a feature": {typed: "a,b", message: `the value "a,b" of the option token of the feature zed holds a comma, which --feature cannot give`},
		} {
			t.Run("the TUI does not accept "+name+", and shows the message", func(t *testing.T) {
				token := pressed(optionsStep(t, nil, map[string]map[string]string{"zed": {}}), "enter", "enter", "enter")
				typed := pressed(token, test.typed)

				stayed := pressed(typed, "enter")

				require.Contains(t, screenOf(stayed), test.message)
				require.Equal(t, "The options of the feature zed", groupTitleOf(stayed))
				require.NotContains(t, screenOf(stayed), "Write the repo?")
			})
		}

		t.Run("an option that a flag gives is not asked, and the confirm step shows it with command line and each answer with answered, also a default that the developer accepted", func(t *testing.T) {
			m := optionsStep(t, map[string]string{"go-version": "1.28"}, map[string]map[string]string{"zed": {"token": "x"}, "alpha": {}})
			require.NotContains(t, screenOf(m), "go-version")

			screen := screenOf(pressed(m, "enter", "enter", "enter"))

			require.Contains(t, screen, `Options:
  starter.go-version: 1.28 (command line)
  starter.module: example.com/my-service (answered)
  zed.mode: fast (answered)
  zed.token: x (command line)
  alpha.level: 1 (answered)
`)
		})

		t.Run("an option with values and no default shows its values in the order of the file, and the first one is the answer of enter", func(t *testing.T) {
			solo := optionsStep(t, map[string]string{"go-version": "1.28", "module": "example.com/a"}, map[string]map[string]string{"solo": {}})
			require.Contains(t, screenOf(solo), "┃ size\n┃ > s\n┃   m\n")

			screen := screenOf(pressed(solo, "enter"))

			require.Contains(t, screen, "  solo.size: s (answered)\n")
		})

		t.Run("an option with one value is a list, so a typed letter does not change its value", func(t *testing.T) {
			uno := optionsStep(t, map[string]string{"go-version": "1.28", "module": "example.com/a"}, map[string]map[string]string{"uno": {}})

			screen := screenOf(pressed(uno, "x", "enter"))

			require.Contains(t, screen, "  uno.kind: only (answered)\n")
		})

		t.Run("one question is asked, at the start and after the feature step", func(t *testing.T) {
			starterValues := map[string]string{"go-version": "1.28", "module": "example.com/a"}
			atStart := optionsStep(t, map[string]string{"go-version": "1.28"}, map[string]map[string]string{"solo": {"size": "m"}})
			m, err := prompt.New(catalog.New(withOptions()), "service", "my-service", starterValues, nil)
			require.NoError(t, err)

			afterFeatures := pressed(started(m, 160, 40), "down", "down", "x", "enter")

			require.Equal(t, "The options of the starter service", groupTitleOf(atStart))
			require.Equal(t, "The options of the feature solo", groupTitleOf(afterFeatures))
		})

		t.Run("with no question, the TUI goes on to the confirm step", func(t *testing.T) {
			m := optionsStep(t, map[string]string{"go-version": "1.28", "module": "example.com/a"}, map[string]map[string]string{"zed": {"token": "x", "mode": "slow"}})

			require.Contains(t, screenOf(m), "Write the repo?")
		})

		t.Run("after the feature step, the options step asks the options of the starter and of the selected features", func(t *testing.T) {
			m, err := prompt.New(catalog.New(withOptions()), "service", "my-service", map[string]string{"go-version": "1.28", "module": "example.com/a"}, nil)
			require.NoError(t, err)

			options := pressed(started(m, 100, 40), "down", "x", "enter", "enter")

			require.Equal(t, "The options of the feature zed", groupTitleOf(options))
		})

		t.Run("after yes, the selection holds each answer, and its arguments name each answer that is not its default", func(t *testing.T) {
			confirmed, quit := pressedAndQuit(optionsStep(t, nil, map[string]map[string]string{"zed": {}}), "enter", "enter", "down", "enter", "t", "enter", "y")

			require.True(t, quit)
			selection, err := confirmed.Confirmed()
			require.NoError(t, err)
			require.Equal(t, "service my-service --feature zed:mode=slow,token=t", selection.Arguments("my-service"))
		})
	})

	t.Run("starter and name", func(t *testing.T) {
		starterStep := func(t *testing.T, givenByFeature map[string]map[string]string) prompt.Model {
			t.Helper()
			m, err := prompt.New(catalog.New(withNeeds()), "", "", nil, givenByFeature)
			require.NoError(t, err)
			return started(m, 100, 40)
		}

		t.Run("with no starter, the TUI starts at the starter step, which lists each starter in the order of the names, with its summary", func(t *testing.T) {
			require.Equal(t, `feature-compose new

┃ Select the starter
┃ > lonely — A starter that no feature targets.
┃   other — Another service.
┃   service — A service.
 ↑/↓ move • enter choose • esc cancel`, screenOf(starterStep(t, nil)))
		})

		t.Run("the lists of starters and of features have no filter, so / changes nothing", func(t *testing.T) {
			starters := starterStep(t, nil)
			features := featureStep(t)

			require.Equal(t, screenOf(starters), screenOf(pressed(starters, "/")))
			require.Equal(t, screenOf(features), screenOf(pressed(features, "/")))
		})

		for _, name := range []string{"", "../escape", "team/service"} {
			t.Run("the name step does not accept the name "+strconv.Quote(name)+", and shows the message of the selection", func(t *testing.T) {
				typed := pressed(starterStep(t, nil), "down", "down", "enter", name)

				stayed := pressed(typed, "enter")

				require.Contains(t, screenOf(stayed), "Enter the name of the repo")
				words := func(text string) string { return strings.Join(strings.Fields(text), " ") }
				require.Contains(t, words(screenOf(stayed)), words(catalog.CheckRepoName(name).Error()))
			})
		}

		t.Run("the name step shows the question, the name as it is typed, a / included, and the keys of the step", func(t *testing.T) {
			typed := pressed(starterStep(t, nil), "down", "down", "enter", "a", "/", "b")

			require.Equal(t, `feature-compose new

┃ Enter the name of the repo
┃ > a/b
the name "a/b" of the repo must start with a letter or a digit, and hold only letters, digits, '.',
'_' and '-'
 enter next • esc cancel`, screenOf(typed))
		})

		t.Run("a starter that cannot be read ends the TUI after the starter step, with the message of the catalog", func(t *testing.T) {
			files := withNeeds()
			files["starters/lonely/starter.yml"] = &fstest.MapFile{Data: []byte("summary: A starter that no feature targets.\noptions:\n  region:\n    pattern: '('\nfolders:\n  go:\n    path: .\n")}
			m, err := prompt.New(catalog.New(files), "", "", nil, nil)
			require.NoError(t, err)

			ended, quit := pressedAndQuit(started(m, 100, 40), "enter")

			require.True(t, quit)
			_, err = ended.Confirmed()
			require.ErrorContains(t, err, "the pattern of the option region")
		})

		t.Run("with a starter and no name, the TUI starts at the name step", func(t *testing.T) {
			m, err := prompt.New(catalog.New(withNeeds()), "service", "", nil, nil)
			require.NoError(t, err)

			require.Contains(t, screenOf(started(m, 100, 40)), "Enter the name of the repo")
		})

		t.Run("after the starter step and the name step, the TUI goes on to the feature step of that starter, and a valid name shows no message", func(t *testing.T) {
			named := pressed(starterStep(t, nil), "down", "down", "enter", "my-service")
			require.NotContains(t, screenOf(named), "must start")

			screen := screenOf(pressed(named, "enter"))

			require.Contains(t, screen, "Select the features")
			require.Contains(t, screen, "a — Feature A.")
		})

		t.Run("a flag that the starter of the TUI does not know ends the TUI after the name step, with the message of the catalog", func(t *testing.T) {
			ended, quit := pressedAndQuit(starterStep(t, map[string]map[string]string{"nope": {}}), "down", "down", "enter", "my-service", "enter")

			require.True(t, quit)
			_, err := ended.Confirmed()
			require.EqualError(t, err, `unknown feature "nope": the features that target the starter service are a, b, c, w`)
		})

		t.Run("after yes, the selection holds the starter and the name that the developer gave", func(t *testing.T) {
			confirmed, quit := pressedAndQuit(starterStep(t, nil), "enter", "my-lonely", "enter", "y")

			require.True(t, quit)
			selection, err := confirmed.Confirmed()
			require.NoError(t, err)
			require.Equal(t, "lonely my-lonely", selection.Arguments("my-lonely"))
		})
	})

	t.Run("cancel", func(t *testing.T) {
		for name, keys := range map[string][]string{
			"n":                     {"n"},
			"no with enter":         {"right", "enter"},
			"esc":                   {"esc"},
			"ctrl+c":                {"ctrl+c"},
			"esc after a scroll":    {"ctrl+d", "esc"},
			"ctrl+c after a scroll": {"pgdown", "ctrl+c"},
		} {
			t.Run(name+" at the confirm step stops the TUI and gives the cancel, with no selection", func(t *testing.T) {
				m := confirmStep(t, withFeatures(), withRegion(), withCLI(), 40)

				canceled, quit := pressedAndQuit(m, keys...)

				require.True(t, quit)
				_, err := canceled.Confirmed()
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		for name, keys := range map[string][]string{
			"esc at the feature step":    {"esc"},
			"ctrl+c at the feature step": {"x", "ctrl+c"},
			"esc at the needs step":      {"x", "enter", "esc"},
			"ctrl+c at the needs step":   {"x", "enter", "ctrl+c"},
		} {
			t.Run(name+" stops the TUI and gives the cancel", func(t *testing.T) {
				canceled, quit := pressedAndQuit(featureStep(t), keys...)

				require.True(t, quit)
				_, err := canceled.Confirmed()
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		for name, keys := range map[string][]string{
			"esc at the options step":    {"esc"},
			"ctrl+c at the options step": {"a", "ctrl+c"},
		} {
			t.Run(name+" stops the TUI and gives the cancel", func(t *testing.T) {
				m, err := prompt.New(catalog.New(withFeatures()), "service", "my-service", nil, withCLI())
				require.NoError(t, err)

				canceled, quit := pressedAndQuit(started(m, 100, 40), keys...)

				require.True(t, quit)
				_, err = canceled.Confirmed()
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		for name, keys := range map[string][]string{
			"esc at the starter step":    {"esc"},
			"ctrl+c at the starter step": {"ctrl+c"},
			"esc at the name step":       {"enter", "esc"},
			"ctrl+c at the name step":    {"enter", "a", "ctrl+c"},
		} {
			t.Run(name+" stops the TUI and gives the cancel", func(t *testing.T) {
				m, err := prompt.New(catalog.New(withNeeds()), "", "", nil, nil)
				require.NoError(t, err)

				canceled, quit := pressedAndQuit(started(m, 100, 40), keys...)

				require.True(t, quit)
				_, err = canceled.Confirmed()
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		t.Run("a TUI that stops before an answer gives the cancel", func(t *testing.T) {
			m := confirmStep(t, withFeatures(), withRegion(), withCLI(), 40)

			_, err := m.Confirmed()

			require.EqualError(t, err, "you canceled new, so it wrote no file")
		})
	})
}

func started(m prompt.Model, width, height int) prompt.Model {
	m = updated(m, m.Init())
	next, cmd := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated(next.(prompt.Model), cmd)
}

var escapeCode = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func screenOf(m prompt.Model) string {
	lines := strings.Split(escapeCode.ReplaceAllString(m.View(), ""), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

func groupTitleOf(m prompt.Model) string {
	for _, line := range strings.Split(screenOf(m), "\n") {
		if title, ok := strings.CutPrefix(line, "The options of "); ok {
			return "The options of " + title
		}
	}
	return ""
}

func lastLineOf(m prompt.Model) string {
	lines := strings.Split(screenOf(m), "\n")
	return lines[len(lines)-1]
}

func pressed(m prompt.Model, keys ...string) prompt.Model {
	pressedModel, _ := pressedAndQuit(m, keys...)
	return pressedModel
}

func pressedAndQuit(m prompt.Model, keys ...string) (prompt.Model, bool) {
	quit := false
	for _, name := range keys {
		next, cmd := m.Update(keyMsg(name))
		m = next.(prompt.Model)
		var stopped bool
		m, stopped = updatedAndQuit(m, cmd)
		quit = quit || stopped
	}
	return m, quit
}

func updated(m prompt.Model, cmd tea.Cmd) prompt.Model {
	m, _ = updatedAndQuit(m, cmd)
	return m
}

func updatedAndQuit(m prompt.Model, cmd tea.Cmd) (prompt.Model, bool) {
	quit := false
	pending := messagesOf(cmd)
	for len(pending) > 0 {
		msg := pending[0]
		pending = pending[1:]
		if _, ok := msg.(tea.QuitMsg); ok {
			quit = true
			continue
		}
		next, cmd := m.Update(msg)
		m = next.(prompt.Model)
		pending = append(pending, messagesOf(cmd)...)
	}
	return m, quit
}

func keyMsg(name string) tea.KeyMsg {
	types := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"tab": tea.KeyTab, "shift+tab": tea.KeyShiftTab, "space": tea.KeySpace, "backspace": tea.KeyBackspace,
		"ctrl+c": tea.KeyCtrlC, "ctrl+d": tea.KeyCtrlD, "ctrl+u": tea.KeyCtrlU, "pgdown": tea.KeyPgDown, "pgup": tea.KeyPgUp,
	}
	if keyType, ok := types[name]; ok {
		return tea.KeyMsg{Type: keyType}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

var commandType = reflect.TypeFor[tea.Cmd]()

// messagesOf runs a command, and each command of a batch or a sequence, and drops a command that waits, such as the blink of the cursor.
func messagesOf(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		if msg == nil {
			return nil
		}
		commands := reflect.ValueOf(msg)
		if commands.Kind() != reflect.Slice || commands.Type().Elem() != commandType {
			return []tea.Msg{msg}
		}
		var messages []tea.Msg
		for i := range commands.Len() {
			messages = append(messages, messagesOf(commands.Index(i).Interface().(tea.Cmd))...)
		}
		return messages
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}
