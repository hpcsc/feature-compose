//go:build unit

package catalog_test

import (
	"testing"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestAnswers(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		t.Run("gives answers with the value of an option of the starter, and leaves the answers that it came from as they were", func(t *testing.T) {
			answers := catalog.Answers{StarterValues: map[string]string{"region": "us"}}

			changed := answers.WithValue("", "region", "eu")

			require.Equal(t, []catalog.Answers{{StarterValues: map[string]string{"region": "us"}}, {StarterValues: map[string]string{"region": "eu"}}}, []catalog.Answers{answers, changed})
		})

		t.Run("gives answers with the value of an option of a feature, and leaves the answers that it came from as they were", func(t *testing.T) {
			answers := catalog.Answers{FeatureValues: map[string]map[string]string{"zed": {"level": "low"}, "alpha": {"mode": "fast"}}}

			changed := answers.WithValue("zed", "level", "high")

			require.Equal(t, []catalog.Answers{
				{FeatureValues: map[string]map[string]string{"zed": {"level": "low"}, "alpha": {"mode": "fast"}}},
				{FeatureValues: map[string]map[string]string{"zed": {"level": "high"}, "alpha": {"mode": "fast"}}},
			}, []catalog.Answers{answers, changed})
		})

		t.Run("gives answers with a value when they had no value yet", func(t *testing.T) {
			require.Equal(t, []catalog.Answers{
				{StarterValues: map[string]string{"region": "eu"}},
				{FeatureValues: map[string]map[string]string{"zed": {"level": "high"}}},
			}, []catalog.Answers{catalog.Answers{}.WithValue("", "region", "eu"), catalog.Answers{}.WithValue("zed", "level", "high")})
		})
	})
}
