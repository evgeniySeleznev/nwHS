package mongo

import (
	"context"
	"time"

	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/domain/events"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// DeadLetterRepository сохраняет неуспешно обработанные события Kafka в MongoDB.
type DeadLetterRepository struct {
	collection *mongo.Collection
}

// NewDeadLetterRepository создаёт репозиторий для DLQ.
func NewDeadLetterRepository(client *mongo.Client, database, collection string) *DeadLetterRepository {
	return &DeadLetterRepository{
		collection: client.Database(database).Collection(collection),
	}
}

// SaveCustomerEvent сохраняет событие о клиенте, которое не удалось записать в Kafka.
func (r *DeadLetterRepository) SaveCustomerEvent(ctx context.Context, event events.CustomerRegistered, payload []byte, publishErr error) error {
	if r == nil || r.collection == nil {
		return nil
	}

	errMsg := ""
	if publishErr != nil {
		errMsg = publishErr.Error()
	}

	doc := bson.M{
		"event": bson.M{
			"customer_id": event.CustomerID,
			"email":       event.Email,
			"full_name":   event.FullName,
			"occurred_at": event.OccurredAt,
		},
		"payload":    payload,
		"error":      errMsg,
		"created_at": time.Now().UTC(),
	}

	_, err := r.collection.InsertOne(ctx, doc)
	return err
}
