package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/urfave/cli/v3"
)

const docsURL = "https://github.com/hpcsc/feature-compose/blob/main/docs/catalog.md"

var entryName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type addedFile struct {
	path     string
	contents string
}

func newAddCommand(configured func() (string, error)) *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "add a starter or a feature to the catalog, with a file to fill in",
		Commands: []*cli.Command{
			{
				Name:      "starter",
				Usage:     "add a starter to the catalog, with a starter.yml to fill in and the folder app",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					catalogFlag("add the starter to the catalog in this folder, not the catalog folder of the config file"),
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return errors.New("add starter needs one argument: add starter <name>")
					}
					name := cmd.Args().First()
					if err := checkEntryName("starter", name); err != nil {
						return err
					}
					catalogDir, err := catalogDirOf(cmd, configured)
					if err != nil {
						return err
					}
					if err := added(catalogDir, "starter", filepath.Join("starters", name), starterFiles(name)); err != nil {
						return err
					}
					return printAdded(cmd, catalogDir, "starter", name, starterFiles(name), name+" my-repo")
				},
			},
			{
				Name:      "feature",
				Usage:     "add a feature to the catalog, with a feature.yml to fill in that targets the starters of --target",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: "target", Usage: "a starter that the feature targets; give the flag one time for each starter"},
					catalogFlag("add the feature to the catalog in this folder, not the catalog folder of the config file"),
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return errors.New("add feature needs one argument: add feature <name> --target <starter>")
					}
					name := cmd.Args().First()
					if err := checkEntryName("feature", name); err != nil {
						return err
					}
					targets := cmd.StringSlice("target")
					if len(targets) == 0 {
						return errors.New("add feature needs at least one --target <starter>")
					}
					catalogDir, err := catalogDirOf(cmd, configured)
					if err != nil {
						return err
					}
					definitions, err := catalog.New(os.DirFS(catalogDir)).Definitions()
					if err != nil {
						return err
					}
					for _, target := range targets {
						if _, err := definitions.ForStarter(target); err != nil {
							return err
						}
					}
					files := featureFiles(name, targets, definitions)
					if err := added(catalogDir, "feature", filepath.Join("features", name), files); err != nil {
						return err
					}
					return printAdded(cmd, catalogDir, "feature", name, files, targets[0]+" my-repo --feature "+name)
				},
			},
		},
	}
}

func checkEntryName(kind, name string) error {
	if !entryName.MatchString(name) {
		return fmt.Errorf("the name %q of the %s must start with a lowercase letter or a digit, and hold only lowercase letters, digits and '-'", name, kind)
	}
	return nil
}

