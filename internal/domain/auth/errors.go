package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrPasswordRequired   = errors.New("password is required")
	ErrPasswordTooShort   = errors.New("password must be at least 6 characters")
)
