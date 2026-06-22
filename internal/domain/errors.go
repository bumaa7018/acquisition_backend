package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token expired")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInternalServer     = errors.New("internal server error")
	ErrDecreeRequired     = errors.New("decree number and date are required")
)
