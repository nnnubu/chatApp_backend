package utils

import (
	"crypto/rand"
	"log"
	"math/big"
	"strings"
)

func GenerateCode(length int) string {
	if length < 1 {
		log.Fatal("长度必须大于1！")
	}
	// 用 strings.Builder 来构造字符串
	// 若采用 "" + 的方式会频繁创建新字符串，效率低
	var builder strings.Builder
	// 预分配长度
	builder.Grow(length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10)) // rand.Reader 读取系统随机源来生成随机数
		if err != nil {
			log.Fatalf("系统随机源不可用：%v", err)
		}
		builder.WriteByte(byte(num.Int64() + '0'))
	}
	code := builder.String()
	return code
}
