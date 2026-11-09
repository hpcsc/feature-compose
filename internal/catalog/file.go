package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type file struct {
	path     string
	contents string
}

func (f file) write(output string) error {
	onDisk := filepath.Join(output, filepath.FromSlash(f.path))
	if err := os.MkdirAll(filepath.Dir(onDisk), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	if err := os.WriteFile(onDisk, []byte(f.contents), f.mode()); err != nil {
		return fmt.Errorf("write %s: %w", f.path, err)
	}
	return nil
}

func (f file) mode() os.FileMode {
	if strings.HasPrefix(f.contents, "#!") {
		return 0o755
	}
	return 0o644
}
