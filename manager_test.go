package gomedia

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMediaManager_NewMediaManager(t *testing.T) {
	mockDriver := new(MockStorageDriver)

	tests := []struct {
		name           string
		defaultStorage string
		storageMap     map[string]StorageDriver
		expectErr      error
		expectNil      bool
	}{
		{
			name:           "should return manager when default storage exists",
			defaultStorage: "default",
			storageMap:     map[string]StorageDriver{"default": mockDriver},
			expectErr:      nil,
			expectNil:      false,
		},
		{
			name:           "should return error when default storage does not exist",
			defaultStorage: "missing",
			storageMap:     map[string]StorageDriver{"default": mockDriver},
			expectErr:      ErrInvalidDefaultStorage,
			expectNil:      true,
		},
		{
			name:           "should return error when storage map is empty",
			defaultStorage: "default",
			storageMap:     map[string]StorageDriver{},
			expectErr:      ErrInvalidDefaultStorage,
			expectNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewMediaManager(tt.defaultStorage, tt.storageMap)

			if tt.expectErr != nil {
				assert.ErrorIs(t, err, tt.expectErr, "expected error to match")
				assert.Nil(t, mgr, "expected manager to be nil")
			} else {
				assert.NoError(t, err, "expected no error")
				assert.NotNil(t, mgr, "expected manager to be non-nil")
			}
		})
	}
}

func TestMediaManager_Storage(t *testing.T) {
	mockDefault := new(MockStorageDriver)
	mockOther := new(MockStorageDriver)

	storageMap := map[string]StorageDriver{
		"default": mockDefault,
		"other":   mockOther,
	}

	manager, err := NewMediaManager("default", storageMap)
	assert.NoError(t, err, "expected no error creating manager")

	tests := []struct {
		name       string
		alias      string
		expectNil  bool
		expectSame bool
	}{
		{
			name:       "should return manager with existing alias storage",
			alias:      "other",
			expectNil:  false,
			expectSame: false,
		},
		{
			name:       "should return manager with nil storage when alias does not exist",
			alias:      "missing",
			expectNil:  true,
			expectSame: false,
		},
		{
			name:       "should return manager with same storage when alias is default",
			alias:      "default",
			expectNil:  false,
			expectSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newMgr := manager.Storage(tt.alias)

			impl, ok := newMgr.(*mediaManagerImpl)
			assert.True(t, ok, "expected returned MediaManager to be *mediaManagerImpl")

			if tt.expectNil {
				assert.Nil(t, impl.defaultStorage, "expected defaultStorage to be nil")
			} else {
				assert.NotNil(t, impl.defaultStorage, "expected defaultStorage to be non-nil")
			}

			if tt.expectSame {
				assert.Equal(t, manager.(*mediaManagerImpl).defaultStorage, impl.defaultStorage, "expected same storage reference")
			} else if !tt.expectNil {
				assert.NotSame(t, manager.(*mediaManagerImpl).defaultStorage, impl.defaultStorage, "expected different storage reference")
			}
		})
	}
}

func TestMediaManager_Delete(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	mockDriver := new(MockStorageDriver)

	manager := &mediaManagerImpl{
		storageMap:     map[string]StorageDriver{"default": mockDriver},
		defaultStorage: mockDriver,
	}

	tests := []struct {
		name       string
		key        string
		mockReturn error
		expectErr  bool
	}{
		{
			name:       "should delete key successfully",
			key:        key,
			mockReturn: nil,
			expectErr:  false,
		},
		{
			name:       "should return error when delete fails",
			key:        key,
			mockReturn: errors.New("delete failed"),
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver.ExpectedCalls = nil // reset calls for isolation
			mockDriver.
				On("Delete", ctx, tt.key).
				Return(tt.mockReturn).
				Once()

			err := manager.Delete(ctx, tt.key)

			if tt.expectErr {
				assert.Error(t, err, "expected error when delete fails")
				assert.EqualError(t, err, tt.mockReturn.Error(), "expected correct error message")
			} else {
				assert.NoError(t, err, "expected no error when delete succeeds")
			}

			mockDriver.AssertExpectations(t)
		})
	}
}

