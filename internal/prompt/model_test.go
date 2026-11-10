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
				screen := screenOf(confirmStep(t, withFeatures(), nil, nil, height))
				if strings.Contains(screen, "Repo: my-service") && strings.Contains(screen, "  main.go: the starter") {
					fits = height
				}
			}
			require.NotZero(t, fits)

			require.NotContains(t, lastLineOf(confirmStep(t, withFeatures(), nil, nil, fits)), "%")
			require.Contains(t, lastLineOf(confirmStep(t, withFeatures(), nil, nil, fits-1)), "0% ·")
		})

		t.Run("fills the rows of the terminal, and cuts each line of the preview at its width", func(t *testing.T) {
			files := withManyFiles()
			files["starters/service/files/go/a-"+strings.Repeat("long-name-", 12)+".txt"] = &fstest.MapFile{Data: []byte("x\n")}
			m, err := prompt.New(catalog.New(files), "service", "my-service", nil, nil)
			require.NoError(t, err)
			m = started(m, 62, 20)

			lines := strings.Split(screenOf(m), "\n")

			require.Len(t, lines, 20)
			require.Contains(t, screenOf(m), "a-long-name-long-name-")
			for _, line := range lines {
				require.LessOrEqual(t, lipgloss.Width(line), 62, line)
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
			m = started(m, 63, 20)

			require.Equal(t, " 0% · ctrl+d/u scroll • ←/→ choose • enter confirm • esc cancel", lastLineOf(m))
			require.Equal(t, " 100% · ctrl+d/u scroll • ←/→ choose • enter confirm …", lastLineOf(pressed(m, "ctrl+d", "ctrl+d", "ctrl+d", "ctrl+d", "ctrl+d")))
			require.Equal(t, " ctrl+d/u scroll • ←/→ choose • enter confirm • esc cancel", lastLineOf(confirmStep(t, withFeatures(), nil, nil, 40)))
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

		t.Run("a template that fails to render stops the TUI before it starts, with the message of the preview", func(t *testing.T) {
			files := withFeatures()
			files["starters/service/files/go/bad.go"] = &fstest.MapFile{Data: []byte("package {{ .Nmae }}\n")}
			starter, err := catalog.New(files).Starter("service")
			require.NoError(t, err)
			selection, err := starter.Select("my-service", nil, nil, catalog.Answers{})
			require.NoError(t, err)
			_, previewErr := selection.Preview()
			require.Error(t, previewErr)

			_, err = prompt.New(catalog.New(files), "service", "my-service", nil, nil)

			require.EqualError(t, err, previewErr.Error())
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
				m := confirmStep(t, withFeatures(), nil, nil, 40)

				canceled, quit := pressedAndQuit(m, keys...)

				require.True(t, quit)
				_, err := canceled.Confirmed()
				require.EqualError(t, err, "you canceled new, so it wrote no file")
			})
		}

		t.Run("a TUI that stops before an answer gives the cancel", func(t *testing.T) {
			m := confirmStep(t, withFeatures(), nil, nil, 40)

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
