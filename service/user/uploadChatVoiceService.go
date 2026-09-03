package user

import (
	"ChatApp/config"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UploadChatVoiceService 聊天语音上传服务
// 与聊天图片一致：只落盘不写库，路径规则 static/chatVoice/日期/uuid.ext
// 返回语音URL，消息 content JSON 由前端组装（url + duration）
type UploadChatVoiceService struct {
	basePath string
}

func NewUploadChatVoiceService() *UploadChatVoiceService {
	return &UploadChatVoiceService{
		basePath: config.Conf.App.BasePath,
	}
}

func (s *UploadChatVoiceService) UploadChatVoice(ctx context.Context, db *gorm.DB, file *multipart.FileHeader, uid string) (string, error) {
	uploadType := "chatVoice"
	dateDir := time.Now().Format("20060102")
	dir := fmt.Sprintf("%s/%s/%s", s.basePath, uploadType, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".m4a"
	}
	// 限制扩展名，避免恶意后缀
	switch ext {
	case ".m4a", ".aac", ".mp3", ".wav", ".amr", ".ogg", ".opus":
	default:
		ext = ".m4a"
	}
	fileUid := uuid.NewString()
	fileName := fmt.Sprintf("%s%s", fileUid, ext)
	savePath := filepath.Join(dir, fileName)

	srcFile, err := file.Open()
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(savePath)
	if err != nil {
		return "", err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return "", err
	}

	url := fmt.Sprintf("/static/%s/%s/%s", uploadType, dateDir, fileName)
	return url, nil
}
