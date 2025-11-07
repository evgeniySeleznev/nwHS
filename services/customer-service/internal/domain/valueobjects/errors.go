package valueobjects

import "errors"

var (
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPhone    = errors.New("invalid phone number")
	ErrEmptyFullName   = errors.New("full name must not be empty")
	ErrInvalidBirthDay = errors.New("birth date must be in the past")
)
