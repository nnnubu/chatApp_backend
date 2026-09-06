package message

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/message"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RecallMessage 撤回消息（仅发送者可撤回自己发送的消息）
func RecallMessage(c *gin.Context) {
	var recallReq dto.RecallMessageReq
	if err := c.ShouldBindJSON(&recallReq); err != nil {
		c.JSON(200, model.CommonResp{
			Code:    400,
			Message: "参数错误",
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
	currentUid, ok := uidAny.(string)
	if !ok {
		c.JSON(200, model.CommonResp{
			Code:    401,
			Message: "登录信息错误，请重新登录",
		})
		return
	}

	reqCtx, db, rc, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	err = message.NewRecallMessageService().RecallMessage(reqCtx, db, rc, currentUid, &recallReq)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "撤回成功",
	})
}
