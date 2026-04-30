package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

type chatPassthroughMeta struct {
	OriginalModel string
	BillingModel  string
	UpstreamModel string
	ClientStream  bool
	IncludeUsage  bool
	StartTime     time.Time
}

func (s *OpenAIGatewayService) tryForwardOpenAICompatibleChatCompletionsPassthrough(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	originalModel string,
	billingModel string,
	upstreamModel string,
	clientStream bool,
	includeUsage bool,
	startTime time.Time,
) (*OpenAIForwardResult, bool, error) {
	if !shouldPassthroughOpenAICompatibleChatCompletions(account) {
		return nil, false, nil
	}
	passthroughBody := body
	if upstreamModel != originalModel {
		var err error
		passthroughBody, err = sjson.SetBytes(body, "model", upstreamModel)
		if err != nil {
			return nil, true, fmt.Errorf("rewrite model in chat completions passthrough body: %w", err)
		}
	}
	result, err := s.forwardOpenAICompatibleChatCompletionsPassthrough(ctx, c, account, passthroughBody, chatPassthroughMeta{
		OriginalModel: originalModel,
		BillingModel:  billingModel,
		UpstreamModel: upstreamModel,
		ClientStream:  clientStream,
		IncludeUsage:  includeUsage,
		StartTime:     startTime,
	})
	return result, true, err
}

func shouldPassthroughOpenAICompatibleChatCompletions(account *Account) bool {
	if account == nil || account.Platform != PlatformOpenAIChat || account.Type != AccountTypeAPIKey {
		return false
	}
	return strings.TrimSpace(account.GetCredential("base_url")) != ""
}

func (s *OpenAIGatewayService) forwardOpenAICompatibleChatCompletionsPassthrough(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	meta chatPassthroughMeta,
) (*OpenAIForwardResult, error) {
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	upstreamReq, err := s.buildOpenAICompatibleChatCompletionsPassthroughRequest(ctx, c, account, body, token)
	if err != nil {
		return nil, fmt.Errorf("build chat completions passthrough request: %w", err)
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		writeChatCompletionsError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
			})
			if s.rateLimitService != nil {
				s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && (isPoolModeRetryableStatus(resp.StatusCode) || isOpenAITransientProcessingError(resp.StatusCode, upstreamMsg, respBody)),
			}
		}
		return s.handleChatCompletionsErrorResponse(resp, c, account)
	}

	if meta.ClientStream {
		return s.handleOpenAICompatibleChatCompletionsPassthroughStream(resp, c, meta)
	}
	return s.handleOpenAICompatibleChatCompletionsPassthroughJSON(resp, c, meta)
}

func (s *OpenAIGatewayService) buildOpenAICompatibleChatCompletionsPassthroughRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
) (*http.Request, error) {
	baseURL := strings.TrimSpace(account.GetCredential("base_url"))
	if baseURL == "" {
		return nil, fmt.Errorf("base_url is required for chat completions passthrough")
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	targetURL := buildOpenAIChatCompletionsURL(validatedURL)
	if account.Platform == PlatformOpenAIChat {
		targetURL = buildOpenAIChatPlatformCompletionsURL(validatedURL)
		body = ensureOpenAIChatStreamIncludeUsage(body)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	allowTimeoutHeaders := s.isOpenAIPassthroughTimeoutHeadersAllowed()
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			lower := strings.ToLower(strings.TrimSpace(key))
			if !isOpenAIPassthroughAllowedRequestHeader(lower, allowTimeoutHeaders) {
				continue
			}
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}

	req.Header.Del("authorization")
	req.Header.Del("x-api-key")
	req.Header.Del("x-goog-api-key")
	req.Header.Set("authorization", "Bearer "+token)
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	return req, nil
}

func ensureOpenAIChatStreamIncludeUsage(body []byte) []byte {
	if !gjson.GetBytes(body, "stream").Bool() {
		return body
	}
	if gjson.GetBytes(body, "stream_options.include_usage").Exists() {
		return body
	}
	updated, err := sjson.SetBytes(body, "stream_options.include_usage", true)
	if err != nil {
		return body
	}
	return updated
}

func (s *OpenAIGatewayService) handleOpenAICompatibleChatCompletionsPassthroughJSON(
	resp *http.Response,
	c *gin.Context,
	meta chatPassthroughMeta,
) (*OpenAIForwardResult, error) {
	requestID := resp.Header.Get("x-request-id")
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeChatCompletionsError(c, http.StatusBadGateway, "api_error", "Failed to read upstream response")
		return nil, fmt.Errorf("read chat completions passthrough response: %w", err)
	}

	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Data(http.StatusOK, firstNonEmptyChatPassthroughValue(resp.Header.Get("content-type"), "application/json"), respBody)

	return &OpenAIForwardResult{
		RequestID:       requestID,
		Usage:           openAICompatibleChatUsageFromBody(respBody),
		Model:           meta.OriginalModel,
		BillingModel:    meta.BillingModel,
		UpstreamModel:   meta.UpstreamModel,
		Stream:          false,
		Duration:        time.Since(meta.StartTime),
		ResponseHeaders: resp.Header,
	}, nil
}

