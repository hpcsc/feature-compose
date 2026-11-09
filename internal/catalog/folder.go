package catalog

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hpcsc/feature-compose/internal/shell"
)

type folder struct {
	name        string
	path        string
	afterRender []string
	test        []string
}

func (f folder) runAfterRender(ctx context.Context, output string) error {
	return f.runAll(ctx, output, "after_render", f.afterRender)
}

func (f folder) runTests(ctx context.Context, output string) error {
	return f.runAll(ctx, output, "test", f.test)
}

func (f folder) runAll(ctx context.Context, output, list string, commands []string) error {
	dir := filepath.Join(output, filepath.FromSlash(f.path))
	for _, command := range commands {
		printed, err := shell.Run(ctx, dir, command)
		if err != nil {
			return fmt.Errorf("the %s command %q of the folder %s failed: %w\n%s", list, command, f.name, err, printed)
		}
	}
	return nil
}
