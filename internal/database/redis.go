package database

import (
	"context"
	"github.com/go-redis/redis/v8"
	"ozMadeBack/config"
)

var RDB *redis.Client

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr: config.GetEnv("REDIS_ADDR"),
	})
}

func GetRedisContext() context.Context {
	return context.Background()
}
