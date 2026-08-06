package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrPasswordRequired   = errors.New("password is required")
	ErrPasswordTooShort   = errors.New("password must be at least 6 characters")
	ErrTokenInvalid        = errors.New("reset link is invalid or has already been used")
	ErrTokenExpired        = errors.New("reset link has expired")
)
