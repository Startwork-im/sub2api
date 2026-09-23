//go:build unit

package migrations

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartworkDeepSeekV41FlashTimePricingMigration(t *testing.T) {
	sqlBytes, err := os.ReadFile("241_startwork_deepseek_v41_flash_time_pricing.sql")
	require.NoError(t, err)
	sql := string(sqlBytes)

	for _, expected := range []string{
		"deepseek-ecosystem-%",
		"deepseek-private-%",
		"deepseek-anthropic-ecosystem-%",
		"deepseek-anthropic-private-%",
		"deepseek-flash",
		"deepseek-v4-flash",
		"deepseek-v4-flash-vision-exp",
		"deepseek-v4-pro",
		"deepseek-chat",
		"deepseek-reasoner",
		"Asia/Shanghai",
		`"weekdays_only": true`,
		`"start_time": "09:00"`,
		`"end_time": "12:00"`,
		`"start_time": "14:00"`,
		`"end_time": "18:00"`,
		`"multiplier": 2`,
		"0.00000015",
		"0.00000060",
		"0.000000003",
	} {
		require.Contains(t, sql, expected)
	}
	require.NotContains(t, sql, "0.00000022::numeric")
	require.Contains(t, strings.ToLower(sql), "deepseek-v4-pro is also routed and billed as v4.1-flash")
	require.Contains(t, sql, "DELETE FROM channel_model_pricing")
	require.Contains(t, sql, "INSERT INTO channel_model_pricing")
}
