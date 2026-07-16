package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
)

// Startwork: patch —— 账务锚点不变量①:入口中间件必须在任何 handler 运行之前，把
// X-Sub2API-Usage-Request-ID 响应头与 ctx 种成【同一个】usage:<uuid> 值。
// 这保证「回传给 Startwork 的头」== 「后续记账读到的 ctx 锚点」，且头先于任何流式写提交。
func TestClientRequestID_StampsUsageAnchor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ClientRequestID())

	var ctxVal string
	r.GET("/x", func(c *gin.Context) {
		ctxVal, _ = c.Request.Context().Value(ctxkey.UsageRequestID).(string)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	hdr := w.Header().Get(Sub2APIUsageRequestIDHeader)
	if !strings.HasPrefix(hdr, "usage:") {
		t.Fatalf("响应头 %s 应以 'usage:' 开头，got %q", Sub2APIUsageRequestIDHeader, hdr)
	}
	if ctxVal != hdr {
		t.Fatalf("ctx 锚点 %q != 响应头 %q —— 二者必须同一个值", ctxVal, hdr)
	}
}

// 账务锚点不变量②:客户端注入的同名请求头必须被【忽略】（生成全新锚点），
// 否则客户端可伪造锚点，两条请求锚同一个 id → Startwork 侧重复认领/双重扣费。
func TestClientRequestID_IgnoresInboundUsageHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ClientRequestID())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(Sub2APIUsageRequestIDHeader, "usage:client-injected")
	r.ServeHTTP(w, req)

	if got := w.Header().Get(Sub2APIUsageRequestIDHeader); got == "usage:client-injected" {
		t.Fatalf("入站请求头被复用了（应忽略并生成新锚点），got %q", got)
	}
}
