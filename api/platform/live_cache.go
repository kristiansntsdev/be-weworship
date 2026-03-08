package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type LiveState struct {
	SongIndex    int       `json:"song_index"`
	ScrollRatio  float64   `json:"scroll_ratio"`
	IsActive     bool      `json:"is_active"`
	LeaderUserID int       `json:"leader_user_id"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type LiveCache struct {
	enabled bool
	client  *redis.Client
	ttl     time.Duration
}

func NewLiveCache() *LiveCache {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		return &LiveCache{enabled: false}
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return &LiveCache{enabled: false}
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return &LiveCache{enabled: false}
	}
	return &LiveCache{enabled: true, client: client, ttl: 4 * time.Hour}
}

func (c *LiveCache) Enabled() bool {
	return c != nil && c.enabled && c.client != nil
}

func liveKey(playlistID int) string {
	return fmt.Sprintf("live:playlist:%d", playlistID)
}

func (c *LiveCache) StartSession(playlistID, leaderUserID int) error {
	if !c.Enabled() {
		return fmt.Errorf("live cache not available")
	}
	state := LiveState{
		SongIndex:    0,
		ScrollRatio:  0,
		IsActive:     true,
		LeaderUserID: leaderUserID,
		UpdatedAt:    time.Now(),
	}
	buf, err := json.Marshal(state)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return c.client.Set(ctx, liveKey(playlistID), buf, c.ttl).Err()
}

func (c *LiveCache) EndSession(playlistID int) error {
	if !c.Enabled() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return c.client.Del(ctx, liveKey(playlistID)).Err()
}

func (c *LiveCache) UpdateState(playlistID, songIndex int, scrollRatio float64) error {
	if !c.Enabled() {
		return fmt.Errorf("live cache not available")
	}
	// Read existing to preserve leader_user_id
	existing, err := c.GetState(playlistID)
	if err != nil || existing == nil {
		return fmt.Errorf("no active session for playlist %d", playlistID)
	}
	existing.SongIndex = songIndex
	existing.ScrollRatio = scrollRatio
	existing.UpdatedAt = time.Now()
	buf, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Reset TTL on each update
	return c.client.Set(ctx, liveKey(playlistID), buf, c.ttl).Err()
}

func (c *LiveCache) GetState(playlistID int) (*LiveState, error) {
	if !c.Enabled() {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	val, err := c.client.Get(ctx, liveKey(playlistID)).Result()
	if err == redis.Nil {
		return nil, nil // no active session
	}
	if err != nil {
		return nil, err
	}
	var state LiveState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return nil, err
	}
	return &state, nil
}
