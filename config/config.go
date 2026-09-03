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
	AI struct {
		ServiceURL         string `json:"service-url"`            // FastAPI AI 服务地址，如 http://127.0.0.1:8000
		ProactiveEnabled   bool   `json:"proactive-enabled"`      // 主动消息总开关
		ProactiveCheckSec  int    `json:"proactive-check-sec"`    // 检查周期（秒）
		ProactiveMinMinute int    `json:"proactive-min-minute"`   // 主动消息最小间隔（分钟）
		ProactiveMaxMinute int    `json:"proactive-max-minute"`   // 主动消息最大间隔（分钟）
		Characters         []AICharacter `json:"characters"`      // AI 角色列表，控制每个角色是否开启主动消息
	}
}

// AICharacter AI 角色配置
type AICharacter struct {
	UID       string `json:"uid"`        // AI 的 UID
	Proactive bool   `json:"proactive"`  // 是否开启主动消息
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
