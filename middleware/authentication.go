package middleware

import (
	"ChatApp/model"
	"ChatApp/utils"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取用户认证 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("认证失败: 缺少 Authorization 头, path=%s, ua=%s",
				c.Request.URL.Path, c.Request.UserAgent())
			c.JSON(200, model.CommonResp{
				Code:    401,
				Message: "未登录，请先登录",
			})
			c.Abort()
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			log.Printf("认证失败: Authorization 格式错误, path=%s", c.Request.URL.Path)
			c.JSON(200, model.CommonResp{Code: 401, Message: "凭证错误！"})
			c.Abort()
			return
		}

		tokenString := authHeader[len(prefix):]

		uid, err := utils.ParseJwtToken(tokenString)
		if err != nil {
			log.Printf("认证失败: token 无效或已过期, path=%s, err=%v",
				c.Request.URL.Path, err)
			c.JSON(200, model.CommonResp{
				Code:    401,
				Message: "token无效或已过期",
			})
			c.Abort()
			return
		}
		c.Set("uid", uid)
		c.Next()
	}
}
