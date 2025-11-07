package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/domain/events"
	mongodlq "github.com/evgeniySeleznev/nwHS/services/customer-service/internal/infrastructure/mongo"
	"github.com/segmentio/kafka-go"
)

// Publisher публикует доменные события в Kafka topic.
type Publisher struct {
	writer *kafka.Writer
	topic  string
	dlq    *mongodlq.DeadLetterRepository
}

// NewPublisher создаёт новый Publisher.
func NewPublisher(writer *kafka.Writer, topic string, dlq *mongodlq.DeadLetterRepository) *Publisher {
	return &Publisher{writer: writer, topic: topic, dlq: dlq}
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
		if p.dlq != nil {
			_ = p.dlq.SaveCustomerEvent(ctx, event, payload, err)
		}
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}
