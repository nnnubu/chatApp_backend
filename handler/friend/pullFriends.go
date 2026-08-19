package friend

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/friend"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PullFriends(c *gin.Context) {
	var req dto.PullFriendsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(200, model.CommonResp{
			Code:    400,
			Message: "参数错误",
		})
		return
	}

	// 修正非法分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
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
	returnList, hasMore, err := friend.NewPullFriendsService().PullFriends(reqCtx, db, currentUid, req.Page, req.PageSize)
	if err != nil {
		c.JSON(200, model.CommonResp{
			Code:    400,
			Message: err.Error(),
		})
		return
	}
	c.JSON(200, model.CommonResp{
		Code:    200,
		Message: "success",
		Data: map[string]any{
			"friends":  returnList,
			"page":     req.Page,
			"pageSize": req.PageSize,
			"hasMore":  hasMore,
		},
	})
}
