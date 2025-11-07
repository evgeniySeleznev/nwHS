package queries

import (
	"context"
	"fmt"

	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/domain/models"
)

// CustomerDTO представляет данные для отдачи наружу.
type CustomerDTO struct {
	ID          string
	FullName    string
	Email       string
	PhoneNumber string
}

// CustomerReadModel описывает операции чтения агрегата.
type CustomerReadModel interface {
	GetByID(ctx context.Context, id string) (*models.Customer, error)
}

// GetCustomerHandler обрабатывает запрос на получение клиента.
type GetCustomerHandler struct {
	readModel CustomerReadModel
}

// NewGetCustomerHandler создаёт обработчик.
func NewGetCustomerHandler(readModel CustomerReadModel) *GetCustomerHandler {
	return &GetCustomerHandler{readModel: readModel}
}

// Handle возвращает DTO клиента.
func (h *GetCustomerHandler) Handle(ctx context.Context, id string) (CustomerDTO, error) {
	customer, err := h.readModel.GetByID(ctx, id)
	if err != nil {
		return CustomerDTO{}, fmt.Errorf("get customer by id: %w", err)
	}

	return CustomerDTO{
		ID:          customer.ID().String(),
		FullName:    customer.FullName(),
		Email:       customer.Email().String(),
		PhoneNumber: customer.PhoneNumber().String(),
	}, nil
}
