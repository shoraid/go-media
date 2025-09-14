package gomedia

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestManager_NewManager(t *testing.T) {
	tests := []struct {
		name                string
		defaultStorageAlias string
		setupStorage        func() map[string]MediaManager
		expectedErr         error
	}{
		{
			name:                "should return manager when default storage exists",
			defaultStorageAlias: "default",
			setupStorage: func() map[string]MediaManager {
				mockMgr := new(MockMediaManager)
				return map[string]MediaManager{"default": mockMgr}
			},
			expectedErr: nil,
		},
		{
			name:                "should return error when default storage does not exist",
			defaultStorageAlias: "nonexistent",
			setupStorage: func() map[string]MediaManager {
				mockMgr := new(MockMediaManager)
				return map[string]MediaManager{"default": mockMgr}
			},
			expectedErr: ErrInvalidDefaultStorage,
		},
		{
			name:                "should return error when storage map is empty",
			defaultStorageAlias: "default",
			setupStorage: func() map[string]MediaManager {
				return map[string]MediaManager{}
			},
			expectedErr: ErrInvalidDefaultStorage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := tt.setupStorage()
			manager, err := NewManager(tt.defaultStorageAlias, storage)

			if tt.expectedErr != nil {
				require.Error(t, err, "expected error to be returned")
				assert.ErrorIs(t, err, tt.expectedErr, "expected error to match")
				assert.Nil(t, manager, "expected manager to be nil")
			} else {
				require.NoError(t, err, "expected no error")
				assert.NotNil(t, manager, "expected manager to be not nil")
			}

			// assert expectations if mock exists
			if mockMgr, ok := storage["default"].(*MockMediaManager); ok {
				mockMgr.AssertExpectations(t)
			}
		})
	}
}

func TestManager_Storage(t *testing.T) {
	mockDefault := new(MockMediaManager)
	mockOther := new(MockMediaManager)

	storage := map[string]MediaManager{
		"default": mockDefault,
		"other":   mockOther,
	}

	manager, err := NewManager("default", storage)
	require.NoError(t, err, "expected no error")

	tests := []struct {
		name            string
		alias           string
		expectedStorage MediaManager
	}{
		{
			name:            "should return manager with other as defaultStorage",
			alias:           "other",
			expectedStorage: mockOther,
		},
		{
			name:            "should return manager with nil defaultStorage when alias does not exist",
			alias:           "nonexistent",
			expectedStorage: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newMgr := manager.Storage(tt.alias)

			// type assert back to *mediaManagerImpl so we can inspect defaultStorage
			impl, ok := newMgr.(*mediaManagerImpl) // you'd need to export this struct
			require.True(t, ok, "expected returned manager to be *mediaManagerImpl")

			assert.Equal(t, tt.expectedStorage, impl.defaultStorage, "expected defaultStorage to match alias")
		})
	}
}

func TestManager_Delete(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockMediaManager)
		expectedError error
	}{
		{
			name: "should delete successfully when defaultStorage returns no error",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Delete", mock.Anything, "file1").
					Return(nil).
					Once()
			},
			expectedError: nil,
		},
		{
			name: "should return error when defaultStorage returns error",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Delete", mock.Anything, "file1").
					Return(errors.New("delete failed")).
					Once()
			},
			expectedError: errors.New("delete failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			// Build a manager with mock as default storage
			manager := &mediaManagerImpl{ // must be exported or test in package gomedia
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			err := manager.Delete(context.Background(), "file1")

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}

			mockMgr.AssertExpectations(t)
		})
	}
}

func TestManager_DeleteMany(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		setupMock     func(*MockMediaManager)
		expectedError error
	}{
		{
			name: "should return nil when no keys provided",
			keys: []string{},
			setupMock: func(mockMgr *MockMediaManager) {
				// no expectations because no keys are passed
			},
			expectedError: nil,
		},
		{
			name: "should delete all keys successfully when no errors occur",
			keys: []string{"file1", "file2"},
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Delete", mock.Anything, "file1").
					Return(nil).
					Once()

				mockMgr.
					On("Delete", mock.Anything, "file2").
					Return(nil).
					Once()
			},
			expectedError: nil,
		},
		{
			name: "should return error when one delete fails",
			keys: []string{"file1", "file2"},
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Delete", mock.Anything, "file1").
					Return(errors.New("delete failed")).
					Once()

				mockMgr.
					On("Delete", mock.Anything, "file2").
					Return(nil).
					Once()
			},
			expectedError: errors.New("delete failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			// Build manager with mock as default storage
			manager := &mediaManagerImpl{ // must be exported OR test in package gomedia
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			err := manager.DeleteMany(context.Background(), tt.keys...)

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}

			mockMgr.AssertExpectations(t)
		})
	}
}

