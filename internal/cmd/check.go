package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/progress"
	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/urfave/cli/v3"
)

const combinationRepoName = "my-repo"

func newCheckCommand(load func() (catalog.Catalogs, error)) *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "find the mistakes of the starters, the features and the templates of the catalog",
		Flags: []cli.Flag{
			catalogFlag("check the catalog in this folder, not the catalog folder of the config file"),
			&cli.BoolFlag{Name: "build", Usage: "also make the repo of each combination of the catalog in a temporary folder, and run its after_render and test commands"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return errors.New("check takes no argument")
			}
			catalogs, err := catalogToRead(cmd, load)
			if err != nil {
				return err
			}
			mistakes, err := catalogs.Check(ctx)
			if err != nil {
				return err
			}
			if len(mistakes) == 0 {
				if _, err := fmt.Fprintln(cmd.Root().Writer, "The catalog has no mistakes."); err != nil || !cmd.Bool("build") {
					return err
				}
				return testCombinations(ctx, cmd.Root().Writer, catalogs)
			}
			if _, err := fmt.Fprintln(cmd.Root().Writer, strings.Join(mistakes, "\n")); err != nil {
				return err
			}
			if len(mistakes) == 1 {
				return errors.New("the catalog has 1 mistake")
			}
			return fmt.Errorf("the catalog has %d mistakes", len(mistakes))
		},
	}
}

func runningLine(w io.Writer, label string) (clear func()) {
	if !isTerminal(w) {
		return func() {}
	}
	line := progress.Start(w, true, label)
	stopTimer := line.Timer(time.Second)
	return func() {
		stopTimer()
		line.Clear()
	}
}

func testCombinations(ctx context.Context, w io.Writer, catalogs catalog.Catalogs) error {
	combinations, err := catalogs.Combinations()
	if err != nil {
		return err
	}
	root, err := os.MkdirTemp("", "feature-compose-check-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(root) }()
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return err
	}
	count := len(combinations)
	if _, err := fmt.Fprintf(w, "The catalog has %s. Make the repo of each, and run its after_render and test commands.\n", screen.Counted(count, "combination")); err != nil {
		return err
	}
	var failed []string
	for i, combination := range combinations {
		if err := ctx.Err(); err != nil {
			return err
		}
		started := time.Now()
		running := runningLine(w, fmt.Sprintf("[%d/%d] %s", i+1, count, combination))
		testErr := catalogs.TestCombination(ctx, combination, combinationRepoName, filepath.Join(root, strconv.Itoa(i+1), combinationRepoName))
		running()
		result := fmt.Sprintf("[%d/%d] ok    %s (%s)\n", i+1, count, combination, time.Since(started).Round(time.Second))
		if testErr != nil {
			failed = append(failed, combination.String())
			result = fmt.Sprintf("[%d/%d] FAIL  %s (%s)\n%s\n", i+1, count, combination, time.Since(started).Round(time.Second), indented(testErr.Error()))
		}
		if _, err := io.WriteString(w, result); err != nil {
			return err
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d of %s failed:\n  %s", len(failed), screen.Counted(count, "combination"), strings.Join(failed, "\n  "))
	}
	_, err = fmt.Fprintln(w, "The repo of each combination passes its commands.")
	return err
}

func indented(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = "      " + line
	}
	return strings.Join(lines, "\n")
}
