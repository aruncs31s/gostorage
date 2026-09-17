//go:build integration

package tests

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aruncs31s/gostorage"
	"github.com/aruncs31s/gostorage/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3StorageWithMinIO(t *testing.T) {
	ctx := context.Background()
	endpoint := envOrDefault("S3_ENDPOINT", "http://localhost:9002")
	accessKey := envOrDefault("AWS_ACCESS_KEY_ID", "admin")
	secretKey := envOrDefault("AWS_SECRET_ACCESS_KEY", "supersecretpassword")
	bucket := envOrDefault("AWS_S3_BUCKET", "gostorage-test")

	client := newMinIOClient(t, ctx, endpoint, accessKey, secretKey)
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("create bucket: %v", err)
	}

	storage := gostorage.NewS3Storage().WithConfig(&config.S3Config{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    envOrDefault("AWS_REGION", "us-east-1"),
		Bucket:    bucket,
		Endpoint:  endpoint,
	})
	key := "integration/hello.txt"
	want := []byte("hello from gostorage")

	gotURL, err := storage.UploadBytes(ctx, want, key, "text/plain")
	if err != nil {
		t.Fatalf("upload bytes: %v", err)
	}
	if gotURL != endpoint+"/"+bucket+"/"+key {
		t.Fatalf("unexpected object URL: got %q", gotURL)
	}

	defer client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	object, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("read uploaded object: %v", err)
	}
	defer object.Body.Close()

	got, err := io.ReadAll(object.Body)
	if err != nil {
		t.Fatalf("read object body: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("unexpected object body: got %q, want %q", got, want)
	}
}

func newMinIOClient(t *testing.T, ctx context.Context, endpoint, accessKey, secretKey string) *s3.Client {
	t.Helper()
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(envOrDefault("AWS_REGION", "us-east-1")),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awsconfig.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		t.Fatalf("load MinIO config: %v", err)
	}

	return s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.UsePathStyle = true
	})
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
