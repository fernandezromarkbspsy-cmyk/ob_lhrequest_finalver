package cache

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

const (
	KeyStats    = "soc5:stats"
	KeyClusters = "soc5:clusters"
	ChannelSSE  = "soc5:events"

	TTLStats    = 15 * time.Second
	TTLClusters = 5 * time.Minute
)

func Connect() {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}

	Client = redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis unavailable (%v) — caching disabled, falling back to direct DB", err)
		Client = nil
		return
	}

	log.Println("Connected to Redis")
}

func Get[T any](ctx context.Context, key string) (T, bool) {
	var zero T
	if Client == nil {
		return zero, false
	}
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return zero, false
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, false
	}
	return result, true
}

func Set(ctx context.Context, key string, value any, ttl time.Duration) {
	if Client == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	Client.Set(ctx, key, data, ttl)
}

func Delete(ctx context.Context, keys ...string) {
	if Client == nil {
		return
	}
	Client.Del(ctx, keys...)
}

func Publish(ctx context.Context, channel string, payload any) {
	if Client == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	Client.Publish(ctx, channel, data)
}

func Subscribe(ctx context.Context, channel string) <-chan string {
	ch := make(chan string, 64)
	if Client == nil {
		close(ch)
		return ch
	}
	sub := Client.Subscribe(ctx, channel)
	go func() {
		defer close(ch)
		for {
			msg, err := sub.ReceiveMessage(ctx)
			if err != nil {
				return
			}
			ch <- msg.Payload
		}
	}()
	return ch
}
