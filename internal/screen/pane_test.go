//go:build unit

package screen_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestPane(t *testing.T) {
	t.Run("draws its title in the top border", func(t *testing.T) {
		pane := screen.Pane{Title: "Repo", Width: 20, Height: 3, Border: screen.PlainBorder}

		require.Equal(t, "╭─ Repo ───────────╮", linesOf(pane.View())[0])
	})

	t.Run("cuts a title that is longer than the pane with …", func(t *testing.T) {
		pane := screen.Pane{Title: "a-very-long-title-for-a-pane", Width: 20, Height: 3, Border: screen.PlainBorder}

		require.Equal(t, "╭─ a-very-long-t… ─╮", linesOf(pane.View())[0])
	})

	t.Run("cuts each line at its inner width", func(t *testing.T) {
		pane := screen.Pane{Lines: []string{"abcdefghijklmnopqrstuvwxyz", "short"}, Width: 12, Height: 4, Border: screen.PlainBorder}

		require.Equal(t, []string{"│ abcdefg… │", "│ short    │"}, linesOf(pane.View())[1:3])
	})

	t.Run("fills its height with empty lines between its borders", func(t *testing.T) {
		pane := screen.Pane{Lines: []string{"one"}, Width: 9, Height: 5, Border: screen.PlainBorder}

		require.Equal(t, []string{"╭─  ────╮", "│ one   │", "│       │", "│       │", "╰───────╯"}, linesOf(pane.View()))
	})

	t.Run("shows its count at the right of its top border", func(t *testing.T) {
		pane := screen.Pane{Title: "Repo", Count: "1 of 2", Width: 20, Height: 3, Border: screen.PlainBorder}

		require.Equal(t, "╭─ Repo ── 1 of 2 ─╮", linesOf(pane.View())[0])
	})

	t.Run("shows its note at the right of its bottom border", func(t *testing.T) {
		pane := screen.Pane{Note: "50%", Width: 20, Height: 3, Border: screen.PlainBorder}

		require.Equal(t, "╰──────────── 50% ─╯", linesOf(pane.View())[2])
	})
}

var escapeCode = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func plain(text string) string {
	return escapeCode.ReplaceAllString(text, "")
}

func linesOf(text string) []string {
	return strings.Split(plain(text), "\n")
}
