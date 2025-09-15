package gomedia

import (
	"context"
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockStorageDriver is a testify.Mock implementation of StorageDriver.
type MockStorageDriver struct {
	mock.Mock
}

func (m *MockStorageDriver) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageDriver) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageDriver) GetSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, key, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockStorageDriver) GetURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockStorageDriver) Put(ctx context.Context, key string, file io.Reader) (string, error) {
	args := m.Called(ctx, key, file)
	return args.String(0), args.Error(1)
}
