package s3driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/rs/zerolog/log"
	gomedia "github.com/shoraid/go-media"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Client interface {
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type presignClient interface {
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

type Visibility string

const (
	VisibilityPrivate Visibility = "private" // Files are private, need signed URL to access
	VisibilityPublic  Visibility = "public"  // Files are publicly accessible via direct URL
)

// ObjectStorageConfig defines the configuration needed to connect to an S3-compatible storage.
// You can use this with AWS S3, Cloudflare R2, MinIO, GCS (S3 API), etc.
type ObjectStorageConfig struct {
	Bucket        string        // bucket name where files will be stored
	Region        string        // AWS region or equivalent
	AccessKey     string        // access key for authentication
	SecretKey     string        // secret key for authentication
	Endpoint      string        // optional custom endpoint (for R2, MinIO, etc.)
	UseSSL        bool          // true = https, false = http
	Visibility    Visibility    // public or private
	DefaultExpiry time.Duration // default expiry duration for signed URLs
}

// ObjectStorage is the concrete implementation of gomedia.StorageDriver for S3-compatible storages.
type ObjectStorage struct {
	client        s3Client
	bucket        string
	config        ObjectStorageConfig
	presignClient presignClient // used to generate signed URLs
}

// NewObjectStorage initializes and returns an ObjectStorage instance using the given config.
// It loads AWS configuration, sets up the S3 client, and prepares a presign client.
// Returns gomedia.ErrInvalidConfig if credentials or config are invalid.
func NewObjectStorage(cfg ObjectStorageConfig) (gomedia.StorageDriver, error) {
	if cfg.AccessKey == "" {
		return nil, gomedia.ErrInvalidConfig
	}

	if cfg.SecretKey == "" {
		return nil, gomedia.ErrInvalidConfig
	}

	storageCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to load config")
		return nil, gomedia.ErrInvalidConfig
	}

	client := s3.NewFromConfig(storageCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // needed for MinIO / R2
		}
	})

	defaultExpiry := cfg.DefaultExpiry
	if defaultExpiry == 0 {
		defaultExpiry = 15 * time.Minute
	}

	return &ObjectStorage{
		client:        client,
		bucket:        cfg.Bucket,
		config:        cfg,
		presignClient: s3.NewPresignClient(client),
	}, nil
}

// Delete permanently removes a file from the bucket.
// Usage: Call when you want to delete a file by its key.
func (s *ObjectStorage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to delete file from S3")
		return gomedia.ErrInternal
	}

	return nil
}

// Exists checks if a file exists in the bucket.
// Usage: Call before uploading or deleting to verify the file's presence.
func (s *ObjectStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiError interface{ ErrorCode() string }
		if errors.As(err, &apiError) && apiError.ErrorCode() == "NotFound" {
			return false, nil
		}

		log.Error().Err(err).Str("key", key).Msg("failed to check if file exists in S3")
		return false, gomedia.ErrInternal
	}

	return true, nil
}

// GetSignedURL generates a temporary signed URL for downloading a file from a private bucket.
// Usage: Call this when you need to share temporary access to a private file.
func (s *ObjectStorage) GetSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if s.config.Visibility != VisibilityPrivate {
		return "", nil
	}

	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to generate signed URL")
		return "", gomedia.ErrInternal
	}

	return req.URL, nil
}

// GetURL returns the direct public URL for a file if the bucket is public.
// Usage: Call this when you want to embed or link a public file directly.
func (s *ObjectStorage) GetURL(ctx context.Context, key string) (string, error) {
	if s.config.Visibility != VisibilityPublic {
		return "", nil
	}

	scheme := "https"
	if !s.config.UseSSL {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.config.Endpoint, s.bucket, key), nil
}

// validateKey ensures that the provided key is valid (not empty, no invalid characters).
// Usage: Called internally by Put to prevent uploading bad file names.
var fileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func validateKey(name string) error {
	switch {
	case len(name) == 0:
		return errors.New("key cannot be empty")
	case !fileNameRegex.MatchString(name):
		return errors.New("key contains invalid characters")
	case name == "." || name == "..":
		return errors.New("invalid key")
	}
	return nil
}

// Put uploads a file to the bucket and returns its URL.
// If the bucket is public, it returns a direct URL.
// If the bucket is private, it returns a signed URL.
// Usage: Call this to save a new file or overwrite an existing file.
func (s *ObjectStorage) Put(ctx context.Context, key string, file io.Reader) (string, error) {
	if err := validateKey(key); err != nil {
		log.Error().Err(err).Str("key", key).Msg("invalid key")
		return "", gomedia.ErrInvalidKey
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(s.bucket),
		Key:                  aws.String(key),
		Body:                 file,
		ServerSideEncryption: "AES256",
	})
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to upload file to S3")
		return "", gomedia.ErrInternal
	}

	// Public bucket: return direct URL
	if s.config.Visibility == VisibilityPublic {
		scheme := "https"
		if !s.config.UseSSL {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s/%s/%s", scheme, s.config.Endpoint, s.bucket, key), nil
	}

	// Private bucket: return signed URL
	return s.GetSignedURL(ctx, key, s.config.DefaultExpiry)
}
