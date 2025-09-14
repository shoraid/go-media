package gomedia

import (
	"context"
	"io"
	"time"

	"golang.org/x/sync/errgroup"
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

// NewManager creates a new MediaManager with a default storage alias.
// Returns an error if the alias does not exist in the provided storage map.
func NewManager(defaultStorageAlias string, storage map[string]MediaManager) (MediaManager, error) {
	defaultStorage, exists := storage[defaultStorageAlias]
	if !exists {
		return nil, ErrInvalidDefaultStorage
	}

	return &mediaManagerImpl{
		storage,
		defaultStorage,
	}, nil
}

// Storage returns a new MediaManager using the given alias as its default storage.
// If alias is not found, defaultStorage will be nil (be careful when calling methods).
func (m *mediaManagerImpl) Storage(alias string) MediaManager {
	return &mediaManagerImpl{
		storageMap:     m.storageMap,
		defaultStorage: m.storageMap[alias],
	}
}

// Delete removes a single file from the storage.
func (m *mediaManagerImpl) Delete(ctx context.Context, key string) error {
	return m.defaultStorage.Delete(ctx, key)
}

// DeleteMany removes multiple files concurrently from the storage.
// Uses errgroup to run deletions in parallel and return the first error encountered.
func (m *mediaManagerImpl) DeleteMany(ctx context.Context, keys ...string) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, key := range keys {
		key := key // avoid closure capture bug
		g.Go(func() error {
			return m.Delete(ctx, key)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// Exists checks whether a file exists in the storage.
func (m *mediaManagerImpl) Exists(ctx context.Context, key string) (bool, error) {
	return m.defaultStorage.Exists(ctx, key)
}

// GetSignedURL returns a temporary signed URL for accessing the file in storage.
func (m *mediaManagerImpl) GetSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return m.defaultStorage.GetSignedURL(ctx, key, expiry)
}

// GetSignedURLs returns signed URLs for multiple files concurrently.
func (m *mediaManagerImpl) GetSignedURLs(ctx context.Context, keys []string, expiry time.Duration) ([]string, error) {
	urls := make([]string, len(keys))
	g, ctx := errgroup.WithContext(ctx)

	for i, key := range keys {
		i, key := i, key // avoid closure capture bug

		g.Go(func() error {
			url, err := m.GetSignedURL(ctx, key, expiry)
			if err != nil {
				return err
			}

			urls[i] = url
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return urls, nil
}

// GetURL returns the direct (public) URL of a file from the storage.
func (m *mediaManagerImpl) GetURL(ctx context.Context, key string) (string, error) {
	return m.defaultStorage.GetURL(ctx, key)
}

// GetURLs returns direct URLs for multiple files concurrently.
func (m *mediaManagerImpl) GetURLs(ctx context.Context, keys []string) ([]string, error) {
	urls := make([]string, len(keys))
	g, ctx := errgroup.WithContext(ctx)

	for i, key := range keys {
		i, key := i, key // avoid closure capture bug

		g.Go(func() error {
			url, err := m.GetURL(ctx, key)
			if err != nil {
				return err
			}

			urls[i] = url
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return urls, nil
}

// Missing returns true if the file does not exist in the storage.
func (m *mediaManagerImpl) Missing(ctx context.Context, key string) (bool, error) {
	exists, err := m.Exists(ctx, key)
	if err != nil {
		return false, err
	}

	return !exists, nil
}

// Put uploads a file to the storage and returns its resulting URL.
func (m *mediaManagerImpl) Put(ctx context.Context, file io.Reader, key string) (string, error) {
	return m.defaultStorage.Put(ctx, file, key)
}
