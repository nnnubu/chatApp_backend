package user

import (
	"ChatApp/config"
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/utils"
	"context"
	"errors"

	"gorm.io/gorm"
)

type LoginService struct{}

func NewLoginService() *LoginService {
	return &LoginService{}
}

func (ls *LoginService) Login(ctx context.Context, db *gorm.DB, lgq *dto.LoginRequest) (*dto.LoginResponse, error) {
	// 校验数据->从数据库中读取用户信息并验证->设置jwt令牌

	if msg := utils.CheckEmail(lgq.Email); msg != "" {
		return nil, errors.New(msg)
	}
	if msg := utils.CheckPassword(lgq.Password); msg != "" {
		return nil, errors.New(msg)
	}

	user, isUserExist, err := model.GetUserByEmail(ctx, db, lgq.Email)
	if err != nil {
		return nil, errors.New("系统繁忙，请稍后重试")
	}

	if !isUserExist {
		return nil, errors.New("该邮箱未注册")
	}

	if !utils.VerifyPwd(lgq.Password, user.Password) {
		return nil, errors.New("邮箱或密码错误")
	}

	jwtToken, err := utils.GenJwtToken(user.UID)
	if err != nil {
		return nil, errors.New("生成登录凭证失败")
	}

	return &dto.LoginResponse{
		Token:    jwtToken,
		Uid:      user.UID,
		NickName: user.Nickname,
		Avatar: dto.ImageResp{
			Url:    user.Avatar,
			ThumbW: config.Conf.ImageResize.AvatarW,
			ThumbH: config.Conf.ImageResize.AvatarH,
		},
		BgImg: dto.ImageResp{
			Url:    user.BgImg,
			ThumbW: config.Conf.ImageResize.BgW,
			ThumbH: config.Conf.ImageResize.BgH,
		},
		Gender:   user.Gender,
		Intro:    user.Intro,
		Birthday: user.Birthday.Format("2006-01-02"),
	}, nil
}
