package friend

import (
	"ChatApp/model"
	"ChatApp/service/friend"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PullOfflineApply(c *gin.Context) {
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

	applyList, err := friend.NewOfflineApplyService().PullOfflineApply(reqCtx, db, currentUid)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, model.CommonResp{
		Code:    200,
		Message: "success",
		Data:    applyList,
	})
}
