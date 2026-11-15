package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/hpcsc/feature-compose/internal/config"
	"github.com/hpcsc/feature-compose/internal/example_catalog"
	"github.com/urfave/cli/v3"
)

func newInitCommand() *cli.Command {
	return &cli.Command{
		Name:      "init",
		Usage:     "write a small catalog with one starter and one feature to a folder, as the start of a catalog of your own",
		ArgsUsage: "<folder>",
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				return errors.New("init needs one argument: init <folder>")
			}
			folder := cmd.Args().First()
			if err := example_catalog.Write(folder); err != nil {
				return err
			}
			catalogDir, err := filepath.Abs(folder)
			if err != nil {
				return err
			}
			configPath, err := config.FilePath()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.Root().Writer, `Made a catalog in %[1]s with the starter %[2]s and the feature %[3]s. Try it:
  feature-compose show --catalog %[1]s
  feature-compose new --catalog %[1]s %[2]s my-repo --feature %[3]s --dry-run
  feature-compose check --catalog %[1]s --build
To use it with no --catalog, write this line in %[4]s:
  catalog: %[5]s
`, folder, example_catalog.StarterName, example_catalog.FeatureName, configPath, catalogDir)
			return err
		},
	}
}
