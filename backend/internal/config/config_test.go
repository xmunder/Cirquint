package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("REVIEW_MIN_CONFIDENCE", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_QUEUE_KEY", "")

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
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("REVIEW_MIN_CONFIDENCE", "0.95")
	t.Setenv("REDIS_ADDR", "redis.internal:6380")
	t.Setenv("REDIS_QUEUE_KEY", "processing:jobs")

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
}

func TestLoadIgnoresInvalidConfidence(t *testing.T) {
	t.Setenv("REVIEW_MIN_CONFIDENCE", "not-a-number")

	cfg := Load()
	if cfg.ReviewMinConfidence != 0.8 {
		t.Fatalf("ReviewMinConfidence = %v, want fallback 0.8", cfg.ReviewMinConfidence)
	}
}
