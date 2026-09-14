package auth

import "errors"

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email taken")
	ErrLocked             = errors.New("account locked")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrRefreshInvalid     = errors.New("refresh invalid")
	ErrNotFound           = errors.New("not found")
)
