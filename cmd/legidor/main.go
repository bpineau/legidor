// Command legidor provides verified access to official French legal sources.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bpineau/legidor/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(app.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
