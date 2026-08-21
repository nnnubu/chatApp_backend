package routers

import (
	"ChatApp/handler/friend"
	"ChatApp/handler/message"
	"ChatApp/handler/msgCategory"
	"ChatApp/handler/user"
	"ChatApp/middleware"
	"ChatApp/socket/service"
	"context"
	"net/http"
	"strings"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

func InitRouters(e *gin.Engine, db *gorm.DB, rc *redis.Client) {
	//为静态资源统一设置缓存策略
	e.Use(func(c *gin.Context) {
		if strings.Contains(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=86400")
		}
		c.Next()
	})

	e.Use(func(c *gin.Context) {
		// 为每一个连接提供一个空链表节点，在此基础上生成可操作的子节点，并在连接超时后取消后续操作
		reqCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		// 将每个请求都强行注入 以下变量 方便进行数据库的调取，以及超时操作的处理
		c.Set("reqCtx", reqCtx)
		c.Set("db", db)
		c.Set("rc", rc)
		c.Next()
	})

	// 配置跨域
	e.Use(cors.New(cors.Config{
		// 允许访问的前端地址
		AllowOrigins: []string{"*"},
		// 允许的HTTP请求方法
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// 允许的请求头
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		// 暴露给前端的响应头
		ExposeHeaders: []string{"Content-Length"},
		// 允许携带cookie，若前端需要发送cookie，需设置为true
		AllowCredentials: true,
		// 跨域缓存时间
		MaxAge: 12 * time.Hour,
	}))

	//设置Uer-Agent反爬
	e.Use(func(c *gin.Context) {
		userAgent := strings.TrimSpace(c.Request.Header.Get("User-Agent"))
		if userAgent == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			c.Abort()
			return
		}

		// 允许的客户端标识（可根据需求扩展）
		allowedAgents := []string{"Mozilla", "Chrome", "Safari", "Edge", "Postman", "Dart", "Flutter"}
		isAllowed := false
		for _, agent := range allowedAgents {
			if strings.Contains(strings.ToLower(userAgent), strings.ToLower(agent)) {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "不支持的客户端"})
			c.Abort()
			return
		}
		c.Next()
	})

	//以上这几个部分其实都是注册中间件

	e.Static("/static", "./static") //注意这个要放在缓存策略后面，否则不会起作用

	GroupApi := e.Group("/api")
	GroupApi.POST("/login", user.LoginHandler)
	GroupApi.POST("/register", user.RegisterHandler)
	GroupApi.POST("/resetPwd", user.ResetPwdHandler)
	GroupApi.POST("/sendCode", user.SendCode)

	GroupAuth := e.Group("/auth", middleware.AuthMiddleware())
	GroupAuth.POST("/uploadImage", user.UploadImage)
	GroupAuth.POST("/updateInfo", user.UpdateInfo)
	GroupAuth.POST("/getUserQR", user.GetUserQR)
	GroupAuth.POST("/visitOthers", user.VisitOthers)
	GroupAuth.POST("/addFriend", friend.AddFriends)
	GroupAuth.POST("/updateApply", friend.UpdateApplyStatus)
	GroupAuth.GET("/pullOfflineApply", friend.PullOfflineApply)
	GroupAuth.GET("/pullCategory", msgCategory.PullCategory)
	GroupAuth.GET("/pullFriends", friend.PullFriends)
	GroupAuth.GET("/pullHistoryMessage", message.PullHistoryMessage)
	GroupAuth.GET("/pullUnReadMessage", message.PullUnReadMessage)
	GroupAuth.POST("/markReadStatus", message.MarkReadStatus)

	GroupWs := e.Group("/ws", middleware.AuthMiddleware())
	GroupWs.GET("/connect", service.WebSocketConnect)

}
