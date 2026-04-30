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
)

func TestOpenAIGatewayService_OpenAIChatPassthrough_UsesChatCompletionsPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{
				"id":"chatcmpl-1",
				"object":"chat.completion",
				"model":"mimo-v2.5-pro",
				"choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],
				"usage":{"prompt_tokens":3,"completion_tokens":5}
			}`)),
		},
	}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"mimo-v2.5-pro","messages":[{"role":"user","content":"hi"}],"stream":false}`))
	c.Request.Header.Set("Content-Type", "application/json")

	account := &Account{
		ID:       11,
		Platform: PlatformOpenAIChat,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "sk-mimo",
			"base_url": "https://api.xiaomimimo.com",
		},
	}

	result, err := svc.ForwardAsChatCompletions(c.Request.Context(), c, account, []byte(`{"model":"mimo-v2.5-pro","messages":[{"role":"user","content":"hi"}],"stream":false}`), "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "https://api.xiaomimimo.com/v1/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "sk-mimo", upstream.lastReq.Header.Get("x-api-key"))
	require.Empty(t, upstream.lastReq.Header.Get("authorization"))
	require.Equal(t, "mimo-v2.5-pro", result.UpstreamModel)
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
	require.Contains(t, rec.Body.String(), "chatcmpl-1")
}
