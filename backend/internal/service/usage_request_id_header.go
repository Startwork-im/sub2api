package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
)

const Sub2APIUsageRequestIDHeader = "X-Sub2API-Usage-Request-ID"

func setUsageRequestIDHeader(ctx context.Context, header http.Header, upstreamRequestID string) string {
	if requestID := strings.TrimSpace(header.Get(Sub2APIUsageRequestIDHeader)); requestID != "" {
		return requestID
	}
	requestID := resolveUsageBillingRequestID(ctx, upstreamRequestID)
	setResolvedUsageRequestIDHeader(header, requestID)
	return requestID
}

func setResolvedUsageRequestIDHeader(header http.Header, requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if header != nil && requestID != "" {
		header.Set(Sub2APIUsageRequestIDHeader, requestID)
	}
	return requestID
}

func resolveStableUsageRequestID(ctx context.Context, upstreamRequestID string) string {
	if requestID := strings.TrimSpace(upstreamRequestID); requestID != "" {
		return requestID
	}
	if ctx != nil {
		if clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
			return "client:" + strings.TrimSpace(clientRequestID)
		}
		if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
			return "local:" + strings.TrimSpace(requestID)
		}
	}
	return ""
}

func setUsageRequestIDHeaderFromGin(c *gin.Context, upstreamRequestID string) string {
	if c == nil {
		return setUsageRequestIDHeader(context.Background(), nil, upstreamRequestID)
	}
	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}
	return setUsageRequestIDHeader(ctx, c.Writer.Header(), upstreamRequestID)
}
