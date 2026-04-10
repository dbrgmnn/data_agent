package queue

import (
	"context"
	"data_agent/internal/models"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type Publisher struct {
	Conn   *amqp.Connection
	Ch     *amqp.Channel
	Q      amqp.Queue
	Ctx    context.Context
	server string
	mu     sync.Mutex
}

// create a new publisher with context
func NewPublisher(ctx context.Context, server string) *Publisher {
	return &Publisher{
		Ctx:    ctx,
		server: server,
	}
}

// connect to RabbitMQ with mutex
func (p *Publisher) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// return nil if connection closed
	if p.Conn != nil && !p.Conn.IsClosed() && p.Ch != nil {
		return nil
	}

	// open connection
	conn, err := amqp.DialConfig(p.server, amqp.Config{
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

	q, err := ch.QueueDeclare("metrics", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	p.Conn = conn
	p.Ch = ch
	p.Q = q
	slog.Info("Publisher connected and queue declared", "queue", q.Name)
	return nil
}

// publish a message without opening a new connection
func (p *Publisher) Publish(metricMsg *models.MetricMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Conn == nil || p.Conn.IsClosed() {
		return fmt.Errorf("cannot publish, connection is closed or nil")
	}

	if p.Ch == nil {
		return fmt.Errorf("cannot publish, channel is closed or nil")
	}

	// marshaling metrics
	body, err := json.Marshal(metricMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// publish metrics
	err = p.Ch.Publish("", p.Q.Name, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("failed to publish metric: %w", err)
	}

	slog.Info("Metric sent to queue", "host", metricMsg.Host.Hostname)
	return nil
}

// publish metrics
func (p *Publisher) StartMetricsPublisher() {
	for {
		if err := p.connect(); err != nil {
			slog.Error("Publisher connection failed, retrying", "error", err)
			select {
			case <-p.Ctx.Done():
				slog.Info("Publisher stopped by context")
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		p.mu.Lock()
		notifyClose := make(chan *amqp.Error, 1)
		p.Conn.NotifyClose(notifyClose)
		p.mu.Unlock()

		select {
		case <-p.Ctx.Done():
			slog.Info("Publisher stopping")
			p.Close()
			return
		case err := <-notifyClose:
			if err != nil {
				slog.Error("Publisher connection closed, reconnecting", "error", err)
			}
		}
	}
}

// close channel and connection gracefully
func (p *Publisher) Close() {
	p.mu.Lock()
	if c := p.Ch; c != nil {
		if err := c.Close(); err != nil {
			slog.Error("Error closing channel", "error", err)
		}
		p.Ch = nil
	}
	if conn := p.Conn; conn != nil && !conn.IsClosed() {
		if err := conn.Close(); err != nil {
			slog.Error("Error closing connection", "error", err)
		}
		p.Conn = nil
	}
	p.mu.Unlock()
	slog.Info("Publisher connection closed")
}
