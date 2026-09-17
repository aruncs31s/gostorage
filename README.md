# Go Storage

### Install

```bash
go get github.com/aruncs31s/gostorage
```

### Use S3
```go
func main() {
	storage := gostorage.NewS3Storage().WithConfig(&config.S3Config{
		AccessKey: "admin",
		SecretKey: "supersecretpassword",
		Region:    "us-east-1",
		Bucket:    "gostorage-test",
		Endpoint:  "http://localhost:9002", // Optional: For MinIO or other S3-compatible services
	})

	url, err := storage.UploadBytes(
		context.Background(),
		[]byte("Hello from GoStorage"),
		"examples/hello.txt",
		"text/plain",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Uploaded:", url)
}
```