//go:build unit

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	writeConfig := func(t *testing.T, configHome, contents string) string {
		t.Helper()
		path := filepath.Join(configHome, "feature-compose", "config.yml")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
		return path
	}

	t.Run("load", func(t *testing.T) {
		t.Run("gives the catalog folder that the config file under XDG_CONFIG_HOME names", func(t *testing.T) {
			configHome := t.TempDir()
			catalogDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			writeConfig(t, configHome, "catalog: "+catalogDir+"\n")

			file, err := config.Load()

			require.NoError(t, err)
			require.Equal(t, config.File{CatalogDir: catalogDir}, file)
		})

		t.Run("with no XDG_CONFIG_HOME, reads the config file under .config in the home folder", func(t *testing.T) {
			home := t.TempDir()
			catalogDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", "")
			t.Setenv("HOME", home)
			writeConfig(t, filepath.Join(home, ".config"), "catalog: "+catalogDir+"\n")

			file, err := config.Load()

			require.NoError(t, err)
			require.Equal(t, config.File{CatalogDir: catalogDir}, file)
		})

		t.Run("a catalog folder that starts with ~/ is in the home folder", func(t *testing.T) {
			home := t.TempDir()
			configHome := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(home, "code", "catalog"), 0o755))
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", configHome)
			writeConfig(t, configHome, "catalog: ~/code/catalog\n")

			file, err := config.Load()

			require.NoError(t, err)
			require.Equal(t, config.File{CatalogDir: filepath.Join(home, "code", "catalog")}, file)
		})

		t.Run("a relative catalog folder is relative to the folder of the config file, and not to the working folder", func(t *testing.T) {
			configHome := t.TempDir()
			t.Chdir(t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", configHome)
			require.NoError(t, os.MkdirAll(filepath.Join(configHome, "feature-compose", "catalog"), 0o755))
			writeConfig(t, configHome, "catalog: catalog\n")

			file, err := config.Load()

			require.NoError(t, err)
			require.Equal(t, config.File{CatalogDir: filepath.Join(configHome, "feature-compose", "catalog")}, file)
		})

		t.Run("no config file stops, and the message names its path and the line to write", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)

			_, err := config.Load()

			require.ErrorContains(t, err, filepath.Join(configHome, "feature-compose", "config.yml"))
			require.ErrorContains(t, err, `catalog: <folder>`)
		})

		t.Run("a config file that names no catalog stops, and the message names the file and the line to write", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			path := writeConfig(t, configHome, "other: value\n")

			_, err := config.Load()

			require.ErrorContains(t, err, path+" names no catalog")
			require.ErrorContains(t, err, `catalog: <folder>`)
		})

		t.Run("a config file that is not valid YAML stops, and the message names the file", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			path := writeConfig(t, configHome, "catalog: [\n")

			_, err := config.Load()

			require.ErrorContains(t, err, "read "+path)
		})

		for name, catalog := range map[string]func(t *testing.T) string{
			"does not exist": func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") },
			"is a file": func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "catalog.yml")
				require.NoError(t, os.WriteFile(path, nil, 0o644))
				return path
			},
		} {
			t.Run("a catalog folder that "+name+" stops, and the message names the folder and the config file", func(t *testing.T) {
				configHome := t.TempDir()
				t.Setenv("XDG_CONFIG_HOME", configHome)
				catalogDir := catalog(t)
				path := writeConfig(t, configHome, "catalog: "+catalogDir+"\n")

				_, err := config.Load()

				require.ErrorContains(t, err, "the catalog folder "+catalogDir+" that "+path+" names is not a folder")
			})
		}
	})
}
