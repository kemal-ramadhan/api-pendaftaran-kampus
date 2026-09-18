package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/config"
)

type HealthHandler struct {
	routesConfig *config.RoutesConfig
	httpClient   *http.Client
}

func NewHealthHandler(routesConfig *config.RoutesConfig) *HealthHandler {
	return &HealthHandler{
		routesConfig: routesConfig,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// LivenessCheck mengembalikan 200 OK jika proses API Gateway aktif
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, "API Gateway is alive", gin.H{
		"status":  "up",
		"service": "api-gateway",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// ReadinessCheck memeriksa ketersediaan seluruh upstream service hilir
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	servicesStatus := make(map[string]string)
	allReady := true

	for svcName, svcCfg := range h.routesConfig.Services {
		checkURL := svcCfg.URL + "/health"
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, checkURL, nil)
		if err != nil {
			servicesStatus[svcName] = "error"
			allReady = false
			continue
		}

		resp, err := h.httpClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			servicesStatus[svcName] = "down"
			allReady = false
		} else {
			servicesStatus[svcName] = "up"
			resp.Body.Close()
		}
	}

	if !allReady {
		c.JSON(http.StatusServiceUnavailable, response.ErrorResponse{
			Success:   false,
			Message:   "Sebagian atau seluruh layanan hilir belum siap",
			ErrorCode: response.ErrCodeUpstreamUnavailable,
		})
		return
	}

	response.Success(c, http.StatusOK, "Seluruh layanan siap melayani permintaan", gin.H{
		"status":   "ready",
		"services": servicesStatus,
	})
}
