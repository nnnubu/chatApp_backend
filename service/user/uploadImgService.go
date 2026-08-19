package user

import (
	"ChatApp/config"
	"ChatApp/model"
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

type UploadImgService struct {
	basePath     string
	avatarHeight int
	avatarWidth  int
	bgHeight     int
	bgWidth      int
}

func NewUploadImgService() *UploadImgService {
	basePath := config.Conf.App.BasePath
	return &UploadImgService{
		basePath:     basePath,
		avatarHeight: config.Conf.ImageResize.AvatarH,
		avatarWidth:  config.Conf.ImageResize.AvatarW,
		bgHeight:     config.Conf.ImageResize.BgH,
		bgWidth:      config.Conf.ImageResize.BgW,
	}
}

func (ups *UploadImgService) UploadImg(ctx context.Context, db *gorm.DB, uploadType string, file *multipart.FileHeader, uid string) (*map[string]int, string, string, error) {
	// 以日期为最后文件夹，方便以后写脚本清理指定日期以前的图片存储
	// 同时避免单文件夹几万张图片导致文件管理器卡顿、磁盘索引变慢；
	dateDir := time.Now().Format("20060102")
	originDir := fmt.Sprintf("%s/%s/origin/%s", ups.basePath, uploadType, dateDir)
	thumbDir := fmt.Sprintf("%s/%s/thumb/%s", ups.basePath, uploadType, dateDir)
	if err := os.MkdirAll(originDir, 0755); err != nil {
		return nil, "", "", err
	}
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return nil, "", "", err
	}
	ext := filepath.Ext(file.Filename) // 提取后缀 .jpg/.png
	fileUid := uuid.NewString()        // UUID全局唯一文件名
	// 原图、缩略图共用同一个 UUID，方便关联查找
	originFileName := fmt.Sprintf("%s%s", fileUid, ext)
	thumbFileName := fmt.Sprintf("%s%s", fileUid, ext)

	originSavePath := filepath.Join(originDir, originFileName)

	srcFile, err := file.Open()
	// 打开前端上传的文件二进制流
	if err != nil {
		return nil, "", "", err
	}
	// 函数结束自动关闭文件句柄
	defer srcFile.Close()

	// 创建本地磁盘文件
	dstFile, err := os.Create(originSavePath)
	if err != nil {
		return nil, "", "", err
	}
	defer dstFile.Close()

	// 把前端上传的二进制拷贝到本地文件
	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return nil, "", "", err
	}

	srcImg, openErr := imaging.Open(originSavePath)
	if openErr != nil {
		return nil, "", "", openErr
	}

	var thumbImg *image.NRGBA
	var thumbSize *map[string]int
	var originUrl string
	var thumbUrl string

	if uploadType == "avatar" {
		// 头像：Fill 居中裁剪，强制填满固定宽高，会裁掉多余画面，输出正方形
		thumbImg = imaging.Fill(srcImg, ups.avatarWidth, ups.avatarHeight, imaging.Center, imaging.Lanczos)
		thumbSize = &map[string]int{
			"thumbW": ups.avatarWidth,
			"thumbH": ups.avatarHeight,
		}
		originUrl = fmt.Sprintf("/static/%s/origin/%s/%s", uploadType, dateDir, originFileName)
		thumbUrl = fmt.Sprintf("/static/%s/thumb/%s/%s", uploadType, dateDir, thumbFileName)

	} else if uploadType == "bgImg" {
		// 背景图：Resize 等比例缩放，完整保留整张图，不裁剪
		thumbImg = imaging.Resize(srcImg, ups.bgWidth, ups.bgHeight, imaging.Lanczos)
		thumbSize = &map[string]int{
			"thumbW": ups.bgWidth,
			"thumbH": ups.bgHeight,
		}
		originUrl = fmt.Sprintf("/static/%s/origin/%s/%s", uploadType, dateDir, originFileName)
		thumbUrl = fmt.Sprintf("/static/%s/thumb/%s/%s", uploadType, dateDir, thumbFileName)

	} else {
		return nil, "", "", fmt.Errorf("不支持的图片类型")
	}

	// 保存缩略图，质量85
	thumbSavePath := filepath.Join(thumbDir, thumbFileName)
	if err = imaging.Save(thumbImg, thumbSavePath, imaging.JPEGQuality(85)); err != nil {
		return nil, "", "", err
	}

	//Go 的函数传参都是 值传递，拷贝的都是副本，即便传入的是指针变量，拷贝的也是
	//使用 goroutine 必须复制成局部变量再传入闭包
	//否则 goroutine 延迟执行时，外层变量可能被后续请求覆盖，出现数据错乱
	//Go 栈上局部变量只有没有任何引用时，函数退出后栈帧才会被回收
	//如果局部变量被闭包函数捕获，编译器会做逃逸分析，把变量从栈 挪到堆上分配，栈帧销毁也不会丢失
	asyncDb := db
	asyncUid := uid
	asyncUploadType := uploadType
	asyncThumbUrl := thumbUrl

	// 此处要单独 开一个 ctx 和 cancel，因为外部的 ctx 的 cancel 持有 *timeCtx 指针
	// 而 外部的 ctx 内部又包装了 *timeCtx 的接口变量， 拷贝外部的 ctx 相当于 包装了 同一个 *timeCtx
	// 因为 *timeCtx 指向的地址都是一样的 所以 当外部进行了 cancel 时， 新拷贝的 ctx 也会同步 取消
	asyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	go func() {
		defer cancel()
		txErr := asyncDb.WithContext(asyncCtx).Transaction(func(tx *gorm.DB) error {
			var dbErr error
			switch asyncUploadType {
			case "avatar":
				dbErr = model.UpdateAvatarUrl(asyncCtx, tx, asyncThumbUrl, asyncUid)
			case "bgImg":
				dbErr = model.UpdateBgImgUrl(asyncCtx, tx, asyncThumbUrl, asyncUid)
			}
			return dbErr
		})
		if txErr != nil {
			fmt.Printf("异步更新图片失败 uid=%s type=%s url=%s err=%v\n", asyncUid, asyncUploadType, asyncThumbUrl, txErr)
		}
	}()

	return thumbSize, originUrl, thumbUrl, nil
}
