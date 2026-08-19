package database

import (
	"ChatApp/config"
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

func ToRedis(ctx context.Context) *redis.Client {
	log.Println("正在连接Redis...")
	rd := config.Conf.Redis
	rc := redis.NewClient(&redis.Options{
		Addr:     rd.Addr,
		Password: rd.Password,
		DB:       rd.DB,
	})
	_, err := rc.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Redis连接失败：%v", err)
	}
	log.Println("Redis已连接！")
	return rc
}

func OutRedis(rc *redis.Client) {
	log.Println("正在关闭redis...")
	err := rc.Close()
	if err != nil {
		log.Fatalf("关闭redis失败：%v", err)
	}
	log.Println("redis已关闭！")
}
