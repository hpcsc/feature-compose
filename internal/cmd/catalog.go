package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/hpcsc/feature-compose/internal/index"
	"github.com/hpcsc/feature-compose/internal/registry"
	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/urfave/cli/v3"
)

type registryOpener func() (*registry.Instance, error)

type indexFetcher func(ctx context.Context) (index.File, error)

func knownCatalogs(ctx context.Context) (index.File, error) {
	url := os.Getenv("FEATURE_COMPOSE_INDEX_URL")
	if url == "" {
		url = index.DefaultURL
	}
	return index.Fetch(ctx, &http.Client{Timeout: 30 * time.Second}, url)
}

func newCatalogCommand(open registryOpener, fetch indexFetcher) *cli.Command {
	pathFlag := &cli.StringFlag{Name: "path", Usage: "the folder of the catalog in the source (default: the root of the source)"}
	refFlag := &cli.StringFlag{Name: "ref", Usage: "a branch, a tag or a commit of a git URL (default: the default branch)"}
	return &cli.Command{
		Name:  "catalog",
		Usage: "add, list, sync, change and remove the catalogs of the config file",
		Commands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "get a catalog from a git URL or a local folder, and add it to the config file with a name; with no source, get the catalog of that name from the index of known catalogs",
				ArgsUsage: "<name> [<source>]",
				Flags:     []cli.Flag{pathFlag, refFlag},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() < 1 || cmd.NArg() > 2 {
						return errors.New("catalog add needs a name, and a source when the index of known catalogs does not have the name: catalog add <name> [<source>]")
					}
					r, err := open()
					if err != nil {
						return err
					}
					entry := config.Entry{Source: cmd.Args().Get(1), Path: cmd.String("path"), Ref: cmd.String("ref")}
					if cmd.NArg() == 1 {
						if entry, err = knownEntry(ctx, cmd, fetch); err != nil {
							return err
						}
					}
					origin, err := r.Add(ctx, cmd.Args().Get(0), entry, stepsOn(cmd.Root().ErrWriter))
					if err != nil {
						return err
					}
					from := "the folder " + origin.Source
					if origin.Commit != "" {
						from = origin.Source + " at the commit " + shortCommit(origin.Commit)
					}
					_, err = fmt.Fprintf(cmd.Root().Writer, `Added the catalog %s from %s.
The starters of a catalog run their commands on this machine. new --dry-run lists them before new runs them.
See the starters and the features of the catalog with:
  feature-compose show
`, origin.Name, from)
					return err
				},
			},
			{
				Name:  "list",
				Usage: "show the name, the source, the path, the ref and the commit of each catalog",
				Flags: []cli.Flag{&cli.BoolFlag{Name: "known", Usage: "show the catalogs of the index of known catalogs, which catalog add <name> gets with no source"}},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() > 0 {
						return errors.New("catalog list takes no argument")
					}
					if cmd.Bool("known") {
						known, err := fetch(ctx)
						if err != nil {
							return err
						}
						return printKnown(cmd.Root().Writer, known)
					}
					r, err := open()
					if err != nil {
						return err
					}
					origins, err := r.List(ctx)
					if err != nil {
						return err
					}
					return printOrigins(cmd.Root().Writer, origins)
				},
			},
			{
				Name:      "sync",
				Usage:     "get the newest commit of the ref of each catalog from a git URL, or of the named catalogs",
				ArgsUsage: "[<name>...]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					r, err := open()
					if err != nil {
						return err
					}
					synced, err := r.Sync(ctx, cmd.Args().Slice(), stepsOn(cmd.Root().ErrWriter))
					if err != nil {
						return err
					}
					return printSynced(cmd.Root().Writer, synced)
				},
			},
			{
				Name:      "remove",
				Usage:     "remove a catalog from the config file, and delete its clone",
				ArgsUsage: "<name>",
				Action: func(_ context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return errors.New("catalog remove needs one argument: catalog remove <name>")
					}
					r, err := open()
					if err != nil {
						return err
					}
					if err := r.Remove(cmd.Args().First()); err != nil {
						return err
					}
					_, err = fmt.Fprintf(cmd.Root().Writer, "Removed the catalog %s.\n", cmd.Args().First())
					return err
				},
			},
			{
				Name:      "set",
				Usage:     "change the path or the ref of a catalog, and get the commit of the new ref",
				ArgsUsage: "<name>",
				Flags:     []cli.Flag{pathFlag, refFlag},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return errors.New("catalog set needs one argument: catalog set <name> [--path <folder>] [--ref <ref>]")
					}
					var change registry.Change
					if cmd.IsSet("path") {
						path := cmd.String("path")
						change.Path = &path
					}
					if cmd.IsSet("ref") {
						ref := cmd.String("ref")
						change.Ref = &ref
					}
					if change.Path == nil && change.Ref == nil {
						return errors.New("catalog set needs --path, --ref or both")
					}
					r, err := open()
					if err != nil {
						return err
					}
					origin, err := r.Set(ctx, cmd.Args().First(), change, stepsOn(cmd.Root().ErrWriter))
					if err != nil {
						return err
					}
					if _, err := fmt.Fprintf(cmd.Root().Writer, "Changed the catalog %s:\n", origin.Name); err != nil {
						return err
					}
					return printOrigins(cmd.Root().Writer, []catalog.Origin{origin})
				},
			},
		},
	}
}

