package middleware

import (
	"ChatApp/model"
	"ChatApp/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取用户认证 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(200, model.CommonResp{
				Code:    401,
				Message: "未登录，请先登录",
			})
			c.Abort()
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			c.JSON(200, model.CommonResp{Code: 401, Message: "凭证错误！"})
			//c.JSON(200, model.CommonResp{Code: 401, Message: "凭证格式错误，需以Bearer开头"})
			c.Abort()
			return
		}

		tokenString := authHeader[len(prefix):]

		uid, err := utils.ParseJwtToken(tokenString)
		if err != nil {
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
