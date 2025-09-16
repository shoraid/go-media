package s3driver

import (
	"context"
	"io"
	"time"
)

// MockObjectStorage is a mock implementation of the gostorage.StorageDriver interface for S3.
type MockObjectStorage struct {
	MockDelete       func(ctx context.Context, key string) error
	MockExists       func(ctx context.Context, key string) (bool, error)
	MockGetSignedURL func(ctx context.Context, key string, expiry time.Duration) (string, error)
	MockGetURL       func(ctx context.Context, key string) (string, error)
	MockPut          func(ctx context.Context, file io.Reader, key string) (url string, err error)
}

// Delete calls the MockDelete function.
func (m *MockObjectStorage) Delete(ctx context.Context, key string) error {
	if m.MockDelete != nil {
		return m.MockDelete(ctx, key)
	}
	return nil
}

// Exists calls the MockExists function.
func (m *MockObjectStorage) Exists(ctx context.Context, key string) (exists bool, err error) {
	if m.MockExists != nil {
		return m.MockExists(ctx, key)
	}
	return false, nil
}

// GetSignedURL calls the MockGetSignedURL function.
func (m *MockObjectStorage) GetSignedURL(ctx context.Context, key string, expiry time.Duration) (url string, err error) {
	if m.MockGetSignedURL != nil {
		return m.MockGetSignedURL(ctx, key, expiry)
	}
	return "", nil
}

// GetURL calls the MockGetURL function.
func (m *MockObjectStorage) GetURL(ctx context.Context, key string) (url string, err error) {
	if m.MockGetURL != nil {
		return m.MockGetURL(ctx, key)
	}
	return "", nil
}

// Put calls the MockPut function.
func (m *MockObjectStorage) Put(ctx context.Context, file io.Reader, key string) (url string, err error) {
	if m.MockPut != nil {
		return m.MockPut(ctx, file, key)
	}
	return "", nil
}
