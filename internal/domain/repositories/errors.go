package repositories

import "errors"

var (
	ErrNotFound = errors.New("resource not found")
	ErrExists   = errors.New("resource already exists")
	ErrInvalid  = errors.New("invalid resource")
)
