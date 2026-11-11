package screen

import "strings"

func Message(text string, width int) string {
	return messageStyle.Render(strings.Join(Wrapped(text, width), "\n"))
}
