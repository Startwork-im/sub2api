//go:build unit

package migrations

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartworkDeepSeekTimePricingMigration(t *testing.T) {
	sqlBytes, err := os.ReadFile("227_startwork_deepseek_v4_time_pricing.sql")
	require.NoError(t, err)
	sql := string(sqlBytes)

	for _, expected := range []string{
		"deepseek-ecosystem-%",
		"deepseek-private-%",
		"deepseek-v4-flash",
		"deepseek-v4-pro",
		"deepseek-chat",
		"deepseek-reasoner",
		"Asia/Shanghai",
		`"start_time": "09:00"`,
		`"end_time": "12:00"`,
		`"start_time": "14:00"`,
		`"end_time": "18:00"`,
		`"multiplier": 2`,
		"0.00000022",
		"0.00000066",
		"0.000000007",
		"0.00000198",
		"0.000000022",
	} {
		require.Contains(t, sql, expected)
	}
	require.Contains(t, strings.ToLower(sql), "group pricing has higher precedence")
	require.Contains(t, sql, "UPDATE groups")
	require.Contains(t, sql, "INSERT INTO channel_model_pricing")
}
