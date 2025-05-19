package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(host string, port string, password string, db int) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	return &RedisCache{client: rdb}
}

func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisCache) Get(key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

func (r *RedisCache) Delete(key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Migrate(key, match string) error {
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}

	if exists == 0 {
		keys, err := r.client.Keys(ctx, match).Result()
		if err != nil {
			return err
		}

		pipe := r.client.Pipeline()
		for _, k := range keys {
			pipe.Del(ctx, k)
		}

		_, err = pipe.Exec(ctx)
		if err != nil {
			return err
		}

		err = r.client.Set(ctx, key, "true", 0).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

func initRedisCache(cfg Config) (*RedisCache, error) {
	cache := NewRedisCache(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, 0)

	err := cache.Migrate("flatty", "*:*")
	if err != nil {
		return nil, err
	}

	err = cache.Migrate("snappyRss", "rss:*")
	if err != nil {
		return nil, err
	}

	err = cache.Migrate("userBuckets", "p:*")
	if err != nil {
		return nil, err
	}

	err = cache.Migrate("profileDates", "p:*")
	if err != nil {
		return nil, err
	}

	err = cache.Migrate("profileStats", "p:*")
	if err != nil {
		return nil, err
	}

	err = cache.Migrate("userType", "p:*")
	if err != nil {
		return nil, err
	}

	return cache, nil
}

func main() {
	cfg := Config{
		RedisHost:     "localhost",
		RedisPort:     "6379",
		RedisPassword: "",
	}

	cache, err := initRedisCache(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v", err)
	}

	// Example usage
	err = cache.Set("example_key", "example_value", 10*time.Minute)
	if err != nil {
		log.Fatalf("Failed to set cache: %v", err)
	}

	var value string
	err = cache.Get("example_key", &value)
	if err != nil {
		log.Fatalf("Failed to get cache: %v", err)
	}

	fmt.Println("Cached value:", value)
}
