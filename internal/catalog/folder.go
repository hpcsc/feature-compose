package catalog

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
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
		if err := f.run(ctx, dir, list, command); err != nil {
			return err
		}
	}
	return nil
}

func (f folder) run(ctx context.Context, dir, list, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// kill the whole process group: a child of sh keeps the output pipe open, so Wait waits for the child
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	printed, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("the %s command %q of the folder %s failed: %w\n%s", list, command, f.name, err, printed)
	}
	return nil
}
