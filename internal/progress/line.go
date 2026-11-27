package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// maxTerminalLabel keeps the line, with its time, inside a terminal of 80
// columns. A longer line wraps, and \r then draws only its last part again.
const maxTerminalLabel = 64

// Line shows work that takes time, such as a download, on one line. On a
// terminal it draws the line again as the work goes on. Elsewhere it writes the
// label one time, so that a log gets no partial lines.
type Line struct {
	w        io.Writer
	terminal bool
	label    string
	shown    string
	drawing  sync.Mutex
}

func Start(w io.Writer, terminal bool, label string) *Line {
	if terminal {
		label = shortened(label)
	}
	l := &Line{w: w, terminal: terminal, label: label}
	if terminal {
		l.draw(l.running())
	} else {
		fmt.Fprintln(w, l.running())
	}
	return l
}

// Bytes shows the bytes that have arrived, of total. total is -1 when it is
// not known.
func (l *Line) Bytes(done, total int64) {
	if !l.terminal {
		return
	}
	text := l.label + ": " + size(done)
	if total > 0 {
		text += fmt.Sprintf(" of %s (%d%%)", size(total), done*100/total)
	}
	l.draw(text)
}

func (l *Line) End() {
	if l.terminal {
		fmt.Fprintln(l.w)
	}
}

// Elapsed shows the time that the work has taken so far.
func (l *Line) Elapsed(elapsed time.Duration) {
	if !l.terminal {
		return
	}
	l.draw(fmt.Sprintf("%s (%s)", l.running(), elapsed.Truncate(time.Second)))
}

// Timer shows the time since now after the label, each interval, until stop.
// It draws nothing outside a terminal.
func (l *Line) Timer(interval time.Duration) (stop func()) {
	if !l.terminal {
		return func() {}
	}
	started := time.Now()
	done, finished := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case now := <-ticker.C:
				l.Elapsed(now.Sub(started))
			}
		}
	}()
	return func() {
		close(done)
		<-finished
	}
}

// Ended shows the work as done, or as failed, with the time that it took, and
// ends the line.
func (l *Line) Ended(ok bool, took time.Duration) {
	mark := "✓"
	if !ok {
		mark = "✗"
	}
	text := fmt.Sprintf("%s %s (%s)", mark, l.label, took.Round(time.Second))
	if !l.terminal {
		fmt.Fprintln(l.w, text)
		return
	}
	l.draw(text)
	fmt.Fprintln(l.w)
}

// Clear removes the line from a terminal, so that the caller can write its own
// line in its place.
func (l *Line) Clear() {
	if l.terminal {
		l.draw("")
	}
}

func (l *Line) draw(text string) {
	l.drawing.Lock()
	defer l.drawing.Unlock()
	if text == l.shown {
		return
	}
	// \r goes back to the start of the line, and \x1b[K clears what a longer
	// text left after it.
	fmt.Fprint(l.w, "\r"+text+"\x1b[K")
	l.shown = text
}

func (l *Line) running() string {
	return strings.TrimSuffix(l.label, "…") + "…"
}

func shortened(label string) string {
	runes := []rune(label)
	if len(runes) <= maxTerminalLabel {
		return label
	}
	return string(runes[:maxTerminalLabel-1]) + "…"
}

func size(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d B", n)
	case n < 1000*1000:
		return fmt.Sprintf("%.1f KB", float64(n)/1000)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/1000/1000)
	}
}
