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
// 压缩策略（针对手机拍摄大图，避免原图几 MB 直传导致加载/上传慢）：
//   - origin 原图最长边超过 maxOriginEdge(1920) 时等比压缩并统一转存为 jpg（质量85）
//   - thumb 缩略图最长边限制为 maxThumbEdge(480)，聊天气泡最大约 220px，足够清晰且体积小
type UploadChatImageService struct {
	basePath      string
	maxThumbEdge  int
	maxOriginEdge int
}

func NewUploadChatImageService() *UploadChatImageService {
	return &UploadChatImageService{
		basePath:      config.Conf.App.BasePath,
		maxThumbEdge:  480,
		maxOriginEdge: 1920,
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

	// ===== 原图压缩：最长边超过阈值时等比缩放并转存 jpg，减小存储与预览加载体积 =====
	originFileName = fmt.Sprintf("%s.jpg", fileUid)
	originUrl := fmt.Sprintf("/static/%s/origin/%s/%s", uploadType, dateDir, originFileName)
	if width > s.maxOriginEdge || height > s.maxOriginEdge {
		w, h := scaleKeepRatio(width, height, s.maxOriginEdge)
		resized := imaging.Resize(srcImg, w, h, imaging.Lanczos)
		compressedPath := filepath.Join(originDir, originFileName)
		if err = imaging.Save(resized, compressedPath, imaging.JPEGQuality(85)); err != nil {
			return "", "", 0, 0, err
		}
		// 删除未压缩的原文件，避免同目录残留两份
		if originSavePath != compressedPath {
			_ = os.Remove(originSavePath)
		}
		width, height = w, h
	} else {
		// 原图较小无需压缩，但非 jpg 后缀统一重命名规则下直接保留原文件
		if originFileName != filepath.Base(originSavePath) {
			keepPath := filepath.Join(originDir, originFileName)
			if err = os.Rename(originSavePath, keepPath); err != nil {
				return "", "", 0, 0, err
			}
		}
	}

	// ===== 生成缩略图：等比例缩放 限制最长边 保持原图比例 不裁剪 =====
	var thumbImg *image.NRGBA
	w, h := width, height
	if width > s.maxThumbEdge || height > s.maxThumbEdge {
		w, h = scaleKeepRatio(width, height, s.maxThumbEdge)
		thumbImg = imaging.Resize(srcImg, w, h, imaging.Lanczos)
	} else {
		// 原图小于缩略图阈值 直接拷贝原图当缩略图 避免无谓的重新编码
		thumbImg = imaging.Clone(srcImg)
	}

	thumbFileName = fmt.Sprintf("%s.jpg", fileUid)
	thumbSavePath := filepath.Join(thumbDir, thumbFileName)
	if err = imaging.Save(thumbImg, thumbSavePath, imaging.JPEGQuality(80)); err != nil {
		return "", "", 0, 0, err
	}

	thumbUrl := fmt.Sprintf("/static/%s/thumb/%s/%s", uploadType, dateDir, thumbFileName)
	return originUrl, thumbUrl, width, height, nil
}

// scaleKeepRatio 等比缩放到最长边为 maxEdge，返回新的宽高
func scaleKeepRatio(width, height, maxEdge int) (int, int) {
	if width >= height {
		return maxEdge, int(float64(height) * float64(maxEdge) / float64(width))
	}
	return int(float64(width) * float64(maxEdge) / float64(height)), maxEdge
}
