package jwks

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type Client interface {
	GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error)
	Refresh(ctx context.Context) error
	SetStaticKey(kid string, pubKey *rsa.PublicKey)
}

type client struct {
	jwksURL    string
	httpClient *http.Client
	logger     *zap.Logger

	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	lastRefresh time.Time
}

func NewClient(jwksURL string, logger *zap.Logger) Client {
	c := &client{
		jwksURL: jwksURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
		keys:   make(map[string]*rsa.PublicKey),
	}

	// Mulai refresh berkala tiap 10 menit
	go c.periodicRefresh()

	return c
}

func (c *client) SetStaticKey(kid string, pubKey *rsa.PublicKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys[kid] = pubKey
}

func (c *client) periodicRefresh() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := c.Refresh(ctx); err != nil {
			c.logger.Warn("Gagal refresh berkala JWKS", zap.Error(err))
		}
		cancel()
	}
}

func (c *client) Refresh(ctx context.Context) error {
	c.mu.Lock()
	// Throttle refresh jika baru saja diperbarui dalam 5 detik terakhir
	if time.Since(c.lastRefresh) < 5*time.Second && len(c.keys) > 0 {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal fetch JWKS dari %s: %w", c.jwksURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS mengembalikan status code %d", resp.StatusCode)
	}

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("gagal decode JWKS: %w", err)
	}

	parsedKeys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pubKey, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			c.logger.Warn("Gagal parse public key JWK", zap.String("kid", k.Kid), zap.Error(err))
			continue
		}
		parsedKeys[k.Kid] = pubKey
	}

	c.mu.Lock()
	c.keys = parsedKeys
	c.lastRefresh = time.Now()
	c.mu.Unlock()

	c.logger.Info("JWKS berhasil diperbarui", zap.Int("total_keys", len(parsedKeys)))
	return nil
}

func (c *client) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	key, ok := c.keys[kid]
	c.mu.RUnlock()

	if ok {
		return key, nil
	}

	// Jika kid tidak dikenal, coba refresh satu kali
	if err := c.Refresh(ctx); err != nil {
		c.logger.Warn("Gagal refresh JWKS saat kid tidak ditemukan", zap.String("kid", kid), zap.Error(err))
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if key, ok := c.keys[kid]; ok {
		return key, nil
	}

	return nil, errors.New("public key dengan kid tersebut tidak ditemukan di JWKS")
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("gagal decode modulus n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("gagal decode exponent e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	var e int
	for _, b := range eBytes {
		e = (e << 8) | int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}
