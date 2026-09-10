package user

import (
	"ChatApp/model"
	"ChatApp/utils"

	"github.com/gin-gonic/gin"
)

// DeleteSticker 删除用户收藏的自定义表情包
// 仅能删除自己收藏的表情包
func DeleteSticker(c *gin.Context) {
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

	stickerId := c.Query("id")
	if stickerId == "" {
		stickerId = c.PostForm("id")
	}
	if stickerId == "" {
		c.JSON(200, model.CommonResp{
			Code:    400,
			Message: "缺少表情包id",
		})
		return
	}

	rowsAffected, err := model.DeleteUserSticker(reqCtx, db, stickerId, uid)
	if err != nil {
		c.JSON(200, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}
	if rowsAffected == 0 {
		c.JSON(200, model.CommonResp{
			Code:    404,
			Message: "表情包不存在或无权删除",
		})
		return
	}

	c.JSON(200, model.CommonResp{
		Code:    200,
		Message: "删除成功",
	})
}
