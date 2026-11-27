package astgrep

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const minimumVersion = "0.45.3"

func Installed() error {
	if _, err := exec.LookPath("ast-grep"); err != nil {
		return errors.New("ast-grep is not on the PATH: install it from https://ast-grep.github.io")
	}
	out, err := exec.Command("ast-grep", "--version").Output()
	if err != nil {
		return fmt.Errorf("run ast-grep --version: %w", err)
	}
	version := strings.TrimPrefix(strings.TrimSpace(string(out)), "ast-grep ")
	older, err := isOlder(version, minimumVersion)
	if err != nil {
		return fmt.Errorf("read the version %q of ast-grep: %w", version, err)
	}
	if older {
		return fmt.Errorf("ast-grep is version %s, and feature-compose needs version %s or later", version, minimumVersion)
	}
	return nil
}

func isOlder(version, minimum string) (bool, error) {
	parts, minimumParts := strings.Split(version, "."), strings.Split(minimum, ".")
	if len(parts) != 3 {
		return false, errors.New("it is not three numbers with dots between them")
	}
	for i := range parts {
		number, err := strconv.Atoi(parts[i])
		if err != nil {
			return false, errors.New("it is not three numbers with dots between them")
		}
		minimumNumber, _ := strconv.Atoi(minimumParts[i])
		if number != minimumNumber {
			return number < minimumNumber, nil
		}
	}
	return false, nil
}
