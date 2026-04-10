package main

import (
	"context"
	"data_agent/internal/config"
	dataBase "data_agent/internal/db"
	"data_agent/internal/queue"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// main function to start the RabbitMQ consumer
func main() {
	// initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// initialize database
	db, err := dataBase.InitDB()
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// load configuration
	cfg := config.LoadConfig()
	rabbitURL := cfg.RabbitURL

	// create a context that is canceled on exit
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// handle termination signals in a separate goroutine
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		slog.Info("Stopping ingestor...")
		cancel()
	}()

	// create and start consumer
	consumer := queue.NewConsumer(ctx, db, rabbitURL)
	consumer.StartMetricsConsumer()
}
