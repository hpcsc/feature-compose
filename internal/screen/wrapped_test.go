//go:build unit

package screen_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestWrapped(t *testing.T) {
	t.Run("wraps a text at a space", func(t *testing.T) {
		require.Equal(t, []string{"one two", "three"}, screen.Wrapped("one two three", 7))
	})

	t.Run("wraps at each column when the width is under one", func(t *testing.T) {
		require.Equal(t, []string{"a", "b"}, screen.Wrapped("ab", 0))
	})
}
