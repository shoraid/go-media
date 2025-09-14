package gomedia

import "errors"

var (
	ErrInternal              = errors.New("media: internal storage error")
	ErrInvalidConfig         = errors.New("media: invalid configuration")
	ErrInvalidDefaultStorage = errors.New("media: invalid default storage")
	ErrInvalidKey            = errors.New("media: invalid key name")
	ErrNotFound              = errors.New("media: file not found")
)
