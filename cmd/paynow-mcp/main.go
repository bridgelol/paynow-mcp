package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bridgelol/paynow-mcp/internal/mcp"
	"github.com/bridgelol/paynow-mcp/internal/paynow"
)

const version = "0.1.0"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := paynow.NewClient(paynow.ConfigFromEnv())
	server := mcp.NewServer("paynow-mcp", version, paynow.Tools(client))

	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
