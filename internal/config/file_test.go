//go:build unit

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/stretchr/testify/require"
)

func loadedWith(t *testing.T, configHome string, catalogs map[string]config.Entry) config.File {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	file, err := config.Load()
	require.NoError(t, err)
	for name, entry := range catalogs {
		file, err = file.WithCatalog(name, entry)
		require.NoError(t, err)
	}
	return file
}

func entriesOf(t *testing.T, file config.File) map[string]config.Entry {
	t.Helper()
	entries := map[string]config.Entry{}
	for _, name := range file.Names() {
		entry, err := file.Entry(name)
		require.NoError(t, err)
		entries[name] = entry
	}
	return entries
}

func TestFile(t *testing.T) {
	writeConfig := func(t *testing.T, configHome, contents string) string {
		t.Helper()
		path := filepath.Join(configHome, "feature-compose", "config.yml")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
		return path
	}

	t.Run("load", func(t *testing.T) {
		t.Run("gives the catalogs that the config file under XDG_CONFIG_HOME names", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			writeConfig(t, configHome, "catalogs:\n  hpcsc:\n    source: https://github.com/hpcsc/feature-compose\n    path: catalog\n    ref: main\n  team:\n    source: /code/team-catalog\n")

			file, err := config.Load()

			require.NoError(t, err)
			require.Equal(t, map[string]config.Entry{
				"hpcsc": {Source: "https://github.com/hpcsc/feature-compose", Path: "catalog", Ref: "main"},
				"team":  {Source: "/code/team-catalog"},
			}, entriesOf(t, file))
		})

		t.Run("with no XDG_CONFIG_HOME, reads the config file under .config in the home folder", func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", "")
			t.Setenv("HOME", home)
			writeConfig(t, filepath.Join(home, ".config"), "catalogs:\n  team:\n    source: /code/team-catalog\n")

			file, err := config.Load()

			require.NoError(t, err)
			require.Equal(t, map[string]config.Entry{"team": {Source: "/code/team-catalog"}}, entriesOf(t, file))
		})

		t.Run("with no config file, the config has no catalog", func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			file, err := config.Load()

			require.NoError(t, err)
			require.Equal(t, map[string]config.Entry{}, entriesOf(t, file))
		})

		t.Run("a config file with the key catalog stops, and the message names the file and catalog add", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			path := writeConfig(t, configHome, "catalog: /code/catalog\n")

			_, err := config.Load()

			require.EqualError(t, err, path+" has the key catalog, which feature-compose does not read: remove the key, and then add the catalog with feature-compose catalog add <name> <folder>")
		})

		t.Run("a catalog name with a slash stops, and the message names the file and the rule of a name", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			path := writeConfig(t, configHome, "catalogs:\n  team/a:\n    source: /code/team-catalog\n")

			_, err := config.Load()

			require.EqualError(t, err, "read "+path+`: the name "team/a" of the catalog must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'`)
		})

		t.Run("a catalog with no source stops, and the message names the file and the catalog", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			path := writeConfig(t, configHome, "catalogs:\n  team:\n    path: catalog\n")

			_, err := config.Load()

			require.EqualError(t, err, "read "+path+": the catalog team has no source")
		})

		t.Run("two catalog names that differ only in case stop, because they share one clone folder on a file system that ignores case", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			path := writeConfig(t, configHome, "catalogs:\n  Team:\n    source: /code/a\n  team:\n    source: /code/b\n")

			_, err := config.Load()

			require.EqualError(t, err, "read "+path+": the config already has the catalog Team, whose name differs from team only in case: two such names share one clone folder on a file system that ignores case")
		})

		t.Run("a config file that is not valid YAML stops, and the message names the file", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			path := writeConfig(t, configHome, "catalogs: [\n")

			_, err := config.Load()

			require.ErrorContains(t, err, "read "+path)
		})
	})

	t.Run("save", func(t *testing.T) {
		t.Run("writes the catalogs that a load then gives, and makes the folder of the config file", func(t *testing.T) {
			file := loadedWith(t, filepath.Join(t.TempDir(), "missing"), map[string]config.Entry{
				"hpcsc": {Source: "git@github.com:hpcsc/feature-compose.git", Path: "catalog", Ref: "v1.0.0"},
				"team":  {Source: "/code/team-catalog"},
			})

			require.NoError(t, file.Save())
			loaded, err := config.Load()

			require.NoError(t, err)
			require.Equal(t, entriesOf(t, file), entriesOf(t, loaded))
		})

		t.Run("writes each catalog in the order of the names, with no empty field", func(t *testing.T) {
			configHome := t.TempDir()
			file := loadedWith(t, configHome, map[string]config.Entry{
				"team":  {Source: "/code/team-catalog"},
				"hpcsc": {Source: "https://github.com/hpcsc/feature-compose", Path: "catalog"},
			})

			require.NoError(t, file.Save())

			contents, err := os.ReadFile(filepath.Join(configHome, "feature-compose", "config.yml"))
			require.NoError(t, err)
			require.Equal(t, "catalogs:\n  hpcsc:\n    source: https://github.com/hpcsc/feature-compose\n    path: catalog\n  team:\n    source: /code/team-catalog\n", string(contents))
		})

		t.Run("writes the target of a config file that is a symlink, and keeps the symlink", func(t *testing.T) {
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			target := filepath.Join(t.TempDir(), "dotfiles", "config.yml")
			require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
			require.NoError(t, os.WriteFile(target, []byte("catalogs: {}\n"), 0o644))
			link := filepath.Join(configHome, "feature-compose", "config.yml")
			require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
			require.NoError(t, os.Symlink(target, link))

			require.NoError(t, loadedWith(t, configHome, map[string]config.Entry{"team": {Source: "/code/team-catalog"}}).Save())

			linked, err := os.Readlink(link)
			require.NoError(t, err)
			contents, err := os.ReadFile(target)
			require.NoError(t, err)
			require.Equal(t, []string{target, "catalogs:\n  team:\n    source: /code/team-catalog\n"}, []string{linked, string(contents)})
		})
	})

	t.Run("with catalog", func(t *testing.T) {
		t.Run("gives a file with the catalog and its clean path, and the file before it keeps its catalogs", func(t *testing.T) {
			file := loadedWith(t, t.TempDir(), nil)

			changed, err := file.WithCatalog("team", config.Entry{Source: "/code/team-catalog", Path: "./catalog/"})

			require.NoError(t, err)
			require.Equal(t, []map[string]config.Entry{{}, {"team": {Source: "/code/team-catalog", Path: "catalog"}}}, []map[string]config.Entry{entriesOf(t, file), entriesOf(t, changed)})
		})

		for name, test := range map[string]struct {
			name    string
			entry   config.Entry
			message string
		}{
			"a name with a slash":         {name: "team/a", entry: config.Entry{Source: "/code/a"}, message: `the name "team/a" of the catalog must start with a letter or a digit, and hold only letters, digits, '.', '_' and '-'`},
			"no source":                   {name: "team", entry: config.Entry{Path: "catalog"}, message: "the catalog team has no source"},
			"a source that starts with -": {name: "team", entry: config.Entry{Source: "--upload-pack=x@host:path"}, message: "the source --upload-pack=x@host:path of the catalog team starts with '-', so git reads it as an option"},
			"a path outside the source":   {name: "team", entry: config.Entry{Source: "/code/a", Path: "../other"}, message: "the path ../other is outside the source: give a folder in the source"},
			"a name that differs from a name of the file only in case": {name: "Team", entry: config.Entry{Source: "/code/b"}, message: "the config already has the catalog team, whose name differs from Team only in case: two such names share one clone folder on a file system that ignores case"},
		} {
			t.Run(name+" stops, and the message says why", func(t *testing.T) {
				file := loadedWith(t, t.TempDir(), map[string]config.Entry{"team": {Source: "/code/team-catalog"}})

				_, err := file.WithCatalog(test.name, test.entry)

				require.EqualError(t, err, test.message)
			})
		}
	})

	t.Run("without catalog", func(t *testing.T) {
		t.Run("gives a file with no such catalog, and the file before it keeps its catalogs", func(t *testing.T) {
			file := loadedWith(t, t.TempDir(), map[string]config.Entry{"team": {Source: "/code/team-catalog"}})

			changed := file.WithoutCatalog("team")

			require.Equal(t, []map[string]config.Entry{{"team": {Source: "/code/team-catalog"}}, {}}, []map[string]config.Entry{entriesOf(t, file), entriesOf(t, changed)})
		})
	})

	t.Run("names", func(t *testing.T) {
		t.Run("gives the name of each catalog in order", func(t *testing.T) {
			file := loadedWith(t, t.TempDir(), map[string]config.Entry{"work": {Source: "/code/work"}, "team": {Source: "/code/team"}})

			require.Equal(t, []string{"team", "work"}, file.Names())
		})
	})

	t.Run("entry", func(t *testing.T) {
		t.Run("gives the entry of the name", func(t *testing.T) {
			file := loadedWith(t, t.TempDir(), map[string]config.Entry{"team": {Source: "/code/team-catalog"}})

			entry, err := file.Entry("team")

			require.NoError(t, err)
			require.Equal(t, config.Entry{Source: "/code/team-catalog"}, entry)
		})

		t.Run("an unknown name stops, and the message lists the names of the file", func(t *testing.T) {
			file := loadedWith(t, t.TempDir(), map[string]config.Entry{"team": {Source: "/code/team-catalog"}, "work": {Source: "/code/work"}})

			_, err := file.Entry("nope")

			require.EqualError(t, err, "the config file "+file.Path()+" has no catalog nope: it has team, work")
		})

		t.Run("an unknown name in a file with no catalog names catalog add", func(t *testing.T) {
			file := loadedWith(t, t.TempDir(), nil)

			_, err := file.Entry("nope")

			require.EqualError(t, err, "the config file "+file.Path()+" has no catalog nope, and no other catalog: add one with feature-compose catalog add <name> <source>")
		})
	})

	t.Run("source folder", func(t *testing.T) {
		t.Run("a source that starts with ~/ is in the home folder", func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			file, err := config.Load()
			require.NoError(t, err)

			folder, err := file.SourceFolder(config.Entry{Source: "~/code/catalog"})

			require.NoError(t, err)
			require.Equal(t, filepath.Join(home, "code", "catalog"), folder)
		})

		t.Run("a relative source is relative to the folder of the config file, and not to the working folder", func(t *testing.T) {
			configHome := t.TempDir()
			t.Chdir(t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", configHome)
			file, err := config.Load()
			require.NoError(t, err)

			folder, err := file.SourceFolder(config.Entry{Source: "catalog"})

			require.NoError(t, err)
			require.Equal(t, filepath.Join(configHome, "feature-compose", "catalog"), folder)
		})

		t.Run("a git URL has no folder", func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			file, err := config.Load()
			require.NoError(t, err)

			_, err = file.SourceFolder(config.Entry{Source: "https://github.com/hpcsc/feature-compose"})

			require.EqualError(t, err, "the source https://github.com/hpcsc/feature-compose is a git URL, not a folder")
		})
	})
}
