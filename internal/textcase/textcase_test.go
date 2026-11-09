//go:build unit

package textcase_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/textcase"
	"github.com/stretchr/testify/require"
)

func TestTextcase(t *testing.T) {
	inEachCase := func(name string) []string {
		return []string{textcase.Kebab(name), textcase.Snake(name), textcase.Camel(name), textcase.Pascal(name)}
	}

	for _, name := range []string{"order-event-store", "order_event_store", "order.event.store", "orderEventStore", "OrderEventStore"} {
		t.Run("the case functions give one result for the name "+name, func(t *testing.T) {
			require.Equal(t, []string{"order-event-store", "order_event_store", "orderEventStore", "OrderEventStore"}, inEachCase(name))
		})
	}

	t.Run("a word of one letter before an upper-case letter is a word of its own", func(t *testing.T) {
		require.Equal(t, []string{"x-ray", "x_ray", "xRay", "XRay"}, inEachCase("xRay"))
	})

	t.Run("an upper-case letter after another upper-case letter stays in the word", func(t *testing.T) {
		require.Equal(t, []string{"order-api", "order_api", "orderApi", "OrderApi"}, inEachCase("OrderAPI"))
	})
}
