package service

// Startwork: patch 新增定价目录快照读取，供 Startwork 后端同步全局模型价格。
// 该文件为 Startwork 自有扩展，升级 upstream 时整体 cherry-pick。

import (
	"strings"
	"time"
)

// SnapshotModelPricing 返回内存中 LiteLLM 定价目录的一份拷贝。
// 参数：
//   - provider: litellm_provider 过滤条件（大小写不敏感）；为空时返回全部模型
//
// 返回：
//   - map[string]LiteLLMModelPricing: 模型名 -> 定价（值拷贝，调用方可安全持有）
//   - time.Time: 定价数据最近一次更新时间
func (s *PricingService) SnapshotModelPricing(provider string) (map[string]LiteLLMModelPricing, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider = strings.ToLower(strings.TrimSpace(provider))
	out := make(map[string]LiteLLMModelPricing, len(s.pricingData))
	for name, pricing := range s.pricingData {
		if pricing == nil {
			continue
		}
		if provider != "" && strings.ToLower(pricing.LiteLLMProvider) != provider {
			continue
		}
		out[name] = *pricing
	}
	return out, s.lastUpdated
}
