package database

import (
	"ChatApp/config"
	"context"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

func ToRedis(ctx context.Context) *redis.Client {
	log.Println("正在连接Redis...")
	rd := config.Conf.Redis
	var rc *redis.Client
	var err error
	if envPwd := os.Getenv("redis_password"); envPwd != "" {
		rd.Password = envPwd
	}

	maxRetry := 10
	for i := 0; i < maxRetry; i++ {
		rc = redis.NewClient(&redis.Options{
			Addr:     rd.Addr,
			Password: rd.Password,
			DB:       rd.DB,
		})
		_, err = rc.Ping(ctx).Result()
		if err == nil {
			log.Println("Redis已连接！")
			return rc
		}
		log.Printf("Redis连接失败 第%d次重试 2秒后继续: %v", i+1, err)
		// 关闭本次失败的client，防止泄漏
		_ = rc.Close()
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("Redis重试%d次全部失败 程序退出: %v", maxRetry, err)
	return nil
}

func OutRedis(rc *redis.Client) {
	log.Println("正在关闭redis...")
	err := rc.Close()
	if err != nil {
		log.Fatalf("关闭redis失败：%v", err)
	}
	log.Println("redis已关闭！")
}
