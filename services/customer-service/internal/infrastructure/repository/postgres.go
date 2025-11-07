package repository

import (
	"context"
	"errors"
	"fmt"

	"time"

	"github.com/company/holo/services/customer-service/internal/domain/models"
	"github.com/company/holo/services/customer-service/internal/domain/valueobjects"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository реализует CustomerRepository поверх PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository создаёт экземпляр.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ExistsByEmail проверяет наличие клиента по email.
func (r *PostgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const query = `SELECT true FROM customers WHERE email = $1 LIMIT 1`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("postgres exists by email: %w", err)
	}

	return exists, nil
}

// Save сохраняет нового клиента.
func (r *PostgresRepository) Save(ctx context.Context, customer *models.Customer) error {
	const stmt = `INSERT INTO customers (
        id, email, full_name, phone_number, birth_date, created_at, updated_at, version
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, stmt,
		customer.ID(),
		customer.Email().String(),
		customer.FullName(),
		customer.PhoneNumber().String(),
		customer.BirthDate(),
		customer.CreatedAt(),
		customer.UpdatedAt(),
		customer.Version(),
	)
	if err != nil {
		return fmt.Errorf("postgres save customer: %w", err)
	}

	return nil
}

// GetByID возвращает клиента по идентификатору.
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*models.Customer, error) {
	const query = `SELECT id, email, full_name, phone_number, birth_date, created_at, updated_at, version
        FROM customers WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)

	var (
		customerID uuid.UUID
		emailRaw   string
		fullName   string
		phoneRaw   string
		birthDate  time.Time
		createdAt  time.Time
		updatedAt  time.Time
		version    int
	)

	if err := row.Scan(&customerID, &emailRaw, &fullName, &phoneRaw, &birthDate, &createdAt, &updatedAt, &version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("customer not found: %w", err)
		}
		return nil, fmt.Errorf("postgres get by id: %w", err)
	}

	email, err := valueobjects.NewEmail(emailRaw)
	if err != nil {
		return nil, err
	}

	phone, err := valueobjects.NewPhoneNumber(phoneRaw)
	if err != nil {
		return nil, err
	}

	customer := models.RehydrateCustomer(
		customerID,
		email,
		fullName,
		phone,
		birthDate,
		createdAt,
		updatedAt,
		version,
	)

	return customer, nil
}
