package user

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/utils"
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type RegisterService struct{}

func NewRegisterService() *RegisterService {
	return &RegisterService{}
}

func (rs *RegisterService) Register(ctx context.Context, db *gorm.DB, rc *redis.Client, rgq *dto.RegisterRequest) error {
	// 逻辑：校验数据格式->用户名是否已经存在->邮箱是否已被注册=>校验验证码->加密密码并开启事务存储

	if msg := utils.CheckNickname(rgq.Nickname); msg != "" {
		return errors.New(msg)
	}
	if msg := utils.CheckPassword(rgq.Password); msg != "" {
		return errors.New(msg)
	}
	if msg := utils.CheckEmail(rgq.Email); msg != "" {
		return errors.New(msg)
	}
	if msg := utils.CheckCode(rgq.Code); msg != "" {
		return errors.New(msg)
	}

	isEmailExists, err := model.IsEmailExists(ctx, db, rgq.Email)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	if isEmailExists {
		return errors.New("邮箱已注册，请直接登录")
	}

	code, err := model.GetCode(ctx, rc, rgq.Email)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return errors.New("验证码不存在或已经过期，请重新获取")
		}
		return errors.New("系统繁忙，请稍后重试")
	}
	if code != rgq.Code {
		return errors.New("验证码错误")
	}

	encryptPwd, err := utils.EncryptPwd(rgq.Password)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	autoUid := utils.GenAutoUuid()

	qrPath := fmt.Sprintf("static/qr/%s.png", autoUid)
	if err = utils.GenUserQRCode(autoUid, qrPath, 512); err != nil {
		return errors.New("生成个人二维码失败")
	}

	// tx := db.Begin() // 手动开启事务，生成tx实例
	// 事务操作
	// if err := tx.Create(user).Error; err != nil {
	//	 tx.Rollback() // 出错回滚
	//	 return err
	// }
	// tx.Commit() // 提交事务

	// Transaction 相当于把上面的操作封装好了
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		avatarPath, bgPath := utils.GetRandomDefaultMedia()
		user := &model.User{
			UID:      autoUid,
			Nickname: rgq.Nickname,
			Password: encryptPwd,
			Email:    rgq.Email,
			Avatar:   avatarPath,
			BgImg:    bgPath,
		}
		if err = model.CreateUser(ctx, tx, user); err != nil {
			return err
		}
		if err = model.InitUserDefaultCategory(ctx, tx, user.UID); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	_ = model.DeleteCode(ctx, rc, rgq.Email)

	return nil
}