func TestManager_Exists(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockMediaManager)
		expected      bool
		expectedError error
	}{
		{
			name: "should return true when defaultStorage returns true",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Exists", mock.Anything, "file1").
					Return(true, nil).
					Once()
			},
			expected:      true,
			expectedError: nil,
		},
		{
			name: "should return false when defaultStorage returns false",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Exists", mock.Anything, "file1").
					Return(false, nil).
					Once()
			},
			expected:      false,
			expectedError: nil,
		},
		{
			name: "should return error when defaultStorage returns error",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Exists", mock.Anything, "file1").
					Return(false, errors.New("exists failed")).
					Once()
			},
			expected:      false,
			expectedError: errors.New("exists failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			manager := &mediaManagerImpl{
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			exists, err := manager.Exists(context.Background(), "file1")

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}
			assert.Equal(t, tt.expected, exists, "expected existence to match")

			mockMgr.AssertExpectations(t)
		})
	}
}

func TestManager_GetSignedURL(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockMediaManager)
		expectedURL   string
		expectedError error
	}{
		{
			name: "should return signed URL when defaultStorage returns one",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("GetSignedURL", mock.Anything, "file1", mock.AnythingOfType("time.Duration")).
					Return("http://signed.url/file1", nil).
					Once()
			},
			expectedURL:   "http://signed.url/file1",
			expectedError: nil,
		},
		{
			name: "should return error when defaultStorage returns error",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("GetSignedURL", mock.Anything, "file1", mock.AnythingOfType("time.Duration")).
					Return("", errors.New("sign failed")).
					Once()
			},
			expectedURL:   "",
			expectedError: errors.New("sign failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			manager := &mediaManagerImpl{
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			url, err := manager.GetSignedURL(context.Background(), "file1", 5*time.Minute)

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}
			assert.Equal(t, tt.expectedURL, url, "expected URL to match")

			mockMgr.AssertExpectations(t)
		})
	}
}

func TestManager_GetSignedURLs(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		setupMock     func(*MockMediaManager)
		expectedURLs  []string
		expectedError error
	}{
		{
			name: "should return empty slice when no keys provided",
			keys: []string{},
			setupMock: func(mockMgr *MockMediaManager) {
				// no expectations because no keys are passed
			},
			expectedURLs:  []string{},
			expectedError: nil,
		},
		{
			name: "should return signed URLs for all keys successfully",
			keys: []string{"file1", "file2"},
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("GetSignedURL", mock.Anything, "file1", mock.AnythingOfType("time.Duration")).
					Return("http://signed.url/file1", nil).
					Once()

				mockMgr.
					On("GetSignedURL", mock.Anything, "file2", mock.AnythingOfType("time.Duration")).
					Return("http://signed.url/file2", nil).
					Once()
			},
			expectedURLs:  []string{"http://signed.url/file1", "http://signed.url/file2"},
			expectedError: nil,
		},
		{
			name: "should return error when one GetSignedURL fails",
			keys: []string{"file1", "file2"},
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("GetSignedURL", mock.Anything, "file1", mock.AnythingOfType("time.Duration")).
					Return("", errors.New("sign failed")).
					Once()

				mockMgr.
					On("GetSignedURL", mock.Anything, "file2", mock.AnythingOfType("time.Duration")).
					Return("http://signed.url/file2", nil).
					Once()
			},
			expectedURLs:  nil,
			expectedError: errors.New("sign failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			manager := &mediaManagerImpl{
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			urls, err := manager.GetSignedURLs(context.Background(), tt.keys, 5*time.Minute)

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}
			assert.Equal(t, tt.expectedURLs, urls, "expected URLs to match")

			mockMgr.AssertExpectations(t)
		})
	}
}

