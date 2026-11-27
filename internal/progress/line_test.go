//go:build unit

package progress_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/hpcsc/feature-compose/internal/progress"
	"github.com/stretchr/testify/require"
)

func TestLine(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		t.Run("on a terminal, draws the label on the line without ending it", func(t *testing.T) {
			var out bytes.Buffer

			progress.Start(&out, true, "Downloading feature-compose v0.2.0")

			require.Equal(t, "\rDownloading feature-compose v0.2.0…\x1b[K", out.String())
		})

		t.Run("elsewhere, writes the label as a whole line", func(t *testing.T) {
			var out bytes.Buffer

			progress.Start(&out, false, "Downloading feature-compose v0.2.0")

			require.Equal(t, "Downloading feature-compose v0.2.0…\n", out.String())
		})
	})

	t.Run("bytes", func(t *testing.T) {
		t.Run("on a terminal, draws the bytes of the total again on the same line", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "Downloading feature-compose v0.2.0")
			out.Reset()

			line.Bytes(500_000, 2_000_000)
			line.Bytes(2_000_000, 2_000_000)

			require.Equal(t,
				"\rDownloading feature-compose v0.2.0: 500.0 KB of 2.0 MB (25%)\x1b[K"+
					"\rDownloading feature-compose v0.2.0: 2.0 MB of 2.0 MB (100%)\x1b[K",
				out.String())
		})

		t.Run("does not draw the same text two times", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "Downloading feature-compose v0.2.0")
			out.Reset()

			line.Bytes(1_000_100, 2_000_000)
			line.Bytes(1_000_200, 2_000_000)

			require.Equal(t, "\rDownloading feature-compose v0.2.0: 1.0 MB of 2.0 MB (50%)\x1b[K", out.String())
		})

		t.Run("shows only the bytes when the total is not known", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "Downloading feature-compose v0.2.0")
			out.Reset()

			line.Bytes(512, -1)

			require.Equal(t, "\rDownloading feature-compose v0.2.0: 512 B\x1b[K", out.String())
		})

		t.Run("elsewhere, writes nothing, so that a log gets no partial lines", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, false, "Downloading feature-compose v0.2.0")

			line.Bytes(500_000, 2_000_000)
			line.End()

			require.Equal(t, "Downloading feature-compose v0.2.0…\n", out.String())
		})
	})

	t.Run("end", func(t *testing.T) {
		t.Run("on a terminal, ends the line", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "Downloading feature-compose v0.2.0")
			out.Reset()

			line.End()

			require.Equal(t, "\n", out.String())
		})
	})

	t.Run("start on a terminal", func(t *testing.T) {
		t.Run("shortens a label longer than 64 characters, so that the line does not wrap in a terminal of 80 columns", func(t *testing.T) {
			var out bytes.Buffer

			progress.Start(&out, true, strings.Repeat("a", 70))

			require.Equal(t, "\r"+strings.Repeat("a", 63)+"…\x1b[K", out.String())
		})
	})

	t.Run("elapsed", func(t *testing.T) {
		t.Run("on a terminal, draws the time so far in whole seconds after the label", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "app: mix deps.get")
			out.Reset()

			line.Elapsed(8*time.Second + 700*time.Millisecond)

			require.Equal(t, "\rapp: mix deps.get… (8s)\x1b[K", out.String())
		})

		t.Run("elsewhere, writes nothing, so that a log gets no partial lines", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, false, "app: mix deps.get")
			out.Reset()

			line.Elapsed(8 * time.Second)

			require.Empty(t, out.String())
		})
	})

	t.Run("timer", func(t *testing.T) {
		t.Run("on a terminal, draws the time again until stop, and draws nothing after stop", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "app: mix deps.get")
			out.Reset()

			stop := line.Timer(10 * time.Millisecond)
			time.Sleep(50 * time.Millisecond)
			stop()
			drawn := out.String()
			time.Sleep(30 * time.Millisecond)

			require.Equal(t, "\rapp: mix deps.get… (0s)\x1b[K", drawn)
			require.Equal(t, drawn, out.String())
		})

		t.Run("elsewhere, draws nothing", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, false, "app: mix deps.get")
			out.Reset()

			stop := line.Timer(10 * time.Millisecond)
			time.Sleep(30 * time.Millisecond)
			stop()

			require.Empty(t, out.String())
		})
	})

	t.Run("ended", func(t *testing.T) {
		t.Run("on a terminal, draws a check mark and the time on the same line, and ends the line", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "app: mix deps.get")
			out.Reset()

			line.Ended(true, 14*time.Second+400*time.Millisecond)

			require.Equal(t, "\r✓ app: mix deps.get (14s)\x1b[K\n", out.String())
		})

		t.Run("on a terminal, draws a cross for work that failed", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "app: mix deps.get")
			out.Reset()

			line.Ended(false, 3*time.Second)

			require.Equal(t, "\r✗ app: mix deps.get (3s)\x1b[K\n", out.String())
		})

		t.Run("elsewhere, writes the mark, the label and the time as a whole line", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, false, "app: mix deps.get")
			out.Reset()

			line.Ended(true, 14*time.Second)

			require.Equal(t, "✓ app: mix deps.get (14s)\n", out.String())
		})
	})

	t.Run("clear", func(t *testing.T) {
		t.Run("on a terminal, removes the text of the line and leaves the cursor at its start", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, true, "[3/25] go --feature cli")
			out.Reset()

			line.Clear()

			require.Equal(t, "\r\x1b[K", out.String())
		})

		t.Run("elsewhere, writes nothing", func(t *testing.T) {
			var out bytes.Buffer
			line := progress.Start(&out, false, "[3/25] go --feature cli")
			out.Reset()

			line.Clear()

			require.Empty(t, out.String())
		})
	})
}
