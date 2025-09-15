package s3driver

import (
	"context"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type mockPresignClient struct {
	url string
	err error
}

func (m *mockPresignClient) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &v4.PresignedHTTPRequest{URL: m.url}, nil
}
