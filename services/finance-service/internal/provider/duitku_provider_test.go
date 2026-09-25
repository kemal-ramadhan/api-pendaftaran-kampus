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
)

func TestDuitkuSignature_HeaderAndCallback(t *testing.T) {
	merchantCode := "D1234"
	apiKey := "test-api-key-12345"
	timestamp := "1700000000000"

	// 1. Header Signature Test
	headerSig := GenerateDuitkuHeaderSignature(merchantCode, timestamp, apiKey)
	if headerSig == "" {
		t.Fatal("Header signature tidak boleh kosong")
	}

	// 2. Callback Signature Test
	amount := "500000"
	merchantOrderID := "INV-20260925-001"
	callbackSig := GenerateDuitkuCallbackSignature(merchantCode, amount, merchantOrderID, apiKey)
	if callbackSig == "" {
		t.Fatal("Callback signature tidak boleh kosong")
	}

	// Verifikasi deterministik
	sameSig := GenerateDuitkuCallbackSignature(merchantCode, amount, merchantOrderID, apiKey)
	if callbackSig != sameSig {
		t.Fatalf("Signature harus deterministik: got %s vs %s", callbackSig, sameSig)
	}
}

func TestDuitkuProvider_CreatePayment_Httptest(t *testing.T) {
	expectedRef := "DUITKU-REF-998877"
	expectedURL := "https://sandbox.duitku.com/pop/pay/DUITKU-REF-998877"

	// Mock HTTP Server Duitku
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.Header.Get("x-duitku-merchantcode") != "TEST_MERCHANT" {
			t.Errorf("Unexpected merchant code header: %s", r.Header.Get("x-duitku-merchantcode"))
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		// Verifikasi customerVaName dipotong <= 20
		vaName, _ := body["customerVaName"].(string)
		if len(vaName) > 20 {
			t.Errorf("customerVaName harus dipotong maksimal 20 karakter, got length %d: %s", len(vaName), vaName)
		}

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
	if err != nil {
		t.Fatalf("NewDuitkuProvider gagal: %v", err)
	}

	req := &PaymentRequest{
		InvoiceID:       uuid.New(),
		MerchantOrderID: "ORDER-12345",
		Amount:          500000,
		ProductDetails:  "Biaya Pendaftaran PMB",
		CustomerName:    "Nama Calon Mahasiswa Yang Sangat Panjang Melebihi Dua Puluh Karakter",
		Email:           "calon@example.com",
	}

	res, err := p.CreatePayment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreatePayment gagal: %v", err)
	}

	if res.ProviderRef != expectedRef {
		t.Errorf("Expected ProviderRef %s, got %s", expectedRef, res.ProviderRef)
	}
	if res.PaymentURL != expectedURL {
		t.Errorf("Expected PaymentURL %s, got %s", expectedURL, res.PaymentURL)
	}
}

func TestDuitkuProvider_ParseWebhook_And_VerifySignature(t *testing.T) {
	merchantCode := "TEST_MERCHANT"
	apiKey := "test-key-secret"
	amount := "750000"
	orderID := "ORD-PMB-999"
	ref := "REF-DUITKU-123"

	validSignature := GenerateDuitkuCallbackSignature(merchantCode, amount, orderID, apiKey)

	cfg := DuitkuConfig{
		MerchantCode: merchantCode,
		APIKey:       apiKey,
	}
	p, _ := NewDuitkuProvider(cfg)

	// 1. Kasus Sukses
	form := url.Values{}
	form.Set("merchantCode", merchantCode)
	form.Set("amount", amount)
	form.Set("merchantOrderId", orderID)
	form.Set("resultCode", "00")
	form.Set("reference", ref)
	form.Set("signature", validSignature)
	form.Set("paymentCode", "014998877")

	body := []byte(form.Encode())

	payload, err := p.ParseWebhook(context.Background(), body)
	if err != nil {
		t.Fatalf("ParseWebhook valid gagal: %v", err)
	}
	if payload.Status != "success" {
		t.Errorf("Expected status success, got %s", payload.Status)
	}
	if payload.Amount != 750000 {
		t.Errorf("Expected amount 750000, got %d", payload.Amount)
	}

	// 2. Kasus Signature Salah
	formInvalidSig := url.Values{}
	formInvalidSig.Set("merchantCode", merchantCode)
	formInvalidSig.Set("amount", amount)
	formInvalidSig.Set("merchantOrderId", orderID)
	formInvalidSig.Set("resultCode", "00")
	formInvalidSig.Set("reference", ref)
	formInvalidSig.Set("signature", "invalid-fake-signature")

	_, err = p.ParseWebhook(context.Background(), []byte(formInvalidSig.Encode()))
	if err == nil {
		t.Fatal("ParseWebhook harus gagal saat signature tidak valid")
	}

	// 3. Kasus Merchant Code Mismatch
	formMismatchMerchant := url.Values{}
	formMismatchMerchant.Set("merchantCode", "WRONG_MERCHANT")
	formMismatchMerchant.Set("amount", amount)
	formMismatchMerchant.Set("merchantOrderId", orderID)
	formMismatchMerchant.Set("resultCode", "00")
	formMismatchMerchant.Set("reference", ref)
	formMismatchMerchant.Set("signature", validSignature)

	_, err = p.ParseWebhook(context.Background(), []byte(formMismatchMerchant.Encode()))
	if err == nil {
		t.Fatal("ParseWebhook harus gagal saat merchant code mismatch")
	}
}

func TestDuitkuProvider_CheckStatus_Httptest(t *testing.T) {
	orderID := "ORDER-CHECK-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "transactionStatus") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
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
	if err != nil {
		t.Fatalf("CheckStatus gagal: %v", err)
	}

	if status.StatusCode != "00" {
		t.Errorf("Expected status 00, got %s", status.StatusCode)
	}
	if status.Amount != 1500000 {
		t.Errorf("Expected amount 1500000, got %d", status.Amount)
	}
}
