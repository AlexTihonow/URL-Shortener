package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr     string
	PostgresDSN  string
	RedisAddr    string
	RedisPass    string
	KafkaBrokers string
	KafkaTopic   string
	CacheTTL     time.Duration
	BaseURL      string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:     env("HTTP_ADDR", ":8080"),
		PostgresDSN:  env("POSTGRES_DSN", ""),
		RedisAddr:    env("REDIS_ADDR", "localhost:6379"),
		RedisPass:    env("REDIS_PASSWORD", ""),
		KafkaBrokers: env("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:   env("KAFKA_TOPIC", "link.clicks"),
		CacheTTL:     envDuration("CACHE_TTL", time.Hour),
		BaseURL:      env("BASE_URL", "http://localhost:8080"),
	}
	if cfg.PostgresDSN == "" {
		return cfg, fmt.Errorf("POSTGRES_DSN is required")
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
