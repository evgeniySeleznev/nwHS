package valueobjects

import "net/mail"

// Email представляет value object для адреса электронной почты.
type Email struct {
	value string
}

// NewEmail валидирует и создаёт email.
func NewEmail(raw string) (Email, error) {
	if _, err := mail.ParseAddress(raw); err != nil {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: raw}, nil
}

// String возвращает строковое представление email.
func (e Email) String() string {
	return e.value
}
