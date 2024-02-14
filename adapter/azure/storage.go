package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type StorageRepository struct {
	client *azblob.Client
}

func NewStorageRepository(client *azblob.Client) *StorageRepository {
	return &StorageRepository{client: client}
}

func (s *StorageRepository) UploadFile(ctx context.Context, containerName string, filename string, body []byte) error {
	_, err := s.client.UploadBuffer(ctx, containerName, filename, body, nil)
	if err != nil {
		return err
	}
	return nil
}
func (s *StorageRepository) DownloadFile(ctx context.Context, containerName string, filename string, buffer []byte) error {
	_, err := s.client.DownloadBuffer(ctx, containerName, filename, buffer, nil)
	if err != nil {
		return err
	}
	return nil
}
func (s *StorageRepository) DeleteFile(ctx context.Context, containerName string, filename string) error {
	_, err := s.client.DeleteBlob(ctx, containerName, filename, nil)
	if err != nil {
		return err
	}
	return nil
}
