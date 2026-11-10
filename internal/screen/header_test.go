//go:build unit

package screen_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestHeader(t *testing.T) {
	t.Run("puts the counts at the right of the line", func(t *testing.T) {
		require.Equal(t, " feature-compose new          1 feature ", plain(screen.Header("feature-compose new", "1 feature", 40)))
	})

	t.Run("keeps the counts when one space is left between them and the title", func(t *testing.T) {
		require.Equal(t, " feature-compose new  1 feature ", plain(screen.Header("feature-compose new", "1 feature", 32)))
	})

	t.Run("keeps only the title when the counts do not fit", func(t *testing.T) {
		require.Equal(t, " feature-compose new ", plain(screen.Header("feature-compose new", "1 feature", 31)))
	})
}
