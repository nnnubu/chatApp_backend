package user

import (
	"ChatApp/config"
	"ChatApp/dto"
	"ChatApp/model"
	"ChatApp/socket/service"
	"ChatApp/utils"
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

	exist, conversationUid, err := model.GetPrivateConversation(ctx, db, currentUid, targetUid)
	if err != nil {
		return nil, err
	}

	// AI 用户：会话不存在时自动创建
	if !exist && service.IsAIUser(targetUid) {
		conversationUid = utils.GenAutoSnowId()
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&model.Conversation{
				ConversationUID:  conversationUid,
				ConversationType: model.MsgTypePrivateChat,
			}).Error; err != nil {
				return err
			}
			members := []model.ConversationMember{
				{ConversationUID: conversationUid, UID: currentUid},
				{ConversationUID: conversationUid, UID: targetUid},
			}
			return tx.Create(&members).Error
		})
		if err != nil {
			return nil, errors.New("系统繁忙，请稍后重试")
		}
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
