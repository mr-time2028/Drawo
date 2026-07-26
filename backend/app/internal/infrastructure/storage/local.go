// Package storage provides concrete implementations of the FileStorage port.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"drawo/config"
	"drawo/internal/core/ports/repositories"
)

// LocalStorageProvider implements repositories.FileStorage using the local filesystem.
// This is perfect for development or small deployments without a dedicated CDN.
type LocalStorageProvider struct {
	cfg config.StorageConfig
}

// NewLocalStorageProvider initializes a local storage handler.
func NewLocalStorageProvider(cfg config.StorageConfig) *LocalStorageProvider {
	// Ensure the base upload directory exists
	_ = os.MkdirAll(cfg.UploadDirectory, 0755)
	return &LocalStorageProvider{cfg: cfg}
}

// Upload saves a file to the local disk in a folder named after the bucket.
func (p *LocalStorageProvider) Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	// 1. Create the full path: /uploads/songs/music.mp3
	fullPath := filepath.Join(p.cfg.UploadDirectory, bucketName, objectName)
	dir := filepath.Dir(fullPath)

	// 2. Ensure the sub-directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create local storage dir: %w", err)
	}

	// 3. Create the file
	out, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create local file: %w", err)
	}
	defer out.Close()

	// 4. Copy the data
	_, err = io.Copy(out, reader)
	if err != nil {
		return "", fmt.Errorf("write local file: %w", err)
	}

	// Return the relative path as the key
	return filepath.Join(bucketName, objectName), nil
}

// Delete removes the file from the local disk.
func (p *LocalStorageProvider) Delete(ctx context.Context, bucketName, objectName string) error {
	fullPath := filepath.Join(p.cfg.UploadDirectory, bucketName, objectName)
	return os.Remove(fullPath)
}

// GetURL returns a local server path.
// In production, your Nginx would serve this folder at /uploads.
func (p *LocalStorageProvider) GetURL(ctx context.Context, bucketName, objectName string) (string, error) {
	return fmt.Sprintf("/uploads/%s/%s", bucketName, objectName), nil
}

var _ repositories.FileStorage = (*LocalStorageProvider)(nil)
