package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aruncs31s/gostorage"
	"github.com/aruncs31s/gostorage/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:embed ui/*
var uiFiles embed.FS

type server struct {
	storage *gostorage.S3Storage
	client  *s3.Client
	bucket  string
}

func main() {
	ctx := context.Background()
	endpoint := envOrDefault("S3_ENDPOINT", "http://localhost:9002")
	accessKey := envOrDefault("AWS_ACCESS_KEY_ID", "admin")
	secretKey := envOrDefault("AWS_SECRET_ACCESS_KEY", "supersecretpassword")
	region := envOrDefault("AWS_REGION", "us-east-1")
	bucket := envOrDefault("AWS_S3_BUCKET", "gostorage-test")

	client := newS3Client(ctx, endpoint, region, accessKey, secretKey)
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") && !strings.Contains(err.Error(), "already exists") {
		log.Fatalf("create bucket: %v", err)
	}

	app := &server{
		storage: gostorage.NewS3Storage().WithConfig(&config.S3Config{
			AccessKey: accessKey,
			SecretKey: secretKey,
			Region:    region,
			Bucket:    bucket,
			Endpoint:  endpoint,
		}),
		client: client,
		bucket: bucket,
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(uiFiles)))
	mux.HandleFunc("/api/upload", app.upload)
	mux.HandleFunc("/api/objects", app.objects)
	mux.HandleFunc("/api/download", app.download)

	address := envOrDefault("WEB_ADDR", ":8080")
	log.Printf("GoStorage UI listening on http://localhost%s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func (s *server) upload(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	file, header, err := request.FormFile("file")
	if err != nil {
		http.Error(response, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(response, "read upload: "+err.Error(), http.StatusBadRequest)
		return
	}
	key := filepath.Base(header.Filename)
	if request.FormValue("key") != "" {
		key = filepath.Base(request.FormValue("key"))
	}
	url, err := s.storage.UploadBytes(request.Context(), content, key, contentType(header, content))
	if err != nil {
		http.Error(response, "upload: "+err.Error(), http.StatusBadGateway)
		return
	}

	writeJSON(response, http.StatusCreated, map[string]string{"key": key, "url": url})
}

func (s *server) objects(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result, err := s.client.ListObjectsV2(request.Context(), &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket)})
	if err != nil {
		http.Error(response, "list objects: "+err.Error(), http.StatusBadGateway)
		return
	}
	objects := make([]map[string]string, 0, len(result.Contents))
	for _, object := range result.Contents {
		if object.Key != nil {
			objects = append(objects, map[string]string{"key": *object.Key})
		}
	}
	writeJSON(response, http.StatusOK, objects)
}

func (s *server) download(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	key := request.URL.Query().Get("key")
	if key == "" {
		http.Error(response, "key is required", http.StatusBadRequest)
		return
	}
	object, err := s.client.GetObject(request.Context(), &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		http.Error(response, "download: "+err.Error(), http.StatusNotFound)
		return
	}
	defer object.Body.Close()
	if object.ContentType != nil {
		response.Header().Set("Content-Type", *object.ContentType)
	}
	response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(key)))
	if _, err := io.Copy(response, object.Body); err != nil {
		log.Printf("download %q: %v", key, err)
	}
}

func newS3Client(ctx context.Context, endpoint, region, accessKey, secretKey string) *s3.Client {
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awsconfig.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		log.Fatalf("load S3 config: %v", err)
	}
	return s3.NewFromConfig(awsConfig, func(options *s3.Options) { options.UsePathStyle = endpoint != "" })
}

func contentType(header *multipart.FileHeader, content []byte) string {
	if header.Header.Get("Content-Type") != "" {
		return header.Header.Get("Content-Type")
	}
	return http.DetectContentType(content)
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
