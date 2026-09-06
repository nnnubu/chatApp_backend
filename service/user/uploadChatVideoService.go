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

// UploadChatVideoService 聊天视频上传服务
// 与聊天图片/语音一致：只落盘不写库，路径规则 static/chatVideo/日期/uuid.ext
// 返回视频URL，消息 content JSON 由前端组装（url）
type UploadChatVideoService struct {
	basePath string
}

func NewUploadChatVideoService() *UploadChatVideoService {
	return &UploadChatVideoService{
		basePath: config.Conf.App.BasePath,
	}
}

func (s *UploadChatVideoService) UploadChatVideo(ctx context.Context, db *gorm.DB, file *multipart.FileHeader, uid string) (string, error) {
	uploadType := "chatVideo"
	dateDir := time.Now().Format("20060102")
	dir := fmt.Sprintf("%s/%s/%s", s.basePath, uploadType, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".mp4"
	}
	// 限制扩展名，避免恶意后缀
	switch ext {
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".3gp", ".m4v":
	default:
		ext = ".mp4"
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
