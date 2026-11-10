//go:build unit

package prompt_test

import (
	"fmt"
	"reflect"
	"regexp"
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
				screen := screenOf(confirmStep(t, withFeatures(), nil, withCLI(), height))
				if strings.Contains(screen, "Repo: my-service") && strings.Contains(screen, "  main.go: the starter") {
					fits = height
				}
			}
			require.NotZero(t, fits)

			require.NotContains(t, lastLineOf(confirmStep(t, withFeatures(), nil, withCLI(), fits)), "%")
			require.Contains(t, lastLineOf(confirmStep(t, withFeatures(), nil, withCLI(), fits-1)), "0% ·")
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
			require.Equal(t, " ctrl+d/u scroll • ←/→ move • enter confirm • esc cancel", lastLineOf(confirmStep(t, withFeatures(), nil, withCLI(), 40)))
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

		t.Run("a template that fails to render stops the TUI before it starts when flags answer the steps before the confirm step, with the message of the preview", func(t *testing.T) {
			_, err := prompt.New(catalog.New(withBadTemplate()), "service", "my-service", nil, withCLI())

			require.EqualError(t, err, previewErrOf(t, withBadTemplate()).Error())
		})

		t.Run("a template that fails to render ends the TUI after the feature step, with the message of the preview", func(t *testing.T) {
			m, err := prompt.New(catalog.New(withBadTemplate()), "service", "my-service", nil, nil)
			require.NoError(t, err)

			ended, quit := pressedAndQuit(started(m, 100, 40), "enter")

			require.True(t, quit)
			_, err = ended.Confirmed()
			require.EqualError(t, err, previewErrOf(t, withBadTemplate()).Error())
		})

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
				m := confirmStep(t, withFeatures(), nil, withCLI(), 40)

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

		t.Run("a TUI that stops before an answer gives the cancel", func(t *testing.T) {
			m := confirmStep(t, withFeatures(), nil, withCLI(), 40)

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
