//go:build unit

package registry_test

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/hpcsc/feature-compose/internal/registry"
	"github.com/stretchr/testify/require"
)

const starterYML = "summary: A service.\nfolders:\n  app:\n    path: .\n"

type source struct {
	dir string
}

func newSource(t *testing.T, files map[string]string) source {
	t.Helper()
	s := source{dir: filepath.Join(t.TempDir(), "source")}
	s.git(t, "init", "--quiet", "--initial-branch=main", s.dir)
	s.commit(t, files)
	return s
}

func (s source) url() string {
	return "file://" + s.dir
}

func (s source) commit(t *testing.T, files map[string]string) string {
	t.Helper()
	for path, contents := range files {
		full := filepath.Join(s.dir, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(contents), 0o644))
	}
	s.git(t, "-C", s.dir, "add", "--all")
	s.git(t, "-C", s.dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "--quiet", "--message", "change")
	return s.git(t, "-C", s.dir, "rev-parse", "HEAD")
}

func (s source) git(t *testing.T, args ...string) string {
	t.Helper()
	printed, err := exec.Command("git", args...).CombinedOutput()
	require.NoError(t, err, string(printed))
	return strings.TrimSpace(string(printed))
}

type folders struct {
	config string
	data   string
}

func (f folders) clone(name string) string {
	return filepath.Join(f.data, "feature-compose", "catalogs", name)
}

func (f folders) configFile(t *testing.T) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(f.config, "feature-compose", "config.yml"))
	require.NoError(t, err)
	return string(contents)
}

func isolated(t *testing.T) folders {
	t.Helper()
	f := folders{config: t.TempDir(), data: t.TempDir()}
	t.Setenv("XDG_CONFIG_HOME", f.config)
	t.Setenv("XDG_DATA_HOME", f.data)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return f
}

func opened(t *testing.T) *registry.Instance {
	t.Helper()
	r, err := registry.Open()
	require.NoError(t, err)
	return r
}

func added(t *testing.T, name string, entry config.Entry) catalog.Origin {
	t.Helper()
	origin, err := opened(t).Add(context.Background(), name, entry)
	require.NoError(t, err)
	return origin
}

func starterNames(t *testing.T, name string) []string {
	t.Helper()
	c, err := opened(t).Catalog(context.Background(), name)
	require.NoError(t, err)
	definitions, err := catalog.NewCatalogs(c).Definitions()
	require.NoError(t, err)
	return slices.Sorted(maps.Keys(definitions.Starters))
}

func listed(t *testing.T, name string) catalog.Origin {
	t.Helper()
	origins, err := opened(t).List(context.Background())
	require.NoError(t, err)
	for _, origin := range origins {
		if origin.Name == name {
			return origin
		}
	}
	require.FailNow(t, "the config has no catalog "+name)
	return catalog.Origin{}
}

func readOnly(t *testing.T, folder string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(folder, 0o755))
	require.NoError(t, os.Chmod(folder, 0o555))
	t.Cleanup(func() { _ = os.Chmod(folder, 0o755) })
}

func writtenConfig(t *testing.T, f folders, contents string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(f.config, "feature-compose"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(f.config, "feature-compose", "config.yml"), []byte(contents), 0o644))
}

func namesOf(synced []registry.Synced) []string {
	var names []string
	for _, s := range synced {
		names = append(names, s.Name)
	}
	return names
}

func pointer(value string) *string {
	return &value
}

