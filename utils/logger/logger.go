package logger

import "github.com/adnilis/logger"

// Logger 日志接口
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// loggerWrapper 包装 github.com/adnilis/logger
type loggerWrapper struct {
	name string
}

func (w *loggerWrapper) Info(msg string, args ...interface{}) {
	logger.Info("[%s] %s", w.name, msg)
}

func (w *loggerWrapper) Error(msg string, args ...interface{}) {
	logger.Error("[%s] %s", w.name, msg)
}

func (w *loggerWrapper) Debug(msg string, args ...interface{}) {
	logger.Debug("[%s] %s", w.name, msg)
}

func (w *loggerWrapper) Warn(msg string, args ...interface{}) {
	logger.Warn("[%s] %s", w.name, msg)
}

// NewLogger 创建日志包装器
func NewLogger(name string) Logger {
	return &loggerWrapper{name: name}
}
