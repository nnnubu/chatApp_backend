package message

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/message"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func MarkReadStatus(c *gin.Context) {
	var markReadReq dto.MarkConversationRead
	if err := c.ShouldBindJSON(&markReadReq); err != nil {
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

	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	err = message.NewMarkReadStatusService().MarkReadStatus(reqCtx, db, markReadReq.ConversationUID, currentUid)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "成功",
	})
}
