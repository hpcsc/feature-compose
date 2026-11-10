//go:build unit

package screen_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestCounted(t *testing.T) {
	t.Run("counts one thing in the singular and each other count in the plural", func(t *testing.T) {
		require.Equal(t, []string{"0 features", "1 feature", "2 features"}, []string{screen.Counted(0, "feature"), screen.Counted(1, "feature"), screen.Counted(2, "feature")})
	})
}
