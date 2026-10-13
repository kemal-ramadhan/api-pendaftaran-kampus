package client

import (
	"time"

	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/config"
)

// Clients menyimpan client eksternal yang dibutuhkan pmb-service
type Clients struct {
	Gateway internalclient.Client
	OCR     internalclient.Client
}

func NewClients(cfg *config.Config) *Clients {
	var gwClient internalclient.Client
	if cfg.GatewayInternalURL != "" {
		gwClient = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.GatewayInternalURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	var ocrClient internalclient.Client
	if cfg.OCRServiceURL != "" {
		ocrClient = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.OCRServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       30 * time.Second,
		})
	}

	return &Clients{
		Gateway: gwClient,
		OCR:     ocrClient,
	}
}
