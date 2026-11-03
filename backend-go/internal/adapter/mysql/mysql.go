// Package mysql 建立 gorm 连接，供管理 API 与 worker 使用。网关热路径不走 ORM。
package mysql

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"cloak/internal/config"
)

// Open 打开连接并校验连通性。
func Open(cfg config.MySQL, log *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger:                 newLogger(log, cfg.LogSlow),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}

// newLogger 只记录慢查询与错误，普通 SQL 不输出。
func newLogger(log *zap.Logger, slow time.Duration) gormlogger.Interface {
	return &zapGormLogger{log: log, slow: slow}
}

type zapGormLogger struct {
	log  *zap.Logger
	slow time.Duration
}

func (l *zapGormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }
func (l *zapGormLogger) Info(_ context.Context, msg string, args ...any) {
	l.log.Sugar().Infof(msg, args...)
}
func (l *zapGormLogger) Warn(_ context.Context, msg string, args ...any) {
	l.log.Sugar().Warnf(msg, args...)
}
func (l *zapGormLogger) Error(_ context.Context, msg string, args ...any) {
	l.log.Sugar().Errorf(msg, args...)
}
func (l *zapGormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	if err != nil && err != gorm.ErrRecordNotFound {
		sql, rows := fc()
		l.log.Error("sql error", zap.Error(err), zap.String("sql", sql), zap.Int64("rows", rows), zap.Duration("elapsed", elapsed))
		return
	}
	if l.slow > 0 && elapsed > l.slow {
		sql, rows := fc()
		l.log.Warn("slow sql", zap.String("sql", sql), zap.Int64("rows", rows), zap.Duration("elapsed", elapsed))
	}
}
