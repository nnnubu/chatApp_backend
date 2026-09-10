package user

import (
	"ChatApp/model"
	"ChatApp/service/user"
	"ChatApp/utils"

	"github.com/gin-gonic/gin"
)

// UploadSticker 上传用户自定义表情包
// 仅支持图片格式（含 GIF），上传后落盘并写入 user_sticker 表
func UploadSticker(c *gin.Context) {
	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(200, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

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

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(200, model.CommonResp{
			Code:    400,
			Message: "未能获取上传文件",
		})
		return
	}

	sticker, err := user.NewUploadStickerService().UploadSticker(reqCtx, db, file, uid)
	if err != nil {
		c.JSON(200, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, model.CommonResp{
		Code:    200,
		Message: "上传成功",
		Data: map[string]any{
			"id":  sticker.ID,
			"url": sticker.URL,
		},
	})
}
