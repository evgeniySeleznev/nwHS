package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	appqueries "github.com/evgeniySeleznev/nwHS/services/customer-service/internal/application/queries"
	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/domain/events"
	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/domain/models"
	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/domain/valueobjects"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fakeRepo struct {
	exists bool
	saved  *models.Customer
	err    error
}

func (f *fakeRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return f.exists, f.err
}

func (f *fakeRepo) Save(ctx context.Context, customer *models.Customer) error {
	f.saved = customer
	return f.err
}

func (f *fakeRepo) GetByID(ctx context.Context, id string) (*models.Customer, error) {
	return f.saved, f.err
}

type fakeIndexer struct{ err error }

func (f *fakeIndexer) Index(ctx context.Context, customer *models.Customer) error {
	return f.err
}

type fakePublisher struct {
	event events.CustomerRegistered
	err   error
}

func (f *fakePublisher) PublishCustomerRegistered(ctx context.Context, event events.CustomerRegistered) error {
	f.event = event
	return f.err
}

func TestRegisterCustomerHandler_Handle(t *testing.T) {
	repo := &fakeRepo{}
	indexer := &fakeIndexer{}
	publisher := &fakePublisher{}
	handler := NewRegisterCustomerHandler(repo, indexer, publisher, zap.NewNop())

	fixedTime := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	handler.WithClock(func() time.Time { return fixedTime })

	id, err := handler.Handle(context.Background(), RegisterCustomer{
		FullName:    "John Doe",
		Email:       "john@example.com",
		PhoneNumber: "+1234567890",
		BirthDate:   time.Date(1990, 5, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id == "" {
		t.Fatalf("expected id to be returned")
	}

	if repo.saved == nil {
		t.Fatalf("expected customer to be saved")
	}

	if publisher.event.CustomerID == "" {
		t.Fatalf("expected event to be published")
	}
}

func TestRegisterCustomerHandler_Duplicate(t *testing.T) {
	repo := &fakeRepo{exists: true}
	handler := NewRegisterCustomerHandler(repo, &fakeIndexer{}, &fakePublisher{}, zap.NewNop())
	_, err := handler.Handle(context.Background(), RegisterCustomer{
		FullName:    "John Doe",
		Email:       "john@example.com",
		PhoneNumber: "+1234567890",
		BirthDate:   time.Date(1990, 5, 10, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected duplication error")
	}
}

func TestRegisterCustomerHandler_InvalidEmail(t *testing.T) {
	handler := NewRegisterCustomerHandler(&fakeRepo{}, &fakeIndexer{}, &fakePublisher{}, zap.NewNop())
	_, err := handler.Handle(context.Background(), RegisterCustomer{Email: "broken"})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestRegisterCustomerHandler_RepositoryError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db down")}
	handler := NewRegisterCustomerHandler(repo, &fakeIndexer{}, &fakePublisher{}, zap.NewNop())
	_, err := handler.Handle(context.Background(), RegisterCustomer{
		FullName:    "John Doe",
		Email:       "john@example.com",
		PhoneNumber: "+1234567890",
		BirthDate:   time.Date(1990, 5, 10, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected repository error")
	}
}

func TestGetCustomerHandler_Handle(t *testing.T) {
	repo := &fakeRepo{}
	handler := appqueries.NewGetCustomerHandler(repo)

	customer, _ := models.NewCustomer("John Doe", mustEmail("john@example.com"), mustPhone("+1234567890"), time.Date(1990, 5, 10, 0, 0, 0, 0, time.UTC))
	repo.saved = models.RehydrateCustomer(uuid.MustParse(customer.ID().String()), customer.Email(), customer.FullName(), customer.PhoneNumber(), customer.BirthDate(), customer.CreatedAt(), customer.UpdatedAt(), customer.Version())

	dto, err := handler.Handle(context.Background(), repo.saved.ID().String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.Email != "john@example.com" {
		t.Fatalf("expected email to match")
	}
}

func mustEmail(val string) valueobjects.Email {
	email, _ := valueobjects.NewEmail(val)
	return email
}

func mustPhone(val string) valueobjects.PhoneNumber {
	phone, _ := valueobjects.NewPhoneNumber(val)
	return phone
}
