package catalog

import (
	"context"
	"fmt"
	"maps"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/shell"
	"github.com/pelletier/go-toml/v2"
)

const toolchainFile = "mise.toml"

type toolchain struct {
	tools []string
}

func newToolchain(label, contents string) (toolchain, error) {
	var parsed struct {
		Tools map[string]any `toml:"tools"`
	}
	if err := toml.Unmarshal([]byte(contents), &parsed); err != nil {
		return toolchain{}, fmt.Errorf("read %s: %w", label, err)
	}
	var tools []string
	for _, name := range slices.Sorted(maps.Keys(parsed.Tools)) {
		version, ok := parsed.Tools[name].(string)
		if !ok || version == "" {
			return toolchain{}, fmt.Errorf("read %s: the tool %s has no version as a text, for example %s = \"1.0.0\"", label, name, name)
		}
		tools = append(tools, name+"@"+version)
	}
	return toolchain{tools: tools}, nil
}

func (t toolchain) run(ctx context.Context, dir, command string) ([]byte, error) {
	if len(t.tools) == 0 {
		return shell.Run(ctx, dir, command)
	}
	if _, err := exec.LookPath("mise"); err != nil {
		return nil, fmt.Errorf("the folder pins its tools in %s, so its commands run with mise, and mise is not on the PATH: install it from https://mise.jdx.dev", toolchainFile)
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("find the path of the folder %s: %w", dir, err)
	}
	real, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("find the real path of the folder %s: %w", dir, err)
	}
	// mise reads no mise.toml in the folder or above it, so it trusts and tracks no file: the versions come from the
	// arguments. mise compares real paths, so a path through a symlink, such as /var on macOS, would not stop it
	env := []string{"MISE_CEILING_PATHS=" + real, "MISE_IGNORED_CONFIG_PATHS=" + filepath.Join(real, toolchainFile)}
	return shell.RunProgram(ctx, dir, env, "mise", t.arguments(command)...)
}

func (t toolchain) shown(command string) string {
	if len(t.tools) == 0 {
		return command
	}
	return "mise exec " + strings.Join(t.tools, " ") + " -- " + command
}

func (t toolchain) arguments(command string) []string {
	arguments := append([]string{"exec"}, t.tools...)
	return append(arguments, "--", "sh", "-c", command)
}
