package events

import "time"

// CustomerRegistered описывает доменное событие регистрации клиента.
type CustomerRegistered struct {
	CustomerID string
	Email      string
	FullName   string
	OccurredAt time.Time
}
