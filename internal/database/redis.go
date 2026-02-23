package database

import (
	"context"
	"log"
	"ozMadeBack/config"

	"github.com/go-redis/redis/v8"
)

var RDB *redis.Client

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr: config.GetEnv("REDIS_ADDR", "redis:6379"),
	})

	_, err := RDB.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
}

func GetRedisContext() context.Context {
	return context.Background()
}
