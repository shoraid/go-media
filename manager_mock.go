package gostorage

import (
	"context"
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockStorageManager is a testify mock implementing gostorage.StorageManager
type MockStorageManager struct {
	mock.Mock
}

var _ StorageManager = (*MockStorageManager)(nil)

func (m *MockStorageManager) Storage(alias string) StorageManager {
	args := m.Called(alias)
	if mgr, ok := args.Get(0).(StorageManager); ok {
		return mgr
	}
	return nil
}

func (m *MockStorageManager) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageManager) DeleteMany(ctx context.Context, keys ...string) error {
	callArgs := append([]any{ctx}, stringSliceToInterface(keys)...)
	args := m.Called(callArgs...)
	return args.Error(0)
}

func (m *MockStorageManager) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageManager) GetSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, key, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockStorageManager) GetSignedURLs(ctx context.Context, keys []string, expiry time.Duration) ([]string, error) {
	args := m.Called(ctx, keys, expiry)
	if urls, ok := args.Get(0).([]string); ok {
		return urls, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStorageManager) GetURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockStorageManager) GetURLs(ctx context.Context, keys []string) ([]string, error) {
	args := m.Called(ctx, keys)
	if urls, ok := args.Get(0).([]string); ok {
		return urls, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStorageManager) Missing(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageManager) Put(ctx context.Context, key string, file io.Reader) (string, error) {
	args := m.Called(ctx, file, key)
	return args.String(0), args.Error(1)
}

func stringSliceToInterface(slice []string) []any {
	res := make([]any, len(slice))
	for i, v := range slice {
		res[i] = v
	}
	return res
}
