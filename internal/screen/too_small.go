package screen

import (
	"fmt"
	"strings"
)

const (
	narrowest = 40
	lowest    = 8
)

func Fits(width, height int) bool {
	return width >= narrowest && height >= lowest
}

func TooSmall(command string, width int) string {
	return strings.Join(Wrapped(fmt.Sprintf("The terminal is too small for %s. Make it at least %d columns wide and %d rows high.", command, narrowest, lowest), width), "\n")
}
