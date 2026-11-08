package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/hpcsc/feature-compose/catalog"
	"github.com/hpcsc/feature-compose/internal/version"
	"github.com/urfave/cli/v3"
)

const releaseRepository = "hpcsc/feature-compose"

func Run(ctx context.Context) int {
	if err := newCommand().Run(ctx, os.Args); err != nil {
		color.Red(err.Error())
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
			newNewCommand(catalog.Embedded()),
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
