package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

var (
	ErrDatabaseURLRequired            = errors.New("database_url is required")
	ErrObjectStorageBucketRequired    = errors.New("object_storage_bucket is required")
	ErrObjectStorageEndpointRequired  = errors.New("object_storage_endpoint is required")
	ErrObjectStorageAccessKeyRequired = errors.New("object_storage_access_key is required")
	ErrObjectStorageSecretKeyRequired = errors.New("object_storage_secret_key is required")
)

type ObjectStorageConfig struct {
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Path      string
}

type Config struct {
	HTTPAddr            string
	ReviewMinConfidence float64
	RedisAddr           string
	RedisQueueKey       string
	DatabaseURL         string
	ObjectStorage       ObjectStorageConfig
}

func Load() Config {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	reviewMinConfidence := 0.8
	if raw := os.Getenv("REVIEW_MIN_CONFIDENCE"); raw != "" {
		if value, err := strconv.ParseFloat(raw, 64); err == nil {
			reviewMinConfidence = value
		}
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	redisQueueKey := os.Getenv("REDIS_QUEUE_KEY")
	databaseURL := os.Getenv("DATABASE_URL")

	return Config{
		HTTPAddr:            addr,
		ReviewMinConfidence: reviewMinConfidence,
		RedisAddr:           redisAddr,
		RedisQueueKey:       redisQueueKey,
		DatabaseURL:         databaseURL,
		ObjectStorage: ObjectStorageConfig{
			Bucket:    os.Getenv("OBJECT_STORAGE_BUCKET"),
			Endpoint:  os.Getenv("OBJECT_STORAGE_ENDPOINT"),
			AccessKey: os.Getenv("OBJECT_STORAGE_ACCESS_KEY"),
			SecretKey: os.Getenv("OBJECT_STORAGE_SECRET_KEY"),
			Path:      os.Getenv("OBJECT_STORAGE_PATH"),
		},
	}
}

func (c Config) ValidateSharedRuntime() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return ErrDatabaseURLRequired
	}
	if strings.TrimSpace(c.ObjectStorage.Bucket) == "" {
		return ErrObjectStorageBucketRequired
	}
	if strings.TrimSpace(c.ObjectStorage.Endpoint) == "" {
		return ErrObjectStorageEndpointRequired
	}
	if strings.TrimSpace(c.ObjectStorage.AccessKey) == "" {
		return ErrObjectStorageAccessKeyRequired
	}
	if strings.TrimSpace(c.ObjectStorage.SecretKey) == "" {
		return ErrObjectStorageSecretKeyRequired
	}
	return nil
}
