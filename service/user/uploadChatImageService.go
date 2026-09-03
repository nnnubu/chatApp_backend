package user

import (
	"ChatApp/config"
	"context"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UploadChatImageService 聊天图片上传服务
// 与原头像/背景图上传不同：头像会异步更新用户字段，聊天图只落盘不写库
// 路径规则：static/chatImg/origin|thumb/日期/uuid.ext
type UploadChatImageService struct {
	basePath string
	// 缩略图最长边限制
	maxThumbEdge int
}

func NewUploadChatImageService() *UploadChatImageService {
	return &UploadChatImageService{
		basePath:     config.Conf.App.BasePath,
		maxThumbEdge: 800,
	}
}

func (s *UploadChatImageService) UploadChatImage(ctx context.Context, db *gorm.DB, file *multipart.FileHeader, uid string) (string, string, int, int, error) {
	uploadType := "chatImg"
	dateDir := time.Now().Format("20060102")
	originDir := fmt.Sprintf("%s/%s/origin/%s", s.basePath, uploadType, dateDir)
	thumbDir := fmt.Sprintf("%s/%s/thumb/%s", s.basePath, uploadType, dateDir)
	if err := os.MkdirAll(originDir, 0755); err != nil {
		return "", "", 0, 0, err
	}
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return "", "", 0, 0, err
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	fileUid := uuid.NewString()
	originFileName := fmt.Sprintf("%s%s", fileUid, ext)
	thumbFileName := fmt.Sprintf("%s%s", fileUid, ext)
	originSavePath := filepath.Join(originDir, originFileName)

	srcFile, err := file.Open()
	if err != nil {
		return "", "", 0, 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(originSavePath)
	if err != nil {
		return "", "", 0, 0, err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return "", "", 0, 0, err
	}

	// 解码原图获取真实宽高
	srcImg, openErr := imaging.Open(originSavePath)
	if openErr != nil {
		return "", "", 0, 0, openErr
	}
	originBounds := srcImg.Bounds()
	width := originBounds.Dx()
	height := originBounds.Dy()

	// 生成缩略图：等比例缩放 限制最长边 保持原图比例 不裁剪
	var thumbImg *image.NRGBA
	w, h := width, height
	if width > s.maxThumbEdge || height > s.maxThumbEdge {
		if width >= height {
			w = s.maxThumbEdge
			h = int(float64(height) * float64(s.maxThumbEdge) / float64(width))
		} else {
			h = s.maxThumbEdge
			w = int(float64(width) * float64(s.maxThumbEdge) / float64(height))
		}
		thumbImg = imaging.Resize(srcImg, w, h, imaging.Lanczos)
	} else {
		// 原图小于缩略图阈值 直接拷贝原图当缩略图 避免无谓的重新编码
		thumbImg = imaging.Clone(srcImg)
	}

	thumbSavePath := filepath.Join(thumbDir, thumbFileName)
	if err = imaging.Save(thumbImg, thumbSavePath, imaging.JPEGQuality(85)); err != nil {
		return "", "", 0, 0, err
	}

	originUrl := fmt.Sprintf("/static/%s/origin/%s/%s", uploadType, dateDir, originFileName)
	thumbUrl := fmt.Sprintf("/static/%s/thumb/%s/%s", uploadType, dateDir, thumbFileName)
	return originUrl, thumbUrl, width, height, nil
}
