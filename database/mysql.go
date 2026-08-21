package database

import (
	"ChatApp/config"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ToMysql(dbName string) *gorm.DB {
	log.Println("正在连接到mysql...")
	msq := config.Conf.Mysql
	// 如果环境变量中有密码优先使用环境变量的
	if envPwd := os.Getenv("mysql_password"); envPwd != "" {
		msq.Password = envPwd
	}

	// 用 url.Values 构建查询参数（自动处理编码）
	params := url.Values{}
	params.Add("parseTime", "True")
	params.Add("loc", "Asia/Shanghai")
	params.Add("charset", "utf8mb4")

	// dsn格式 用户名:密码@tcp(数据库服务地址:端口号)/数据库名	?后面的都是额外的参数，parseTime是自动处理时间格式 loc用于指定时区 这里用Asia/Shanghai 注意这里的/要转换成url编码
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", msq.Username, msq.Password, msq.Host, msq.Port, dbName, params.Encode())

	var db *gorm.DB
	var err error

	maxRetry := 10
	for i := 0; i < maxRetry; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Error),
		})
		if err == nil {
			// 额外ping确认连接真正可用
			sqlDB, pingErr := db.DB()
			if pingErr == nil {
				if pingErr = sqlDB.Ping(); pingErr == nil {
					log.Println("MySQL已连接！")
					return db
				}
			}
		}

		log.Printf("MySQL连接失败 第%d次重试 2秒后继续: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("MySQL重试%d次全部失败 程序退出: %v", maxRetry, err)
	return db
}

func OutMySQL(db *gorm.DB) {
	log.Println("正在关闭MySQL...")
	msq, _ := db.DB()
	err := msq.Close()
	if err != nil {
		log.Fatalf("关闭MySQL失败：%v", err)
	}
	log.Println("MySQL已关闭！")
}
