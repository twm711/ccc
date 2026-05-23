package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
)

// MinIOClient wraps MinIO SDK for recording storage.
type MinIOClient struct {
	client *minio.Client
	bucket string
	logger zerolog.Logger
}

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	Logger    zerolog.Logger
}

func NewMinIOClient(cfg Config) (*MinIOClient, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: init: %w", err)
	}

	return &MinIOClient{client: mc, bucket: cfg.Bucket, logger: cfg.Logger}, nil
}

// EnsureBucket creates the bucket if it doesn't exist.
func (m *MinIOClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("minio: check bucket: %w", err)
	}
	if !exists {
		if err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("minio: create bucket: %w", err)
		}
		m.logger.Info().Str("bucket", m.bucket).Msg("bucket created")
	}
	return nil
}

// Upload stores a file in MinIO.
func (m *MinIOClient) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio: upload %s: %w", objectName, err)
	}
	m.logger.Debug().Str("object", objectName).Msg("uploaded")
	return nil
}

// Download retrieves a file from MinIO.
func (m *MinIOClient) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio: download %s: %w", objectName, err)
	}
	return obj, nil
}

// GetPresignedURL returns a temporary download URL.
func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectName string, expirySec int) (string, error) {
	u, err := m.client.PresignedGetObject(ctx, m.bucket, objectName, 0, nil)
	if err != nil {
		return "", fmt.Errorf("minio: presign %s: %w", objectName, err)
	}
	return u.String(), nil
}