func TestRegistry(t *testing.T) {
	ctx := context.Background()

	t.Run("add", func(t *testing.T) {
		t.Run("a git URL gets a clone at the default branch, and the config records the entry", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"catalog/starters/service/starter.yml": starterYML})
			head := src.git(t, "-C", src.dir, "rev-parse", "HEAD")

			origin := added(t, "hpcsc", config.Entry{Source: src.url(), Path: "catalog"})

			require.Equal(t, catalog.Origin{Name: "hpcsc", Source: src.url(), Path: "catalog", Commit: head}, origin)
			require.Equal(t, "catalogs:\n  hpcsc:\n    source: "+src.url()+"\n    path: catalog\n", f.configFile(t))
			require.Equal(t, []string{"hpcsc/service"}, starterNames(t, "hpcsc"))
		})

		t.Run("a ref that names a tag gets the commit of the tag", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/old/starter.yml": starterYML})
			tagged := src.git(t, "-C", src.dir, "rev-parse", "HEAD")
			src.git(t, "-C", src.dir, "tag", "v1.0.0")
			src.commit(t, map[string]string{"starters/new/starter.yml": starterYML})

			origin := added(t, "hpcsc", config.Entry{Source: src.url(), Ref: "v1.0.0"})

			require.Equal(t, catalog.Origin{Name: "hpcsc", Source: src.url(), Ref: "v1.0.0", Commit: tagged}, origin)
			require.Equal(t, []string{"hpcsc/old"}, starterNames(t, "hpcsc"))
		})

		t.Run("a local folder needs no clone, and the config records its full path", func(t *testing.T) {
			f := isolated(t)
			work := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(work, "team-catalog", "starters", "worker"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(work, "team-catalog", "starters", "worker", "starter.yml"), []byte(starterYML), 0o644))
			t.Chdir(work)

			origin := added(t, "team", config.Entry{Source: "team-catalog"})

			require.Equal(t, catalog.Origin{Name: "team", Source: filepath.Join(work, "team-catalog")}, origin)
			require.NoDirExists(t, f.clone("team"))
			require.Equal(t, []string{"team/worker"}, starterNames(t, "team"))
		})

		t.Run("a name that the config already has stops, and the message names set and remove", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: src.url()})

			_, err := opened(t).Add(ctx, "hpcsc", config.Entry{Source: src.url()})

			require.EqualError(t, err, "the config already has the catalog hpcsc: change it with feature-compose catalog set, or remove it with feature-compose catalog remove")
		})

		t.Run("a git URL that git cannot read stops with the message about git clone, and leaves no clone and no config file", func(t *testing.T) {
			f := isolated(t)
			missing := "file://" + filepath.Join(t.TempDir(), "missing")

			_, err := opened(t).Add(ctx, "hpcsc", config.Entry{Source: missing})

			require.ErrorContains(t, err, "git cannot read "+missing+": check that git clone "+missing+" works\n")
			require.NoDirExists(t, f.clone("hpcsc"))
			require.NoFileExists(t, filepath.Join(f.config, "feature-compose", "config.yml"))
		})

		t.Run("a source with no folder starters at the path stops, the message names --path, and the clone is gone", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"catalog/starters/service/starter.yml": starterYML})

			_, err := opened(t).Add(ctx, "hpcsc", config.Entry{Source: src.url()})

			require.EqualError(t, err, "the source "+src.url()+" has no catalog at its root, because it has no folder starters: give the folder of the catalog in the source with --path")
			require.NoDirExists(t, f.clone("hpcsc"))
		})

		t.Run("a ref that the source does not have stops, and the config gets no entry", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})

			_, err := opened(t).Add(ctx, "hpcsc", config.Entry{Source: src.url(), Ref: "v9"})

			require.EqualError(t, err, "the source "+src.url()+" has no ref v9")
			require.NoDirExists(t, f.clone("hpcsc"))
			require.NoFileExists(t, filepath.Join(f.config, "feature-compose", "config.yml"))
		})

		t.Run("a ref with a local folder stops", func(t *testing.T) {
			isolated(t)
			folder := t.TempDir()

			_, err := opened(t).Add(ctx, "team", config.Entry{Source: folder, Ref: "main"})

			require.EqualError(t, err, "the source "+folder+" is a local folder, so it has no ref: give --ref only with a git URL")
		})

		t.Run("a local folder that does not exist stops, and the config gets no entry", func(t *testing.T) {
			f := isolated(t)
			missing := filepath.Join(t.TempDir(), "missing")

			_, err := opened(t).Add(ctx, "team", config.Entry{Source: missing})

			require.EqualError(t, err, "the source "+missing+" is not a git URL and not a folder")
			require.NoFileExists(t, filepath.Join(f.config, "feature-compose", "config.yml"))
		})

		t.Run("a path with no folder starters stops, the message names the path and --path, and the clone is gone", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"catalog/starters/service/starter.yml": starterYML})

			_, err := opened(t).Add(ctx, "hpcsc", config.Entry{Source: src.url(), Path: "other"})

			require.EqualError(t, err, "the source "+src.url()+" has no catalog at the path other, because that path has no folder starters: give the folder of the catalog in the source with --path")
			require.NoDirExists(t, f.clone("hpcsc"))
		})

		t.Run("a name that is not a catalog name stops, and the config gets no entry", func(t *testing.T) {
			f := isolated(t)

			_, err := opened(t).Add(ctx, "team/a", config.Entry{Source: t.TempDir()})

			require.ErrorContains(t, err, `the name "team/a" of the catalog must start with a letter or a digit`)
			require.NoFileExists(t, filepath.Join(f.config, "feature-compose", "config.yml"))
		})

		t.Run("a name that differs from a name of the config only in case stops, and the clone of the other catalog stays", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "team", config.Entry{Source: src.url()})

			_, err := opened(t).Add(ctx, "Team", config.Entry{Source: src.url()})

			require.EqualError(t, err, "the config already has the catalog team, whose name differs from Team only in case: two such names share one clone folder on a file system that ignores case")
			require.DirExists(t, filepath.Join(f.clone("team"), ".git"))
		})

		t.Run("a config file that cannot be saved stops, and the clone is gone", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			readOnly(t, filepath.Join(f.config, "feature-compose"))

			_, err := opened(t).Add(ctx, "hpcsc", config.Entry{Source: src.url()})

			require.ErrorContains(t, err, "write "+filepath.Join(f.config, "feature-compose", "config.yml"))
			require.NoDirExists(t, f.clone("hpcsc"))
		})

		t.Run("an empty source stops, and the config gets no entry", func(t *testing.T) {
			f := isolated(t)
			work := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(work, "starters"), 0o755))
			t.Chdir(work)

			_, err := opened(t).Add(ctx, "team", config.Entry{})

			require.EqualError(t, err, "the catalog team has no source")
			require.NoFileExists(t, filepath.Join(f.config, "feature-compose", "config.yml"))
		})

		t.Run("the message of a git URL that git cannot read holds no password or token", func(t *testing.T) {
			isolated(t)
			server := httptest.NewServer(http.NotFoundHandler())
			t.Cleanup(server.Close)
			source := strings.Replace(server.URL, "://", "://deploy:secret@", 1) + "/team/catalog.git"

			_, err := opened(t).Add(ctx, "team", config.Entry{Source: source})

			require.ErrorContains(t, err, "git cannot read "+server.URL+"/team/catalog.git")
			require.NotContains(t, err.Error(), "secret")
		})

		t.Run("a path outside the source stops", func(t *testing.T) {
			isolated(t)

			_, err := opened(t).Add(ctx, "team", config.Entry{Source: t.TempDir(), Path: "../other"})

			require.EqualError(t, err, "the path ../other is outside the source: give a folder in the source")
		})

		t.Run("with no XDG_DATA_HOME, the clone is in .local/share of the home folder", func(t *testing.T) {
			isolated(t)
			home := t.TempDir()
			t.Setenv("XDG_DATA_HOME", "")
			t.Setenv("HOME", home)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})

			added(t, "hpcsc", config.Entry{Source: src.url()})

			require.DirExists(t, filepath.Join(home, ".local", "share", "feature-compose", "catalogs", "hpcsc", ".git"))
		})

		t.Run("a change by another process between open and save stays in the config", func(t *testing.T) {
			isolated(t)
			first, second := opened(t), opened(t)

			_, firstErr := first.Add(ctx, "one", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})
			_, secondErr := second.Add(ctx, "two", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})

			require.NoError(t, errors.Join(firstErr, secondErr))
			require.Equal(t, []string{"one", "two"}, []string{listed(t, "one").Name, listed(t, "two").Name})
		})

		t.Run("stops at once while another process changes the catalogs", func(t *testing.T) {
			f := isolated(t)
			lockFolder := filepath.Join(f.data, "feature-compose", "catalogs")
			require.NoError(t, os.MkdirAll(lockFolder, 0o755))
			lock, err := os.OpenFile(filepath.Join(lockFolder, ".lock"), os.O_CREATE|os.O_RDWR, 0o644)
			require.NoError(t, err)
			t.Cleanup(func() { _ = lock.Close() })
			require.NoError(t, syscall.Flock(int(lock.Fd()), syscall.LOCK_EX))

			_, err = opened(t).Add(ctx, "team", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})

			require.EqualError(t, err, "another feature-compose catalog command changes the catalogs now: run this command again when it ends")
		})
	})

	t.Run("catalogs", func(t *testing.T) {
		t.Run("give each catalog of the config, with the full names of its starters", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			folder := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(folder, "starters", "worker"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(folder, "starters", "worker", "starter.yml"), []byte(starterYML), 0o644))
			added(t, "hpcsc", config.Entry{Source: src.url()})
			added(t, "team", config.Entry{Source: folder})

			catalogs, err := opened(t).Catalogs(ctx)
			require.NoError(t, err)
			definitions, err := catalogs.Definitions()

			require.NoError(t, err)
			require.Equal(t, []string{"hpcsc/service", "team/worker"}, slices.Sorted(maps.Keys(definitions.Starters)))
		})

		t.Run("a catalog whose clone is gone stops, and the message names catalog sync", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: src.url()})
			require.NoError(t, os.RemoveAll(f.clone("hpcsc")))

			_, err := opened(t).Catalogs(ctx)

			require.EqualError(t, err, "the catalog hpcsc has no clone in "+f.clone("hpcsc")+": run feature-compose catalog sync hpcsc")
		})

		t.Run("a config with no catalog stops, and the message names catalog add", func(t *testing.T) {
			f := isolated(t)

			_, err := opened(t).Catalogs(ctx)

			require.EqualError(t, err, "the config file "+filepath.Join(f.config, "feature-compose", "config.yml")+" has no catalog: add one with feature-compose catalog add <name> <source>")
		})
	})

	t.Run("catalog", func(t *testing.T) {
		t.Run("an unknown name stops, and the message lists the catalogs of the config", func(t *testing.T) {
			f := isolated(t)
			added(t, "team", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})

			_, err := opened(t).Catalog(ctx, "nope")

			require.EqualError(t, err, "the config file "+filepath.Join(f.config, "feature-compose", "config.yml")+" has no catalog nope: it has team")
		})

		t.Run("the origin of a source with a password or a token holds neither, so .feature-compose.yml holds neither", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML, "starters/service/files/app/README.md": "# {{ .Name }}\n"})
			added(t, "team", config.Entry{Source: src.url()})
			writtenConfig(t, f, "catalogs:\n  team:\n    source: https://deploy:secret@example.com/team/catalog.git\n")
			c, err := opened(t).Catalog(ctx, "team")
			require.NoError(t, err)
			starter, err := catalog.NewCatalogs(c).Starter("team/service")
			require.NoError(t, err)
			selection, err := starter.Select("my-repo", nil, nil, catalog.Answers{})
			require.NoError(t, err)
			output := filepath.Join(t.TempDir(), "my-repo")

			require.NoError(t, selection.MakeRepo(ctx, output, "v1.2.3", catalog.NoSteps{}))

			contents, err := os.ReadFile(filepath.Join(output, ".feature-compose.yml"))
			require.NoError(t, err)
			require.Contains(t, string(contents), "  source: https://example.com/team/catalog.git\n")
			require.NotContains(t, string(contents), "secret")
		})
	})

	t.Run("sync", func(t *testing.T) {
		t.Run("a branch gets the newest commit of the source, and gives the old and the new commit", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			old := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit
			newest := src.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})

			synced, err := opened(t).Sync(ctx, nil)

			require.NoError(t, err)
			require.Equal(t, []registry.Synced{{Name: "hpcsc", OldCommit: old, NewCommit: newest}}, synced)
			require.Equal(t, []string{"hpcsc/service", "hpcsc/worker"}, starterNames(t, "hpcsc"))
		})

		t.Run("a tag keeps its commit", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			src.git(t, "-C", src.dir, "tag", "v1.0.0")
			tagged := added(t, "hpcsc", config.Entry{Source: src.url(), Ref: "v1.0.0"}).Commit
			src.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})

			synced, err := opened(t).Sync(ctx, []string{"hpcsc"})

			require.NoError(t, err)
			require.Equal(t, []registry.Synced{{Name: "hpcsc", OldCommit: tagged, NewCommit: tagged}}, synced)
		})

		t.Run("a local folder gets nothing", func(t *testing.T) {
			isolated(t)
			added(t, "team", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})

			synced, err := opened(t).Sync(ctx, nil)

			require.NoError(t, err)
			require.Equal(t, []registry.Synced{{Name: "team", Local: true}}, synced)
		})

		t.Run("a clone that is gone is cloned again", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			head := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit
			require.NoError(t, os.RemoveAll(f.clone("hpcsc")))

			synced, err := opened(t).Sync(ctx, nil)

			require.NoError(t, err)
			require.Equal(t, []registry.Synced{{Name: "hpcsc", NewCommit: head}}, synced)
			require.Equal(t, []string{"hpcsc/service"}, starterNames(t, "hpcsc"))
		})

		t.Run("a source that git cannot read gives its error, and the other catalogs still sync", func(t *testing.T) {
			isolated(t)
			gone := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			kept := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			goneCommit := added(t, "gone", config.Entry{Source: gone.url()}).Commit
			keptCommit := added(t, "kept", config.Entry{Source: kept.url()}).Commit
			newest := kept.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})
			require.NoError(t, os.RemoveAll(gone.dir))

			synced, err := opened(t).Sync(ctx, nil)

			require.NoError(t, err)
			require.Equal(t, []string{"gone", goneCommit, ""}, []string{synced[0].Name, synced[0].OldCommit, synced[0].NewCommit})
			require.ErrorContains(t, synced[0].Err, "git cannot read "+gone.url())
			require.Equal(t, registry.Synced{Name: "kept", OldCommit: keptCommit, NewCommit: newest}, synced[1])
		})

		t.Run("a new commit with no catalog at the path gives an error, and the clone stays at its commit", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			old := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit
			src.git(t, "-C", src.dir, "rm", "--quiet", "-r", "starters")
			src.commit(t, map[string]string{"README.md": "no catalog\n"})

			synced, err := opened(t).Sync(ctx, nil)

			require.NoError(t, err)
			require.ErrorContains(t, synced[0].Err, "has no catalog at its root")
			require.Equal(t, old, listed(t, "hpcsc").Commit)
			require.Equal(t, []string{"hpcsc/service"}, starterNames(t, "hpcsc"))
		})

		t.Run("a clone that is gone and cannot check out its ref gives an error, and leaves no new clone", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			src.git(t, "-C", src.dir, "branch", "next")
			added(t, "hpcsc", config.Entry{Source: src.url(), Ref: "next"})
			require.NoError(t, os.RemoveAll(f.clone("hpcsc")))
			src.git(t, "-C", src.dir, "branch", "--delete", "next")

			synced, err := opened(t).Sync(ctx, nil)

			require.NoError(t, err)
			require.EqualError(t, synced[0].Err, "the source "+src.url()+" has no ref next")
			require.NoDirExists(t, f.clone("hpcsc"))
		})

		t.Run("an unknown name stops before a catalog syncs", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			old := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit
			src.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})

			_, err := opened(t).Sync(ctx, []string{"hpcsc", "nope"})

			require.ErrorContains(t, err, "has no catalog nope: it has hpcsc")
			require.Equal(t, old, listed(t, "hpcsc").Commit)
		})

		t.Run("syncs only the named catalogs", func(t *testing.T) {
			isolated(t)
			first := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			second := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "first", config.Entry{Source: first.url()})
			secondCommit := added(t, "second", config.Entry{Source: second.url()}).Commit
			second.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})

			synced, err := opened(t).Sync(ctx, []string{"first"})

			require.NoError(t, err)
			require.Equal(t, []string{"first"}, namesOf(synced))
			require.Equal(t, secondCommit, listed(t, "second").Commit)
		})

		t.Run("a new default branch of the source gives its newest commit", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: src.url()})
			src.git(t, "-C", src.dir, "checkout", "--quiet", "-b", "trunk")
			trunk := src.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})
			src.git(t, "-C", src.dir, "checkout", "--quiet", "main")
			src.git(t, "-C", src.dir, "symbolic-ref", "HEAD", "refs/heads/trunk")

			synced, err := opened(t).Sync(ctx, nil)

			require.NoError(t, err)
			require.Equal(t, trunk, synced[0].NewCommit)
		})

		t.Run("a source that the config file changes by hand gives the newest commit of the new source", func(t *testing.T) {
			f := isolated(t)
			old := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			moved := newSource(t, map[string]string{"starters/worker/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: old.url()})
			writtenConfig(t, f, "catalogs:\n  hpcsc:\n    source: "+moved.url()+"\n")

			synced, err := opened(t).Sync(ctx, nil)

			require.NoError(t, err)
			require.Equal(t, moved.git(t, "-C", moved.dir, "rev-parse", "HEAD"), synced[0].NewCommit)
			require.Equal(t, []string{"hpcsc/worker"}, starterNames(t, "hpcsc"))
		})
	})

	t.Run("set", func(t *testing.T) {
		t.Run("a ref checks out the commit of the new ref, and the config records it", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: src.url()})
			src.git(t, "-C", src.dir, "checkout", "--quiet", "-b", "next")
			next := src.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})

			origin, err := opened(t).Set(ctx, "hpcsc", registry.Change{Ref: pointer("next")})

			require.NoError(t, err)
			require.Equal(t, catalog.Origin{Name: "hpcsc", Source: src.url(), Ref: "next", Commit: next}, origin)
			require.Equal(t, "catalogs:\n  hpcsc:\n    source: "+src.url()+"\n    ref: next\n", f.configFile(t))
		})

		t.Run("a ref that the source does not have stops, and the entry and the clone stay as they were", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			old := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit
			before := f.configFile(t)

			_, err := opened(t).Set(ctx, "hpcsc", registry.Change{Ref: pointer("v9")})

			require.EqualError(t, err, "the source "+src.url()+" has no ref v9")
			require.Equal(t, []string{before, old}, []string{f.configFile(t), listed(t, "hpcsc").Commit})
		})

		t.Run("a ref with no catalog at the path stops, and the clone goes back to its commit", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			old := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit
			src.git(t, "-C", src.dir, "checkout", "--quiet", "-b", "empty")
			src.git(t, "-C", src.dir, "rm", "--quiet", "-r", "starters")
			src.commit(t, map[string]string{"README.md": "no catalog\n"})

			_, err := opened(t).Set(ctx, "hpcsc", registry.Change{Ref: pointer("empty")})

			require.ErrorContains(t, err, "has no catalog at its root")
			require.Equal(t, catalog.Origin{Name: "hpcsc", Source: src.url(), Commit: old}, listed(t, "hpcsc"))
			require.Equal(t, []string{"hpcsc/service"}, starterNames(t, "hpcsc"))
		})

		t.Run("a path of a local folder changes the folder of the catalog", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML, "other/starters/worker/starter.yml": starterYML})
			added(t, "team", config.Entry{Source: src.dir})

			origin, err := opened(t).Set(ctx, "team", registry.Change{Path: pointer("other/")})

			require.NoError(t, err)
			require.Equal(t, catalog.Origin{Name: "team", Source: src.dir, Path: "other"}, origin)
			require.Equal(t, []string{"team/worker"}, starterNames(t, "team"))
		})

		t.Run("an empty ref goes back to the default branch", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			src.git(t, "-C", src.dir, "tag", "v1")
			newest := src.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: src.url(), Ref: "v1"})

			origin, err := opened(t).Set(ctx, "hpcsc", registry.Change{Ref: pointer("")})

			require.NoError(t, err)
			require.Equal(t, catalog.Origin{Name: "hpcsc", Source: src.url(), Commit: newest}, origin)
			require.Equal(t, "catalogs:\n  hpcsc:\n    source: "+src.url()+"\n", f.configFile(t))
		})

		t.Run("a path of a local folder that the config gives relative to its own folder keeps that source", func(t *testing.T) {
			f := isolated(t)
			catalogDir := filepath.Join(f.config, "feature-compose", "team-catalog")
			require.NoError(t, os.MkdirAll(filepath.Join(catalogDir, "other", "starters", "worker"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(catalogDir, "other", "starters", "worker", "starter.yml"), []byte(starterYML), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(f.config, "feature-compose", "config.yml"), []byte("catalogs:\n  team:\n    source: team-catalog\n"), 0o644))
			t.Chdir(t.TempDir())

			origin, err := opened(t).Set(ctx, "team", registry.Change{Path: pointer("other")})

			require.NoError(t, err)
			require.Equal(t, catalog.Origin{Name: "team", Source: "team-catalog", Path: "other"}, origin)
			require.Equal(t, []string{"team/worker"}, starterNames(t, "team"))
		})

		t.Run("an unknown name stops, and the message lists the catalogs", func(t *testing.T) {
			isolated(t)
			added(t, "team", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})

			_, err := opened(t).Set(ctx, "nope", registry.Change{Ref: pointer("main")})

			require.ErrorContains(t, err, "has no catalog nope: it has team")
		})

		t.Run("a config file that cannot be saved stops, and the clone goes back to its commit", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			old := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit
			src.git(t, "-C", src.dir, "checkout", "--quiet", "-b", "next")
			src.commit(t, map[string]string{"starters/worker/starter.yml": starterYML})
			readOnly(t, filepath.Join(f.config, "feature-compose"))

			_, err := opened(t).Set(ctx, "hpcsc", registry.Change{Ref: pointer("next")})

			require.ErrorContains(t, err, "write "+filepath.Join(f.config, "feature-compose", "config.yml"))
			require.Equal(t, catalog.Origin{Name: "hpcsc", Source: src.url(), Commit: old}, listed(t, "hpcsc"))
		})

		t.Run("a ref of a local folder stops", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "team", config.Entry{Source: src.dir})

			_, err := opened(t).Set(ctx, "team", registry.Change{Ref: pointer("main")})

			require.EqualError(t, err, "the source "+src.dir+" is a local folder, so it has no ref: give --ref only with a git URL")
		})
	})

	t.Run("remove", func(t *testing.T) {
		t.Run("deletes the entry and the clone", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: src.url()})

			err := opened(t).Remove("hpcsc")

			require.NoError(t, err)
			require.NoDirExists(t, f.clone("hpcsc"))
			require.Equal(t, "catalogs: {}\n", f.configFile(t))
		})

		t.Run("an unknown name stops, and the message lists the catalogs", func(t *testing.T) {
			isolated(t)
			added(t, "team", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})

			err := opened(t).Remove("nope")

			require.ErrorContains(t, err, "has no catalog nope: it has team")
		})

		t.Run("a config file that cannot be saved stops, and the clone stays", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: src.url()})
			readOnly(t, filepath.Join(f.config, "feature-compose"))

			err := opened(t).Remove("hpcsc")

			require.ErrorContains(t, err, "write "+filepath.Join(f.config, "feature-compose", "config.yml"))
			require.DirExists(t, filepath.Join(f.clone("hpcsc"), ".git"))
		})

		t.Run("of a local folder keeps the folder", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "team", config.Entry{Source: src.dir})

			err := opened(t).Remove("team")

			require.NoError(t, err)
			require.FileExists(t, filepath.Join(src.dir, "starters", "service", "starter.yml"))
		})
	})

	t.Run("folder", func(t *testing.T) {
		t.Run("with no name, gives the folder of the one catalog, with its path", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"catalog/starters/service/starter.yml": starterYML})
			added(t, "team", config.Entry{Source: src.dir, Path: "catalog"})

			folder, err := opened(t).Folder("")

			require.NoError(t, err)
			require.Equal(t, filepath.Join(src.dir, "catalog"), folder)
		})

		t.Run("an unknown name stops, and the message lists the catalogs", func(t *testing.T) {
			isolated(t)
			added(t, "team", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})

			_, err := opened(t).Folder("nope")

			require.ErrorContains(t, err, "has no catalog nope: it has team")
		})

		t.Run("a catalog from a git URL stops, because the next sync removes each change in the clone", func(t *testing.T) {
			isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			added(t, "hpcsc", config.Entry{Source: src.url()})

			_, err := opened(t).Folder("hpcsc")

			require.EqualError(t, err, "the catalog hpcsc is a clone of "+src.url()+", and the next catalog sync removes each change in the clone: give --catalog <folder> with a clone of your own")
		})

		t.Run("with no name and more than one catalog, stops and lists the names", func(t *testing.T) {
			f := isolated(t)
			added(t, "team", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})
			added(t, "work", config.Entry{Source: newSource(t, map[string]string{"starters/service/starter.yml": starterYML}).dir})

			_, err := opened(t).Folder("")

			require.EqualError(t, err, "the config file "+filepath.Join(f.config, "feature-compose", "config.yml")+" has more than one catalog: give one with --catalog <name>, one of team, work")
		})

		t.Run("with no name and no catalog, stops and names --catalog", func(t *testing.T) {
			f := isolated(t)

			_, err := opened(t).Folder("")

			require.EqualError(t, err, "the config file "+filepath.Join(f.config, "feature-compose", "config.yml")+" has no catalog: give the folder of the catalog with --catalog <folder>")
		})
	})

	t.Run("list", func(t *testing.T) {
		for source, recorded := range map[string]string{
			"https://x-access-token:secret@github.com/team/catalog.git": "https://github.com/team/catalog.git",
			"https://secret@github.com/team/catalog.git":                "https://github.com/team/catalog.git",
			"ssh://git:secret@github.com/team/catalog.git":              "ssh://git@github.com/team/catalog.git",
			"git@github.com:team/catalog.git":                           "git@github.com:team/catalog.git",
		} {
			t.Run("gives the source "+recorded+" with no password or token", func(t *testing.T) {
				f := isolated(t)
				require.NoError(t, os.MkdirAll(filepath.Join(f.config, "feature-compose"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(f.config, "feature-compose", "config.yml"), []byte("catalogs:\n  team:\n    source: "+source+"\n"), 0o644))

				require.Equal(t, recorded, listed(t, "team").Source)
			})
		}

		t.Run("gives each catalog in the order of the names, with the commit of each clone", func(t *testing.T) {
			f := isolated(t)
			src := newSource(t, map[string]string{"starters/service/starter.yml": starterYML})
			folder := newSource(t, map[string]string{"starters/worker/starter.yml": starterYML}).dir
			head := added(t, "hpcsc", config.Entry{Source: src.url(), Ref: "main"}).Commit
			added(t, "gone", config.Entry{Source: src.url()})
			added(t, "team", config.Entry{Source: folder})
			require.NoError(t, os.RemoveAll(f.clone("gone")))

			origins, err := opened(t).List(ctx)

			require.NoError(t, err)
			require.Equal(t, []catalog.Origin{
				{Name: "gone", Source: src.url()},
				{Name: "hpcsc", Source: src.url(), Ref: "main", Commit: head},
				{Name: "team", Source: folder},
			}, origins)
		})
	})
}
