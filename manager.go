package gomedia

import (
	"context"
	"io"
	"time"
)

// MediaManager is the main interface for managing media files.
// It provides methods for uploading, deleting, checking existence,
// and retrieving signed or public URLs for files across different storages.
type MediaManager interface {
	// Storage returns a new MediaManager that uses the storage alias provided.
	// Useful when you have multiple storage backends and need to switch dynamically.
	Storage(alias string) MediaManager

	// Delete removes a single file identified by key.
	Delete(ctx context.Context, key string) error

	// DeleteMany removes multiple files concurrently.
	DeleteMany(ctx context.Context, keys ...string) error

	// Exists checks if a file exists by key.
	Exists(ctx context.Context, key string) (bool, error)

	// GetSignedURL returns a temporary signed URL for accessing a file.
	// This is typically used for private storages with time-limited access.
	GetSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)

	// GetSignedURLs returns signed URLs for multiple files concurrently.
	GetSignedURLs(ctx context.Context, keys []string, expiry time.Duration) ([]string, error)

	// GetURL returns a public URL for a file.
	// This is typically used for public storages where files can be accessed directly.
	GetURL(ctx context.Context, key string) (string, error)

	// GetURLs returns public URLs for multiple files concurrently.
	GetURLs(ctx context.Context, keys []string) ([]string, error)

	// Missing returns true if a file does NOT exist (inverse of Exists).
	Missing(ctx context.Context, key string) (bool, error)

	// Put uploads a file to the storage with the given key and returns its URL.
	Put(ctx context.Context, file io.Reader, key string) (string, error)
}

// mediaManagerImpl is the concrete implementation of MediaManager.
// It delegates calls to the defaultStorage or a selected storage from storageMap.
type mediaManagerImpl struct {
	storageMap     map[string]MediaManager // all available storages by alias
	defaultStorage MediaManager            // the currently selected storage
}
