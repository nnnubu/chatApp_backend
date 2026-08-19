package user

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/service/user"
	"ChatApp/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ResetPwdHandler(c *gin.Context) {
	var req *dto.RestPwdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误：请输入正确信息",
		})
		return
	}
	fmt.Println("重置密码信息：", req)

	reqCtx, db, rc, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	err = user.NewResetPwdService().ResetPwd(reqCtx, db, rc, req)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 400, Message: err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "重置成功",
	})
}
