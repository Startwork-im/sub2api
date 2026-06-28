package service

import (
	"context"
	"net/http"
	"strings"
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

func TestSetUsageRequestIDHeaderGeneratesRequestIDWhenAllSourcesMissing(t *testing.T) {
	header := http.Header{}

	got := setUsageRequestIDHeader(context.Background(), header, "")

	require.NotEmpty(t, got)
	require.True(t, strings.HasPrefix(got, "generated:"))
	require.Equal(t, got, header.Get(Sub2APIUsageRequestIDHeader))
}

func TestSetResolvedUsageRequestIDHeaderTrimsValue(t *testing.T) {
	header := http.Header{}

	got := setResolvedUsageRequestIDHeader(header, " usage-log-request-id ")

	require.Equal(t, "usage-log-request-id", got)
	require.Equal(t, "usage-log-request-id", header.Get(Sub2APIUsageRequestIDHeader))
}
