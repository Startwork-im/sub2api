package admin

// Startwork: patch 定价目录查询接口测试。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupPricingCatalogRouter(pricingSvc *service.PricingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &ChannelHandler{pricingService: pricingSvc}
	router.GET("/channels/pricing/catalog", h.GetPricingCatalog)
	return router
}

type pricingCatalogResponse struct {
	Data struct {
		Models      map[string]service.LiteLLMModelPricing `json:"models"`
		ModelCount  int                                    `json:"model_count"`
		LastUpdated string                                 `json:"last_updated"`
	} `json:"data"`
}

func TestGetPricingCatalog_EmptyService(t *testing.T) {
	svc := service.NewPricingService(nil, nil)
	router := setupPricingCatalogRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/channels/pricing/catalog", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body pricingCatalogResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.Data.Models, "models must not be null")
	require.Empty(t, body.Data.Models)
	require.Zero(t, body.Data.ModelCount)
}

func TestGetPricingCatalog_ProviderFilter_EmptyService(t *testing.T) {
	svc := service.NewPricingService(nil, nil)
	router := setupPricingCatalogRouter(svc)

	for _, provider := range []string{"anthropic", "openai", "gemini", "Anthropic"} {
		req := httptest.NewRequest(http.MethodGet, "/channels/pricing/catalog?provider="+provider, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, "provider=%s", provider)

		var body pricingCatalogResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.NotNil(t, body.Data.Models, "models must not be null for provider=%s", provider)
	}
}
