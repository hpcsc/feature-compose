package shell

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func Run(ctx context.Context, dir, command string) ([]byte, error) {
	return RunProgram(ctx, dir, nil, "sh", "-c", command)
}

func RunProgram(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// kill the whole process group: a child of sh keeps the output pipe open, so Wait waits for the child
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	return cmd.CombinedOutput()
}
