//go:build unit

package catalogscreen_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/catalogscreen"
	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/hpcsc/feature-compose/internal/index"
	"github.com/hpcsc/feature-compose/internal/registry"
	"github.com/stretchr/testify/require"
)

const starterYML = "summary: A service.\nfolders:\n  app:\n    path: .\n"

type source struct {
	dir string
}

func newSource(t *testing.T) source {
	t.Helper()
	s := source{dir: filepath.Join(t.TempDir(), "source")}
	s.git(t, "init", "--quiet", "--initial-branch=main", s.dir)
	s.commit(t, "service")
	return s
}

func (s source) url() string {
	return "file://" + s.dir
}

func (s source) commit(t *testing.T, starterName string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(s.dir, "starters", starterName), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(s.dir, "starters", starterName, "starter.yml"), []byte(starterYML), 0o644))
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

func isolated(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func opened(t *testing.T) *registry.Instance {
	t.Helper()
	r, err := registry.Open()
	require.NoError(t, err)
	return r
}

func added(t *testing.T, name string, entry config.Entry) catalog.Origin {
	t.Helper()
	origin, err := opened(t).Add(context.Background(), name, entry, catalog.NoSteps{})
	require.NoError(t, err)
	return origin
}

func indexWith(t *testing.T, status int, contents string) func(context.Context) (index.File, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(contents))
	}))
	t.Cleanup(server.Close)
	return func(ctx context.Context) (index.File, error) {
		return index.Fetch(ctx, http.DefaultClient, server.URL+"/catalogs.yml")
	}
}

func started(t *testing.T, fetch func(context.Context) (index.File, error)) catalogscreen.Model {
	t.Helper()
	m, err := catalogscreen.New(context.Background(), opened(t), fetch)
	require.NoError(t, err)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 240, Height: 24})
	return settled(t, next, m.Init())
}

func settled(t *testing.T, m tea.Model, cmd tea.Cmd) catalogscreen.Model {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if _, quit := msg.(tea.QuitMsg); quit {
			break
		}
		m, cmd = m.Update(msg)
	}
	return m.(catalogscreen.Model)
}

func pressed(t *testing.T, m catalogscreen.Model, keys ...string) catalogscreen.Model {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, cmd := m.Update(msg)
		m = settled(t, next, cmd)
	}
	return m
}

