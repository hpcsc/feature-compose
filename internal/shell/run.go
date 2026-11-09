package shell

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

func Run(ctx context.Context, dir, command string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// kill the whole process group: a child of sh keeps the output pipe open, so Wait waits for the child
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	return cmd.CombinedOutput()
}
