//go:build unit

package screen_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestTooSmall(t *testing.T) {
	t.Run("names the command and the smallest size", func(t *testing.T) {
		require.Equal(t, "The terminal is too small for new. Make it at least 40 columns wide and 8 rows high.", screen.TooSmall("new", 200))
	})

	t.Run("wraps the message at the width of the terminal", func(t *testing.T) {
		message := screen.TooSmall("show", 30)

		require.Greater(t, len(strings.Split(message, "\n")), 1)
		for _, line := range strings.Split(message, "\n") {
			require.LessOrEqual(t, lipgloss.Width(line), 30)
		}
		require.Equal(t, "The terminal is too small for show. Make it at least 40 columns wide and 8 rows high.", strings.Join(strings.Fields(message), " "))
	})
}

func TestFits(t *testing.T) {
	t.Run("a screen of 40 columns and 8 rows or more fits, and a smaller one does not", func(t *testing.T) {
		require.Equal(t, []bool{true, false, false}, []bool{screen.Fits(40, 8), screen.Fits(39, 8), screen.Fits(40, 7)})
	})
}
