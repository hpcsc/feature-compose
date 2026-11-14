package screen

import "strings"

func Message(text string, width int) string {
	return ProblemStyle.Render(strings.Join(Wrapped(text, width), "\n"))
}
