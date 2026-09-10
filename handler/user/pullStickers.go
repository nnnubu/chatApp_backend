package user

import (
	"ChatApp/model"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PullStickers 拉取当前用户收藏的自定义表情包列表
func PullStickers(c *gin.Context) {
	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	uidAny, hasUid := c.Get("uid")
	if !hasUid {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    401,
			Message: "登录态缺失，请重新登录",
		})
		return
	}
	uid, ok := uidAny.(string)
	if !ok {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    401,
			Message: "登录信息错误，请重新登录",
		})
		return
	}

	stickers, err := model.GetUserStickersByUID(reqCtx, db, uid)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "获取成功",
		Data: map[string]any{
			"list": stickers,
		},
	})
}
