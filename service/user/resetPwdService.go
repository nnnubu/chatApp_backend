package user

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/utils"
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type ResetPwdService struct{}

func NewResetPwdService() *ResetPwdService {
	return &ResetPwdService{}
}

func (rps *ResetPwdService) ResetPwd(ctx context.Context, db *gorm.DB, rc *redis.Client, rsp *dto.RestPwdRequest) error {
	if msg := utils.CheckEmail(rsp.Email); msg != "" {
		return errors.New(msg)
	}
	if msg := utils.CheckCode(rsp.Code); msg != "" {
		return errors.New(msg)
	}
	if msg := utils.CheckPassword(rsp.Password); msg != "" {
		return errors.New(msg)
	}

	isEmailExists, err := model.IsEmailExists(ctx, db, rsp.Email)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	if !isEmailExists {
		return errors.New("邮箱尚未注册，请先注册")
	}

	code, err := model.GetCode(ctx, rc, rsp.Email)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return errors.New("验证码不存在或已经过期，请重新获取")
		}
		return errors.New("系统繁忙，请稍后重试")
	}
	if code != rsp.Code {
		return errors.New("验证码错误")
	}

	encryptPwd, err := utils.EncryptPwd(rsp.Password)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = model.UpdatePassword(ctx, tx, rsp.Email, encryptPwd); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	_ = model.DeleteCode(ctx, rc, rsp.Email)

	return nil
}
