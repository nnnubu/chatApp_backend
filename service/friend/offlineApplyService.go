package friend

import (
	"ChatApp/dto"
	"ChatApp/model"
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

type OfflineApplyService struct{}

func NewOfflineApplyService() *OfflineApplyService {
	return &OfflineApplyService{}
}

func (oas *OfflineApplyService) PullOfflineApply(ctx context.Context, db *gorm.DB, currentUid string) (ApplyList []dto.WebsocketChange, err error) {
	tempList1, err := model.GetUnReadApplyByUid(ctx, db, currentUid)
	if err != nil {
		return nil, err
	}

	tempList2, err := model.GetPendingApplyByUid(ctx, db, currentUid)
	if err != nil {
		return nil, err
	}

	// 收集全部需要查询的uid map去重
	uidSet := make(map[string]struct{})
	for _, v := range tempList1 {
		uidSet[v.TargetUid] = struct{}{}
	}
	for _, v := range tempList2 {
		uidSet[v.ApplyUid] = struct{}{}
	}
	// 集合转列表
	var needUids []string
	for uid := range uidSet {
		needUids = append(needUids, uid)
	}
	// 批量查用户
	var userList []model.User
	if len(needUids) > 0 {
		userList, err = model.GetUserByList(ctx, db, needUids)
		if err != nil {
			return nil, errors.New("系统繁忙，请稍后重试")
		}
	}

	// 构建uid → user内存映射
	userMap := make(map[string]*model.User, len(userList))
	for i := range userList {
		userMap[userList[i].UID] = &userList[i]
	}

	for _, v := range tempList1 {
		targetUser, isTargetUserExist := userMap[v.TargetUid]

		if !isTargetUserExist {
			continue
		}
		data, err := json.Marshal(dto.AddFriendResponse{
			Uid:       v.TargetUid,
			ApplyUid:  v.ApplyUid,
			TargetUid: targetUser.UID,
			Nickname:  targetUser.Nickname,
			Content:   v.Msg,
			AvatarUrl: targetUser.Avatar,
			Status:    v.Status,
		})
		if err != nil {
			return nil, err
		}
		ApplyList = append(ApplyList, dto.WebsocketChange{
			MsgType: "addFriend",
			MsgId:   v.MsgId,
			Data:    data,
		})
	}

	for _, v := range tempList2 {
		applyUser, isApplyUserExist := userMap[v.ApplyUid]
		if !isApplyUserExist {
			return nil, errors.New("该用户不存在")
		}
		data, err := json.Marshal(dto.AddFriendResponse{
			Uid:       v.ApplyUid,
			ApplyUid:  applyUser.UID,
			TargetUid: v.TargetUid,
			Nickname:  applyUser.Nickname,
			Content:   v.Msg,
			AvatarUrl: applyUser.Avatar,
			Status:    0,
		})
		if err != nil {
			return nil, err
		}
		ApplyList = append(ApplyList, dto.WebsocketChange{
			MsgType: "addFriend",
			MsgId:   v.MsgId,
			Data:    data,
		})
	}

	return ApplyList, nil
}
