package agent

import (
	"context"
	"data_agent/internal/models"
	"data_agent/internal/queue"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"time"
)

// run the agent to collect and send metrics to RabbitMQ
func Run(ctx context.Context, rabbitURL string, interval time.Duration) {
	// create and start publisher
	publisher := queue.NewPublisher(ctx, rabbitURL)
	go publisher.StartMetricsPublisher()

	// send metrics every N seconds
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Agent stopped")
			return
		case <-ticker.C:
			if err := collectAndSend(publisher); err != nil {
				slog.Error("Failed to collect and send metrics", "error", err)
			}
		}
	}
}

// collect and send metrics
func collectAndSend(publisher *queue.Publisher) error {
	host, err := CollectHostInfo()
	if err != nil {
		return err
	}
	metric, err := CollectMetricInfo()
	if err != nil {
		return err
	}
	metricMsg := models.NewMetricMessage(&host, &metric)
	return publisher.Publish(metricMsg)
}

// parse flag --url and --interval
func ParseFlags() (string, time.Duration, error) {
	rabbitURL := flag.String("url", "", "RabbitMQ URL")
	interval := flag.Int("interval", 2, "Interval in seconds between metric collections")
	flag.Parse()

	if *interval <= 0 {
		*interval = 2
	}

	// validate URL format
	if *rabbitURL == "" {
		return "", 0, fmt.Errorf("rabbitMQ URL must be specified with --url")
	}
	u, err := url.Parse(*rabbitURL)
	if err != nil || u.Scheme != "amqp" {
		return "", 0, fmt.Errorf("invalid RabbitMQ URL: expected format amqp://user:pass@host:port/")
	}
	return *rabbitURL, time.Duration(*interval) * time.Second, nil
}
