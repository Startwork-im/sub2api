package handler

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func requestLogger(c *gin.Context, component string, fields ...zap.Field) *zap.Logger {
	base := logger.L()
	if c != nil && c.Request != nil {
		base = logger.FromContext(c.Request.Context())
	}

	if component != "" {
		fields = append([]zap.Field{zap.String("component", component)}, fields...)
	}
	return base.With(fields...)
}

func logStickySessionBindFailure(reqLog *zap.Logger, event string, accountID int64, err error) {
	if reqLog == nil || err == nil {
		return
	}
	fields := []zap.Field{
		zap.Int64("account_id", accountID),
		zap.Error(err),
	}
	if errors.Is(err, context.Canceled) {
		reqLog.Info(event, append(fields, zap.Bool("request_canceled", true))...)
		return
	}
	reqLog.Warn(event, fields...)
}
