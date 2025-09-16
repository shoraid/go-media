package gostorage

import "errors"

var (
	ErrInternal              = errors.New("storage: internal storage error")
	ErrInvalidConfig         = errors.New("storage: invalid configuration")
	ErrInvalidDefaultStorage = errors.New("storage: invalid default storage")
	ErrInvalidKey            = errors.New("storage: invalid key name")
	ErrNotFound              = errors.New("storage: file not found")
)
