package user

import (
	"ChatApp/model"
	"ChatApp/utils"
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type CodeService struct{}

func NewCodeService() *CodeService {
	return &CodeService{}
}

func (cs *CodeService) SendCode(ctx context.Context, rc *redis.Client, db *gorm.DB, email string, reqType string) error {
	// 逻辑：校验数据格式->检查发送频率->检查邮箱是否存在->生成验证码并设置时间与频率限制->发送验证码
	if msg := utils.CheckEmail(email); msg != "" {
		return errors.New(msg)
	}

	isLimited, err := model.CheckSendLimit(ctx, rc, email)
	if err != nil {
		return errors.New("校验发送频率失败，请稍后重试")
	}
	if isLimited {
		return errors.New("发送过于频繁，请1分钟后再试")
	}

	isExists, err := model.IsEmailExists(ctx, db, email)
	if err != nil {
		return errors.New("查询用户信息超时")
	}
	if isExists && reqType == "register" {
		return errors.New("该邮箱已注册，请直接登录")
	}

	if !isExists && reqType == "resetPwd" {
		return errors.New("邮箱尚未注册，请先注册")
	}

	code := utils.GenerateCode(6)

	err = model.SaveCode(ctx, rc, email, code)
	if err != nil {
		return errors.New("验证码存储失败")
	}

	err = model.SetSendLimit(ctx, rc, email)
	if err != nil {
		return errors.New("限流设置失败")
	}

	err = utils.SendQQEmail(email, code)
	if err != nil {
		return errors.New("邮件发送失败，请检查邮箱")
	}

	return nil
}
