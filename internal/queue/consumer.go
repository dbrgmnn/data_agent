package queue

import (
	"context"
	dataBase "data_agent/internal/db"
	"data_agent/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/streadway/amqp"
)

type Consumer struct {
	Conn      *amqp.Connection
	Ch        *amqp.Channel
	Db        *sql.DB
	Ctx       context.Context
	RabbitURL string
}

// create a new consumer with context
func NewConsumer(ctx context.Context, db *sql.DB, rabbitURL string) *Consumer {
	return &Consumer{
		Db:        db,
		Ctx:       ctx,
		RabbitURL: rabbitURL,
	}
}

// connect to RabbitMQ
func (c *Consumer) connect() error {
	// open connection
	conn, err := amqp.DialConfig(c.RabbitURL, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// open channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	c.Conn = conn
	c.Ch = ch
	return nil
}

// saves a metric to the database
func (c *Consumer) ConsumeMetrics() error {
	// declare a queue
	q, err := c.Ch.QueueDeclare("metrics", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// subscribe to the queue
	msgs, err := c.Ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to the queue: %w", err)
	}

	// process messages
	for {
		select {
		case <-c.Ctx.Done():
			return nil

		case d, ok := <-msgs:
			if !ok {
				slog.Info("Message channel closed")
				return nil
			}

			// decode message
			var metric models.MetricMessage
			if err := json.Unmarshal(d.Body, &metric); err != nil {
				slog.Error("Failed to decode metric", "error", err)
				// don't send to queue
				d.Nack(false, false)
				continue
			}

			// send metric to database
			if err := dataBase.SaveMetric(c.Ctx, c.Db, &metric); err != nil {
				slog.Error("Failed to save metric", "error", err, "host", metric.Host.Hostname)
				// send to queue again
				d.Nack(false, true)
				continue
			}

			// acknowledge message
			d.Ack(false)
			slog.Info("Metric saved from queue", "host", metric.Host.Hostname)
		}
	}
}

// consume metrics
func (c *Consumer) StartMetricsConsumer() {
	for {
		if err := c.connect(); err != nil {
			slog.Error("Consumer connection failed, retrying in 5s", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		slog.Info("Connected to RabbitMQ")
		if err := c.ConsumeMetrics(); err != nil {
			slog.Warn("Consume error, reconnecting", "error", err)
		}

		select {
		case <-c.Ctx.Done():
			slog.Info("Consumer stopped by context")
			c.Close()
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// close channel and connection gracefully
func (c *Consumer) Close() {
	if c.Ch != nil {
		if err := c.Ch.Close(); err != nil {
			slog.Error("Error closing channel", "error", err)
		}
		c.Ch = nil
	}
	if c.Conn != nil {
		if err := c.Conn.Close(); err != nil {
			slog.Error("Error closing connection", "error", err)
		}
		c.Conn = nil
	}
	slog.Info("Consumer connection closed")
}
