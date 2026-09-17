package gostorage

import (
	"context"

	"github.com/aruncs31s/gostorage/config"
	"github.com/aruncs31s/gostorage/storages"
)

type Storage interface {
	UploadFile(
		ctx context.Context,
		filePath string,
		s3Key string,
		contentType string,
	) (string, error)
	UploadBytes(
		ctx context.Context,
		fileBytes []byte,
		s3Key string,
		contentType string,
	) (string, error)
	GetURL(ctx context.Context, s3Key string) (string, error)
}

func NewS3Storage() *S3Storage {
	return &S3Storage{}
}

type S3Storage struct {
	Config *config.S3Config
}

func (s *S3Storage) WithConfig(cfg *config.S3Config) *S3Storage {
	s.Config = cfg
	return s
}
func (s *S3Storage) Get() Storage {
	return s
}

func (s *S3Storage) UploadFile(ctx context.Context, filePath string, s3Key string, contentType string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	return storages.UploadFileToS3(s.Config, ctx, filePath, s3Key, contentType)
}
func (s *S3Storage) UploadBytes(ctx context.Context, fileBytes []byte, s3Key string, contentType string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return storages.UploadToS3(s.Config, ctx, fileBytes, s3Key, contentType)
}
func (s *S3Storage) GetURL(ctx context.Context, s3Key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	url := storages.GetS3URL(s.Config, s3Key)
	return url, nil
}