func (s *OpenAIGatewayService) handleOpenAICompatibleChatCompletionsPassthroughStream(
	resp *http.Response,
	c *gin.Context,
	meta chatPassthroughMeta,
) (*OpenAIForwardResult, error) {
	requestID := resp.Header.Get("x-request-id")
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	contentType := firstNonEmptyChatPassthroughValue(resp.Header.Get("content-type"), "text/event-stream")
	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	var usage OpenAIUsage
	var firstTokenMs *int
	firstChunk := true

	for scanner.Scan() {
		line := scanner.Text()
		if _, err := fmt.Fprint(c.Writer, line+"\n"); err != nil {
			logger.L().Info("openai chat_completions passthrough stream: client disconnected",
				zap.String("request_id", requestID),
			)
			break
		}
		if firstChunk && strings.HasPrefix(line, "data: ") && line != "data: [DONE]" {
			firstChunk = false
			ms := int(time.Since(meta.StartTime).Milliseconds())
			firstTokenMs = &ms
		}
		if strings.HasPrefix(line, "data: ") && line != "data: [DONE]" {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if parsedUsage := openAICompatibleChatUsageFromBody([]byte(payload)); parsedUsage != (OpenAIUsage{}) {
				usage = parsedUsage
			}
		}
		c.Writer.Flush()
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		logger.L().Warn("openai chat_completions passthrough stream: read error",
			zap.Error(err),
			zap.String("request_id", requestID),
		)
	}

	return &OpenAIForwardResult{
		RequestID:       requestID,
		Usage:           usage,
		Model:           meta.OriginalModel,
		BillingModel:    meta.BillingModel,
		UpstreamModel:   meta.UpstreamModel,
		Stream:          true,
		Duration:        time.Since(meta.StartTime),
		FirstTokenMs:    firstTokenMs,
		ResponseHeaders: resp.Header,
	}, nil
}

// buildOpenAIChatCompletionsURL 组装 OpenAI-compatible Chat Completions 端点。
// - base 已是 /chat/completions：原样返回
// - base 已是 /responses：替换为同级 /chat/completions
// - base 以 /v1 结尾：追加 /chat/completions
// - 其他情况：追加 /v1/chat/completions
func buildOpenAIChatCompletionsURL(base string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(normalized, "/chat/completions") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/responses") {
		return strings.TrimSuffix(normalized, "/responses") + "/chat/completions"
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/chat/completions"
	}
	return normalized + "/v1/chat/completions"
}

func buildOpenAIChatPlatformCompletionsURL(base string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(normalized, "/chat/completions") {
		return normalized
	}
	return normalized + "/chat/completions"
}

func openAICompatibleChatUsageFromBody(body []byte) OpenAIUsage {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return OpenAIUsage{}
	}
	usage := OpenAIUsage{
		InputTokens:  firstPositiveGJSONInt(body, "usage.prompt_tokens", "usage.input_tokens"),
		OutputTokens: firstPositiveGJSONInt(body, "usage.completion_tokens", "usage.output_tokens"),
	}
	usage.CacheReadInputTokens = firstPositiveGJSONInt(body, "usage.prompt_tokens_details.cached_tokens", "usage.input_tokens_details.cached_tokens")
	return usage
}

func firstPositiveGJSONInt(body []byte, paths ...string) int {
	for _, path := range paths {
		value := int(gjson.GetBytes(body, path).Int())
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmptyChatPassthroughValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
