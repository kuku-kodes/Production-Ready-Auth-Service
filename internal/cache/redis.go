package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kaushlender/auth-service/internal/config"
)

type Cache struct {
	client *redis.Client
}

func New(cfg *config.RedisConfig) *Cache {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &Cache{client: client}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Close() error {
	return c.client.Close()
}

// Token Blacklist
func (c *Cache) BlacklistToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	return c.client.Set(ctx, fmt.Sprintf("blacklist:%s", tokenID), true, ttl).Err()
}

func (c *Cache) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	exists, err := c.client.Exists(ctx, fmt.Sprintf("blacklist:%s", tokenID)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Login Attempt Tracking
func (c *Cache) IncrementLoginAttempts(ctx context.Context, email string) (int, error) {
	key := fmt.Sprintf("login_attempts:%s", email)
	attempts, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set expiry for the key
	c.client.Expire(ctx, key, 15*time.Minute)
	return int(attempts), nil
}

func (c *Cache) ResetLoginAttempts(ctx context.Context, email string) error {
	key := fmt.Sprintf("login_attempts:%s", email)
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) IsLoginLocked(ctx context.Context, email string) (bool, error) {
	key := fmt.Sprintf("login_attempts:%s", email)
	attempts, err := c.client.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	return attempts >= 5, nil
}

// Generic Set/Get with TTL
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return fmt.Errorf("failed to get value: %w", err)
	}
	return json.Unmarshal(data, dest)
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Rate Limiting
func (c *Cache) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int, error) {
	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, fmt.Sprintf("ratelimit:%s", key))
	pipe.Expire(ctx, fmt.Sprintf("ratelimit:%s", key), window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int(incr.Val()), nil
}