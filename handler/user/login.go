package user

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/user"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func LoginHandler(c *gin.Context) {
	//ShouldBindJSON	Content-Type: application/json	表单上传不可用
	//ShouldBindQuery	URL 问号后的 query 参数
	//ShouldBind	自动兼容 form-data、x-www-form-urlencoded、json 上传表单使用

	var lgq *dto.LoginRequest
	var lgp *dto.LoginResponse
	if err := c.ShouldBindJSON(&lgq); err != nil {
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

	lgp, err = user.NewLoginService().Login(reqCtx, db, lgq)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "登录成功",
		Data:    lgp,
	})
}
