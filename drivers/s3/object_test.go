package s3driver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	gostorage "github.com/shoraid/go-storage"
	"github.com/stretchr/testify/assert"
)

func TestNewObjectStorage(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ObjectStorageConfig
		expectedErr error
	}{
		{
			name: "should create new object storage successfully",
			cfg: ObjectStorageConfig{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
			expectedErr: nil,
		},
		{
			name: "should create new object storage successfully with custom endpoint",
			cfg: ObjectStorageConfig{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
				Endpoint:  "http://localhost:9000",
			},
			expectedErr: nil,
		},
		{
			name: "should return error when access key is missing",
			cfg: ObjectStorageConfig{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				SecretKey: "test-secret-key",
			},
			expectedErr: gostorage.ErrInvalidConfig,
		},
		{
			name: "should return error when secret key is missing",
			cfg: ObjectStorageConfig{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				AccessKey: "test-access-key",
			},
			expectedErr: gostorage.ErrInvalidConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewObjectStorage(tt.cfg)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr, "expected error when config is invalid")
				assert.Nil(t, storage, "expected storage to be nil on error")
			} else {
				assert.NoError(t, err, "expected no error when config is valid")
				assert.NotNil(t, storage, "expected storage to be not nil on success")
			}
		})
	}
}

func TestObjectStorage_Delete(t *testing.T) {
	tests := []struct {
		name        string
		mockErr     error
		expectedErr error
	}{
		{
			name:        "should delete file successfully when no error returned",
			mockErr:     nil,
			expectedErr: nil,
		},
		{
			name:        "should return internal error when DeleteObject fails",
			mockErr:     errors.New("delete error"),
			expectedErr: gostorage.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ObjectStorage{
				bucket: "test-bucket",
				client: &mockS3Client{
					err: tt.mockErr,
				},
			}

			err := storage.Delete(context.Background(), "file.txt")

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr, "expected internal error when DeleteObject fails")
			} else {
				assert.NoError(t, err, "expected no error when DeleteObject succeeds")
			}
		})
	}
}

func TestObjectStorage_Exists(t *testing.T) {
	tests := []struct {
		name        string
		mockErr     error
		expected    bool
		expectedErr error
	}{
		{
			name:        "should return true when file exists",
			mockErr:     nil,
			expected:    true,
			expectedErr: nil,
		},
		{
			name:        "should return false when file does not exist",
			mockErr:     &mockNotFoundError{},
			expected:    false,
			expectedErr: nil,
		},
		{
			name:        "should return internal error on unexpected S3 error",
			mockErr:     errors.New("some AWS error"),
			expected:    false,
			expectedErr: gostorage.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ObjectStorage{
				bucket: "test-bucket",
				client: &mockS3Client{
					err: tt.mockErr,
				},
			}

			got, err := storage.Exists(context.Background(), "file.txt")
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr, "expected error")
			} else {
				assert.NoError(t, err, "expected no error")
			}
			assert.Equal(t, tt.expected, got, "expected existence result")
		})
	}
}

func TestObjectStorage_GetSignedURL(t *testing.T) {
	tests := []struct {
		name        string
		visibility  Visibility
		mockURL     string
		mockErr     error
		expected    string
		expectedErr error
	}{
		{
			name:        "should return empty string when bucket is public",
			visibility:  VisibilityPublic,
			expected:    "",
			expectedErr: nil,
		},
		{
			name:        "should return signed URL when bucket is private",
			visibility:  VisibilityPrivate,
			mockURL:     "https://example.com/signed",
			expected:    "https://example.com/signed",
			expectedErr: nil,
		},
		{
			name:        "should return error when presign fails for private bucket",
			visibility:  VisibilityPrivate,
			mockErr:     errors.New("presign failed"),
			expected:    "",
			expectedErr: gostorage.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ObjectStorage{
				bucket: "test-bucket",
				config: ObjectStorageConfig{
					Visibility: tt.visibility,
				},
				presignClient: &mockPresignClient{
					url: tt.mockURL,
					err: tt.mockErr,
				},
			}

			got, err := s.GetSignedURL(context.Background(), "file.txt", 5*time.Minute)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr, "expected error when presign fails or is restricted")
			} else {
				assert.NoError(t, err, "expected no error when presign succeeds or public bucket returns empty string")
			}

			assert.Equal(t, tt.expected, got, "expected signed URL or empty string to match")
		})
	}
}

