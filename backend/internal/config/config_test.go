package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("REVIEW_MIN_CONFIDENCE", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_QUEUE_KEY", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OBJECT_STORAGE_BUCKET", "")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "")
	t.Setenv("OBJECT_STORAGE_PATH", "")

	cfg := Load()
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.ReviewMinConfidence != 0.8 {
		t.Fatalf("ReviewMinConfidence = %v, want 0.8", cfg.ReviewMinConfidence)
	}
	if cfg.RedisAddr != "127.0.0.1:6379" {
		t.Fatalf("RedisAddr = %q, want default", cfg.RedisAddr)
	}
	if cfg.RedisQueueKey != "" {
		t.Fatalf("RedisQueueKey = %q, want empty", cfg.RedisQueueKey)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
	if cfg.ObjectStorage.Bucket != "" {
		t.Fatalf("ObjectStorage.Bucket = %q, want empty", cfg.ObjectStorage.Bucket)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("REVIEW_MIN_CONFIDENCE", "0.95")
	t.Setenv("REDIS_ADDR", "redis.internal:6380")
	t.Setenv("REDIS_QUEUE_KEY", "processing:jobs")
	t.Setenv("DATABASE_URL", "postgres://app:secret@db.internal:5432/cirquint")
	t.Setenv("OBJECT_STORAGE_BUCKET", "cirquint-dev")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://r2.example.com")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "access-key")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "secret-key")
	t.Setenv("OBJECT_STORAGE_PATH", "/var/lib/cirquint/objects")

	cfg := Load()
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9090")
	}
	if cfg.ReviewMinConfidence != 0.95 {
		t.Fatalf("ReviewMinConfidence = %v, want 0.95", cfg.ReviewMinConfidence)
	}
	if cfg.RedisAddr != "redis.internal:6380" {
		t.Fatalf("RedisAddr = %q, want env value", cfg.RedisAddr)
	}
	if cfg.RedisQueueKey != "processing:jobs" {
		t.Fatalf("RedisQueueKey = %q, want env value", cfg.RedisQueueKey)
	}
	if cfg.DatabaseURL != "postgres://app:secret@db.internal:5432/cirquint" {
		t.Fatalf("DatabaseURL = %q, want env value", cfg.DatabaseURL)
	}
	if cfg.ObjectStorage.Bucket != "cirquint-dev" {
		t.Fatalf("ObjectStorage.Bucket = %q, want env value", cfg.ObjectStorage.Bucket)
	}
	if cfg.ObjectStorage.Endpoint != "https://r2.example.com" {
		t.Fatalf("ObjectStorage.Endpoint = %q, want env value", cfg.ObjectStorage.Endpoint)
	}
	if cfg.ObjectStorage.AccessKey != "access-key" {
		t.Fatalf("ObjectStorage.AccessKey = %q, want env value", cfg.ObjectStorage.AccessKey)
	}
	if cfg.ObjectStorage.SecretKey != "secret-key" {
		t.Fatalf("ObjectStorage.SecretKey = %q, want env value", cfg.ObjectStorage.SecretKey)
	}
	if cfg.ObjectStorage.Path != "/var/lib/cirquint/objects" {
		t.Fatalf("ObjectStorage.Path = %q, want env value", cfg.ObjectStorage.Path)
	}
}

func TestLoadIgnoresInvalidConfidence(t *testing.T) {
	t.Setenv("REVIEW_MIN_CONFIDENCE", "not-a-number")

	cfg := Load()
	if cfg.ReviewMinConfidence != 0.8 {
		t.Fatalf("ReviewMinConfidence = %v, want fallback 0.8", cfg.ReviewMinConfidence)
	}
}

func TestValidateSharedRuntime(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want error
	}{
		{
			name: "missing database url",
			cfg:  Config{ObjectStorage: ObjectStorageConfig{Bucket: "review-bucket", Endpoint: "https://r2.example.com", AccessKey: "key", SecretKey: "secret"}},
			want: ErrDatabaseURLRequired,
		},
		{
			name: "missing object storage bucket",
			cfg:  Config{DatabaseURL: "postgres://app:secret@db.internal:5432/cirquint", ObjectStorage: ObjectStorageConfig{Endpoint: "https://r2.example.com", AccessKey: "key", SecretKey: "secret"}},
			want: ErrObjectStorageBucketRequired,
		},
		{
			name: "missing object storage endpoint",
			cfg:  Config{DatabaseURL: "postgres://app:secret@db.internal:5432/cirquint", ObjectStorage: ObjectStorageConfig{Bucket: "review-bucket", AccessKey: "key", SecretKey: "secret"}},
			want: ErrObjectStorageEndpointRequired,
		},
		{
			name: "missing object storage access key",
			cfg:  Config{DatabaseURL: "postgres://app:secret@db.internal:5432/cirquint", ObjectStorage: ObjectStorageConfig{Bucket: "review-bucket", Endpoint: "https://r2.example.com", SecretKey: "secret"}},
			want: ErrObjectStorageAccessKeyRequired,
		},
		{
			name: "missing object storage secret key",
			cfg:  Config{DatabaseURL: "postgres://app:secret@db.internal:5432/cirquint", ObjectStorage: ObjectStorageConfig{Bucket: "review-bucket", Endpoint: "https://r2.example.com", AccessKey: "key"}},
			want: ErrObjectStorageSecretKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.ValidateSharedRuntime(); err != tt.want {
				t.Fatalf("ValidateSharedRuntime() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestValidateSharedRuntimeAcceptsCompleteConfig(t *testing.T) {
	cfg := Config{
		DatabaseURL: "postgres://app:secret@db.internal:5432/cirquint",
		ObjectStorage: ObjectStorageConfig{
			Bucket:    "review-bucket",
			Endpoint:  "https://r2.example.com",
			AccessKey: "key",
			SecretKey: "secret",
		},
	}

	if err := cfg.ValidateSharedRuntime(); err != nil {
		t.Fatalf("ValidateSharedRuntime() error = %v, want nil", err)
	}
}
