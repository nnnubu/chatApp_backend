package main

import (
	"ChatApp/config"
	"ChatApp/database"
	"ChatApp/global"
	"ChatApp/model"
	"ChatApp/routers"
	"ChatApp/utils"
	"context"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {

	// 初始化日志，必须放在最前面
	global.InitLogger()
	// 程序退出刷新缓冲区日志
	defer global.Log.Sync()

	// 测试日志输出
	global.Log.Info("服务启动成功")

	e := gin.Default()

	e.MaxMultipartMemory = 10 << 20 // 10 向左移动 20 位 即为 10 * 1024 * 1024 byte
	//srv := &http.Server{
	//	Addr:           "localhost:8080",
	//	Handler:        e,
	//	MaxHeaderBytes: 1 << 20,
	//}
	//_ = srv.ListenAndServe()

	if err := config.LoadConfig("config/config.json"); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}
	if err := utils.InitSnowflake(); err != nil {
		log.Fatal("雪花算法初始化失败：", err)
	}
	db := database.ToMysql("chatapp")
	rc := database.ToRedis(context.Background())
	// context.Background() 是创建一个链表头节点，为后续的可操作节点提供基础
	defer func() {
		database.OutMySQL(db)
		database.OutRedis(rc)
	}()
	// defer 后面跟着的如果是带实参的函数
	// 那么该参数会被立即求值并获取快照存入延迟调用栈中
	// 如果后续该实参有所修改 释放的永远是旧的资源
	// 除非修改实参时通过指针修改其内部的值，而不是重新为他开辟新的地址（创建新的实例对象）
	// 跟着匿名函数就只会在最后调用时求值，保证释放的都是新的资源
	// 普通结构体字符串修改数据之后，旧的数据没有被引用，其占据的内存会被 GC 自动回收
	// MySQL/Redis 连接池：属于操作系统句柄资源，GC 不会自动关闭连接
	// 不手动 Close 会造成长期连接泄漏，数据库挂大量无效连接

	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("自动创建用户表单失败：%v", err)
	}
	if err := db.AutoMigrate(&model.FriendApply{}); err != nil {
		log.Fatalf("自动创建好友请求表单失败：%v", err)
	}
	if err := db.AutoMigrate(&model.Friend{}); err != nil {
		log.Fatalf("自动创建好友表单失败：%v", err)
	}
	if err := db.AutoMigrate(&model.MsgCategory{}); err != nil {
		log.Fatalf("自动创建消息分类表单失败：%v", err)
	}
	if err := db.AutoMigrate(&model.Message{}); err != nil {
		log.Fatalf("自动创建消息表单失败：%v", err)
	}
	if err := db.AutoMigrate(&model.MessageRead{}); err != nil {
		log.Fatalf("自动创建消息已读表单失败：%v", err)
	}
	if err := db.AutoMigrate(&model.Conversation{}); err != nil {
		log.Fatalf("自动创建会话表单失败：%v", err)
	}
	if err := db.AutoMigrate(&model.ConversationMember{}); err != nil {
		log.Fatalf("自动创建会话成员表单失败：%v", err)
	}

	routers.InitRouters(e, db, rc)
	if err := e.Run(strings.Replace(config.Conf.App.PreFixUrl, "http://", "", 1)); err != nil {
		log.Println("HTTP启动失败！")
		log.Fatal(err)
		return
	}
}
