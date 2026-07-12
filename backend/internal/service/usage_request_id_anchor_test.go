package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

// Startwork: patch —— 账务锚点不变量③:落库时 resolveUsageBillingRequestID 必须【优先】
// 返回 ctx 里的 usage 锚点，且【原样返回不加前缀】，使 usage_logs.request_id == 回传给
// Startwork 的响应头值。即便同时存在 client/upstream id，锚点也必须胜出。
// 两条计费线（Anthropic/OpenAI）都经此 resolver，故此不变量覆盖全部计费路径。
func TestResolveUsageBillingRequestID_PrefersUsageAnchorRaw(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.UsageRequestID, "usage:abc-123")
	ctx = context.WithValue(ctx, ctxkey.ClientRequestID, "cli-xyz") // 干扰项:也存在 client id
	ctx = context.WithValue(ctx, ctxkey.RequestID, "local-req-777")  // 干扰项:也存在 local id

	got := resolveUsageBillingRequestID(ctx, "upstream-req-999")
	if got != "usage:abc-123" {
		t.Fatalf("应原样优先返回 usage 锚点 'usage:abc-123'，got %q", got)
	}
}

// 无锚点时保持上游原有回退语义（client:/local:/upstream/generated），不破坏既有行为。
func TestResolveUsageBillingRequestID_FallsBackWhenNoAnchor(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "cli-xyz")
	if got := resolveUsageBillingRequestID(ctx, ""); got != "client:cli-xyz" {
		t.Fatalf("无锚点时应回退到 client:cli-xyz，got %q", got)
	}
}
