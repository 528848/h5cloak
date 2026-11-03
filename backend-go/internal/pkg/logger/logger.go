// Package logger 封装 zap，所有服务统一输出 JSON 日志。
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New 按级别与格式创建 logger。format 为 console 时输出易读格式，用于本地开发。
func New(level, format, service string) (*zap.Logger, error) {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = zapcore.InfoLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.DisableStacktrace = true
	if format == "console" {
		cfg.Encoding = "console"
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return l.With(zap.String("service", service)), nil
}
