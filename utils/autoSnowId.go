package utils

import (
	"github.com/bwmarrin/snowflake"
)

var node *snowflake.Node

// InitSnowflake 初始化雪花节点
// 雪花 ID 是 int64 纯数字，结构：
// 符号位 (1位)：固定 0 + 时间戳(41位) + 机器ID(10位) + 序列号(12位)
func InitSnowflake() error {
	var err error
	// 机器ID 0~1023	分布式相关 就一台机子用0就好了
	node, err = snowflake.NewNode(0)
	return err
}

// GenAutoSnowId 自动生成唯一username
func GenAutoSnowId() string {
	return node.Generate().String()
}
