package cmd

import (
	"io"
	"time"

	"github.com/hpcsc/feature-compose/internal/progress"
)

func stepsOn(w io.Writer) stepLines {
	return stepLines{w: w, terminal: isTerminal(w)}
}

type stepLines struct {
	w        io.Writer
	terminal bool
}

func (s stepLines) Start(label string) func(error) {
	started := time.Now()
	line := progress.Start(s.w, s.terminal, label)
	stopTimer := line.Timer(time.Second)
	return func(err error) {
		stopTimer()
		line.Ended(err == nil, time.Since(started))
	}
}
