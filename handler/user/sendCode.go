package user

import (
	"ChatApp/model"
	"ChatApp/service/user"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SendCode(c *gin.Context) {

	// 逻辑： 接收前端数据并验证数据格式，将email传入service层->service层进行邮箱格式验证之类的操作并生成code,将code传入model层->model层将code存入redis，设置5分钟时限
	//   	 ->成功之后则发送邮件->等待用户注册

	var data map[string]string
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求JSON格式错误",
		})
		return
	}
	email := data["email"]
	reqType := data["type"]

	reqCtx, db, rc, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{Code: 500, Message: err.Error()})
		return
	}

	err = user.NewCodeService().SendCode(reqCtx, rc, db, email, reqType)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "验证码发送成功，5分钟内有效",
	})
}
