package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildOpenAIChatCompletionsURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{name: "base host", base: "https://api.deepseek.com", want: "https://api.deepseek.com/v1/chat/completions"},
		{name: "v1 base", base: "https://api.deepseek.com/v1", want: "https://api.deepseek.com/v1/chat/completions"},
		{name: "chat path", base: "https://api.deepseek.com/v1/chat/completions", want: "https://api.deepseek.com/v1/chat/completions"},
		{name: "responses path", base: "https://api.deepseek.com/v1/responses", want: "https://api.deepseek.com/v1/chat/completions"},
		{name: "trailing slash", base: "https://api.deepseek.com/v1/", want: "https://api.deepseek.com/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, buildOpenAIChatCompletionsURL(tt.base))
		})
	}
}

func TestBuildOpenAIChatPlatformCompletionsURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{name: "base host", base: "https://api.deepseek.com", want: "https://api.deepseek.com/chat/completions"},
		{name: "v1 base", base: "https://token-plan-cn.xiaomimimo.com/v1", want: "https://token-plan-cn.xiaomimimo.com/v1/chat/completions"},
		{name: "chat path", base: "https://api.deepseek.com/chat/completions", want: "https://api.deepseek.com/chat/completions"},
		{name: "trailing slash", base: "https://api.deepseek.com/", want: "https://api.deepseek.com/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, buildOpenAIChatPlatformCompletionsURL(tt.base))
		})
	}
}

func TestShouldPassthroughOpenAICompatibleChatCompletions(t *testing.T) {
	require.False(t, shouldPassthroughOpenAICompatibleChatCompletions(nil))
	require.False(t, shouldPassthroughOpenAICompatibleChatCompletions(&Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}))
	require.False(t, shouldPassthroughOpenAICompatibleChatCompletions(&Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.openai-compatible.example/v1"},
	}))
	require.True(t, shouldPassthroughOpenAICompatibleChatCompletions(&Account{
		Platform:    PlatformOpenAIChat,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.deepseek.com/v1"},
	}))
}

func TestBuildOpenAICompatibleChatCompletionsPassthroughRequestUsesChatPath(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{Enabled: false},
		},
	}
	svc := &OpenAIGatewayService{cfg: cfg}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"deepseek-v4-flash"}`))
	c.Request.Header.Set("Authorization", "Bearer client-key")
	c.Request.Header.Set("Content-Type", "application/json")

	account := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.deepseek.com/v1"},
	}

	req, err := svc.buildOpenAICompatibleChatCompletionsPassthroughRequest(c.Request.Context(), c, account, []byte(`{"model":"deepseek-v4-flash"}`), "upstream-key")
	require.NoError(t, err)
	require.Equal(t, "https://api.deepseek.com/v1/chat/completions", req.URL.String())
	require.Equal(t, "Bearer upstream-key", req.Header.Get("Authorization"))
	require.Empty(t, req.Header.Get("x-api-key"))
}

func TestBuildOpenAICompatibleChatCompletionsPassthroughRequestAddsStreamUsageForOpenAIChat(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{Enabled: false},
		},
	}
	svc := &OpenAIGatewayService{cfg: cfg}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"mimo-v2.5-pro"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	account := &Account{
		Platform:    PlatformOpenAIChat,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://token-plan-cn.xiaomimimo.com/v1"},
	}

	body := []byte(`{"model":"mimo-v2.5-pro","messages":[{"role":"user","content":"hi"}],"stream":true}`)
	req, err := svc.buildOpenAICompatibleChatCompletionsPassthroughRequest(c.Request.Context(), c, account, body, "sk-mimo")
	require.NoError(t, err)
	require.NotNil(t, req)

	updated, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(updated, "stream_options.include_usage").Bool())
}