func TestMediaManager_DeleteMany(t *testing.T) {
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}

	tests := []struct {
		name       string
		keys       []string
		mockReturn error
		expectErr  bool
	}{
		{
			name:       "should delete many keys successfully",
			keys:       keys,
			mockReturn: nil,
			expectErr:  false,
		},
		{
			name:       "should return error when delete many fails",
			keys:       keys,
			mockReturn: errors.New("delete many failed"),
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver := new(MockStorageDriver)

			// Setup expectations only if keys exist
			for _, k := range tt.keys {
				if tt.expectErr && k == tt.keys[len(tt.keys)-1] {
					mockDriver.On("Delete", mock.Anything, k).Return(tt.mockReturn).Once()
				} else {
					mockDriver.On("Delete", mock.Anything, k).Return(nil).Once()
				}
			}

			manager := &mediaManagerImpl{
				defaultStorage: mockDriver, // direct use, no adapter
			}

			err := manager.DeleteMany(ctx, tt.keys...)

			if tt.expectErr {
				assert.Error(t, err, "expected error when delete fails")
				assert.Contains(t, err.Error(), tt.mockReturn.Error(), "expected correct error message")
			} else {
				assert.NoError(t, err, "expected no error when delete many succeeds")
			}

			mockDriver.AssertExpectations(t)
		})
	}
}

func TestMediaManager_Exists(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	mockDriver := new(MockStorageDriver)

	manager := &mediaManagerImpl{
		storageMap:     map[string]StorageDriver{"default": mockDriver},
		defaultStorage: mockDriver,
	}

	tests := []struct {
		name          string
		key           string
		mockReturnVal bool
		mockReturnErr error
		expectResult  bool
		expectErr     bool
	}{
		{
			name:          "should return true if key exists",
			key:           key,
			mockReturnVal: true,
			mockReturnErr: nil,
			expectResult:  true,
			expectErr:     false,
		},
		{
			name:          "should return false if key does not exist",
			key:           key,
			mockReturnVal: false,
			mockReturnErr: nil,
			expectResult:  false,
			expectErr:     false,
		},
		{
			name:          "should return error if store returns an error",
			key:           key,
			mockReturnVal: false,
			mockReturnErr: errors.New("some other error"),
			expectResult:  false,
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver.ExpectedCalls = nil // reset calls for isolation
			mockDriver.
				On("Exists", ctx, tt.key).
				Return(tt.mockReturnVal, tt.mockReturnErr).
				Once()

			result, err := manager.Exists(ctx, tt.key)

			if tt.expectErr {
				assert.Error(t, err, "expected error")
				assert.EqualError(t, err, tt.mockReturnErr.Error(), "expected correct error message")
				assert.Equal(t, tt.expectResult, result, "expected correct result on error")
			} else {
				assert.NoError(t, err, "expected no error")
				assert.Equal(t, tt.expectResult, result, "expected correct result")
			}

			mockDriver.AssertExpectations(t)
		})
	}
}

func TestMediaManager_GetSignedURL(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	expiry := 5 * time.Minute
	expectedURL := "https://signed.example.com/test-key"
	mockDriver := new(MockStorageDriver)

	manager := &mediaManagerImpl{
		storageMap:     map[string]StorageDriver{"default": mockDriver},
		defaultStorage: mockDriver,
	}

	tests := []struct {
		name          string
		key           string
		expiry        time.Duration
		mockReturnVal string
		mockReturnErr error
		expectURL     string
		expectErr     bool
	}{
		{
			name:          "should get signed URL successfully",
			key:           key,
			expiry:        expiry,
			mockReturnVal: expectedURL,
			mockReturnErr: nil,
			expectURL:     expectedURL,
			expectErr:     false,
		},
		{
			name:          "should return error when get signed URL fails",
			key:           key,
			expiry:        expiry,
			mockReturnVal: "",
			mockReturnErr: errors.New("signed URL failed"),
			expectURL:     "",
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver.ExpectedCalls = nil // reset calls for isolation
			mockDriver.
				On("GetSignedURL", ctx, tt.key, tt.expiry).
				Return(tt.mockReturnVal, tt.mockReturnErr).
				Once()

			url, err := manager.GetSignedURL(ctx, tt.key, tt.expiry)

			if tt.expectErr {
				assert.Error(t, err, "expected error when get signed URL fails")
				assert.EqualError(t, err, tt.mockReturnErr.Error(), "expected correct error message")
				assert.Equal(t, tt.expectURL, url, "expected empty URL on error")
			} else {
				assert.NoError(t, err, "expected no error when get signed URL succeeds")
				assert.Equal(t, tt.expectURL, url, "expected correct URL")
			}

			mockDriver.AssertExpectations(t)
		})
	}
}

