package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/hpcsc/feature-compose/internal/version"
	"github.com/urfave/cli/v3"
)

const releaseRepository = "hpcsc/feature-compose"

func Run(ctx context.Context) int {
	if err := newCommand().Run(ctx, os.Args); err != nil {
		writeError(os.Stderr, err)
		return 1
	}

	return 0
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:                  "feature-compose",
		Usage:                 "make a repo from one starter and the features that you select",
		Version:               version.Current(),
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			newNewCommand(catalogInConfig, version.Current()),
			newShowCommand(catalogInConfig),
			newCheckCommand(catalogInConfig),
			newInitCommand(),
			newAddCommand(catalogDirInConfig),
			newVersionCommand(),
			newUpdateCommand(),
		},
	}
}

func newVersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print the tag feature-compose was built from, or its commit when it has no tag",
		Action: func(_ context.Context, cmd *cli.Command) error {
			_, err := fmt.Fprintln(cmd.Root().Writer, version.Current())
			return err
		},
	}
}

func catalogInConfig() (catalog.Catalogs, error) {
	catalogDir, err := catalogDirInConfig()
	if err != nil {
		return catalog.Catalogs{}, err
	}
	return catalog.NewCatalogs(catalog.New(os.DirFS(catalogDir))), nil
}

func catalogDirInConfig() (string, error) {
	file, err := config.Load()
	if err != nil {
		return "", err
	}
	return file.CatalogDir, nil
}

func writeError(w io.Writer, err error) {
	red := color.New(color.FgRed)
	if os.Getenv("NO_COLOR") != "" || !isTerminal(w) {
		red.DisableColor()
	} else {
		red.EnableColor()
	}
	_, _ = red.Fprintln(w, err.Error())
}
