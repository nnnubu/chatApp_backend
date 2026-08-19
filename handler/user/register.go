package user

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/user"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterHandler(c *gin.Context) {
	var rgq *dto.RegisterRequest
	if err := c.ShouldBindJSON(&rgq); err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    400,
			Message: "参数错误：请输入正确信息",
		})
		return
	}

	reqCtx, db, rc, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	err = user.NewRegisterService().Register(reqCtx, db, rc, rgq)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "注册成功！",
	})
}
