package cloudinary

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/tamirat-dejene/veritas/shared/pkg/storage"
)

type cloudinaryStorage struct {
	client *cloudinary.Cloudinary
	folder string
}

// NewCloudinaryStorage creates a new Cloudinary-backed storage implementation.
func NewCloudinaryStorage(cloudName, apiKey, apiSecret, folder string) (storage.FileStorage, error) {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	return &cloudinaryStorage{
		client: cld,
		folder: folder,
	}, nil
}

func (s *cloudinaryStorage) Upload(ctx context.Context, fileName string, content io.Reader) (string, error) {
	b := true
	resp, err := s.client.Upload.Upload(ctx, content, uploader.UploadParams{
		Folder:         s.folder,
		PublicID:       fileName,
		ResourceType:   "auto",
		UniqueFilename: &b,
		Overwrite:      &b,
		Invalidate:     &b,
	})

	if err != nil {
		return "", fmt.Errorf("cloudinary upload failed: %w", err)
	}
	
	return resp.SecureURL, nil
}

func (s *cloudinaryStorage) Delete(ctx context.Context, publicID string) error {
	_, err := s.client.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	if err != nil {
		return fmt.Errorf("cloudinary delete failed: %w", err)
	}

	return nil
}
