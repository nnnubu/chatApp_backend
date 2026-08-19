package user

import (
	"ChatApp/model"
	"ChatApp/utils"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func GetUserQR(c *gin.Context) {
	uidAny, hasUid := c.Get("uid")
	if !hasUid {
		c.JSON(200, model.CommonResp{
			Code:    401,
			Message: "登录态缺失，请重新登录",
		})
		return
	}

	uid, ok := uidAny.(string)
	if !ok {
		c.JSON(200, model.CommonResp{
			Code:    401,
			Message: "登录信息错误，请重新登录",
		})
		return
	}
	// 二维码不存在则重新生成
	qrPath := fmt.Sprintf("static/qr/%s.png", uid)
	if _, err := os.Stat(qrPath); os.IsNotExist(err) {
		_ = utils.GenUserQRCode(uid, qrPath, 512)
	}
	c.JSON(200, model.CommonResp{
		Code:    200,
		Message: "二维码获取成功",
		Data: map[string]any{
			"url":    qrPath,
			"thumbW": 512,
			"thumbH": 512,
		},
	})
}
