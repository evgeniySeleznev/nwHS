package models

import (
	"time"

	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/domain/valueobjects"
	"github.com/google/uuid"
)

// Customer отражает агрегат клиента оздоровительного центра.
type Customer struct {
	id          uuid.UUID
	email       valueobjects.Email
	fullName    string
	birthDate   time.Time
	phoneNumber valueobjects.PhoneNumber
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// NewCustomer создаёт нового клиента и валидирует входные данные.
func NewCustomer(fullName string, email valueobjects.Email, phone valueobjects.PhoneNumber, birthDate time.Time) (*Customer, error) {
	if fullName == "" {
		return nil, valueobjects.ErrEmptyFullName
	}

	now := time.Now().UTC()

	return &Customer{
		id:          uuid.New(),
		email:       email,
		fullName:    fullName,
		birthDate:   birthDate,
		phoneNumber: phone,
		createdAt:   now,
		updatedAt:   now,
		version:     1,
	}, nil
}

// RehydrateCustomer восстанавливает агрегат из слоя хранения.
func RehydrateCustomer(id uuid.UUID, email valueobjects.Email, fullName string, phone valueobjects.PhoneNumber, birthDate, createdAt, updatedAt time.Time, version int) *Customer {
	return &Customer{
		id:          id,
		email:       email,
		fullName:    fullName,
		birthDate:   birthDate,
		phoneNumber: phone,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}
}

// UpdateEmail обновляет email и инкрементирует версию.
func (c *Customer) UpdateEmail(email valueobjects.Email) {
	c.email = email
	c.touch()
}

// UpdatePhoneNumber обновляет номер телефона.
func (c *Customer) UpdatePhoneNumber(phone valueobjects.PhoneNumber) {
	c.phoneNumber = phone
	c.touch()
}

// UpdateFullName обновляет имя клиента.
func (c *Customer) UpdateFullName(name string) {
	if name == "" {
		return
	}
	c.fullName = name
	c.touch()
}

// ID возвращает идентификатор клиента.
func (c *Customer) ID() uuid.UUID { return c.id }

// Email возвращает email клиента.
func (c *Customer) Email() valueobjects.Email { return c.email }

// FullName возвращает имя клиента.
func (c *Customer) FullName() string { return c.fullName }

// PhoneNumber возвращает номер телефона.
func (c *Customer) PhoneNumber() valueobjects.PhoneNumber { return c.phoneNumber }

// BirthDate возвращает дату рождения клиента.
func (c *Customer) BirthDate() time.Time { return c.birthDate }

// CreatedAt возвращает время создания клиента.
func (c *Customer) CreatedAt() time.Time { return c.createdAt }

// UpdatedAt возвращает время последнего обновления.
func (c *Customer) UpdatedAt() time.Time { return c.updatedAt }

// Version возвращает текущую версию агрегата.
func (c *Customer) Version() int { return c.version }

func (c *Customer) touch() {
	c.updatedAt = time.Now().UTC()
	c.version++
}