func TestMediaManager_GetSignedURLs(t *testing.T) {
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}
	expiry := 5 * time.Minute
	expectedURLs := []string{
		"https://signed.example.com/key1",
		"https://signed.example.com/key2",
		"https://signed.example.com/key3",
	}

	tests := []struct {
		name           string
		keys           []string
		expiry         time.Duration
		mockReturnURLs []string
		mockReturnErr  error
		expectURLs     []string
		expectErr      bool
	}{
		{
			name:           "should get signed URLs successfully for all keys",
			keys:           keys,
			expiry:         expiry,
			mockReturnURLs: expectedURLs,
			mockReturnErr:  nil,
			expectURLs:     expectedURLs,
			expectErr:      false,
		},
		{
			name:           "should handle empty keys list",
			keys:           []string{},
			expiry:         expiry,
			mockReturnURLs: []string{},
			mockReturnErr:  nil,
			expectURLs:     []string{},
			expectErr:      false,
		},
		{
			name:           "should return error if any GetSignedURL fails",
			keys:           keys,
			expiry:         expiry,
			mockReturnURLs: []string{"https://signed.example.com/key1"}, // Only one success
			mockReturnErr:  errors.New("signed URL failed for key2"),
			expectURLs:     nil,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver := new(MockStorageDriver)

			// Mock GetSignedURL for each key in order
			for i, k := range tt.keys {
				if i < len(tt.mockReturnURLs) {
					mockDriver.On("GetSignedURL", mock.Anything, k, tt.expiry).Return(tt.mockReturnURLs[i], nil).Once()
				} else if tt.mockReturnErr != nil {
					mockDriver.On("GetSignedURL", mock.Anything, k, tt.expiry).Return("", tt.mockReturnErr).Once()
				}
			}

			manager := &mediaManagerImpl{
				defaultStorage: mockDriver,
			}

			urls, err := manager.GetSignedURLs(ctx, tt.keys, tt.expiry)

			if tt.expectErr {
				assert.Error(t, err, "expected error when getting signed URLs")
				if tt.mockReturnErr != nil {
					assert.Contains(t, err.Error(), tt.mockReturnErr.Error(), "expected correct error message")
				}
			} else {
				assert.NoError(t, err, "expected no error when getting signed URLs succeeds")
			}

			assert.Equal(t, tt.expectURLs, urls, "expected URLs to match in order")
			mockDriver.AssertExpectations(t)
		})
	}
}

func TestMediaManager_GetURL(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	expectedURL := "http://example.com/test-key"
	mockDriver := new(MockStorageDriver)

	manager := &mediaManagerImpl{
		storageMap:     map[string]StorageDriver{"default": mockDriver},
		defaultStorage: mockDriver,
	}

	tests := []struct {
		name          string
		key           string
		mockReturnVal string
		mockReturnErr error
		expectURL     string
		expectErr     bool
	}{
		{
			name:          "should get URL successfully",
			key:           key,
			mockReturnVal: expectedURL,
			mockReturnErr: nil,
			expectURL:     expectedURL,
			expectErr:     false,
		},
		{
			name:          "should return error when get URL fails",
			key:           key,
			mockReturnVal: "",
			mockReturnErr: errors.New("get URL failed"),
			expectURL:     "",
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver.ExpectedCalls = nil // reset calls for isolation
			mockDriver.
				On("GetURL", ctx, tt.key).
				Return(tt.mockReturnVal, tt.mockReturnErr).
				Once()

			url, err := manager.GetURL(ctx, tt.key)

			if tt.expectErr {
				assert.Error(t, err, "expected error when get URL fails")
				assert.EqualError(t, err, tt.mockReturnErr.Error(), "expected correct error message")
				assert.Equal(t, tt.expectURL, url, "expected empty URL on error")
			} else {
				assert.NoError(t, err, "expected no error when get URL succeeds")
				assert.Equal(t, tt.expectURL, url, "expected correct URL")
			}

			mockDriver.AssertExpectations(t)
		})
	}
}

