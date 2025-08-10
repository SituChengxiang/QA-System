package extension

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	pluginLogger     *slog.Logger
	pluginLoggerOnce sync.Once
	// 日志目录常量
	logDir = "logs"
)

// LogConfig 插件日志配置结构体
type LogConfig struct {
	Level      slog.Level
	AddSource  bool
	LogPath    string
	MaxSize    int  // 最大日志大小，单位MB
	MaxAge     int  // 最大保留天数
	Compress   bool // 是否压缩
	Format     string
	LoggerName string
}

// GetDefaultLogConfig 获取默认日志配置
func GetDefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      slog.LevelInfo,
		AddSource:  true,
		LogPath:    filepath.Join(logDir, "pluginLog.log"),
		MaxSize:    10, // 10MB
		MaxAge:     7,  // 7天
		Compress:   true,
		Format:     "json", // 默认json格式
		LoggerName: "plugin",
	}
}

// GetPluginLogger 获取用于插件系统的slog日志记录器的单例实例
func GetPluginLogger() *slog.Logger {
	pluginLoggerOnce.Do(func() {
		if pluginLogger == nil {
			pluginLogger = initPluginLogger(GetDefaultLogConfig())
		}
	})
	return pluginLogger
}

// GetPluginLoggerWithConfig 使用自定义配置获取日志记录器
func GetPluginLoggerWithConfig(cfg *LogConfig) *slog.Logger {
	if pluginLogger != nil {
		return pluginLogger // 如果已初始化，返回现有实例
	}

	// 否则创建新的实例
	return initPluginLogger(cfg)
}

// createLogWriter 创建日志写入器，支持日志滚动
func createLogWriter(cfg *LogConfig) io.Writer {
	// 确保日志目录存在
	dir := filepath.Dir(cfg.LogPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		return os.Stdout // 失败时使用标准输出
	}

	// 使用lumberjack进行日志滚动
	return &lumberjack.Logger{
		Filename:  cfg.LogPath,
		MaxSize:   cfg.MaxSize,
		MaxAge:    cfg.MaxAge,
		Compress:  cfg.Compress,
		LocalTime: true,
	}
}

// initPluginLogger 初始化插件日志记录器
func initPluginLogger(cfg *LogConfig) *slog.Logger {
	// 创建多路日志写入器（同时写入文件和控制台）
	writer := createLogWriter(cfg)
	multiWriter := io.MultiWriter(writer, os.Stdout)

	// 创建日志处理器选项
	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	// 根据配置选择日志格式
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(multiWriter, opts)
	} else {
		handler = slog.NewTextHandler(multiWriter, opts)
	}

	// 创建日志记录器
	logger := slog.New(handler)

	// 记录日志系统初始化成功的消息
	logger.Info("Plugin Logger initalized successfully",
		"path", cfg.LogPath,
		"level", cfg.Level.String(),
		"format", cfg.Format,
	)

	return logger
}

// 以下是一些辅助日志方法，简化常用日志记录

// Info 记录信息日志
func Info(msg string, args ...any) {
	GetPluginLogger().Info(msg, args...)
}

// Error 记录错误日志
func Error(msg string, args ...any) {
	GetPluginLogger().Error(msg, args...)
}

// Warn 记录警告日志
func Warn(msg string, args ...any) {
	GetPluginLogger().Warn(msg, args...)
}

// Debug 记录调试日志
func Debug(msg string, args ...any) {
	GetPluginLogger().Debug(msg, args...)
}
