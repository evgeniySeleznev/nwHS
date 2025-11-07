package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/domain/models"
	opensearch "github.com/opensearch-project/opensearch-go/v2"
)

// Indexer отвечает за индексацию клиентов в OpenSearch/Elasticsearch.
type Indexer struct {
	client *opensearch.Client
	index  string
}

// NewIndexer создаёт новый индексатор.
func NewIndexer(client *opensearch.Client, index string) *Indexer {
	return &Indexer{client: client, index: index}
}

// Index публикует клиента в поисковый индекс.
func (i *Indexer) Index(ctx context.Context, customer *models.Customer) error {
	payload := map[string]interface{}{
		"id":           customer.ID().String(),
		"email":        customer.Email().String(),
		"full_name":    customer.FullName(),
		"phone_number": customer.PhoneNumber().String(),
		"birth_date":   customer.BirthDate(),
		"created_at":   customer.CreatedAt(),
		"updated_at":   customer.UpdatedAt(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal search payload: %w", err)
	}

	response, err := i.client.Index(i.index, bytesReader(body), i.client.Index.WithContext(ctx), i.client.Index.WithDocumentID(customer.ID().String()), i.client.Index.WithRefresh("true"))
	if err != nil {
		return fmt.Errorf("index customer: %w", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		return fmt.Errorf("index customer: status %s", response.Status())
	}

	return nil
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
