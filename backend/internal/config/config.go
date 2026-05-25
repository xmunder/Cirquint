package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr            string
	ReviewMinConfidence float64
	RedisAddr           string
	RedisQueueKey       string
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

	return Config{
		HTTPAddr:            addr,
		ReviewMinConfidence: reviewMinConfidence,
		RedisAddr:           redisAddr,
		RedisQueueKey:       redisQueueKey,
	}
}
