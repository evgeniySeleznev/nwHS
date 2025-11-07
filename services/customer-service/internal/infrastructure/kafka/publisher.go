package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/company/holo/services/customer-service/internal/domain/events"
	"github.com/segmentio/kafka-go"
)

// Publisher публикует доменные события в Kafka topic.
type Publisher struct {
	writer *kafka.Writer
	topic  string
}

// NewPublisher создаёт новый Publisher.
func NewPublisher(writer *kafka.Writer, topic string) *Publisher {
	return &Publisher{writer: writer, topic: topic}
}

// PublishCustomerRegistered реализует DomainEventPublisher.
func (p *Publisher) PublishCustomerRegistered(ctx context.Context, event events.CustomerRegistered) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	message := kafka.Message{
		Topic: p.topic,
		Key:   []byte(event.CustomerID),
		Value: payload,
	}

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}
