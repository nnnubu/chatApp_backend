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

// aiCharacterList AI 角色列表（与 main.go 中初始化的保持一致）
var aiCharacterList = []struct {
	UID    string
	Name   string
	Avatar string
}{
	{"ai_debug_001", "小助手", "/static/default/avatar/df_avatar1.jpg"},
}

// ensureAIFriend 确保用户与所有 AI 角色建立好友关系，不存在则自动创建
func ensureAIFriend(ctx context.Context, db *gorm.DB, uid string) {
	for _, ai := range aiCharacterList {
		isFriend, err := model.IsFriend(ctx, db, uid, ai.UID)
		if err != nil || isFriend {
			continue
		}
		// 自动创建双向好友关系（忽略错误，不阻塞好友列表返回）
		_ = db.Transaction(func(tx *gorm.DB) error {
			friends := []model.Friend{
				{Uid: uid, FriendUid: ai.UID},
				{Uid: ai.UID, FriendUid: uid},
			}
			return tx.Create(&friends).Error
		})
	}
}

func (pfs *PullFriendsService) PullFriends(ctx context.Context, db *gorm.DB, uid string, page int, pageSize int) ([]dto.PullFriendsResp, bool, error) {
	// 确保 AI 好友关系存在（首次拉取时自动建立）
	ensureAIFriend(ctx, db, uid)

	// 用户的好友列表
	friendsList, hasMore, err := model.PullFriendsByUid(ctx, db, uid, page, pageSize)
	if err != nil {
		return nil, false, errors.New("系统繁忙，请稍后再试")
	}

	var result []dto.PullFriendsResp

	if len(friendsList) > 0 {
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
	}

	// 将所有 AI 角色插入好友列表最前面（AI 永远在线）
	var aiFriends []dto.PullFriendsResp
	for _, ai := range aiCharacterList {
		aiFriends = append(aiFriends, dto.PullFriendsResp{
			Uid:       ai.UID,
			Nickname:  ai.Name,
			AvatarUrl: ai.Avatar,
			IsOnline:  true,
		})
	}
	result = append(aiFriends, result...)

	return result, hasMore, nil
}