func added(catalogDir, kind, folder string, files []addedFile) error {
	if _, err := os.Lstat(filepath.Join(catalogDir, folder)); !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("the catalog already has the %s %s in %s, so add writes no file", kind, filepath.Base(folder), filepath.ToSlash(folder))
	}
	for _, f := range files {
		target := filepath.Join(catalogDir, filepath.FromSlash(f.path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(f.contents), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func printAdded(cmd *cli.Command, catalogDir, kind, name string, files []addedFile, newArguments string) error {
	var printed strings.Builder
	fmt.Fprintf(&printed, "Added the %s %s to the catalog in %s:\n", kind, name, catalogDir)
	for _, f := range files {
		fmt.Fprintf(&printed, "  %s\n", f.path)
	}
	catalogWords := ""
	if given := cmd.String(catalogFlagName); given != "" {
		catalogWords = " --catalog " + given
	}
	fmt.Fprintf(&printed, "Fill in %s.yml, and then try the %s:\n", kind, kind)
	fmt.Fprintf(&printed, "  feature-compose new%s %s --dry-run\n", catalogWords, newArguments)
	fmt.Fprintf(&printed, "  feature-compose check%s --build\n", catalogWords)
	_, err := fmt.Fprint(cmd.Root().Writer, printed.String())
	return err
}

func starterFiles(name string) []addedFile {
	return []addedFile{
		{path: "starters/" + name + "/starter.yml", contents: `# The fields of this file: ` + docsURL + `#starteryml
summary: The starter ` + name + `. Change this sentence to what a repo of this starter gets.

# options:
#   owner:
#     default: nobody
#     pattern: '^[a-z][a-z0-9-]*$'

folders:
  app:
    path: .
    # after_render: [<command>]
    # test: [<command>]

# extension_points:
#   readme.sections:
#     summary: A section of README.md, as Markdown.
#     file: app/README.md
#     item: [text]
`},
		{path: "starters/" + name + "/files/app/README.md", contents: "# {{ .Name }}\n"},
	}
}

func featureFiles(name string, targets []string, definitions catalog.Definitions) []addedFile {
	folder, hasFolder := folderOfEachTarget(targets, definitions)
	var yml strings.Builder
	fmt.Fprintf(&yml, "# The fields of this file: %s#featureyml\n", docsURL)
	fmt.Fprintf(&yml, "summary: The feature %s. Change this sentence to what the feature adds to a repo.\n", name)
	fmt.Fprintf(&yml, "targets: [%s]\n", strings.Join(targets, ", "))
	yml.WriteString(`
# needs: [<feature>]

# options:
#   mode:
#     values: [fast, safe]
#     default: fast

`)
	if hasFolder {
		yml.WriteString("files:\n  - path: files/\n")
	} else {
		yml.WriteString("# No folder is in each target. Put each file under files/<folder>/, where <folder> is a folder of each target.\n# files:\n#   - path: files/\n")
	}
	yml.WriteString(addToExample(targets, definitions))
	files := []addedFile{{path: "features/" + name + "/feature.yml", contents: yml.String()}}
	if hasFolder {
		files = append(files, addedFile{
			path:     "features/" + name + "/files/" + folder + "/docs/" + name + ".md",
			contents: "# " + name + "\n\nThe feature " + name + " writes this file. Change it into the files of the feature.\n",
		})
	}
	return files
}

func folderOfEachTarget(targets []string, definitions catalog.Definitions) (string, bool) {
	for _, folder := range slices.Sorted(maps.Keys(definitions.Starters[targets[0]].Folders)) {
		inEach := true
		for _, target := range targets[1:] {
			if _, ok := definitions.Starters[target].Folders[folder]; !ok {
				inEach = false
			}
		}
		if inEach {
			return folder, true
		}
	}
	return "", false
}

func addToExample(targets []string, definitions catalog.Definitions) string {
	var example strings.Builder
	for _, target := range targets {
		points := definitions.Starters[target].ExtensionPoints
		if len(points) == 0 {
			fmt.Fprintf(&example, "\n# The starter %s declares no extension point.\n", target)
			continue
		}
		fmt.Fprintf(&example, "\n# The extension points of the starter %s, with the fields of an item:\n", target)
		for _, point := range slices.Sorted(maps.Keys(points)) {
			fmt.Fprintf(&example, "#   %s: [%s]\n", point, strings.Join(points[point].Item, ", "))
		}
	}
	point, item, ok := pointOfEachTarget(targets, definitions)
	if !ok {
		return example.String()
	}
	fmt.Fprintf(&example, "\n# add_to:\n#   %s:\n", point)
	for i, field := range item {
		marker := "  "
		if i == 0 {
			marker = "- "
		}
		fmt.Fprintf(&example, "#     %s%s: <value>\n", marker, field)
	}
	return example.String()
}

func pointOfEachTarget(targets []string, definitions catalog.Definitions) (string, []string, bool) {
	points := definitions.Starters[targets[0]].ExtensionPoints
	for _, point := range slices.Sorted(maps.Keys(points)) {
		inEach := true
		for _, target := range targets[1:] {
			other, ok := definitions.Starters[target].ExtensionPoints[point]
			if !ok || !slices.Equal(other.Item, points[point].Item) {
				inEach = false
			}
		}
		if inEach {
			return point, points[point].Item, true
		}
	}
	return "", nil, false
}
