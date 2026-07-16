package middleware

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const clientRequestIDHeader = "X-Client-Request-ID"

// Sub2APIUsageRequestIDHeader 是 Sub2API 回传给 Startwork 的记账锚点响应头（Startwork: patch）。
// Startwork 侧读取该头作为 usage_logs.request_id 的对账锚点。
const Sub2APIUsageRequestIDHeader = "X-Sub2API-Usage-Request-ID"

// ClientRequestID ensures every request has a unique client_request_id in request.Context().
//
// This is used by the Ops monitoring module for end-to-end request correlation.
//
// Startwork: patch —— 该中间件已挂在全部计费转发路由链上，故顺带在请求入口种下 usage 记账锚点
// （stampUsageRequestID）：生成 Sub2API 自有的 usage:<uuid>，写响应头 X-Sub2API-Usage-Request-ID
// 与 ctx，先于任何 handler 写响应体。由此一处统一收敛，结构性保证「锚点头先于任何流式写提交」，
// 无需在每条转发路径逐处插入（避免散点接线在升级 cherry-pick 时被上游重构悄悄冲掉）。
func ClientRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request == nil {
			c.Next()
			return
		}

		stampUsageRequestID(c) // Startwork: patch usage 记账锚点

		if v, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(v) != "" {
			c.Header(clientRequestIDHeader, strings.TrimSpace(v))
			c.Next()
			return
		}

		id := uuid.New().String()
		c.Header(clientRequestIDHeader, id)
		ctx := context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, id)
		requestLogger := logger.FromContext(ctx).With(zap.String("client_request_id", strings.TrimSpace(id)))
		ctx = logger.IntoContext(ctx, requestLogger)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// stampUsageRequestID 在请求入口生成（或复用已有的）Sub2API 记账锚点，写响应头 + ctx（Startwork: patch）。
// 关键:只读 ctx（内部值），绝不读入站请求头 —— 否则客户端可注入锚点导致重复认领/双重扣费。
func stampUsageRequestID(c *gin.Context) {
	ctx := c.Request.Context()
	if v, _ := ctx.Value(ctxkey.UsageRequestID).(string); strings.TrimSpace(v) != "" {
		c.Header(Sub2APIUsageRequestIDHeader, strings.TrimSpace(v))
		return
	}
	usageID := "usage:" + uuid.New().String()
	c.Header(Sub2APIUsageRequestIDHeader, usageID)
	c.Request = c.Request.WithContext(context.WithValue(ctx, ctxkey.UsageRequestID, usageID))
}
