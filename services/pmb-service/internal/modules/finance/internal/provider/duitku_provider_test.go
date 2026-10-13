package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuitkuSignature_HeaderAndCallback(t *testing.T) {
	merchantCode := "D1234"
	apiKey := "test-api-key-12345"
	timestamp := "1700000000000"

	// 1. Header Signature Test
	headerSig := GenerateDuitkuHeaderSignature(merchantCode, timestamp, apiKey)
	assert.NotEmpty(t, headerSig)

	// 2. Callback Signature Test (HMAC & MD5)
	amount := "500000"
	merchantOrderID := "INV-20260925-001"
	callbackHMAC := GenerateDuitkuCallbackSignature(merchantCode, amount, merchantOrderID, apiKey)
	assert.NotEmpty(t, callbackHMAC)

	callbackMD5 := GenerateDuitkuCallbackMD5Signature(merchantCode, amount, merchantOrderID, apiKey)
	assert.NotEmpty(t, callbackMD5)

	// Verifikasi deterministik
	sameSig := GenerateDuitkuCallbackSignature(merchantCode, amount, merchantOrderID, apiKey)
	assert.Equal(t, callbackHMAC, sameSig)
}

func TestDuitkuProvider_CreatePayment_Httptest(t *testing.T) {
	expectedRef := "DUITKU-REF-998877"
	expectedURL := "https://sandbox.duitku.com/pop/pay/DUITKU-REF-998877"

	// Mock HTTP Server Duitku
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "TEST_MERCHANT", r.Header.Get("x-duitku-merchantcode"))

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		// Verifikasi customerVaName dipotong <= 20
		vaName, _ := body["customerVaName"].(string)
		assert.LessOrEqual(t, len(vaName), 20)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"statusCode":    "00",
			"statusMessage": "SUCCESS",
			"reference":     expectedRef,
			"paymentUrl":    expectedURL,
			"vaNumber":      "98812345678",
			"paymentCode":   "98812345678",
			"amount":        "500000",
		})
	}))
	defer server.Close()

	cfg := DuitkuConfig{
		Env:             "sandbox",
		MerchantCode:    "TEST_MERCHANT",
		APIKey:          "test-key-secret",
		ExpiryMinutes:   60,
		BaseURLOverride: server.URL,
	}

	p, err := NewDuitkuProvider(cfg)
	require.NoError(t, err)

	req := &PaymentRequest{
		InvoiceID:       uuid.New(),
		MerchantOrderID: "ORDER-12345",
		Amount:          500000,
		ProductDetails:  "Biaya Pendaftaran PMB",
		CustomerName:    "Nama Calon Mahasiswa Yang Sangat Panjang Melebihi Dua Puluh Karakter",
		Email:           "calon@example.com",
	}

	res, err := p.CreatePayment(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, expectedRef, res.ProviderRef)
	assert.Equal(t, expectedURL, res.PaymentURL)
}

func TestDuitkuProvider_ParseWebhook_And_VerifySignature(t *testing.T) {
	merchantCode := "TEST_MERCHANT"
	apiKey := "test-key-secret"
	amount := "750000"
	orderID := "ORD-PMB-999"
	ref := "REF-DUITKU-123"

	validHMAC := GenerateDuitkuCallbackSignature(merchantCode, amount, orderID, apiKey)
	validMD5 := GenerateDuitkuCallbackMD5Signature(merchantCode, amount, orderID, apiKey)

	cfg := DuitkuConfig{
		MerchantCode: merchantCode,
		APIKey:       apiKey,
	}
	p, _ := NewDuitkuProvider(cfg)

	// 1. Kasus Sukses dengan Signature HMAC (POP)
	form := url.Values{}
	form.Set("merchantCode", merchantCode)
	form.Set("amount", amount)
	form.Set("merchantOrderId", orderID)
	form.Set("resultCode", "00")
	form.Set("reference", ref)
	form.Set("signature", validHMAC)
	form.Set("paymentCode", "014998877")

	payload, err := p.ParseWebhook(context.Background(), []byte(form.Encode()))
	require.NoError(t, err)
	assert.Equal(t, "success", payload.Status)
	assert.Equal(t, int64(750000), payload.Amount)

	// 2. Kasus Sukses dengan Signature MD5 (WebAPI)
	formMD5 := url.Values{}
	formMD5.Set("merchantCode", merchantCode)
	formMD5.Set("amount", amount)
	formMD5.Set("merchantOrderId", orderID)
	formMD5.Set("resultCode", "00")
	formMD5.Set("reference", ref)
	formMD5.Set("signature", validMD5)

	payloadMD5, err := p.ParseWebhook(context.Background(), []byte(formMD5.Encode()))
	require.NoError(t, err)
	assert.Equal(t, "success", payloadMD5.Status)

	// 3. Kasus Signature Salah
	formInvalidSig := url.Values{}
	formInvalidSig.Set("merchantCode", merchantCode)
	formInvalidSig.Set("amount", amount)
	formInvalidSig.Set("merchantOrderId", orderID)
	formInvalidSig.Set("resultCode", "00")
	formInvalidSig.Set("reference", ref)
	formInvalidSig.Set("signature", "invalid-fake-signature")

	_, err = p.ParseWebhook(context.Background(), []byte(formInvalidSig.Encode()))
	assert.Error(t, err)

	// 4. Kasus Merchant Code Mismatch
	formMismatchMerchant := url.Values{}
	formMismatchMerchant.Set("merchantCode", "WRONG_MERCHANT")
	formMismatchMerchant.Set("amount", amount)
	formMismatchMerchant.Set("merchantOrderId", orderID)
	formMismatchMerchant.Set("resultCode", "00")
	formMismatchMerchant.Set("reference", ref)
	formMismatchMerchant.Set("signature", validHMAC)

	_, err = p.ParseWebhook(context.Background(), []byte(formMismatchMerchant.Encode()))
	assert.Error(t, err)
}

