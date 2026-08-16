package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:         os.Getenv("REDIS_ADDR"),
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		Protocol:     2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	log.Println("Connected To Redis Successfully")
}

func CloseRedis() {
	if RedisClient != nil {
		RedisClient.Close()
	}
}

// InvalidateCacheByPattern deletes every key matching pattern (e.g.
// "user:profile:<id>:page:*"). Used to evict all paginated cache entries
// for a resource after a write that changes it.
func InvalidateCacheByPattern(ctx context.Context, pattern string) {
	var keys []string
	iter := RedisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Printf("cache scan failed for pattern %s: %v", pattern, err)
		return
	}

	if len(keys) == 0 {
		return
	}
	if err := RedisClient.Del(ctx, keys...).Err(); err != nil {
		log.Printf("cache invalidation failed for pattern %s: %v", pattern, err)
	}
}
