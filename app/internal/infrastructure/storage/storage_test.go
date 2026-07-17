package storage

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"drawo/config"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"
)

type mockMinio struct {
	mock.Mock
}

func (m *mockMinio) BucketExists(ctx context.Context, n string) (bool, error) {
	args := m.Called(ctx, n)
	return args.Bool(0), args.Error(1)
}
func (m *mockMinio) MakeBucket(ctx context.Context, n string, o minio.MakeBucketOptions) error {
	return m.Called(ctx, n, o).Error(0)
}
func (m *mockMinio) PutObject(ctx context.Context, b, n string, r io.Reader, s int64, o minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, b, n, r, s, o)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}
func (m *mockMinio) RemoveObject(ctx context.Context, b, n string, o minio.RemoveObjectOptions) error {
	return m.Called(ctx, b, n, o).Error(0)
}
func (m *mockMinio) PresignedGetObject(ctx context.Context, b, n string, e time.Duration, r url.Values) (*url.URL, error) {
	args := m.Called(ctx, b, n, e, r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*url.URL), args.Error(1)
}

func TestMinioProvider(t *testing.T) {
	m := new(mockMinio)
	p := &MinioProvider{client: m, cfg: config.StorageConfig{BucketName: "b"}}
	ctx := context.Background()
	m.On("BucketExists", mock.Anything, "b").Return(true, nil)
	m.On("PutObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(minio.UploadInfo{Key: "o"}, nil)
	p.Upload(ctx, "b", "o", strings.NewReader("data"), 4, "text")
	m.On("RemoveObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	p.Delete(ctx, "b", "o")
	u, _ := url.Parse("http://test.com")
	m.On("PresignedGetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(u, nil)
	p.GetURL(ctx, "b", "o")
}

func TestLocalStorageProvider(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewLocalStorageProvider(config.StorageConfig{UploadDirectory: tmpDir})
	ctx := context.Background()
	p.Upload(ctx, "b", "o", strings.NewReader("data"), 4, "text")
	p.GetURL(ctx, "b", "o")
	p.Delete(ctx, "b", "o")
}
