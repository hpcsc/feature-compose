//go:build unit

package screen_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestOffsetShowing(t *testing.T) {
	t.Run("keeps the offset while the line is in view", func(t *testing.T) {
		require.Equal(t, []int{2, 2}, []int{screen.OffsetShowing(2, 2, 4), screen.OffsetShowing(2, 5, 4)})
	})

	t.Run("moves down just enough to show a line below the view", func(t *testing.T) {
		require.Equal(t, 3, screen.OffsetShowing(2, 6, 4))
	})

	t.Run("moves up to a line above the view", func(t *testing.T) {
		require.Equal(t, 1, screen.OffsetShowing(2, 1, 4))
	})
}
