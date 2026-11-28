package registry

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/config"
)

const gitTimeout = 10 * time.Minute

type clone struct {
	source                   string
	sourceWithoutCredentials string
	folder                   string
}

func newClone(ctx context.Context, entry config.Entry, folder string, steps catalog.Steps) (clone, error) {
	c := clone{source: entry.Source, sourceWithoutCredentials: entry.SourceWithoutCredentials(), folder: folder}
	if err := os.RemoveAll(folder); err != nil {
		return clone{}, fmt.Errorf("remove the old clone %s: %w", folder, err)
	}
	if err := os.MkdirAll(filepath.Dir(folder), 0o755); err != nil {
		return clone{}, fmt.Errorf("make the folder of the clones: %w", err)
	}
	end := steps.Start("Clone " + c.sourceWithoutCredentials)
	if printed, err := c.run(ctx, "clone", "--quiet", "--", c.source, folder); err != nil {
		_ = os.RemoveAll(folder)
		err = c.cannotRead(printed)
		end(err)
		return clone{}, err
	}
	end(nil)
	return c, nil
}

func (c clone) exists() bool {
	info, err := os.Stat(filepath.Join(c.folder, ".git"))
	return err == nil && info.IsDir()
}

func (c clone) commit(ctx context.Context) (string, error) {
	printed, err := c.run(ctx, "-C", c.folder, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("read the commit of the clone %s: %s", c.folder, printed)
	}
	return printed, nil
}

func (c clone) fetch(ctx context.Context, steps catalog.Steps) error {
	end := steps.Start("Fetch " + c.sourceWithoutCredentials)
	err := c.fetched(ctx)
	end(err)
	return err
}

func (c clone) fetched(ctx context.Context) error {
	if printed, err := c.run(ctx, "-C", c.folder, "remote", "set-url", "origin", c.source); err != nil {
		return fmt.Errorf("set the source %s in the clone %s: %s", c.sourceWithoutCredentials, c.folder, printed)
	}
	if printed, err := c.run(ctx, "-C", c.folder, "fetch", "--quiet", "--tags", "--force", "--prune", "origin"); err != nil {
		return c.cannotRead(printed)
	}
	if printed, err := c.run(ctx, "-C", c.folder, "remote", "set-head", "origin", "--auto"); err != nil {
		return c.cannotRead(printed)
	}
	return nil
}

func (c clone) refs(ctx context.Context) ([]string, error) {
	printed, err := c.run(ctx, "-C", c.folder, "for-each-ref", "--format=%(refname)", "refs/remotes/origin", "refs/tags")
	if err != nil {
		return nil, fmt.Errorf("read the refs of the clone %s: %s", c.folder, printed)
	}
	var branches, tags []string
	for _, line := range strings.Split(printed, "\n") {
		switch {
		case strings.HasPrefix(line, "refs/remotes/origin/") && line != "refs/remotes/origin/HEAD":
			branches = append(branches, strings.TrimPrefix(line, "refs/remotes/origin/"))
		case strings.HasPrefix(line, "refs/tags/"):
			tags = append(tags, strings.TrimPrefix(line, "refs/tags/"))
		}
	}
	return append(branches, tags...), nil
}

func (c clone) checkout(ctx context.Context, ref string) error {
	revision, err := c.revision(ctx, ref)
	if err != nil {
		return err
	}
	if printed, err := c.run(ctx, "-C", c.folder, "checkout", "--quiet", "--detach", revision); err != nil {
		return fmt.Errorf("check out %s in the clone %s: %s", revision, c.folder, printed)
	}
	return nil
}

func (c clone) revision(ctx context.Context, ref string) (string, error) {
	candidates := []string{"origin/HEAD"}
	if ref != "" {
		candidates = []string{"origin/" + ref, ref}
	}
	for _, candidate := range candidates {
		printed, err := c.run(ctx, "-C", c.folder, "rev-parse", "--verify", "--quiet", "--end-of-options", candidate+"^{commit}")
		if err == nil {
			return printed, nil
		}
	}
	if ref == "" {
		return "", fmt.Errorf("the source %s has no default branch: give a branch, a tag or a commit with --ref", c.sourceWithoutCredentials)
	}
	return "", fmt.Errorf("the source %s has no ref %s", c.sourceWithoutCredentials, ref)
}

func (c clone) environment() []string {
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	server := strings.TrimSuffix(os.Getenv("GITHUB_SERVER_URL"), "/")
	if server == "" {
		server = "https://github.com"
	}
	if token == "" || !strings.HasPrefix(c.source, server+"/") {
		return env
	}
	// the environment, and not -c, holds the header, so that other users cannot read the token in the process list
	count, err := strconv.Atoi(os.Getenv("GIT_CONFIG_COUNT"))
	if err != nil {
		count = 0
	}
	header := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token))
	return append(env,
		"GIT_CONFIG_COUNT="+strconv.Itoa(count+1),
		fmt.Sprintf("GIT_CONFIG_KEY_%d=http.%s/.extraheader", count, server),
		fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", count, header),
	)
}

func (c clone) cannotRead(printed string) error {
	return fmt.Errorf("git cannot read %s: check that git clone %s works\n%s", c.sourceWithoutCredentials, c.sourceWithoutCredentials, strings.ReplaceAll(printed, c.source, c.sourceWithoutCredentials))
}

func (c clone) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = c.environment()
	// a new session has no terminal, so neither git nor ssh can ask for a password that nobody sees
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	printed, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Sprintf("git ran for more than %s, so feature-compose stopped it", gitTimeout), ctx.Err()
	}
	var notStarted *exec.Error
	if errors.As(err, &notStarted) {
		return fmt.Sprintf("%v: install git, and put it on the PATH", err), err
	}
	return strings.TrimSpace(string(printed)), err
}