func TestManager_GetURL(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockMediaManager)
		expectedURL   string
		expectedError error
	}{
		{
			name: "should return URL when defaultStorage returns one",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("GetURL", mock.Anything, "file1").
					Return("http://public.url/file1", nil).
					Once()
			},
			expectedURL:   "http://public.url/file1",
			expectedError: nil,
		},
		{
			name: "should return error when defaultStorage returns error",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("GetURL", mock.Anything, "file1").
					Return("", errors.New("get URL failed")).
					Once()
			},
			expectedURL:   "",
			expectedError: errors.New("get URL failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			manager := &mediaManagerImpl{
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			url, err := manager.GetURL(context.Background(), "file1")

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}
			assert.Equal(t, tt.expectedURL, url, "expected URL to match")

			mockMgr.AssertExpectations(t)
		})
	}
}

func TestManager_GetURLs(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		setupMock     func(*MockMediaManager)
		expectedURLs  []string
		expectedError error
	}{
		{
			name: "should return empty slice when no keys provided",
			keys: []string{},
			setupMock: func(mockMgr *MockMediaManager) {
				// no expectations because no keys are passed
			},
			expectedURLs:  []string{},
			expectedError: nil,
		},
		{
			name: "should return URLs for all keys successfully",
			keys: []string{"file1", "file2"},
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("GetURL", mock.Anything, "file1").
					Return("http://public.url/file1", nil).
					Once()

				mockMgr.
					On("GetURL", mock.Anything, "file2").
					Return("http://public.url/file2", nil).
					Once()
			},
			expectedURLs:  []string{"http://public.url/file1", "http://public.url/file2"},
			expectedError: nil,
		},
		{
			name: "should return error when one GetURL fails",
			keys: []string{"file1", "file2"},
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("GetURL", mock.Anything, "file1").
					Return("", errors.New("get URL failed")).
					Once()

				mockMgr.
					On("GetURL", mock.Anything, "file2").
					Return("http://public.url/file2", nil).
					Once()
			},
			expectedURLs:  nil,
			expectedError: errors.New("get URL failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			manager := &mediaManagerImpl{
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			urls, err := manager.GetURLs(context.Background(), tt.keys)

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}
			assert.Equal(t, tt.expectedURLs, urls, "expected URLs to match")

			mockMgr.AssertExpectations(t)
		})
	}
}

func TestManager_Missing(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockMediaManager)
		expected      bool
		expectedError error
	}{
		{
			name: "should return true when file is missing",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Exists", mock.Anything, "file1").
					Return(false, nil).
					Once()
			},
			expected:      true,
			expectedError: nil,
		},
		{
			name: "should return false when file exists",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Exists", mock.Anything, "file1").
					Return(true, nil).
					Once()
			},
			expected:      false,
			expectedError: nil,
		},
		{
			name: "should return error when Exists returns error",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Exists", mock.Anything, "file1").
					Return(false, errors.New("exists failed")).
					Once()
			},
			expected:      false,
			expectedError: errors.New("exists failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			manager := &mediaManagerImpl{
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			missing, err := manager.Missing(context.Background(), "file1")

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}
			assert.Equal(t, tt.expected, missing, "expected missing status to match")

			mockMgr.AssertExpectations(t)
		})
	}
}

func TestManager_Put(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockMediaManager)
		expectedURL   string
		expectedError error
	}{
		{
			name: "should put file successfully and return URL",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Put", mock.Anything, mock.Anything, "file1").
					Return("http://put.url/file1", nil).
					Once()
			},
			expectedURL:   "http://put.url/file1",
			expectedError: nil,
		},
		{
			name: "should return error when defaultStorage returns error",
			setupMock: func(mockMgr *MockMediaManager) {
				mockMgr.
					On("Put", mock.Anything, mock.Anything, "file1").
					Return("", errors.New("put failed")).
					Once()
			},
			expectedURL:   "",
			expectedError: errors.New("put failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := new(MockMediaManager)
			tt.setupMock(mockMgr)

			// Build manager with mock as default storage
			manager := &mediaManagerImpl{
				storageMap:     map[string]MediaManager{"default": mockMgr},
				defaultStorage: mockMgr,
			}

			url, err := manager.Put(context.Background(), strings.NewReader("dummy"), "file1")

			if tt.expectedError != nil {
				require.Error(t, err, "expected error to be returned")
				assert.EqualError(t, err, tt.expectedError.Error(), "expected error to match")
			} else {
				require.NoError(t, err, "expected no error")
			}
			assert.Equal(t, tt.expectedURL, url, "expected URL to match")

			mockMgr.AssertExpectations(t)
		})
	}
}
