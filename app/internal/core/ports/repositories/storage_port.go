// Package repositories defines the data access layers.
package repositories

import (
	"context"
	"io"
)

// FileStorage defines a common interface for storing large files like songs or avatars.
// This allows Drawo to switch between MinIO, AWS S3, or Local Disk storage easily.
type FileStorage interface {
	// Upload saves an object to the storage provider and returns its key/path.
	Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) (string, error)
	// Delete removes an object from the storage provider.
	Delete(ctx context.Context, bucketName, objectName string) error
	// GetURL returns a public or pre-signed URL to access the file.
	GetURL(ctx context.Context, bucketName, objectName string) (string, error)
}