func knownEntry(ctx context.Context, cmd *cli.Command, fetch indexFetcher) (config.Entry, error) {
	known, err := fetch(ctx)
	if err != nil {
		return config.Entry{}, err
	}
	found, err := known.Entry(cmd.Args().First())
	if err != nil {
		return config.Entry{}, err
	}
	entry := found.ConfigEntry()
	if cmd.IsSet("path") {
		entry.Path = cmd.String("path")
	}
	if cmd.IsSet("ref") {
		entry.Ref = cmd.String("ref")
	}
	return entry, nil
}

func printKnown(w io.Writer, known index.File) error {
	names := known.Names()
	if len(names) == 0 {
		_, err := fmt.Fprintln(w, "The index of known catalogs has no catalog.")
		return err
	}
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "NAME\tSOURCE\tPATH\tSUMMARY")
	for _, name := range names {
		entry, err := known.Entry(name)
		if err != nil {
			return err
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", name, entry.Source, orDash(entry.Path), entry.Summary)
	}
	if err := table.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "Add one with: feature-compose catalog add <name>")
	return err
}

func printOrigins(w io.Writer, origins []catalog.Origin) error {
	if len(origins) == 0 {
		_, err := fmt.Fprintln(w, "The config file has no catalog. Add one with: feature-compose catalog add <name> <source>")
		return err
	}
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "NAME\tSOURCE\tPATH\tREF\tCOMMIT")
	var withNoClone []string
	for _, origin := range origins {
		commit := shortCommit(origin.Commit)
		switch {
		case !config.IsGitURL(origin.Source):
			commit = "-"
		case origin.Commit == "":
			commit = "no clone"
			withNoClone = append(withNoClone, origin.Name)
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", origin.Name, origin.Source, orDash(origin.Path), orDash(origin.Ref), commit)
	}
	if err := table.Flush(); err != nil {
		return err
	}
	if len(withNoClone) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "Get the clone of %s with: feature-compose catalog sync %s\n", screen.Counted(len(withNoClone), "catalog"), strings.Join(withNoClone, " "))
	return err
}

func printSynced(w io.Writer, synced []registry.Synced) error {
	var printed strings.Builder
	var failed []string
	for _, s := range synced {
		switch {
		case s.Err != nil:
			failed = append(failed, s.Name)
			fmt.Fprintf(&printed, "%s: FAIL\n%s\n", s.Name, indented(s.Err.Error()))
		case s.Local:
			fmt.Fprintf(&printed, "%s: a local folder, so sync gets nothing\n", s.Name)
		case s.OldCommit == "":
			fmt.Fprintf(&printed, "%s: cloned again, at %s\n", s.Name, shortCommit(s.NewCommit))
		case s.OldCommit == s.NewCommit:
			fmt.Fprintf(&printed, "%s: no new commit, at %s\n", s.Name, shortCommit(s.NewCommit))
		default:
			fmt.Fprintf(&printed, "%s: from %s to %s\n", s.Name, shortCommit(s.OldCommit), shortCommit(s.NewCommit))
		}
	}
	if _, err := io.WriteString(w, printed.String()); err != nil {
		return err
	}
	if len(failed) > 0 {
		return fmt.Errorf("the sync of %d of %s failed: %s", len(failed), screen.Counted(len(synced), "catalog"), strings.Join(failed, ", "))
	}
	return nil
}

func shortCommit(commit string) string {
	return commit[:min(len(commit), 7)]
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
