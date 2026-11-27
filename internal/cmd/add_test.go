//go:build unit

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/example_catalog"
	"github.com/stretchr/testify/require"
)

func TestAddCommand(t *testing.T) {
	noConfigFile := func() (string, error) { return "", errors.New("no config file") }
	runAdd := func(t *testing.T, configured func() (string, error), arguments ...string) (string, error) {
		t.Helper()
		var printed bytes.Buffer
		command := newAddCommand(configured)
		command.Writer, command.ErrWriter = &printed, io.Discard
		err := command.Run(context.Background(), append([]string{"add"}, arguments...))
		return printed.String(), err
	}
	exampleCatalog := func(t *testing.T) string {
		t.Helper()
		catalogDir := t.TempDir()
		require.NoError(t, example_catalog.Write(catalogDir))
		return catalogDir
	}
	builtLines := func(t *testing.T, catalogDir string) []string {
		t.Helper()
		var printed bytes.Buffer
		check := newCheckCommand(func() (catalog.Catalogs, error) { return catalog.Catalogs{}, errors.New("no config file") })
		check.Writer, check.ErrWriter = &printed, io.Discard
		require.NoError(t, check.Run(context.Background(), []string{"check", "--catalog", catalogDir, "--build"}), printed.String())
		var lines []string
		for _, line := range strings.Split(printed.String(), "\n") {
			if strings.HasPrefix(line, "[") {
				lines = append(lines, regexp.MustCompile(`^\[\d+/\d+\] | \(\d+s\)$`).ReplaceAllString(line, ""))
			}
		}
		return lines
	}
	definitionsOf := func(t *testing.T, catalogDir string) catalog.Definitions {
		t.Helper()
		definitions, err := catalog.New(os.DirFS(catalogDir)).Definitions()
		require.NoError(t, err)
		return definitions
	}

	t.Run("starter", func(t *testing.T) {
		t.Run("writes a starter with a summary to fill in and the folder app, whose repo passes check --build", func(t *testing.T) {
			catalogDir := exampleCatalog(t)

			_, err := runAdd(t, noConfigFile, "starter", "service", "--catalog", catalogDir)

			require.NoError(t, err)
			require.Equal(t, catalog.StarterDefinition{
				Summary:         "The starter service. Change this sentence to what a repo of this starter gets.",
				Folders:         map[string]catalog.FolderDefinition{"app": {Path: ".", AfterRender: []string{}, Test: []string{}}},
				Options:         map[string]catalog.OptionDefinition{},
				ExtensionPoints: map[string]catalog.ExtensionPointDefinition{},
			}, definitionsOf(t, catalogDir).Starters["service"])
			require.Contains(t, builtLines(t, catalogDir), "ok    service")
		})

		t.Run("makes a catalog of an empty folder", func(t *testing.T) {
			catalogDir := t.TempDir()

			_, err := runAdd(t, noConfigFile, "starter", "service", "--catalog", catalogDir)

			require.NoError(t, err)
			require.Equal(t, []string{"ok    service"}, builtLines(t, catalogDir))
		})

		t.Run("prints the files that it writes and the commands that try the starter, with the --catalog that the flag gives", func(t *testing.T) {
			catalogDir := t.TempDir()

			printed, err := runAdd(t, noConfigFile, "starter", "service", "--catalog", catalogDir)

			require.NoError(t, err)
			require.Equal(t, `Added the starter service to the catalog in `+catalogDir+`:
  starters/service/starter.yml
  starters/service/files/app/README.md
Fill in starter.yml, and then try the starter:
  feature-compose new --catalog `+catalogDir+` service my-repo --dry-run
  feature-compose check --catalog `+catalogDir+` --build
`, printed)
		})

		t.Run("with no --catalog, writes to the catalog folder of the config file, and prints the commands with no --catalog", func(t *testing.T) {
			catalogDir := t.TempDir()

			printed, err := runAdd(t, func() (string, error) { return catalogDir, nil }, "starter", "service")

			require.NoError(t, err)
			require.FileExists(t, filepath.Join(catalogDir, "starters", "service", "starter.yml"))
			require.Contains(t, printed, "\n  feature-compose new service my-repo --dry-run\n  feature-compose check --build\n")
		})
	})

	t.Run("feature", func(t *testing.T) {
		t.Run("writes a feature that targets each --target, with a file in a folder of each target, whose repos pass check --build", func(t *testing.T) {
			catalogDir := exampleCatalog(t)
			_, err := runAdd(t, noConfigFile, "starter", "service", "--catalog", catalogDir)
			require.NoError(t, err)

			_, err = runAdd(t, noConfigFile, "feature", "metrics", "--target", "example", "--target", "service", "--catalog", catalogDir)

			require.NoError(t, err)
			feature := definitionsOf(t, catalogDir).Features["metrics"]
			require.Equal(t, []string{"The feature metrics. Change this sentence to what the feature adds to a repo.", "example service"}, []string{feature.Summary, strings.Join(feature.Targets, " ")})
			require.Subset(t, builtLines(t, catalogDir), []string{"ok    example --feature metrics", "ok    service --feature metrics"})
			require.Equal(t, "# metrics\n\nThe feature metrics writes this file. Change it into the files of the feature.\n", filesIn(t, catalogDir)["features/metrics/files/app/docs/metrics.md"])
		})

		t.Run("lists the extension points of each target, and its add_to example passes check --build once a person removes the comment marks", func(t *testing.T) {
			catalogDir := exampleCatalog(t)
			_, err := runAdd(t, noConfigFile, "feature", "metrics", "--target", "example", "--catalog", catalogDir)
			require.NoError(t, err)
			path := filepath.Join(catalogDir, "features", "metrics", "feature.yml")
			written, err := os.ReadFile(path)
			require.NoError(t, err)
			example := "# add_to:\n#   readme.sections:\n#     - text: <value>\n"
			require.True(t, strings.HasSuffix(string(written), "\n# The extension points of the starter example, with the fields of an item:\n#   readme.sections: [text]\n\n"+example), string(written))

			uncommented := strings.Replace(string(written), example, "add_to:\n  readme.sections:\n    - text: \"## Metrics\"\n", 1)
			require.NoError(t, os.WriteFile(path, []byte(uncommented), 0o644))

			require.Contains(t, builtLines(t, catalogDir), "ok    example --feature metrics")
			require.Equal(t, map[string][]map[string]string{"readme.sections": {{"text": "## Metrics"}}}, definitionsOf(t, catalogDir).Features["metrics"].AddTo)
		})

		t.Run("with no folder in each target, writes no files entry and no file, and says why in feature.yml", func(t *testing.T) {
			catalogDir := exampleCatalog(t)
			require.NoError(t, os.MkdirAll(filepath.Join(catalogDir, "starters", "site", "files", "web"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(catalogDir, "starters", "site", "starter.yml"), []byte("folders:\n  web:\n    path: .\n"), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(catalogDir, "starters", "site", "files", "web", "index.html"), []byte("<h1>{{ .Name }}</h1>\n"), 0o644))

			_, err := runAdd(t, noConfigFile, "feature", "metrics", "--target", "example", "--target", "site", "--catalog", catalogDir)

			require.NoError(t, err)
			written := filesIn(t, filepath.Join(catalogDir, "features", "metrics"))
			require.Equal(t, []string{"feature.yml"}, slices.Sorted(maps.Keys(written)))
			require.Contains(t, written["feature.yml"], "# No folder is in each target. Put each file under files/<folder>/, where <folder> is a folder of each target.\n# files:\n#   - path: files/\n")
			require.Contains(t, written["feature.yml"], "\n# The starter site declares no extension point.\n")
			require.NotContains(t, written["feature.yml"], "# add_to:")
			require.Subset(t, builtLines(t, catalogDir), []string{"ok    example --feature metrics", "ok    site --feature metrics"})
		})

		t.Run("prints the files that it writes and the commands that try the feature with its first target", func(t *testing.T) {
			catalogDir := exampleCatalog(t)

			printed, err := runAdd(t, noConfigFile, "feature", "metrics", "--target", "example", "--catalog", catalogDir)

			require.NoError(t, err)
			require.Equal(t, `Added the feature metrics to the catalog in `+catalogDir+`:
  features/metrics/feature.yml
  features/metrics/files/app/docs/metrics.md
Fill in feature.yml, and then try the feature:
  feature-compose new --catalog `+catalogDir+` example my-repo --feature metrics --dry-run
  feature-compose check --catalog `+catalogDir+` --build
`, printed)
		})
	})

	t.Run("stop", func(t *testing.T) {
		for name, test := range map[string]struct {
			arguments []string
			message   string
		}{
			"a starter that the catalog has":     {[]string{"starter", "example"}, "the catalog already has the starter example in starters/example, so add writes no file"},
			"a feature that the catalog has":     {[]string{"feature", "greeting", "--target", "example"}, "the catalog already has the feature greeting in features/greeting, so add writes no file"},
			"a name of a starter with a capital": {[]string{"starter", "Service"}, `the name "Service" of the starter must start with a lowercase letter or a digit, and hold only lowercase letters, digits and '-'`},
			"a name of a feature with a slash":   {[]string{"feature", "a/b", "--target", "example"}, `the name "a/b" of the feature must start with a lowercase letter or a digit, and hold only lowercase letters, digits and '-'`},
			"a target that is not a starter":     {[]string{"feature", "metrics", "--target", "nope"}, `unknown starter "nope": the catalog has example`},
			"a feature with no target":           {[]string{"feature", "metrics"}, "add feature needs at least one --target <starter>"},
			"a starter with no name":             {[]string{"starter"}, "add starter needs one argument: add starter <name>"},
			"a feature with two names":           {[]string{"feature", "a", "b", "--target", "example"}, "add feature needs one argument: add feature <name> --target <starter>"},
		} {
			t.Run(name+" stops add, which writes no file", func(t *testing.T) {
				catalogDir := exampleCatalog(t)
				before := filesIn(t, catalogDir)

				_, err := runAdd(t, noConfigFile, append(test.arguments, "--catalog", catalogDir)...)

				require.EqualError(t, err, test.message)
				require.Equal(t, before, filesIn(t, catalogDir))
			})
		}

		t.Run("a config file that cannot load stops add with its error", func(t *testing.T) {
			_, err := runAdd(t, noConfigFile, "starter", "service")

			require.EqualError(t, err, "no config file")
		})
	})

	t.Run("help", func(t *testing.T) {
		t.Run("--help of add names its two commands", func(t *testing.T) {
			printed, err := runAdd(t, noConfigFile, "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "starter  add a starter to the catalog, with a starter.yml to fill in and the folder app\n")
			require.Contains(t, printed, "feature  add a feature to the catalog, with a feature.yml to fill in that targets the starters of --target\n")
		})

		t.Run("--help of add feature names --target", func(t *testing.T) {
			printed, err := runAdd(t, noConfigFile, "feature", "--help")

			require.NoError(t, err)
			require.Contains(t, printed, "a starter that the feature targets; give the flag one time for each starter")
		})
	})
}
