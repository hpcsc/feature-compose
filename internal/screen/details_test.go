//go:build unit

package screen_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/feature-compose/internal/screen"
	"github.com/stretchr/testify/require"
)

func TestDetails(t *testing.T) {
	t.Run("shows the summary first and a blank line before each section", func(t *testing.T) {
		details := screen.Details("A summary.", 60,
			screen.Section{Heading: "Needs", Entries: []screen.Entry{{Name: "cli"}}},
			screen.Section{Heading: "Options", None: "none"},
		)

		require.Equal(t, "A summary.\n\nNeeds\n  cli\n\nOptions\n  none", plain(details))
	})

	t.Run("starts with the first section when there is no summary", func(t *testing.T) {
		details := screen.Details("", 60, screen.Section{Heading: "Features", Entries: []screen.Entry{{Name: "cli", Summary: "A command line program."}}})

		require.Equal(t, "Features\n  cli  A command line program.", plain(details))
	})

	t.Run("the names of the entries share one column, and so do the labels", func(t *testing.T) {
		details := screen.Details("", 60, screen.Section{Heading: "Options", Entries: []screen.Entry{
			{Name: "tier", Fields: []screen.Field{{Label: "values", Values: []string{"standard"}}}},
			{Name: "go-version", Fields: []screen.Field{{Label: "default", Values: []string{"1.27.1"}}}},
		}})

		require.Equal(t, "Options\n  tier        values   standard\n  go-version  default  1.27.1", plain(details))
	})

	t.Run("a field with more than one value puts each value on a line of its own", func(t *testing.T) {
		details := screen.Details("", 60, screen.Section{Heading: "Folders", Entries: []screen.Entry{
			{Name: "go", Fields: []screen.Field{{Label: "after_render", Values: []string{"go mod tidy", "gofmt -w ."}}}},
		}})

		require.Equal(t, "Folders\n  go  after_render  go mod tidy\n                    gofmt -w .", plain(details))
	})

	t.Run("a value stays beside its name while the pane leaves it 20 columns", func(t *testing.T) {
		details := screen.Details("", 26, screen.Section{Heading: "Options", Entries: []screen.Entry{{Name: "go", Fields: []screen.Field{{Label: "default", Values: []string{"x"}}}}}})

		require.Equal(t, "Options\n  go  default  x", plain(details))
	})

	t.Run("a value goes under its name when the pane is too narrow for both, and wraps at the width of the pane", func(t *testing.T) {
		details := screen.Details("", 25, screen.Section{Heading: "Options", Entries: []screen.Entry{{Name: "go", Fields: []screen.Field{{Label: "default", Values: []string{"aaaaa bbbbbb", "aaaaa bbbbbbb"}}}}}})

		require.Equal(t, "Options\n  go\n    default  aaaaa bbbbbb\n             aaaaa\n             bbbbbbb", plain(details))
	})

	t.Run("wraps a long text at a space", func(t *testing.T) {
		require.Equal(t, "one two\nthree\nfour", plain(screen.Details("one two three four", 9)))
	})
}

func TestEntry(t *testing.T) {
	t.Run("gives its summary and then its fields, each label as wide as the widest and each value wrapped after it", func(t *testing.T) {
		entry := screen.Entry{Summary: "A summary.", Fields: []screen.Field{
			{Label: "file", Values: []string{"x.go"}},
			{Label: "pattern", Values: []string{"aaaaa bbbbb", "aaaaa bbbbbb"}},
		}}

		require.Equal(t, []string{"A summary.", "file     x.go", "pattern  aaaaa bbbbb", "         aaaaa", "         bbbbbb"}, linesOf(strings.Join(entry.Lines(20), "\n")))
	})

	t.Run("wraps a value at each column when its label leaves it no room", func(t *testing.T) {
		entry := screen.Entry{Fields: []screen.Field{{Label: "file", Values: []string{"ab"}}}}

		require.Equal(t, []string{"file  a", "      b"}, linesOf(strings.Join(entry.Lines(6), "\n")))
	})
}
