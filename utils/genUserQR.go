package utils

import (
	"fmt"
	"os"
	"path"

	"github.com/skip2/go-qrcode"
)

// GenUserQRCode 生成用户加好友二维码图片
// uid: 用户完整雪花UID字符串
// savePath: 图片保存路径
// size: 图片像素尺寸（256/512）
func GenUserQRCode(uid string, savePath string, size int) error {
	// 约定扫码文本
	content := fmt.Sprintf("uid:%s", uid)
	png, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(path.Dir(savePath), 0755)
	return os.WriteFile(savePath, png, 0644)
}
