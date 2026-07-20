package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(addr, password string, ttl time.Duration) *Cache {
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: password})
	return &Cache{rdb: rdb, ttl: ttl}
}

func key(code string) string { return "link:" + code }

func (c *Cache) Get(ctx context.Context, code string) (string, bool) {
	v, err := c.rdb.Get(ctx, key(code)).Result()
	if errors.Is(err, redis.Nil) || err != nil {
		return "", false
	}
	return v, true
}

func (c *Cache) Set(ctx context.Context, code, url string) {
	_ = c.rdb.Set(ctx, key(code), url, c.ttl).Err()
}

func (c *Cache) Invalidate(ctx context.Context, code string) {
	_ = c.rdb.Del(ctx, key(code)).Err()
}

func (c *Cache) Ping(ctx context.Context) error { return c.rdb.Ping(ctx).Err() }

func (c *Cache) Close() error { return c.rdb.Close() }
