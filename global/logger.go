package global

import (
	"log"
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log 全局日志变量
var Log *zap.Logger

// InitLogger 初始化日志
func InitLogger() {
	// 1. 日志文件切割配置
	fileWriter := &lumberjack.Logger{
		Filename:   "./logs/app.log", // 日志存放路径
		MaxSize:    100,              // 单个日志最大100MB
		MaxBackups: 10,               // 最多保留10个备份文件
		MaxAge:     7,                // 日志保留7天
		Compress:   true,             // 过期日志压缩
	}

	// 2. 日志编码配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		CallerKey:     "caller", // 打印代码文件+行号
		MessageKey:    "msg",
		StacktraceKey: "stack",
		EncodeTime:    zapcore.ISO8601TimeEncoder, // 标准时间格式
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	// 区分开发/生产编码
	var encoder zapcore.Encoder
	// 开发模式：控制台彩色易读
	encoder = zapcore.NewConsoleEncoder(encoderConfig)
	// 生产模式：json结构化日志（注释上面一行，放开下面）
	// encoder = zapcore.NewJSONEncoder(encoderConfig)

	// 3. 输出位置：控制台 + 文件双输出
	multiWriter := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
		zapcore.AddSync(fileWriter),
	)

	// 4. 日志级别：Debug(调试) < Info(正常) < Warn(警告) < Error(错误)
	core := zapcore.NewCore(encoder, multiWriter, zapcore.DebugLevel)

	// 5. 创建日志实例
	Log = zap.New(
		core,
		zap.AddCaller(),                       // 开启代码行号
		zap.AddStacktrace(zapcore.ErrorLevel), // Error自动打印堆栈
	)
	// 让标准库log全部走zap
	stdLog := zap.NewStdLog(Log)
	log.SetOutput(stdLog.Writer())

}
