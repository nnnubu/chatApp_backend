package user

import (
	"ChatApp/config"
	"ChatApp/model"
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

// UploadStickerService 用户自定义表情包上传服务
// 落盘 static/userSticker/日期/uuid.ext，同时写入 user_sticker 表
// 表情包发送时直接引用 url，无需每次上传
type UploadStickerService struct {
	basePath string
}

func NewUploadStickerService() *UploadStickerService {
	return &UploadStickerService{
		basePath: config.Conf.App.BasePath,
	}
}

func (s *UploadStickerService) UploadSticker(ctx context.Context, db *gorm.DB, file *multipart.FileHeader, uid string) (*model.UserSticker, error) {
	uploadType := "userSticker"
	dateDir := time.Now().Format("20060102")
	dir := fmt.Sprintf("%s/%s/%s", s.basePath, uploadType, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".png"
	}
	// 限制扩展名，表情包只支持图片格式（含 GIF 动图）
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
	default:
		ext = ".png"
	}
	stickerId := uuid.NewString()
	fileName := fmt.Sprintf("%s%s", stickerId, ext)
	savePath := filepath.Join(dir, fileName)

	srcFile, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(savePath)
	if err != nil {
		return nil, err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("/static/%s/%s/%s", uploadType, dateDir, fileName)
	sticker := &model.UserSticker{
		ID:  stickerId,
		UID: uid,
		URL: url,
	}
	if err := model.CreateUserSticker(ctx, db, sticker); err != nil {
		return nil, err
	}
	return sticker, nil
}
