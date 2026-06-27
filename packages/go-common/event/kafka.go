// Package event provides Kafka producer and consumer abstractions.
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"
)

// Event represents a domain event.
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// Producer sends events to Kafka topics.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer.
func NewProducer(brokers []string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	return &Producer{writer: writer}
}

// Publish sends an event to a Kafka topic.
func (p *Producer) Publish(ctx context.Context, topic string, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.ID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "event_source", Value: []byte(event.Source)},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish event to topic %s: %w", topic, err)
	}

	logger.Log.Debug("Event published",
		zap.String("topic", topic),
		zap.String("event_id", event.ID),
		zap.String("event_type", event.Type),
	)

	return nil
}

// Close closes the producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// EventHandler is a function that handles consumed events.
type EventHandler func(ctx context.Context, event *Event) error

// Consumer reads events from Kafka topics.
type Consumer struct {
	reader   *kafka.Reader
	handlers map[string]EventHandler
}

// NewConsumer creates a new Kafka consumer.
func NewConsumer(brokers []string, groupID string, topics ...string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupTopics:    topics,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: 1 * time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &Consumer{
		reader:   reader,
		handlers: make(map[string]EventHandler),
	}
}

// RegisterHandler registers an event handler for a specific event type.
func (c *Consumer) RegisterHandler(eventType string, handler EventHandler) {
	c.handlers[eventType] = handler
}

// Start begins consuming events. Blocks until the context is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	logger.Log.Info("Kafka consumer started")

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Kafka consumer stopping")
			return c.reader.Close()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return c.reader.Close()
				}
				logger.Log.Error("Failed to read message", zap.Error(err))
				continue
			}

			var event Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				logger.Log.Error("Failed to unmarshal event",
					zap.Error(err),
					zap.ByteString("value", msg.Value),
				)
				continue
			}

			handler, exists := c.handlers[event.Type]
			if !exists {
				logger.Log.Debug("No handler for event type",
					zap.String("event_type", event.Type),
					zap.String("event_id", event.ID),
				)
				continue
			}

			if err := handler(ctx, &event); err != nil {
				logger.Log.Error("Event handler failed",
					zap.String("event_type", event.Type),
					zap.String("event_id", event.ID),
					zap.Error(err),
				)
				continue
			}

			logger.Log.Debug("Event processed",
				zap.String("event_type", event.Type),
				zap.String("event_id", event.ID),
			)
		}
	}
}

// Close closes the consumer.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// NewEvent creates a new event with the given type, source, and data.
func NewEvent(eventType, source string, data interface{}) (*Event, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return &Event{
		ID:        fmt.Sprintf("%s_%d", eventType, time.Now().UnixNano()),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Data:      dataBytes,
	}, nil
}
