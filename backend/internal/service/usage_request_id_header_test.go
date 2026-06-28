package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestSetUsageRequestIDHeaderUsesResolvedUsageLogRequestID(t *testing.T) {
	header := http.Header{}
	got := setUsageRequestIDHeader(context.Background(), header, "upstream-req")

	require.Equal(t, "upstream-req", got)
	require.Equal(t, "upstream-req", header.Get(Sub2APIUsageRequestIDHeader))
}

func TestSetUsageRequestIDHeaderFallsBackToClientRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-req")
	header := http.Header{}

	got := setUsageRequestIDHeader(ctx, header, "")

	require.Equal(t, "client:client-req", got)
	require.Equal(t, "client:client-req", header.Get(Sub2APIUsageRequestIDHeader))
}
