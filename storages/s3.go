package storages

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aruncs31s/gologger"
	"github.com/aruncs31s/gostorage/config"
	"github.com/aruncs31s/gostorage/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	ErrMissingS3Config      = errors.New("AWS credentials or S3 bucket configurations are missing in environment")
	ErrFailedToUpload       = errors.New("failed to upload file to S3")
	ErrFailedToLoadS3Config = errors.New("failed to load s3 configuration")
	ErrFailedToReadFile     = errors.New("failed to read local file")
)

var CFG *config.S3Config

// UploadToS3 uploads a byte slice to S3 (or a MinIO-compatible store when S3_ENDPOINT is set).
// Returns the public URL of the uploaded file on success.
func UploadToS3(CFG *config.S3Config, ctx context.Context, fileBytes []byte, s3Key string, contentType string) (string, error) {
	var cfg config.S3Config
	if CFG == nil {
		cfg = config.GetFromEnv()
	} else {
		cfg = *CFG
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		gologger.Error("AWS credentials or S3 bucket configurations are missing in environment")
		return "", ErrMissingS3Config
	}

	// Build credentials and base config
	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(creds),
	}
	if cfg.Endpoint != "" {
		options = append(options, awsconfig.WithBaseEndpoint(cfg.Endpoint))
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		gologger.Error(ErrFailedToLoadS3Config.Error())
		return "", fmt.Errorf("%w: %w", ErrFailedToLoadS3Config, err)
	}

	// UsePathStyle is required for MinIO and most S3-compatible stores
	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = cfg.Endpoint != ""
	})
	gologger.Info("Uploading file to S3")

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailedToUpload, err)
	}

	// Return appropriate public URL
	if cfg.Endpoint != "" {
		// MinIO / S3-compatible: <endpoint>/<bucket>/<key>
		return fmt.Sprintf("%s/%s/%s", cfg.Endpoint, cfg.Bucket, s3Key), nil
	}
	// Real AWS S3
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.Bucket, cfg.Region, s3Key), nil
}

// GetS3URL returns the public URL of a given key in the configured S3 or MinIO bucket.
func GetS3URL(CFG *config.S3Config, s3Key string) string {
	var cfg config.S3Config
	if CFG == nil {
		cfg = config.GetFromEnv()
	} else {
		cfg = *CFG
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		gologger.Error("AWS credentials or S3 bucket configurations are missing in environment")
		return ""
	}
	endpoint := utils.GetEnv("S3_ENDPOINT", "")
	bucket := utils.GetEnv("AWS_S3_BUCKET", "")
	if endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", endpoint, bucket, s3Key)
	}
	region := utils.GetEnv("AWS_REGION", "us-east-1")
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, s3Key)
}

func UploadFileToS3(CFG *config.S3Config, ctx context.Context, localFilePath string, s3Key string, contentType string) (string, error) {
	var cfg config.S3Config
	if CFG == nil {
		cfg = config.GetFromEnv()
	} else {
		cfg = *CFG
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		gologger.Error("AWS credentials or S3 bucket configurations are missing in environment")
		return "", ErrMissingS3Config
	}

	fileBytes, err := os.ReadFile(localFilePath)
	if err != nil {
		gologger.Error(ErrFailedToReadFile.Error())
		return "", fmt.Errorf("%w: %w", ErrFailedToReadFile, err)
	}

	return UploadToS3(&cfg, ctx, fileBytes, s3Key, contentType)
}
