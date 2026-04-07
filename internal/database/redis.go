package database

import (
	"context"
	"log"
	"ozMadeBack/config"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var RDB *redis.Client

func InitRedis() {
	dbIndex, err := strconv.Atoi(config.GetEnv("REDIS_DB", "0"))
	if err != nil {
		log.Printf("invalid REDIS_DB value, using default 0: %v", err)
		dbIndex = 0
	}

	RDB = redis.NewClient(&redis.Options{
		Addr:         config.GetEnv("REDIS_ADDR", "redis:6379"),
		Password:     config.GetEnv("REDIS_PASSWORD", ""),
		DB:           dbIndex,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = RDB.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
}

func GetRedisContext() context.Context {
	return context.Background()
}
