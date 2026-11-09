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
					Summary: "API keys.", Targets: []string{"api"}, Needs: []string{"cli"},
					Options: map[string]catalog.OptionDefinition{"method": {Default: "api-key", Values: []string{"api-key", "oauth"}}},
					AddTo:   map[string][]map[string]string{"api.middleware": {{"code": "auth.Middleware"}}, "api.config": {{"name": "Auth"}}},
				},
				"docs":       {Summary: "Docs.", Targets: []string{"api"}},
				"zone-tools": {Summary: "Tools of the zone.", Targets: []string{"zone"}},
				"other":      {Summary: "Only for another starter.", Targets: []string{"web"}},
			},
		}
	}

	t.Run("list of the starters", func(t *testing.T) {
		t.Run("shows each starter with its summary, in the order of the names", func(t *testing.T) {
			screen := screenOf(opened(browser.New(definitions())))

			require.Contains(t, screen, "api")
			require.Contains(t, screen, "An API that a worker calls.")
			require.Contains(t, screen, "A worker for jobs.")
			require.Less(t, strings.Index(screen, "An API that a worker calls."), strings.Index(screen, "A worker for jobs."))
			require.Contains(t, screen, "Starters")
			require.Contains(t, screen, "3 starters")
		})

		t.Run("/ starts a filter on the name, and a starter whose summary holds the text but whose name does not is hidden", func(t *testing.T) {
			m := pressed(opened(browser.New(definitions())), "/", "w", "o", "r")

			screen := screenOf(m)
			require.Contains(t, screen, "A worker for jobs.")
			require.NotContains(t, screen, "An API that a worker calls.")
		})

		t.Run("while the filter is typed, h, j, k, l and q go to the filter and do not stop the TUI", func(t *testing.T) {
			m, quit := pressedAndQuit(opened(browser.New(definitions())), "/", "h", "j", "k", "l", "q")

			require.False(t, quit)
			screen := screenOf(m)
			require.Contains(t, screen, "hjklq")
			require.NotContains(t, screen, "A worker for jobs.")
			require.NotContains(t, screen, "An API that a worker calls.")
		})

		t.Run("enter ends the typing and keeps the filtered list, where l opens the view of the starter", func(t *testing.T) {
			m := pressed(opened(browser.New(definitions())), "/", "w", "o", "enter", "l")

			require.Contains(t, screenOf(m), "No feature targets this starter.")
		})

		t.Run("esc clears the filter, and the list holds each starter again", func(t *testing.T) {
			m := pressed(opened(browser.New(definitions())), "/", "w", "o", "enter", "esc")

			screen := screenOf(m)
			require.Contains(t, screen, "A worker for jobs.")
			require.Contains(t, screen, "An API that a worker calls.")
		})

		for name, keys := range map[string][]string{
			"j and then l opens the view of the second starter":        {"j", "l"},
			"down and then enter opens the view of the second starter": {"down", "enter"},
		} {
			t.Run(name, func(t *testing.T) {
				require.Contains(t, screenOf(pressed(opened(browser.New(definitions())), keys...)), "No feature targets this starter.")
			})
		}

		for name, keys := range map[string][]string{
			"j, k and then l opens the view of the first starter":     {"j", "k", "l"},
			"down, up and then l opens the view of the first starter": {"down", "up", "l"},
		} {
			t.Run(name, func(t *testing.T) {
				require.Contains(t, screenOf(pressed(opened(browser.New(definitions())), keys...)), "Extension points")
				require.Contains(t, screenOf(pressed(opened(browser.New(definitions())), keys...)), "An API that a worker calls.")
			})
		}

		t.Run("h does nothing at the list of the starters", func(t *testing.T) {
			m, quit := pressedAndQuit(opened(browser.New(definitions())), "h")

			require.False(t, quit)
			require.Equal(t, screenOf(opened(browser.New(definitions()))), screenOf(m))
		})

		t.Run("the full help of a list with more than one page names no h or l for the pages", func(t *testing.T) {
			many := catalog.Definitions{Starters: map[string]catalog.StarterDefinition{}}
			for i := range 12 {
				many.Starters[fmt.Sprintf("starter-%02d", i)] = catalog.StarterDefinition{Summary: fmt.Sprintf("Summary %02d.", i)}
			}

			help := screenOf(pressed(sized(browser.New(many), 120, 14), "?"))

			require.Contains(t, help, "←/pgup")
			require.Contains(t, help, "→/pgdn")
			require.NotContains(t, help, "←/h/pgup")
			require.NotContains(t, help, "→/l/pgdn")
		})

		t.Run("esc at the list with no filter does not stop the TUI", func(t *testing.T) {
			_, quit := pressedAndQuit(opened(browser.New(definitions())), "esc")

			require.False(t, quit)
		})

		t.Run("l at a list with no starter keeps the list", func(t *testing.T) {
			empty := opened(browser.New(catalog.Definitions{}))

			require.Equal(t, screenOf(empty), screenOf(pressed(empty, "l")))
		})

		t.Run("h does not change the page of a list with more than one page", func(t *testing.T) {
			many := catalog.Definitions{Starters: map[string]catalog.StarterDefinition{}}
			for i := range 12 {
				many.Starters[fmt.Sprintf("starter-%02d", i)] = catalog.StarterDefinition{Summary: fmt.Sprintf("Summary %02d.", i)}
			}
			onSecondPage := pressed(sized(browser.New(many), 80, 12), "right")

			require.NotEqual(t, screenOf(sized(browser.New(many), 80, 12)), screenOf(onSecondPage))
			require.Equal(t, screenOf(onSecondPage), screenOf(pressed(onSecondPage, "h")))
		})

		t.Run("q stops the TUI when no filter is typed", func(t *testing.T) {
			_, quit := pressedAndQuit(opened(browser.New(definitions())), "q")

			require.True(t, quit)
		})

		t.Run("ctrl+c stops the TUI also while the filter is typed", func(t *testing.T) {
			_, quit := pressedAndQuit(opened(browser.New(definitions())), "/", "w", "ctrl+c")

			require.True(t, quit)
		})
	})

	t.Run("view of a starter", func(t *testing.T) {
		t.Run("shows the summary, the folders with their commands, the options, the extension points of the starter and of its features, and its features", func(t *testing.T) {
			screen := screenOf(pressed(opened(browser.New(definitions())), "l"))

			require.Equal(t, `api
An API that a worker calls.

Folders
  docs  path: docs
  go  path: .
    after_render: go mod tidy, gofmt -w .
    test: task test:unit
  web  path: web
    after_render: npm install

Options
  module  default: example.com/{{ .Name }}  pattern: ^[a-z./]+$
  region  default: us  values: us, eu
  tier  default: standard  values: standard

Extension points
  api.config  item: name  declared by the starter
    A config field.
  api.middleware  item: code, import  declared by the starter
    A middleware.
  cli.commands  item: code  declared by the feature cli
    A command.
  cli.flags  item: name  declared by the feature cli
    A flag.

Features
> auth  API keys.
  cli  A command line program.
  docs  Docs.

h back · j/k move · l open · q quit
`, screen)
		})

		t.Run("a starter with no folder, option, extension point or feature says none for each", func(t *testing.T) {
			screen := screenOf(pressed(opened(browser.New(definitions())), "j", "l"))

			require.Equal(t, `worker
A worker for jobs.

Folders
  none

Options
  none

Extension points
  none

Features
  No feature targets this starter.

h back · j/k move · l open · q quit
`, screen)
		})

		for keys, selected := range map[string]string{
			"j":            "> cli  A command line program.",
			"j j":          "> docs  Docs.",
			"j j j down":   "> docs  Docs.",
			"j j k":        "> cli  A command line program.",
			"down down up": "> cli  A command line program.",
			"up k":         "> auth  API keys.",
		} {
			t.Run("j and k move the selection in the list of the features by one, as down and up do, and stop at its ends: "+keys, func(t *testing.T) {
				screen := screenOf(pressed(opened(browser.New(definitions())), append([]string{"l"}, strings.Fields(keys)...)...))

				require.Contains(t, screen, selected)
			})
		}

		t.Run("with one feature, j and k keep the selection on it", func(t *testing.T) {
			screen := screenOf(pressed(opened(browser.New(definitions())), "j", "j", "l", "j", "k", "j"))

			require.Contains(t, screen, "> zone-tools  Tools of the zone.")
		})

		for _, back := range []string{"h", "esc"} {
			t.Run(back+" goes back to the list of the starters, with the same starter selected and the same filter", func(t *testing.T) {
				m := pressed(opened(browser.New(definitions())), "/", "w", "o", "enter", "l", back)

				screen := screenOf(m)
				require.Contains(t, screen, "A worker for jobs.")
				require.NotContains(t, screen, "An API that a worker calls.")
				require.Contains(t, screenOf(pressed(m, "l")), "No feature targets this starter.")
			})
		}

		for _, stop := range []string{"q", "ctrl+c"} {
			t.Run(stop+" stops the TUI from the view of a starter", func(t *testing.T) {
				_, quit := pressedAndQuit(opened(browser.New(definitions())), "l", stop)

				require.True(t, quit)
			})
		}
	})

	t.Run("view of a feature", func(t *testing.T) {
		t.Run("l on a feature opens its view: the summary, the needs, the options, and the extension points that it declares or adds items to", func(t *testing.T) {
			screen := screenOf(pressed(opened(browser.New(definitions())), "l", "l"))

			require.Equal(t, `auth
API keys.

Needs
  cli

Options
  method  default: api-key  values: api-key, oauth

Extension points that it declares
  none

Extension points that it adds items to
  api.config
  api.middleware

h back · q quit
`, screen)
		})

		t.Run("j and then enter opens the view of the second feature, which says none where it has nothing", func(t *testing.T) {
			screen := screenOf(pressed(opened(browser.New(definitions())), "l", "j", "enter"))

			require.Equal(t, `cli
A command line program.

Needs
  none

Options
  none

Extension points that it declares
  cli.commands  item: code
    A command.
  cli.flags  item: name
    A flag.

Extension points that it adds items to
  none

h back · q quit
`, screen)
		})

		for _, back := range []string{"h", "esc"} {
			t.Run(back+" goes back to the view of the starter with the same feature selected, and a second h goes back to the list of the starters", func(t *testing.T) {
				atStarter := pressed(opened(browser.New(definitions())), "l", "j", "l", back)

				require.Contains(t, screenOf(atStarter), "> cli  A command line program.")
				require.Contains(t, screenOf(pressed(atStarter, "h")), "A worker for jobs.")
			})
		}

		t.Run("after show <starter>, h goes back from the view of a feature to the view of that starter, and then to the list", func(t *testing.T) {
			atStarter, err := browser.NewAtStarter(definitions(), "api")
			require.NoError(t, err)
			m := pressed(opened(atStarter), "l", "h")

			require.Contains(t, screenOf(m), "> auth  API keys.")
			require.Contains(t, screenOf(pressed(m, "h")), "A worker for jobs.")
		})

		t.Run("l at a starter with no feature keeps the view of the starter", func(t *testing.T) {
			screen := screenOf(pressed(opened(browser.New(definitions())), "j", "l", "l"))

			require.Contains(t, screen, "No feature targets this starter.")
		})

		for _, stop := range []string{"q", "ctrl+c"} {
			t.Run(stop+" stops the TUI from the view of a feature", func(t *testing.T) {
				_, quit := pressedAndQuit(opened(browser.New(definitions())), "l", "l", stop)

				require.True(t, quit)
			})
		}

		t.Run("j and k in the view of a feature do not change the view", func(t *testing.T) {
			m := pressed(opened(browser.New(definitions())), "l", "l")

			require.Equal(t, screenOf(m), screenOf(pressed(m, "j", "k")))
		})

		t.Run("j in the view of a feature does not move the selection that h goes back to", func(t *testing.T) {
			screen := screenOf(pressed(opened(browser.New(definitions())), "l", "l", "j", "h"))

			require.Contains(t, screen, "> auth  API keys.")
		})
	})

	t.Run("at a starter", func(t *testing.T) {
		t.Run("opens the view of that starter, and h then shows the list of all starters", func(t *testing.T) {
			atStarter, err := browser.NewAtStarter(definitions(), "worker")
			require.NoError(t, err)
			m := opened(atStarter)

			require.Contains(t, screenOf(m), "No feature targets this starter.")
			screen := screenOf(pressed(m, "h"))
			require.Contains(t, screen, "A worker for jobs.")
			require.Contains(t, screen, "An API that a worker calls.")
			require.Contains(t, screenOf(pressed(m, "h", "l")), "No feature targets this starter.")
		})

		t.Run("an unknown starter is an error that lists the starters", func(t *testing.T) {
			_, err := browser.NewAtStarter(definitions(), "nope")

			require.EqualError(t, err, `unknown starter "nope": the catalog has api, worker, zone`)
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
	return escapeCode.ReplaceAllString(m.View(), "")
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
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
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
