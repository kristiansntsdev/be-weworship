package platform

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type SongCache struct {
	enabled bool
	client  *redis.Client
	ttl     time.Duration
}

func NewSongCache() *SongCache {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		return &SongCache{enabled: false}
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return &SongCache{enabled: false}
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return &SongCache{enabled: false}
	}
	return &SongCache{enabled: true, client: client, ttl: 5 * time.Minute}
}

func (c *SongCache) Enabled() bool {
	return c != nil && c.enabled && c.client != nil
}

func (c *SongCache) Get(key string, out any) bool {
	if !c.Enabled() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(val), out) == nil
}

func (c *SongCache) Set(key string, value any) {
	if !c.Enabled() {
		return
	}
	buf, err := json.Marshal(value)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.client.Set(ctx, key, buf, c.ttl).Err()
}

func (c *SongCache) InvalidateSongsList() {
	if !c.Enabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, "songs:list:*", 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			_ = c.client.Del(ctx, keys...).Err()
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}
