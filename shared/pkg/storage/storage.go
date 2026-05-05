package storage

import (
	"context"
	"io"
)

// FileStorage defines the interface for file storage operations.
type FileStorage interface {
	// Upload uploads a file and returns its public URL and an optional error.
	Upload(ctx context.Context, fileName string, content io.Reader) (string, error)

	// Delete deletes a file by its public ID or URL.
	Delete(ctx context.Context, fileID string) error
}
