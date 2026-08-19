package errorConst

import "errors"

var ErrUserOffline = errors.New("用户无任何在线连接")
var ErrSendFailed = errors.New("消息推送失败，管道已满或连接已下线")
var ErrConnOffline = errors.New("该连接不存在，已下线")
