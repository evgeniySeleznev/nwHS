package valueobjects

import (
	"regexp"
	"strings"
)

var phoneRegexp = regexp.MustCompile(`^[0-9+\-()\s]{7,20}$`)

// PhoneNumber представляет value object номера телефона.
type PhoneNumber struct {
	value string
}

// NewPhoneNumber валидирует номер телефона по простому паттерну.
func NewPhoneNumber(raw string) (PhoneNumber, error) {
	normalized := strings.TrimSpace(raw)
	if !phoneRegexp.MatchString(normalized) {
		return PhoneNumber{}, ErrInvalidPhone
	}
	return PhoneNumber{value: normalized}, nil
}

// String возвращает строковое представление номера телефона.
func (p PhoneNumber) String() string {
	return p.value
}
