//go:build unit

package cmd

import (
	"bytes"
	"context"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestCheckCommand(t *testing.T) {
	runCheck := func(t *testing.T, files fstest.MapFS, arguments ...string) (string, error) {
		t.Helper()
		var printed, stderr bytes.Buffer
		command := newCheckCommand(func() (catalog.Catalog, error) { return catalog.New(files), nil })
		command.Writer = &printed
		command.ErrWriter = &stderr
		err := command.Run(context.Background(), append([]string{"check"}, arguments...))
		return printed.String(), err
	}
	withFeatures := func(features map[string]string) fstest.MapFS {
		files := fstest.MapFS{
			"starters/service/starter.yml":        &fstest.MapFile{Data: []byte("folders:\n  go:\n    path: .\nextension_points:\n  api.middleware:\n    file: go/server.go\n    item: [code, import]\n")},
			"starters/service/files/go/server.go": &fstest.MapFile{Data: []byte("package server\n")},
		}
		for featureName, definition := range features {
			files["features/"+featureName+"/feature.yml"] = &fstest.MapFile{Data: []byte(definition)}
		}
		return files
	}

	t.Run("check", func(t *testing.T) {
		t.Run("with no mistake, prints that the catalog has no mistakes and stops with no error", func(t *testing.T) {
			printed, err := runCheck(t, withFeatures(map[string]string{
				"auth": "targets: [service]\nneeds: [cli]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n  cli.commands:\n    - code: newAuthCommand()\n",
				"cli":  "targets: [service]\nextension_points:\n  cli.commands:\n    file: go/cli.go\n    item: [code]\n",
			}))

			require.NoError(t, err)
			require.Equal(t, "The catalog has no mistakes.\n", printed)
		})

		for name, features := range map[string]map[string]string{
			"with no mistake":   {"auth": "targets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n      import: internal/auth\n"},
			"with two mistakes": {"auth": "targets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n  cli.commands:\n    - code: newAuthCommand()\n"},
		} {
			t.Run(name+", a write to stdout that fails stops check with its cause", func(t *testing.T) {
				command := newCheckCommand(func() (catalog.Catalog, error) { return catalog.New(withFeatures(features)), nil })
				command.Writer, command.ErrWriter = failingWriter{}, failingWriter{}

				err := command.Run(context.Background(), []string{"check"})

				require.EqualError(t, err, "closed output")
			})
		}

		t.Run("prints each mistake on its own line, and stops with an error that counts the mistakes", func(t *testing.T) {
			printed, err := runCheck(t, withFeatures(map[string]string{
				"auth":  "targets: [service]\nadd_to:\n  api.middleware:\n    - code: auth.Middleware\n",
				"tools": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n",
			}))

			require.EqualError(t, err, "the catalog has 2 mistakes")
			require.Equal(t, `the starter service: the item 1 that the feature auth adds to the extension point "api.middleware" has the fields [code], but the extension point declares the fields [code, import]
the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare
`, printed)
		})

		t.Run("with one mistake, the error counts one mistake", func(t *testing.T) {
			_, err := runCheck(t, withFeatures(map[string]string{"tools": "targets: [service]\nadd_to:\n  cli.commands:\n    - code: newToolsCommand()\n"}))

			require.EqualError(t, err, "the catalog has 1 mistake")
		})
	})

	t.Run("stop", func(t *testing.T) {
		t.Run("an argument stops check before it reads the catalog", func(t *testing.T) {
			command := newCheckCommand(func() (catalog.Catalog, error) {
				require.FailNow(t, "check read the catalog")
				return catalog.Catalog{}, nil
			})
			var printed bytes.Buffer
			command.Writer, command.ErrWriter = &printed, &printed

			err := command.Run(context.Background(), []string{"check", "go-event-sourcing-service"})

			require.EqualError(t, err, "check takes no argument")
		})

		t.Run("a catalog that check cannot read stops it with the cause, and prints nothing", func(t *testing.T) {
			files := withFeatures(nil)
			files["starters/service/starter.yml"] = &fstest.MapFile{Data: []byte("folders: [go\n")}

			printed, err := runCheck(t, files)

			require.ErrorContains(t, err, "starters/service/starter.yml")
			require.Empty(t, printed)
		})

		t.Run("a files entry that leads out of the feature stops check, and the message names the feature.yml and the files entry", func(t *testing.T) {
			printed, err := runCheck(t, withFeatures(map[string]string{"cli": "targets: [service]\nfiles:\n  - path: ../docs/files/\n"}))

			require.ErrorContains(t, err, `read features/cli/feature.yml: the files entry "../docs/files/" is outside the folder of the feature`)
			require.Empty(t, printed)
		})
	})
}
