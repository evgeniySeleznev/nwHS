package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/company/holo/services/customer-service/internal/domain/events"
	"github.com/company/holo/services/customer-service/internal/domain/models"
	"github.com/company/holo/services/customer-service/internal/domain/valueobjects"
	"go.uber.org/zap"
)

// RegisterCustomer описывает команду создания клиента.
type RegisterCustomer struct {
	FullName    string
	Email       string
	PhoneNumber string
	BirthDate   time.Time
}

// CustomerRepository определяет контракты с инфраструктурой хранения.
type CustomerRepository interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Save(ctx context.Context, customer *models.Customer) error
}

// CustomerSearchIndexer индексирует сущность в поисковом движке.
type CustomerSearchIndexer interface {
	Index(ctx context.Context, customer *models.Customer) error
}

// DomainEventPublisher публикует доменные события в шину Kafka.
type DomainEventPublisher interface {
	PublishCustomerRegistered(ctx context.Context, event events.CustomerRegistered) error
}

// RegisterCustomerHandler реализует бизнес-логику регистрации клиента.
type RegisterCustomerHandler struct {
	repo     CustomerRepository
	indexer  CustomerSearchIndexer
	events   DomainEventPublisher
	logger   *zap.Logger
	clockNow func() time.Time
}

// NewRegisterCustomerHandler создаёт обработчик с зависимостями.
func NewRegisterCustomerHandler(repo CustomerRepository, indexer CustomerSearchIndexer, events DomainEventPublisher, logger *zap.Logger) *RegisterCustomerHandler {
	return &RegisterCustomerHandler{
		repo:     repo,
		indexer:  indexer,
		events:   events,
		logger:   logger,
		clockNow: time.Now,
	}
}

// Handle выполняет команду регистрации клиента.
func (h *RegisterCustomerHandler) Handle(ctx context.Context, cmd RegisterCustomer) (string, error) {
	email, err := valueobjects.NewEmail(cmd.Email)
	if err != nil {
		return "", err
	}

	phone, err := valueobjects.NewPhoneNumber(cmd.PhoneNumber)
	if err != nil {
		return "", err
	}

	if cmd.BirthDate.After(h.clockNow()) {
		return "", valueobjects.ErrInvalidBirthDay
	}

	exists, err := h.repo.ExistsByEmail(ctx, email.String())
	if err != nil {
		return "", fmt.Errorf("check email existence: %w", err)
	}
	if exists {
		return "", fmt.Errorf("customer with email %s already exists", email.String())
	}

	customer, err := models.NewCustomer(cmd.FullName, email, phone, cmd.BirthDate)
	if err != nil {
		return "", err
	}

	if err := h.repo.Save(ctx, customer); err != nil {
		return "", fmt.Errorf("save customer: %w", err)
	}

	if err := h.indexer.Index(ctx, customer); err != nil {
		h.logger.Warn("failed to index customer", zap.Error(err), zap.String("customer_id", customer.ID().String()))
	}

	domainEvent := events.CustomerRegistered{
		CustomerID: customer.ID().String(),
		Email:      customer.Email().String(),
		FullName:   customer.FullName(),
		OccurredAt: h.clockNow().UTC(),
	}

	if err := h.events.PublishCustomerRegistered(ctx, domainEvent); err != nil {
		return "", fmt.Errorf("publish event: %w", err)
	}

	return customer.ID().String(), nil
}

// WithClock позволяет переопределить таймер в тестах.
func (h *RegisterCustomerHandler) WithClock(clock func() time.Time) {
	if clock != nil {
		h.clockNow = clock
	}
}
