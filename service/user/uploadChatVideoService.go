package user

import (
	"ChatApp/config"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UploadChatVideoService 聊天视频上传服务
// 与聊天图片/语音一致：只落盘不写库，路径规则 static/chatVideo/日期/uuid.ext
// 视频保存后调用 ffmpeg 抽帧生成缩略图（同目录 uuid_thumb.jpg），
// 返回视频URL 与缩略图URL；ffmpeg 缺失/抽帧失败时降级（thumbUrl 为空），
// 由前端回退到旧逻辑（直接用视频首帧）。
type UploadChatVideoService struct {
	basePath string
}

func NewUploadChatVideoService() *UploadChatVideoService {
	return &UploadChatVideoService{
		basePath: config.Conf.App.BasePath,
	}
}

func (s *UploadChatVideoService) UploadChatVideo(ctx context.Context, db *gorm.DB, file *multipart.FileHeader, uid string) (string, string, error) {
	uploadType := "chatVideo"
	dateDir := time.Now().Format("20060102")
	dir := fmt.Sprintf("%s/%s/%s", s.basePath, uploadType, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
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
		return "", "", err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(savePath)
	if err != nil {
		return "", "", err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return "", "", err
	}

	url := fmt.Sprintf("/static/%s/%s/%s", uploadType, dateDir, fileName)

	// 抽帧生成缩略图（失败不阻断上传，前端回退旧逻辑）
	thumbUrl := ""
	thumbName := fmt.Sprintf("%s_thumb.jpg", fileUid)
	thumbPath := filepath.Join(dir, thumbName)
	if err := s.generateThumb(ctx, savePath, thumbPath); err != nil {
		log.Printf("视频缩略图生成失败, 降级处理: %v", err)
	} else {
		thumbUrl = fmt.Sprintf("/static/%s/%s/%s", uploadType, dateDir, thumbName)
	}

	return url, thumbUrl, nil
}

// generateThumb 用 ffmpeg 抽取视频第 0.1 秒一帧保存为 jpg
// 带超时控制，避免 ffmpeg 卡死阻塞上传请求
func (s *UploadChatVideoService) generateThumb(ctx context.Context, videoPath, thumbPath string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",              // 覆盖已存在文件
		"-ss", "0.1",      // 取第 0.1 秒（短于 0.1s 的视频取第一帧）
		"-i", videoPath,   // 输入
		"-vframes", "1",   // 只抽一帧
		"-q:v", "2",       // jpg 质量（2 高质量）
		thumbPath,         // 输出
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg 抽帧失败: %v, output: %s", err, string(out))
	}
	// 校验输出文件确实生成
	if _, err := os.Stat(thumbPath); err != nil {
		return fmt.Errorf("ffmpeg 输出文件不存在: %v", err)
	}
	return nil
}
