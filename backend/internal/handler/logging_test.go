//go:build unit

package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogStickySessionBindFailure_ContextCanceledIsInfo(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)

	logStickySessionBindFailure(zap.New(core), "gateway.bind_sticky_session_failed", 9, context.Canceled)

	entries := logs.All()
	require.Len(t, entries, 1)
	require.Equal(t, zap.InfoLevel, entries[0].Level)
	require.Equal(t, "gateway.bind_sticky_session_failed", entries[0].Message)
	require.Equal(t, true, entries[0].ContextMap()["request_canceled"])
}

func TestLogStickySessionBindFailure_RealFailureRemainsWarning(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)

	logStickySessionBindFailure(zap.New(core), "gateway.bind_sticky_session_failed", 9, errors.New("redis unavailable"))

	entries := logs.All()
	require.Len(t, entries, 1)
	require.Equal(t, zap.WarnLevel, entries[0].Level)
	require.Equal(t, "gateway.bind_sticky_session_failed", entries[0].Message)
	require.NotContains(t, entries[0].ContextMap(), "request_canceled")
}
