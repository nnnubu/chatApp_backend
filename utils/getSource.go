package utils

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

func GetRequestSource(c *gin.Context) (ctx context.Context, db *gorm.DB, rc *redis.Client, err error) {
	reqCtxAny, hasCtx := c.Get("reqCtx")
	dbAny, hasDb := c.Get("db")
	rcAny, hasRc := c.Get("rc")

	if !hasCtx || !hasDb || !hasRc {
		return nil, nil, nil, errors.New("服务初始化异常")
	}

	reqCtx, ok1 := reqCtxAny.(context.Context)
	dbIns, ok2 := dbAny.(*gorm.DB)
	rcIns, ok3 := rcAny.(*redis.Client)

	if !ok1 || !ok2 || !ok3 {
		return nil, nil, nil, errors.New("服务依赖类型错误")
	}

	return reqCtx, dbIns, rcIns, nil
}
