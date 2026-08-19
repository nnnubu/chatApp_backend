package friend

import (
	"ChatApp/dto"
	"ChatApp/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type PullFriendsService struct {
}

func NewPullFriendsService() *PullFriendsService {
	return &PullFriendsService{}
}

func (pfs *PullFriendsService) PullFriends(ctx context.Context, db *gorm.DB, uid string, page int, pageSize int) ([]dto.PullFriendsResp, bool, error) {
	// 用户的好友列表
	friendsList, hasMore, err := model.PullFriendsByUid(ctx, db, uid, page, pageSize)
	if err != nil {
		return nil, false, errors.New("系统繁忙，请稍后再试")
	}

	// 好友数为 0
	if len(friendsList) == 0 {
		return []dto.PullFriendsResp{}, hasMore, nil
	}

	// 构建好友 uid 列表
	var uidList []string
	for _, v := range friendsList {
		uidList = append(uidList, v.FriendUid)
	}

	// 批量获取好友信息
	list, err := model.GetUserByList(ctx, db, uidList)
	if err != nil {
		return nil, false, errors.New("系统繁忙，请稍后再试")
	}

	// 将好友信息映射到表中
	userMap := make(map[string]model.User, len(list))
	for _, u := range list {
		userMap[u.UID] = u
	}

	// 批量获取好友信息以及在线状态
	var result []dto.PullFriendsResp
	for _, v := range friendsList {
		user, ok := userMap[v.FriendUid]
		if !ok {
			//用户记录存在与否 有可能B是A的好友，但是A在查询时，B已经注销
			continue
		}
		result = append(result, dto.PullFriendsResp{
			Uid:       user.UID,
			Nickname:  user.Nickname,
			AvatarUrl: user.Avatar,
			IsOnline:  false,
			// 在线状态暂时先用 false 代替
		})
	}
	return result, hasMore, nil
}
