package friend

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/friend"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddFriends(c *gin.Context) {
	var afq dto.AddFriendRequest
	if err := c.ShouldBindJSON(&afq); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    400,
			Message: "请求参数格式错误",
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

	if err = friend.NewAddFriendService().AddFriend(reqCtx, db, &afq, currentUid); err != nil {
		c.JSON(200, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, model.CommonResp{
		Code:    200,
		Message: "已发送好友请求",
	})
}
