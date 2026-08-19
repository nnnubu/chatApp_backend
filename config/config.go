package config

import (
	"encoding/json"
	"log"
	"os"
)

var Conf *Config

type Config struct {
	Mysql struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"` // 此处本地跑就写本地的 host 若是 docker 跑就写 mysql
		Port     string `json:"port"`
	}
	Redis struct {
		Addr     string `json:"addr"` // 此处本地跑就写 localhost:6379 若是 docker 就写 redis:6379
		Password string `json:"password"`
		DB       int    `json:"db"`
	}
	Smtp struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     int    `json:"port"` //http 用 25	https 用 465
	}
	Jwt struct {
		Secret string `json:"secret"`
		Expire string `json:"expire"`
	}
	App struct {
		BasePath  string `json:"base-path"`
		PreFixUrl string `json:"pre-fix-url"` // 容器内部需要使用 0.0.0.0 监听容器内所有网卡的流量
	}
	ImageResize struct {
		AvatarW int `json:"avatar_w"`
		AvatarH int `json:"avatar_h"`
		BgW     int `json:"bg_w"`
		BgH     int `json:"bg_h"`
	}
}

func LoadConfig(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("配置文件关闭警告: %v", err)
		}
	}()

	// 创建一个临时变量来存储配置信息，等待解析全部成功，再给全局变量Conf，防止直接给全局Conf出现脏数据
	newConf := &Config{}
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&newConf); err != nil {
		return err
	}
	Conf = newConf
	return nil
}
