package common

import (
	"context"
)

type DBService interface {
	Create(ctx context.Context, media *Media) error
	Read(ctx context.Context, id string) (*Media, error)
}

type StorageService interface {
	UploadFile(ctx context.Context, containerName string, filename string, body []byte) error
	DownloadFile(ctx context.Context, containerName string, filename string, buffer []byte) error
	DeleteFile(ctx context.Context, containerName string, filename string) error
}

type Media struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	DurationString string  `json:"duration_string"`
	Duration       float64 `json:"duration"`
	BucketPath     string  `json:"bucket_path"`
}
