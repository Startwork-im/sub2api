package service

// Startwork: patch 定价目录快照测试。

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newPricingCatalogTestService(t *testing.T) *PricingService {
	t.Helper()
	svc := NewPricingService(nil, nil)
	data, err := svc.parsePricingData([]byte(`{
		"claude-sonnet-4-20250514": {
			"input_cost_per_token": 3e-06,
			"output_cost_per_token": 1.5e-05,
			"cache_creation_input_token_cost": 3.75e-06,
			"cache_read_input_token_cost": 3e-07,
			"litellm_provider": "anthropic",
			"mode": "chat",
			"supports_prompt_caching": true
		},
		"gpt-4o": {
			"input_cost_per_token": 2.5e-06,
			"output_cost_per_token": 1e-05,
			"litellm_provider": "openai",
			"mode": "chat"
		},
		"gemini-2.0-flash": {
			"input_cost_per_token": 1e-07,
			"output_cost_per_token": 4e-07,
			"litellm_provider": "gemini",
			"mode": "chat"
		}
	}`))
	require.NoError(t, err)
	svc.pricingData = data
	svc.lastUpdated = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return svc
}

func TestSnapshotModelPricing_All(t *testing.T) {
	svc := newPricingCatalogTestService(t)

	models, lastUpdated := svc.SnapshotModelPricing("")
	require.Len(t, models, 3)
	require.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), lastUpdated)

	claude, ok := models["claude-sonnet-4-20250514"]
	require.True(t, ok)
	require.InDelta(t, 3e-06, claude.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.5e-05, claude.OutputCostPerToken, 1e-12)
	require.InDelta(t, 3.75e-06, claude.CacheCreationInputTokenCost, 1e-12)
	require.InDelta(t, 3e-07, claude.CacheReadInputTokenCost, 1e-12)
	require.Equal(t, "anthropic", claude.LiteLLMProvider)
	require.True(t, claude.SupportsPromptCaching)
}

func TestSnapshotModelPricing_ProviderFilter(t *testing.T) {
	svc := newPricingCatalogTestService(t)

	models, _ := svc.SnapshotModelPricing("Anthropic")
	require.Len(t, models, 1)
	_, ok := models["claude-sonnet-4-20250514"]
	require.True(t, ok)

	models, _ = svc.SnapshotModelPricing("openai")
	require.Len(t, models, 1)
	_, ok = models["gpt-4o"]
	require.True(t, ok)

	models, _ = svc.SnapshotModelPricing("unknown-provider")
	require.NotNil(t, models)
	require.Empty(t, models)
}

func TestSnapshotModelPricing_ReturnsCopy(t *testing.T) {
	svc := newPricingCatalogTestService(t)

	models, _ := svc.SnapshotModelPricing("")
	entry := models["gpt-4o"]
	entry.InputCostPerToken = 999
	models["gpt-4o"] = entry

	again, _ := svc.SnapshotModelPricing("")
	require.InDelta(t, 2.5e-06, again["gpt-4o"].InputCostPerToken, 1e-12)
}

func TestSnapshotModelPricing_EmptyService(t *testing.T) {
	svc := NewPricingService(nil, nil)

	models, lastUpdated := svc.SnapshotModelPricing("")
	require.NotNil(t, models)
	require.Empty(t, models)
	require.True(t, lastUpdated.IsZero())
}
