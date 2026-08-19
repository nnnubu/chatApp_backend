package friend

import (
	"ChatApp/dto"
	"ChatApp/global"
	"ChatApp/model"
	"ChatApp/socket/service"
	"ChatApp/utils"
	"context"
	"errors"

	"gorm.io/gorm"
)

type AddFriendService struct{}

func NewAddFriendService() *AddFriendService {
	return &AddFriendService{}
}

func (afs *AddFriendService) AddFriend(ctx context.Context, db *gorm.DB, afq *dto.AddFriendRequest, currentUid string) error {
	if afq.TargetUid == currentUid {
		return errors.New("您已经是自己的好友")
	}

	if msg := utils.CheckVerifyMsg(afq.Msg); msg != "" {
		return errors.New(msg)
	}

	targetUser, isTargetUserExist, err := model.GetUserByUID(ctx, db, afq.TargetUid)

	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	if !isTargetUserExist {
		return errors.New("该用户不存在")
	}

	// 先判断是否已经是好友，再判断好友申请重复与否
	isFriend, err := model.IsFriend(ctx, db, currentUid, afq.TargetUid)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	if isFriend {
		return errors.New("你们已经是好友")
	}

	hasPending, err := model.FriendApplyHasPending(ctx, db, currentUid, afq.TargetUid)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	if hasPending {
		return errors.New("已发送待处理好友申请，请勿重复提交")
	}

	isTargetPending, err := model.IsTargetPending(ctx, db, currentUid, afq.TargetUid)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	if isTargetPending {
		return errors.New("对方已向您发送好友请求，请先处理")
	}

	msgId := utils.GenAutoSnowId()
	if err = model.CreateFriendApply(ctx, db, &model.FriendApply{
		MsgId:     msgId,
		ApplyUid:  currentUid,
		TargetUid: afq.TargetUid,
		Msg:       afq.Msg,
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("请勿重复发送好友申请")
		}
		return err
	}

	currentUser, isCurrentUidExist, err := model.GetUserByUID(ctx, db, currentUid)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}
	if !isCurrentUidExist {
		return errors.New("当前用户不存在")
	}

	// 数据库执行完毕则返回

	// 开启异步推送协程 防止阻塞入库函数
	go func() {
		// 将当前发起请求的用户的部分信息渲染给目标用户
		if err = service.PushBroadCastMsg(afq.TargetUid, "addFriend", msgId, &dto.AddFriendResponse{
			Uid:       currentUser.UID,
			ApplyUid:  currentUser.UID,
			TargetUid: targetUser.UID,
			Nickname:  currentUser.Nickname,
			Content:   afq.Msg,
			AvatarUrl: currentUser.Avatar,
		}); err != nil {
			global.Log.Error(err.Error())
		}

		// 将当前用户自己发送的消息渲染到当前用户自己的设备
		if err = service.PushBroadCastMsg(currentUid, "addFriend", msgId, &dto.AddFriendResponse{
			Uid:       targetUser.UID,
			ApplyUid:  currentUser.UID,
			TargetUid: targetUser.UID,
			Nickname:  targetUser.Nickname,
			Content:   afq.Msg,
			AvatarUrl: targetUser.Avatar,
		}); err != nil {
			global.Log.Error(err.Error())
		}

		// 上述操作不可再做了，因为前端目前没有区分自己发送的和对面发送的，不如等待对方同意之后再渲染
		// 想了一下还是需要渲染的，不然用户发出去可能会忘了自己有没有添加好友
		// 另外后续还需要通过状态的更新来区分双方的读取与否，这个时候就需要从原来的只有一个uid字段更新为发起者和接收者的uid字段
	}()

	return nil
}
