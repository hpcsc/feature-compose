package screen

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func Wrapped(text string, width int) []string {
	return strings.Split(ansi.Wrap(text, max(width, 1), ""), "\n")
}
