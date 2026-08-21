package message

import (
	"ChatApp/dto"
	"ChatApp/model"
	"context"
	"encoding/json"

	"gorm.io/gorm"
)

type PullHistoryMessageService struct{}

func NewPullHistoryMessageService() *PullHistoryMessageService {
	return &PullHistoryMessageService{}
}

func (pms *PullHistoryMessageService) PullHistoryMessage(ctx context.Context, db *gorm.DB, currentUid string, pageSize int, cursorMsgId string, conversationUid string) (returnList []*dto.WebsocketChange, hasMore bool, err error) {
	// 查询时校验是否是私聊会话以及当前用户是否属于当前会话
	_, err = model.GetPrivateConversationByUID(ctx, db, currentUid, conversationUid)
	if err != nil {
		return nil, false, err
	}

	uidList, err := model.GetConversationMember(ctx, db, conversationUid)
	if err != nil {
		return nil, false, err
	}

	userList, err := model.GetUserByList(ctx, db, uidList)
	if err != nil {
		return nil, false, err
	}
	// 私聊会话若 没有对应的聊天成员 或聊天成员 大于 2
	if len(userList) == 0 || len(uidList) > 2 {
		return nil, false, nil
	}

	userMap := make(map[string]model.User)
	//peerUser := userList[1]
	for _, v := range userList {
		userMap[v.UID] = v
	}

	messageList, hasMore, err := model.PullHistoryMessage(ctx, db, pageSize, cursorMsgId, conversationUid)
	if err != nil {
		return nil, false, err
	}

	for _, message := range messageList {
		sender := userMap[message.SenderUID]
		hashData, err := json.Marshal(dto.ChatResp{
			Uid:             sender.UID,
			SenderUID:       sender.UID,
			ReceiverUID:     message.ReceiverUID,
			Nickname:        sender.Nickname,
			AvatarUrl:       sender.Avatar,
			ConversationUID: message.ConversationUID,
			Content:         message.Content,
			IsInsertToTop:   true,
		})
		if err != nil {
			return nil, false, err
		}
		returnList = append(returnList, &dto.WebsocketChange{
			MsgType: "chat",
			MsgId:   message.MsgID,
			Data:    hashData,
		})
	}
	return returnList, hasMore, nil
}
