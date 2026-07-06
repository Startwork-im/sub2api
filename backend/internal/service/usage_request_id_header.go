package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const Sub2APIUsageRequestIDHeader = "X-Sub2API-Usage-Request-ID"

func setUsageRequestIDHeader(ctx context.Context, header http.Header) string {
	_, requestID := bindUsageRequestID(ctx, header)
	return requestID
}

func setResolvedUsageRequestIDHeader(header http.Header, requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if header != nil && requestID != "" {
		header.Set(Sub2APIUsageRequestIDHeader, requestID)
	}
	return requestID
}

func newUsageRequestID() string {
	return "usage:" + uuid.NewString()
}

func usageRequestContext(ctx context.Context, requestID string) context.Context {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxkey.UsageRequestID, requestID)
}

func bindUsageRequestID(ctx context.Context, header http.Header) (context.Context, string) {
	if requestID := strings.TrimSpace(header.Get(Sub2APIUsageRequestIDHeader)); requestID != "" {
		return usageRequestContext(ctx, requestID), requestID
	}
	if ctx != nil {
		if requestID, _ := ctx.Value(ctxkey.UsageRequestID).(string); strings.TrimSpace(requestID) != "" {
			requestID = strings.TrimSpace(requestID)
			setResolvedUsageRequestIDHeader(header, requestID)
			return ctx, requestID
		}
	}
	requestID := newUsageRequestID()
	setResolvedUsageRequestIDHeader(header, requestID)
	return usageRequestContext(ctx, requestID), requestID
}

func bindUsageRequestIDFromGin(c *gin.Context) (context.Context, string) {
	if c == nil {
		return bindUsageRequestID(context.Background(), nil)
	}
	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}
	boundCtx, requestID := bindUsageRequestID(ctx, c.Writer.Header())
	if c.Request != nil && boundCtx != c.Request.Context() {
		c.Request = c.Request.WithContext(boundCtx)
	}
	return boundCtx, requestID
}

func setUsageRequestIDHeaderFromGin(c *gin.Context) string {
	_, requestID := bindUsageRequestIDFromGin(c)
	return requestID
}
