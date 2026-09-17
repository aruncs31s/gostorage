package utils

import (
	"os"

	"github.com/aruncs31s/gologger"
)

func GetEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
func MustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		gologger.Fatal("provided key is empty")
	}
	return val
}
