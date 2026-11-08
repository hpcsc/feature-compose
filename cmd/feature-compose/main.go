package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/hpcsc/feature-compose/internal/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	os.Exit(cmd.Run(ctx))
}
