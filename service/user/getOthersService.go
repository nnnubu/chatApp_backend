package user

import (
	"ChatApp/config"
	"ChatApp/dto"
	"ChatApp/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type VisitOthersService struct{}

func NewGetOthersService() *VisitOthersService {
	return &VisitOthersService{}
}

func (gos *VisitOthersService) VisitOthers(ctx context.Context, db *gorm.DB, currentUid string, targetUid string) (*dto.GetOthersResponse, error) {
	user, isExist, err := model.GetUserByUID(ctx, db, targetUid)
	if err != nil {
		return nil, errors.New("系统繁忙，请稍后重试")
	}
	if !isExist {
		return nil, errors.New("该用户不存在")
	}

	isFriend, err := model.IsFriend(ctx, db, currentUid, targetUid)
	if err != nil {
		return nil, errors.New("系统繁忙，请稍后重试")
	}

	_, conversationUid, err := model.GetPrivateConversation(ctx, db, currentUid, targetUid)
	if err != nil {
		return nil, err
	}
	return &dto.GetOthersResponse{
		Uid:      user.UID,
		Nickname: user.Nickname,
		Avatar: dto.ImageResp{
			Url:    user.Avatar,
			ThumbW: config.Conf.ImageResize.AvatarW,
			ThumbH: config.Conf.ImageResize.AvatarH,
		},
		Intro: user.Intro,
		BgImg: dto.ImageResp{
			Url:    user.BgImg,
			ThumbW: config.Conf.ImageResize.BgW,
			ThumbH: config.Conf.ImageResize.BgH,
		},
		IsFriend:        isFriend,
		ConversationUid: conversationUid,
	}, nil
}
