package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

type mockProvider struct{}

func NewMockProvider() PaymentProvider {
	return &mockProvider{}
}

func (m *mockProvider) CreateVA(ctx context.Context, invoiceID uuid.UUID, amount int64) (*VACreationResult, error) {
	// Format VA Bank PMB: 98800 + 8 digit acak
	randomSuffix := 10000000 + rand.Intn(90000000)
	vaNumber := fmt.Sprintf("98800%d", randomSuffix)

	providerRef := fmt.Sprintf("MOCK-VA-%s", uuid.New().String()[:8])
	expiresAt := time.Now().Add(24 * time.Hour)

	return &VACreationResult{
		ProviderRef: providerRef,
		VANumber:    vaNumber,
		ExpiresAt:   expiresAt,
	}, nil
}

func (m *mockProvider) ParseWebhook(ctx context.Context, body []byte) (*WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("payload webhook bukan format JSON yang valid: %w", err)
	}

	if payload.ProviderRef == "" {
		return nil, fmt.Errorf("provider_ref wajib ada pada payload webhook")
	}

	if payload.PaidAt.IsZero() {
		payload.PaidAt = time.Now()
	}

	return &payload, nil
}

func (m *mockProvider) VerifySignature(secret string, signatureHeader string, body []byte) bool {
	if secret == "" {
		return true // abaikan jika secret kosong dalam dev lokal
	}
	if signatureHeader == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Bersihkan prefix jika ada (misal "sha256=")
	cleanSig := strings.TrimPrefix(strings.TrimSpace(signatureHeader), "sha256=")
	return hmac.Equal([]byte(cleanSig), []byte(expectedSignature))
}
