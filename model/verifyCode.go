package model

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	CodePrefix  = "code:"
	LimitPrefix = "limit:"
	CodeExpire  = 5 * time.Minute
	LimitExpire = 60 * time.Second
)

func SaveCode(ctx context.Context, rc *redis.Client, email string, code string) error {
	key := CodePrefix + email
	return rc.Set(ctx, key, code, CodeExpire).Err()
}

func GetCode(ctx context.Context, rc *redis.Client, email string) (string, error) {
	key := CodePrefix + email
	return rc.Get(ctx, key).Result()
}

func SetSendLimit(ctx context.Context, rc *redis.Client, email string) error {
	key := LimitPrefix + email
	return rc.Set(ctx, key, "1", LimitExpire).Err()
}

func CheckSendLimit(ctx context.Context, rc *redis.Client, email string) (bool, error) {
	key := LimitPrefix + email
	count, err := rc.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func DeleteCode(ctx context.Context, rc *redis.Client, email string) error {
	key := CodePrefix + email
	return rc.Del(ctx, key).Err()
}