func TestMediaManager_GetURLs(t *testing.T) {
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}
	expectedURLs := []string{
		"http://example.com/key1",
		"http://example.com/key2",
		"http://example.com/key3",
	}

	tests := []struct {
		name           string
		keys           []string
		mockReturnURLs []string
		mockReturnErr  error
		expectURLs     []string
		expectErr      bool
	}{
		{
			name:           "should get URLs successfully for all keys",
			keys:           keys,
			mockReturnURLs: expectedURLs,
			mockReturnErr:  nil,
			expectURLs:     expectedURLs,
			expectErr:      false,
		},
		{
			name:           "should handle empty keys list",
			keys:           []string{},
			mockReturnURLs: []string{},
			mockReturnErr:  nil,
			expectURLs:     []string{},
			expectErr:      false,
		},
		{
			name:           "should return error if any GetURL fails",
			keys:           keys,
			mockReturnURLs: []string{"http://example.com/key1"},
			mockReturnErr:  errors.New("get URL failed for key2"),
			expectURLs:     nil,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver := new(MockStorageDriver)

			// Mock GetURL for each key in order
			for i, k := range tt.keys {
				if i < len(tt.mockReturnURLs) {
					mockDriver.On("GetURL", mock.Anything, k).Return(tt.mockReturnURLs[i], nil).Once()
				} else if tt.mockReturnErr != nil {
					mockDriver.On("GetURL", mock.Anything, k).Return("", tt.mockReturnErr).Once()
				}
			}

			manager := &mediaManagerImpl{
				defaultStorage: mockDriver,
			}

			urls, err := manager.GetURLs(ctx, tt.keys)

			if tt.expectErr {
				assert.Error(t, err, "expected error when getting URLs")
				if tt.mockReturnErr != nil {
					assert.Contains(t, err.Error(), tt.mockReturnErr.Error(), "expected correct error message")
				}
			} else {
				assert.NoError(t, err, "expected no error when getting URLs succeeds")
			}

			assert.Equal(t, tt.expectURLs, urls, "expected URLs to match in order")
			mockDriver.AssertExpectations(t)
		})
	}
}

func TestMediaManager_Missing(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	mockDriver := new(MockStorageDriver)

	manager := &mediaManagerImpl{
		storageMap:     map[string]StorageDriver{"default": mockDriver},
		defaultStorage: mockDriver,
	}

	tests := []struct {
		name          string
		key           string
		mockReturnVal bool
		mockReturnErr error
		expectResult  bool
		expectErr     bool
	}{
		{
			name:          "should return false if key does not exist",
			key:           key,
			mockReturnVal: false,
			mockReturnErr: nil,
			expectResult:  true, // Missing means it does not exist
			expectErr:     false,
		},
		{
			name:          "should return true if key exists",
			key:           key,
			mockReturnVal: true,
			mockReturnErr: nil,
			expectResult:  false, // Missing means it does not exist, so if it exists, it's not missing
			expectErr:     false,
		},
		{
			name:          "should return error if store returns an error",
			key:           key,
			mockReturnVal: false,
			mockReturnErr: errors.New("some other error"),
			expectResult:  false, // Default value on error
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver.ExpectedCalls = nil // reset calls for isolation
			mockDriver.
				On("Exists", ctx, tt.key).
				Return(tt.mockReturnVal, tt.mockReturnErr).
				Once()

			result, err := manager.Missing(ctx, tt.key)

			if tt.expectErr {
				assert.Error(t, err, "expected error")
				assert.EqualError(t, err, tt.mockReturnErr.Error(), "expected correct error message")
				assert.Equal(t, tt.expectResult, result, "expected correct result on error")
			} else {
				assert.NoError(t, err, "expected no error")
				assert.Equal(t, tt.expectResult, result, "expected correct result")
			}

			mockDriver.AssertExpectations(t)
		})
	}
}

func TestMediaManager_Put(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	content := "upload content"
	reader := strings.NewReader(content)
	mockDriver := new(MockStorageDriver)

	manager := &mediaManagerImpl{
		defaultStorage: mockDriver,
	}

	tests := []struct {
		name      string
		key       string
		content   io.Reader
		mockURL   string
		mockErr   error
		expectErr bool
	}{
		{
			name:      "should put content successfully",
			key:       key,
			content:   reader,
			mockURL:   "http://example.com/test-key",
			mockErr:   nil,
			expectErr: false,
		},
		{
			name:      "should return error when put fails",
			key:       key,
			content:   reader,
			mockURL:   "",
			mockErr:   errors.New("put failed"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDriver.ExpectedCalls = nil // reset calls
			mockDriver.
				On("Put", mock.Anything, tt.key, mock.Anything).
				Return(tt.mockURL, tt.mockErr).
				Once()

			url, err := manager.Put(ctx, tt.key, tt.content)

			if tt.expectErr {
				assert.Error(t, err, "expected error when put fails")
				assert.EqualError(t, err, tt.mockErr.Error(), "expected correct error message")
				assert.Empty(t, url, "expected empty URL on error")
			} else {
				assert.NoError(t, err, "expected no error when put succeeds")
				assert.Equal(t, tt.mockURL, url, "expected correct URL to be returned")
			}

			mockDriver.AssertExpectations(t)
		})
	}
}
