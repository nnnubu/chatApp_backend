package message

import (
	"ChatApp/dto"
	"ChatApp/model"
	"context"
	"encoding/json"

	"gorm.io/gorm"
)

type PullUnReadMessageService struct{}

func NewPullUnReadMessageService() *PullUnReadMessageService {
	return &PullUnReadMessageService{}
}

func (service *PullUnReadMessageService) PullUnReadMessage(ctx context.Context, db *gorm.DB, currentUid string) (returnList []*dto.WebsocketChange, err error) {
	messageList, _ := model.PullUnReadMessage(ctx, db, currentUid)
	var uidList []string
	for _, v := range messageList {
		uidList = append(uidList, v.SenderUID)
	}
	userList, err := model.GetUserByList(ctx, db, uidList)
	if err != nil {
		return nil, err
	}
	userMap := make(map[string]model.User)
	for _, v := range userList {
		userMap[v.UID] = v
	}
	for _, message := range messageList {
		sender := userMap[message.SenderUID]
		hashData, err := json.Marshal(dto.ChatResp{
			Uid:             sender.UID,
			SenderUID:       message.SenderUID,
			ReceiverUID:     message.ReceiverUID,
			Nickname:        sender.Nickname,
			AvatarUrl:       sender.Avatar,
			ConversationUID: message.ConversationUID,
			Content:         message.Content,
			IsInsertToTop:   false,
		})
		if err != nil {
			return nil, err
		}
		returnList = append(returnList, &dto.WebsocketChange{
			MsgType: "chat",
			MsgId:   message.MsgID,
			Data:    hashData,
		})
	}
	return returnList, nil
}
