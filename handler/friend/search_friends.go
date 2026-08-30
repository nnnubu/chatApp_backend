package friend

import (
	"ChatApp/model"
	"ChatApp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SearchFriendsHandler 搜索好友（按昵称或邮箱）
func SearchFriendsHandler(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    400,
			Message: "关键词不能为空",
		})
		return
	}

	uidAny, hasUid := c.Get("uid")
	if !hasUid {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    401,
			Message: "登录态缺失，请重新登录",
		})
		return
	}
	currentUid, ok := uidAny.(string)
	if !ok {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    401,
			Message: "登录信息错误，请重新登录",
		})
		return
	}

	reqCtx, db, _, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	// 搜索用户（排除自己，最多20条）
	users, err := model.SearchUsers(reqCtx, db, keyword, currentUid, 20)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: "搜索失败",
		})
		return
	}

	// 查询当前用户的好友列表，用于标记是否已是好友
	friendList, _, err := model.PullFriendsByUid(reqCtx, db, currentUid, 1, 1000)
	if err != nil {
		friendList = nil
	}

	// 构建好友 UID 集合
	friendUidSet := make(map[string]bool)
	if friendList != nil {
		for _, f := range friendList {
			friendUidSet[f.FriendUid] = true
		}
	}

	// 构建返回结果
	type SearchResult struct {
		UID      string `json:"uid"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Intro    string `json:"intro"`
		Email    string `json:"email"`
		IsFriend bool   `json:"isFriend"`
	}

	results := make([]SearchResult, 0, len(users))
	for _, u := range users {
		results = append(results, SearchResult{
			UID:      u.UID,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Intro:    u.Intro,
			Email:    u.Email,
			IsFriend: friendUidSet[u.UID],
		})
	}

	c.JSON(http.StatusOK, model.CommonResp{
		Code:    200,
		Message: "success",
		Data: map[string]any{
			"list":  results,
			"total": len(results),
		},
	})
}
