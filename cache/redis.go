// cache/redis.go
package cache

import (
	"context"
	"fmt"

	"github.com/kirant400/rp-wrapper/config"
	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	ctx    = context.Background()
)

// Init initializes the Redis connection once at startup
// Called from main.go
func Init() {
	if config.Global.Redis.URL == "" {
		panic("Redis URL is not configured in config.yaml")
	}

	opt, err := redis.ParseURL(config.Global.Redis.URL)
	if err != nil {
		panic(fmt.Sprintf("Invalid Redis URL '%s': %v", config.Global.Redis.URL, err))
	}

	Client = redis.NewClient(opt)

	// Test connection
	_, err = Client.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis at %s: %v", config.Global.Redis.URL, err))
	}

	// Optional: log success in production
	// log.Printf("Connected to Redis: %s", config.Global.Redis.URL)
}

// Optional helper: Close connection gracefully on shutdown
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}