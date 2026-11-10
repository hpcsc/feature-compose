//go:build unit

package screen_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestRowLine(t *testing.T) {
	t.Run("a row starts after one space with its prefix and its text", func(t *testing.T) {
		require.Equal(t, " ├─ api-auth", plain(screen.RowLine("├─ ", "api-auth", 30)))
	})

	t.Run("a row that is wider than the pane ends with …", func(t *testing.T) {
		require.Equal(t, " ├─ a-very-…", plain(screen.RowLine("├─ ", "a-very-long-feature-name", 12)))
	})

	t.Run("the selected row starts with a bar and fills the width of the pane", func(t *testing.T) {
		require.Equal(t, "▌▾ go     ", plain(screen.SelectedRowLine("▾ go", 10)))
	})

	t.Run("a selected row that is wider than the pane ends with …", func(t *testing.T) {
		require.Equal(t, "▌abcd…", plain(screen.SelectedRowLine("abcdefghijkl", 6)))
	})
}