func TestDuitkuProvider_CheckStatus_Httptest(t *testing.T) {
	orderID := "ORDER-CHECK-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.Contains(r.URL.Path, "transactionStatus"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"statusCode":      "00",
			"statusMessage":   "SUCCESS",
			"merchantOrderId": orderID,
			"reference":       "REF-STATUS-001",
			"amount":          "1500000",
		})
	}))
	defer server.Close()

	cfg := DuitkuConfig{
		MerchantCode:    "TEST_MERCHANT",
		APIKey:          "test-key-secret",
		BaseURLOverride: server.URL,
	}
	p, _ := NewDuitkuProvider(cfg)

	status, err := p.CheckStatus(context.Background(), orderID)
	require.NoError(t, err)
	assert.Equal(t, "00", status.StatusCode)
	assert.Equal(t, int64(1500000), status.Amount)
}

func TestDuitkuProvider_GetPaymentMethods_Httptest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.Contains(r.URL.Path, "paymentmethod/getpaymentmethod"))

		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "TEST_MERCHANT", req["merchantcode"])
		assert.Equal(t, "250000", req["amount"])
		assert.NotEmpty(t, req["datetime"])
		assert.NotEmpty(t, req["signature"])

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"paymentFee": []map[string]string{
				{
					"paymentMethod": "VC",
					"paymentName":   "BCA Virtual Account",
					"paymentImage":  "https://images.duitku.com/icon/BCA.png",
					"totalFee":      "4000",
				},
				{
					"paymentMethod": "VA",
					"paymentName":   "BNI Virtual Account",
					"paymentImage":  "https://images.duitku.com/icon/BNI.png",
					"totalFee":      "3000",
				},
			},
			"responseCode":    "00",
			"responseMessage": "SUCCESS",
		})
	}))
	defer server.Close()

	cfg := DuitkuConfig{
		MerchantCode:    "TEST_MERCHANT",
		APIKey:          "test-key-secret",
		BaseURLOverride: server.URL,
	}
	p, _ := NewDuitkuProvider(cfg)

	methods, err := p.GetPaymentMethods(context.Background(), 250000)
	require.NoError(t, err)
	assert.Len(t, methods, 2)
	assert.Equal(t, "VC", methods[0].PaymentMethod)
	assert.Equal(t, "BCA Virtual Account", methods[0].PaymentName)
}

func TestDuitkuProvider_ErrorMapping_502(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"statusCode":"99","statusMessage":"Internal Duitku Gateway Error"}`))
	}))
	defer server.Close()

	cfg := DuitkuConfig{
		MerchantCode:    "TEST_MERCHANT",
		APIKey:          "test-key-secret",
		BaseURLOverride: server.URL,
	}
	p, _ := NewDuitkuProvider(cfg)

	_, err := p.CreatePayment(context.Background(), &PaymentRequest{
		Amount: 100000,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PAYMENT_PROVIDER_ERROR")

	_, errMethods := p.GetPaymentMethods(context.Background(), 100000)
	assert.Error(t, errMethods)
	assert.Contains(t, errMethods.Error(), "PAYMENT_PROVIDER_ERROR")
}