var escapeCode = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func linesOf(m catalogscreen.Model) []string {
	lines := strings.Split(escapeCode.ReplaceAllString(m.View(), ""), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return lines
}

func panesOf(m catalogscreen.Model) (list, details []string) {
	lines := linesOf(m)
	for _, line := range lines[2 : len(lines)-3] {
		left, right, _ := strings.Cut(line, "││")
		list = append(list, strings.TrimRight(strings.TrimPrefix(left, "│ "), " "))
		details = append(details, strings.TrimRight(strings.TrimSuffix(strings.TrimPrefix(right, " "), "│"), " "))
	}
	return withoutEmptyEnd(list), withoutEmptyEnd(details)
}

func statusOf(m catalogscreen.Model) string {
	lines := linesOf(m)
	return strings.TrimSpace(lines[len(lines)-2])
}

func withoutEmptyEnd(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func TestModel(t *testing.T) {
	const knownIndex = "catalogs:\n  other:\n    summary: Another catalog.\n    source: https://example.com/other.git\n    path: catalog\n"

	t.Run("show", func(t *testing.T) {
		t.Run("lists the catalogs of the config, and then the known catalogs that the config does not have", func(t *testing.T) {
			isolated(t)
			src := newSource(t)
			added(t, "hpcsc", config.Entry{Source: src.url()})

			list, _ := panesOf(started(t, indexWith(t, http.StatusOK, knownIndex+"  hpcsc:\n    summary: Added.\n    source: https://example.com/hpcsc.git\n")))

			require.Equal(t, []string{"▌ hpcsc", "", "Known catalogs", "  other"}, list)
		})

		t.Run("the details of a catalog give its source, its path, its ref and its commit", func(t *testing.T) {
			isolated(t)
			src := newSource(t)
			head := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit

			_, details := panesOf(started(t, indexWith(t, http.StatusOK, knownIndex)))

			require.Equal(t, []string{"source  " + src.url(), "path    the root of the source", "ref     the default branch", "commit  " + head}, details)
		})

		t.Run("the details of a known catalog give its summary and its source, and say that enter adds it", func(t *testing.T) {
			isolated(t)

			_, details := panesOf(started(t, indexWith(t, http.StatusOK, knownIndex)))

			require.Equal(t, []string{"Another catalog.", "", "source  https://example.com/other.git", "path    catalog", "ref     the default branch", "", "Press enter to add it to the config file."}, details)
		})

		t.Run("an index that the tool cannot get says so, and the catalogs of the config still show", func(t *testing.T) {
			isolated(t)
			added(t, "hpcsc", config.Entry{Source: newSource(t).url()})

			list, _ := panesOf(started(t, indexWith(t, http.StatusNotFound, "")))

			require.Equal(t, []string{"▌ hpcsc", "", "Known catalogs", "The tool cannot", "get the index."}, list)
		})

		t.Run("a terminal that is too small shows only a message", func(t *testing.T) {
			isolated(t)
			m := started(t, indexWith(t, http.StatusOK, knownIndex))

			next, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 5})

			require.Equal(t, "The terminal is too small for catalog. Make it at least 40 columns wide and 8 rows high.", strings.Join(linesOf(next.(catalogscreen.Model)), " "))
		})
	})

	t.Run("sync", func(t *testing.T) {
		t.Run("s gets the newest commit of the selected catalog, and the status gives the old and the new commit", func(t *testing.T) {
			isolated(t)
			src := newSource(t)
			old := added(t, "hpcsc", config.Entry{Source: src.url()}).Commit
			newest := src.commit(t, "worker")

			m := pressed(t, started(t, indexWith(t, http.StatusOK, knownIndex)), "s")

			_, details := panesOf(m)
			require.Equal(t, "hpcsc: from "+old[:7]+" to "+newest[:7], statusOf(m))
			require.Equal(t, "commit  "+newest, details[3])
		})

		t.Run("S syncs each catalog, and the status gives each result", func(t *testing.T) {
			isolated(t)
			added(t, "first", config.Entry{Source: newSource(t).url()})
			team := newSource(t)
			added(t, "team", config.Entry{Source: team.dir})

			m := pressed(t, started(t, indexWith(t, http.StatusOK, knownIndex)), "S")

			require.Contains(t, statusOf(m), " · team: a local folder, so sync gets nothing")
		})
	})

	t.Run("ref", func(t *testing.T) {
		t.Run("r lists the default branch, the branches and the tags, and enter sets the chosen ref", func(t *testing.T) {
			isolated(t)
			src := newSource(t)
			tagged := src.git(t, "-C", src.dir, "rev-parse", "HEAD")
			src.git(t, "-C", src.dir, "tag", "v1")
			src.commit(t, "worker")
			added(t, "hpcsc", config.Entry{Source: src.url()})
			m := pressed(t, started(t, indexWith(t, http.StatusOK, knownIndex)), "r")
			_, refs := panesOf(m)

			chosen := pressed(t, m, "j", "j", "enter")

			require.Equal(t, []string{"▌ the default branch", "  main", "  v1"}, refs[len(refs)-3:])
			require.Equal(t, "The catalog hpcsc has the ref v1, at the commit "+tagged[:7]+".", statusOf(chosen))
		})

		t.Run("esc closes the list of refs, and the ref stays", func(t *testing.T) {
			isolated(t)
			added(t, "hpcsc", config.Entry{Source: newSource(t).url()})

			m := pressed(t, started(t, indexWith(t, http.StatusOK, knownIndex)), "r", "j", "esc")

			_, details := panesOf(m)
			require.Equal(t, "ref     the default branch", details[2])
		})
	})

	t.Run("remove", func(t *testing.T) {
		t.Run("x asks first, and y removes the catalog", func(t *testing.T) {
			isolated(t)
			added(t, "hpcsc", config.Entry{Source: newSource(t).url()})
			m := pressed(t, started(t, indexWith(t, http.StatusOK, knownIndex)), "x")
			asked := statusOf(m)

			removed := pressed(t, m, "y")

			list, _ := panesOf(removed)
			require.Equal(t, []string{"Remove the catalog hpcsc? Press y to remove it, or another key to keep it.", "Removed the catalog hpcsc."}, []string{asked, statusOf(removed)})
			require.Equal(t, []string{"The config file", "has no catalog."}, list[:2])
		})

		t.Run("another key than y keeps the catalog", func(t *testing.T) {
			isolated(t)
			added(t, "hpcsc", config.Entry{Source: newSource(t).url()})

			m := pressed(t, started(t, indexWith(t, http.StatusOK, knownIndex)), "x", "n")

			list, _ := panesOf(m)
			require.Equal(t, []string{"Kept the catalog hpcsc.", "▌ hpcsc"}, []string{statusOf(m), list[0]})
		})
	})

	t.Run("add", func(t *testing.T) {
		t.Run("enter at a known catalog adds it to the config file, and it moves to the catalogs of the config", func(t *testing.T) {
			isolated(t)
			src := newSource(t)
			head := src.git(t, "-C", src.dir, "rev-parse", "HEAD")

			m := pressed(t, started(t, indexWith(t, http.StatusOK, "catalogs:\n  team:\n    summary: The catalog of the team.\n    source: "+src.url()+"\n")), "enter")

			list, _ := panesOf(m)
			require.Equal(t, "Added the catalog team at the commit "+head[:7]+".", statusOf(m))
			require.Equal(t, []string{"▌ team", "", "Known catalogs", "The config file", "has each known", "catalog."}, list)
		})

		t.Run("a known catalog that git cannot read shows the error, and the config gets no catalog", func(t *testing.T) {
			isolated(t)
			missing := "file://" + filepath.Join(t.TempDir(), "missing")

			m := pressed(t, started(t, indexWith(t, http.StatusOK, "catalogs:\n  team:\n    summary: The catalog of the team.\n    source: "+missing+"\n")), "enter")

			list, _ := panesOf(m)
			require.True(t, strings.HasPrefix(statusOf(m), "git cannot read "+missing), statusOf(m))
			require.Equal(t, []string{"The config file", "has no catalog."}, list[:2])
		})
	})
}
