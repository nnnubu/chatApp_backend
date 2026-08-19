package user

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/user"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func VisitOthers(c *gin.Context) {
	var data map[string]string
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求JSON格式错误",
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
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	targetUid := data["targetUid"]

	var other *dto.GetOthersResponse
	other, err = user.NewGetOthersService().VisitOthers(reqCtx, db, currentUid, targetUid)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "访问成功",
		Data:    other,
	})
}
