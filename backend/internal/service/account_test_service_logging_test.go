package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestAccountTestServiceSendErrorAndEndSkipsCanceledRequestError(t *testing.T) {
	core, entries := observer.New(zap.InfoLevel)
	requestContext := logger.IntoContext(context.Background(), zap.New(core))
	requestContext, cancel := context.WithCancel(requestContext)
	cancel()

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = (&http.Request{}).WithContext(requestContext)

	err := (&AccountTestService{}).sendErrorAndEnd(ginContext, `Request failed: Post "https://nextrouter.io/v1/messages?beta=true": context canceled`)

	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, recorder.Body.String())
	require.Len(t, entries.All(), 1)
	require.Equal(t, "account_test.request_canceled", entries.All()[0].Message)
	require.Equal(t, true, entries.All()[0].ContextMap()["request_canceled"])
}
