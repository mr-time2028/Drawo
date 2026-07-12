package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
)

// MinioClient defines the subset of MinIO methods we use.
// This allows us to mock the SDK in unit tests.
type MinioClient interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, options minio.MakeBucketOptions) error
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)
}

// MinioProvider implements repositories.FileStorage using the MinIO SDK.
type MinioProvider struct {
	client MinioClient
	cfg    config.StorageConfig
}

// NewMinioProvider initializes a connection to a MinIO server.
func NewMinioProvider(cfg config.StorageConfig) (*MinioProvider, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio connection: %w", err)
	}

	return &MinioProvider{
		client: client,
		cfg:    cfg,
	}, nil
}

// Upload transfers a file (like an MP3) to a MinIO bucket.
func (p *MinioProvider) Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	exists, err := p.client.BucketExists(ctx, bucketName)
	if err != nil {
		return "", err
	}
	if !exists {
		if err := p.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return "", err
		}
	}

	info, err := p.client.PutObject(ctx, bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload to minio: %w", err)
	}

	return info.Key, nil
}

// Delete removes the file from MinIO.
func (p *MinioProvider) Delete(ctx context.Context, bucketName, objectName string) error {
	return p.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}

// GetURL generates a temporary (pre-signed) URL for the frontend to play the song.
func (p *MinioProvider) GetURL(ctx context.Context, bucketName, objectName string) (string, error) {
	presignedURL, err := p.client.PresignedGetObject(ctx, bucketName, objectName, time.Hour*24, nil)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

var _ repositories.FileStorage = (*MinioProvider)(nil)
