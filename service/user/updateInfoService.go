package user

import (
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/utils"
	"context"
	"errors"

	"gorm.io/gorm"
)

type UpdateInfoService struct{}

func NewUpdateInfoService() *UpdateInfoService {
	return &UpdateInfoService{}
}

func (uis *UpdateInfoService) UpdateInfo(ctx context.Context, db *gorm.DB, uiq *dto.UpdateInfoRequest, uid string) error {
	updateMap := make(map[string]any)
	// 局部更新，只更新需要修改的部分
	if uiq.Nickname != nil {
		nickname := *uiq.Nickname
		updateMap["nickname"] = nickname
		if msg := utils.CheckNickname(nickname); msg != "" {
			return errors.New(msg)
		}
	}

	if uiq.Intro != nil {
		Intro := *uiq.Intro
		updateMap["Intro"] = Intro
		if msg := utils.CheckIntro(Intro); msg != "" {
			return errors.New(msg)
		}
	}

	if uiq.Gender != nil {
		Gender := *uiq.Gender
		updateMap["Gender"] = Gender
		if msg := utils.CheckGender(Gender); msg != "" {
			return errors.New(msg)
		}
	}

	if uiq.Birthday != nil {
		Birthday := *uiq.Birthday
		updateMap["Birthday"] = Birthday
		if msg := utils.CheckBirthday(Birthday); msg != "" {
			return errors.New(msg)
		}
	}

	if len(updateMap) == 0 {
		return errors.New("您未修改任何资料")
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := model.UpdateInfo(ctx, tx, updateMap, uid); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	return nil
}
