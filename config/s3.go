package config

import "github.com/aruncs31s/gostorage/utils"

type S3Config struct {
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string
	Endpoint  string // Optional: For MinIO or other S3-compatible services
}

func GetFromEnv() S3Config {
	return S3Config{
		AccessKey: utils.GetEnv("AWS_ACCESS_KEY_ID", ""),
		SecretKey: utils.GetEnv("AWS_SECRET_ACCESS_KEY", ""),
		Region:    utils.GetEnv("AWS_REGION", "us-east-1"),
		Bucket:    utils.GetEnv("AWS_S3_BUCKET", ""),
		Endpoint:  utils.GetEnv("S3_ENDPOINT", ""), // Optional: For MinIO or other S3-compatible services
	}
}
