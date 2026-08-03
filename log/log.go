package log

import (
	"os"
	"path"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.SugaredLogger

var LogFile = path.Join(os.Getenv("ARM_SF"), "/data/log/", "web.log")

func InitLog() *zap.SugaredLogger {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.InfoLevel)

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	logger = zapLogger.Sugar()
	return logger
}

func getEncoder() zapcore.Encoder {
	// NewJSONEncoder创建了一个快速、低分配的JSON编码器。编码器适当地转义所有字段键和值。
	// NewProductionEncoderConfig returns an opinionated EncoderConfig for
	// production environments.
	//return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// NewConsoleEncoder创建一个编码器，其输出是为人类而不是机器设计的。
	// 它以纯文本格式序列化核心日志条目数据(消息、级别、时间戳等)，并将结构化上下文保留为JSON。
	/*
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}*/

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {
	// Logger is an io.WriteCloser that writes to the specified filename.
	// 日志记录器在第一次写入时打开或创建日志文件。如果文件存在并且小于MaxSize兆字节，则lumberjack将打开并追加该文件。
	// 如果该文件存在并且其大小为>= MaxSize兆字节，
	// 则通过将当前时间放在文件扩展名(或者如果没有扩展名则放在文件名的末尾)的名称中的时间戳中来重命名该文件。
	// 然后使用原始文件名创建一个新的日志文件。
	// 每当写操作导致当前日志文件超过MaxSize兆字节时，将关闭当前文件，重新命名，并使用原始名称创建新的日志文件。
	// 因此，您给Logger的文件名始终是“当前”日志文件。
	// 如果MaxBackups和MaxAge均为0，则不会删除旧的日志文件。

	lumberJackLogger := &lumberjack.Logger{
		Filename:   LogFile, // Filename是要写入日志的文件。备份日志文件将保留在同一目录下
		MaxSize:    10,      // MaxSize是日志文件旋转之前的最大大小(以兆字节为单位)。默认为100兆字节。
		MaxBackups: 5,       // MaxBackups是要保留的旧日志文件的最大数量。默认是保留所有旧的日志文件(尽管MaxAge仍然可能导致它们被删除)。
		MaxAge:     30,      // MaxAge是根据文件名中编码的时间戳保留旧日志文件的最大天数。请注意，一天被定义为24小时，由于夏令时、闰秒等原因，可能与日历日不完全对应。默认情况下，不根据时间删除旧的日志文件。
		Compress:   false,   // Compress决定是否应该使用gzip压缩旋转的日志文件。默认情况下不执行压缩。
	}
	return zapcore.AddSync(lumberJackLogger)
}

func GetLogger() *zap.SugaredLogger {
	return logger
}

func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

func Warn(args ...interface{}) {
	logger.Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	logger.Fatalf(template, args...)
}
