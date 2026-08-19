package friend

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/friend"
	"ChatApp/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func UpdateApplyStatus(c *gin.Context) {
	var ufq dto.UpdateFriendApplyRequest
	if err := c.ShouldBindJSON(&ufq); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    400,
			Message: "请求参数格式错误",
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

	if err = friend.NewUpdateApplyService().UpdateApply(reqCtx, db, &ufq, currentUid); err != nil {
		c.JSON(200, model.CommonResp{
			Code:    400,
			Message: err.Error(),
		})
		return
	}
	var msg string
	if ufq.IsAgree {
		msg = "你们已成为好友"
	} else {
		msg = "已拒绝该好友申请"
	}

	c.JSON(200, model.CommonResp{
		Code:    200,
		Message: msg,
	})
}
