package gomedia

import (
	"context"
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockMediaManager is a testify mock implementing gomedia.MediaManager
type MockMediaManager struct {
	mock.Mock
}

var _ MediaManager = (*MockMediaManager)(nil)

func (m *MockMediaManager) Storage(alias string) MediaManager {
	args := m.Called(alias)
	if mgr, ok := args.Get(0).(MediaManager); ok {
		return mgr
	}
	return nil
}

func (m *MockMediaManager) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockMediaManager) DeleteMany(ctx context.Context, keys ...string) error {
	callArgs := append([]any{ctx}, stringSliceToInterface(keys)...)
	args := m.Called(callArgs...)
	return args.Error(0)
}

func (m *MockMediaManager) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockMediaManager) GetSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, key, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockMediaManager) GetSignedURLs(ctx context.Context, keys []string, expiry time.Duration) ([]string, error) {
	args := m.Called(ctx, keys, expiry)
	if urls, ok := args.Get(0).([]string); ok {
		return urls, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMediaManager) GetURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockMediaManager) GetURLs(ctx context.Context, keys []string) ([]string, error) {
	args := m.Called(ctx, keys)
	if urls, ok := args.Get(0).([]string); ok {
		return urls, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMediaManager) Missing(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockMediaManager) Put(ctx context.Context, file io.Reader, key string) (string, error) {
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
