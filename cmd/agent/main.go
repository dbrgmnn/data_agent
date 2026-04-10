package main

import (
	"context"
	"data_agent/internal/agent"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// main function to run the agent
func main() {
	// initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// create a context that is canceled on exit
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// handle termination signals in a separate goroutine
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		slog.Info("Stopping agent...")
		cancel()
	}()

	// parse flags and run the agent
	url, interval, err := agent.ParseFlags()
	if err != nil {
		slog.Error("Failed to parse flags", "error", err)
		os.Exit(1)
	}
	agent.Run(ctx, url, interval)
}