func TestObjectStorage_GetURL(t *testing.T) {
	tests := []struct {
		name       string
		visibility Visibility
		useSSL     bool
		key        string
		expected   string
	}{
		{
			name:       "should return empty string for private bucket",
			visibility: VisibilityPrivate,
			useSSL:     true,
			key:        "file.txt",
			expected:   "",
		},
		{
			name:       "should return empty string for unknown visibility",
			visibility: Visibility("unknown"),
			useSSL:     true,
			key:        "file.txt",
			expected:   "",
		},
		{
			name:       "should return https URL for public bucket with SSL",
			visibility: VisibilityPublic,
			useSSL:     true,
			key:        "file.txt",
			expected:   "https://endpoint/test-bucket/file.txt",
		},
		{
			name:       "should return http URL for public bucket without SSL",
			visibility: VisibilityPublic,
			useSSL:     false,
			key:        "file.txt",
			expected:   "http://endpoint/test-bucket/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ObjectStorage{
				bucket: "test-bucket",
				config: ObjectStorageConfig{
					Endpoint:   "endpoint",
					UseSSL:     tt.useSSL,
					Visibility: tt.visibility,
				},
			}

			got, err := storage.GetURL(context.Background(), tt.key)
			assert.NoError(t, err, "expected no error")
			assert.Equal(t, tt.expected, got, "expected URL to match")
		})
	}
}

func TestObjectStorage_Put(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		visibility  Visibility
		mockPutErr  error
		mockSignURL string
		mockSignErr error
		expected    string
		expectedErr error
	}{
		{
			name:        "should return error when key is empty",
			key:         "",
			visibility:  VisibilityPrivate,
			expected:    "",
			expectedErr: gostorage.ErrInvalidKey,
		},
		{
			name:        "should return error when key contains invalid characters",
			key:         "bad/key.txt",
			visibility:  VisibilityPrivate,
			expected:    "",
			expectedErr: gostorage.ErrInvalidKey,
		},
		{
			name:        "should return internal error when PutObject fails",
			key:         "file.txt",
			visibility:  VisibilityPrivate,
			mockPutErr:  errors.New("s3 error"),
			expected:    "",
			expectedErr: gostorage.ErrInternal,
		},
		{
			name:        "should return https public URL when bucket is public and SSL enabled",
			key:         "file.txt",
			visibility:  VisibilityPublic,
			expected:    "https://endpoint/test-bucket/file.txt",
			expectedErr: nil,
		},
		{
			name:        "should return http public URL when bucket is public and SSL disabled",
			key:         "file.txt",
			visibility:  VisibilityPublic,
			expected:    "http://endpoint/test-bucket/file.txt",
			expectedErr: nil,
		},
		{
			name:        "should return signed URL when bucket is private and presign succeeds",
			key:         "file.txt",
			visibility:  VisibilityPrivate,
			mockSignURL: "https://signed-url",
			expected:    "https://signed-url",
			expectedErr: nil,
		},
		{
			name:        "should return internal error when presign fails for private bucket",
			key:         "file.txt",
			visibility:  VisibilityPrivate,
			mockSignErr: errors.New("presign error"),
			expected:    "",
			expectedErr: gostorage.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &ObjectStorage{
				bucket: "test-bucket",
				config: ObjectStorageConfig{
					Endpoint:      "endpoint",
					UseSSL:        !(tt.name == "should return http public URL when bucket is public and SSL disabled"),
					Visibility:    tt.visibility,
					DefaultExpiry: 10 * time.Minute,
				},
				client: &mockS3Client{err: tt.mockPutErr},
				presignClient: &mockPresignClient{
					url: tt.mockSignURL,
					err: tt.mockSignErr,
				},
			}

			data := io.NopCloser(bytes.NewBufferString("testdata"))
			got, err := storage.Put(context.Background(), tt.key, data)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr, "expected matching error when Put fails or key is invalid")
			} else {
				assert.NoError(t, err, "expected no error when Put succeeds")
			}
			assert.Equal(t, tt.expected, got, "expected returned URL or empty string to match")
		})
	}
}
