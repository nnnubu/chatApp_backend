package user

import (
	"ChatApp/model"
	"ChatApp/service/user"
	"ChatApp/utils"

	"github.com/gin-gonic/gin"
)

func UploadImage(c *gin.Context) {
	uploadType := c.PostForm("uploadType")
	if uploadType == "" || (uploadType != "avatar" && uploadType != "cover" && uploadType != "bgImg") {
		c.JSON(200, model.CommonResp{
			Code:    400,
			Message: "参数错误，uploadType仅支持avatar、cover、bgImg",
		})
		return
	}

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

	// 读取上传文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(200, model.CommonResp{
			Code:    400,
			Message: "未能获取上传文件",
		})
		return
	}

	//thumbSize,originUrl, thumbUrl, err := service.NewUploadImgService().UploadImg(reqCtx, db, uploadType, file, uid)
	thumbSize, _, thumbUrl, err := user.NewUploadImgService().UploadImg(reqCtx, db, uploadType, file, uid)
	if err != nil {
		c.JSON(200, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	var w, h int
	if thumbSize != nil {
		w = (*thumbSize)["thumbW"]
		h = (*thumbSize)["thumbH"]
	} else {
		// 空指针兜底宽高0
		w = 0
		h = 0
	}

	c.JSON(200, model.CommonResp{
		Code:    200,
		Message: "更新成功",
		//Data: map[string]any{
		//	"originUrl":   originUrl,
		//	"thumbUrl":    thumbUrl,
		//	"thumbSize":   thumbSize,
		//},
		Data: map[string]any{
			"url":    thumbUrl,
			"thumbW": w,
			"thumbH": h,
		},
	})
}
