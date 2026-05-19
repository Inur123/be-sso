package database

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"sso.pelajarnumagetan.or.id/internal/config"
)

var Redis *redis.Client

func ConnectRedis() *redis.Client {
	cfg := config.Get()

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx := context.Background()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("✅ Redis connected")
	Redis = rdb
	return rdb
}
