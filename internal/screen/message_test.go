//go:build unit

package screen_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestMessage(t *testing.T) {
	t.Run("wraps the text of a message at a space at the width", func(t *testing.T) {
		require.Equal(t, []string{"zed stays", "selected."}, linesOf(screen.Message("zed stays selected.", 10)))
	})
}
