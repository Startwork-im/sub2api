package admin

// Startwork: patch 新增定价目录查询接口，供 Startwork 后端同步全局模型价格。
// 该文件为 Startwork 自有扩展，升级 upstream 时整体 cherry-pick。

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetPricingCatalog 返回 LiteLLM 定价目录快照
// GET /api/v1/admin/channels/pricing/catalog?provider=anthropic
// 参数：
//   - provider: 可选，litellm_provider 过滤（如 anthropic/openai/gemini）；为空返回全部模型
//
// 价格单位与 LiteLLM 原始数据一致（USD / token）。
func (h *ChannelHandler) GetPricingCatalog(c *gin.Context) {
	provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
	models, lastUpdated := h.pricingService.SnapshotModelPricing(provider)
	response.Success(c, gin.H{
		"models":       models,
		"model_count":  len(models),
		"last_updated": lastUpdated,
	})
}
