package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/config"
	"go.uber.org/zap"
)

type ProxyHandler struct {
	routesConfig *config.RoutesConfig
	logger       *zap.Logger
	proxies      map[string]*httputil.ReverseProxy
}

func NewProxyHandler(routesConfig *config.RoutesConfig, logger *zap.Logger) (*ProxyHandler, error) {
	ph := &ProxyHandler{
		routesConfig: routesConfig,
		logger:       logger,
		proxies:      make(map[string]*httputil.ReverseProxy),
	}

	for svcName, svcCfg := range routesConfig.Services {
		targetURL, err := url.Parse(svcCfg.URL)
		if err != nil {
			return nil, err
		}

		timeout := svcCfg.Timeout
		if timeout == 0 {
			timeout = 10 * time.Second
		}

		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: timeout,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
		}

		revProxy := httputil.NewSingleHostReverseProxy(targetURL)
		revProxy.Transport = transport

		revProxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
			logger.Warn("Kesalahan reverse proxy upstream",
				zap.String("service", svcName),
				zap.String("path", req.URL.Path),
				zap.Error(err),
			)

			w.Header().Set("Content-Type", "application/json")

			var netErr net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
				w.WriteHeader(http.StatusGatewayTimeout)
				_ = response.WriteJSON(w, http.StatusGatewayTimeout, response.ErrorResponse{
					Success:   false,
					Message:   "Layanan hilir melebihi batas waktu (upstream timeout)",
					ErrorCode: response.ErrCodeUpstreamTimeout,
					RequestID: req.Header.Get("X-Request-ID"),
				})
				return
			}

			w.WriteHeader(http.StatusBadGateway)
			_ = response.WriteJSON(w, http.StatusBadGateway, response.ErrorResponse{
				Success:   false,
				Message:   "Layanan hilir sedang tidak dapat dihubungi (bad gateway)",
				ErrorCode: response.ErrCodeUpstreamUnavailable,
				RequestID: req.Header.Get("X-Request-ID"),
			})
		}

		ph.proxies[svcName] = revProxy
	}

	return ph, nil
}

// ForwardRequest meneruskan request Gin ke upstream proxy yang sesuai
func (ph *ProxyHandler) Forward(c *gin.Context, matchedRoute *config.RouteConfig) {
	proxy, ok := ph.proxies[matchedRoute.Service]
	if !ok {
		response.Error(c, http.StatusBadGateway, response.ErrCodeUpstreamUnavailable, "Layanan upstream tidak terkonfigurasi", nil)
		return
	}

	svcCfg := ph.routesConfig.Services[matchedRoute.Service]
	targetURL, _ := url.Parse(svcCfg.URL)

	req := c.Request

	// Rewrite path: buang prefix route dan ganti dengan rewrite
	originalPath := req.URL.Path
	remainder := strings.TrimPrefix(originalPath, matchedRoute.Prefix)
	if !strings.HasPrefix(remainder, "/") && remainder != "" {
		remainder = "/" + remainder
	}

	newPath := matchedRoute.Rewrite + remainder
	if !strings.HasPrefix(newPath, "/") {
		newPath = "/" + newPath
	}

	req.URL.Scheme = targetURL.Scheme
	req.URL.Host = targetURL.Host
	req.URL.Path = newPath
	req.Host = targetURL.Host

	proxy.ServeHTTP(c.Writer, req)
}
