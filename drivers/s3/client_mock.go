package s3driver

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// mockS3Client simulates s3.Client's PutObject behavior
type mockS3Client struct {
	err error
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.HeadObjectOutput{}, nil
}

func (m *mockS3Client) PutObject(ctx context.Context, in *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &s3.PutObjectOutput{}, nil
}
