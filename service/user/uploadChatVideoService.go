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
// 处理链路：
//  1. 保存上传原视频；
//  2. ffmpeg 转码压缩（最长边 1280、libx264 CRF28、faststart 前置 moov），
//     转码产物更小才替换原文件，超时/失败/转码后更大则保留原文件降级；
//  3. 对最终视频抽帧生成缩略图（uuid_thumb.jpg），失败降级 thumbUrl 为空，
//     由前端回退到旧逻辑（直接用视频首帧）。
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

	// ===== 抽帧生成缩略图（从原文件抽，同步执行，失败不阻断上传） =====
	thumbUrl := ""
	thumbName := fmt.Sprintf("%s_thumb.jpg", fileUid)
	thumbPath := filepath.Join(dir, thumbName)
	if err := s.generateThumb(ctx, savePath, thumbPath); err != nil {
		log.Printf("视频缩略图生成失败, 降级处理: %v", err)
	} else {
		thumbUrl = fmt.Sprintf("/static/%s/%s/%s", uploadType, dateDir, thumbName)
	}

	// ===== 转码压缩（异步后台执行，不阻塞上传） =====
	// 服务器内存仅 1.6G、chatapp 容器限 700m，同步转码 1080p 视频会被 OOM 杀掉；
	// 异步 + 全局并发限制为 1：上传立即返回，转码排队逐个后台跑，
	// 转码完成自动替换为压缩版（url 不变，前端无感），失败保留原文件。
	transcodeSem <- struct{}{}
	go func() {
		defer func() { <-transcodeSem }()
		if err := s.transcode(context.Background(), savePath); err != nil {
			log.Printf("视频转码压缩失败, 使用原文件: %v", err)
		}
	}()

	return url, thumbUrl, nil
}

// transcodeSem 转码并发信号量：同时只允许一个 ffmpeg 转码任务，
// 避免多个视频同时转码导致容器内存超限被 OOM 杀掉。
var transcodeSem = make(chan struct{}, 1)

// transcode 用 ffmpeg 转码压缩视频：
//   - 最长边缩到 1280（约 720p 级别，手机播放足够清晰）
//   - H.264 veryfast + CRF28（体积/画质平衡）
//   - 音频 aac 96k、30fps 限帧、单线程（降低内存峰值，适配低配服务器）
//   - faststart：把 moov 原子前置，网络播放起播/拖动更快
//
// 转码产物体积小于原文件才替换，否则保留原文件（避免画质换体积）；
// 异步执行 + 10 分钟超时（低配服务器转码慢），失败直接返回，由调用方降级。
func (s *UploadChatVideoService) transcode(ctx context.Context, videoPath string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	tmpPath := videoPath + ".transcode.tmp.mp4"
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",   // 覆盖已存在文件
		"-i", videoPath, // 输入
		// 限制最长边 1280（横屏限宽、竖屏限高），避免高分辨率全量转码
		// fps=30：60fps 等高帧率视频减帧，解码/编码工作量减半，降低内存与耗时
		"-vf", "scale=if(gt(iw\\,ih)\\,min(1280\\,iw)\\,-2):if(gt(iw\\,ih)\\,-2\\,min(1280\\,ih)),fps=30",
		// 单线程：低配服务器 + 降低内存峰值，避免容器 OOM 被杀
		"-threads", "1",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "28",
		"-c:a", "aac",
		"-b:a", "96k",
		"-movflags", "+faststart",
		tmpPath, // 输出到临时文件，转码成功再替换
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("ffmpeg 转码失败: %v, output: %s", err, string(out))
	}

	origStat, err1 := os.Stat(videoPath)
	tmpStat, err2 := os.Stat(tmpPath)
	if err1 != nil || err2 != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("转码产物检查失败: %v %v", err1, err2)
	}

	// 转码后没变小：保留原文件，避免画质损失换更大体积
	if tmpStat.Size() >= origStat.Size() {
		_ = os.Remove(tmpPath)
		return nil
	}

	if err := os.Rename(tmpPath, videoPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// generateThumb 用 ffmpeg 抽取视频第 0.1 秒一帧保存为 jpg
// 带超时控制，避免 ffmpeg 卡死阻塞上传请求
func (s *UploadChatVideoService) generateThumb(ctx context.Context, videoPath, thumbPath string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",            // 覆盖已存在文件
		"-ss", "0.1",    // 取第 0.1 秒（短于 0.1s 的视频取第一帧）
		"-i", videoPath, // 输入
		"-vframes", "1", // 只抽一帧
		"-q:v", "2", // jpg 质量（2 高质量）
		thumbPath, // 输出
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
