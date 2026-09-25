package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDuitkuConfigMissing  = errors.New("DUITKU_MERCHANT_CODE dan DUITKU_API_KEY wajib dikonfigurasi")
	ErrDuitkuCreateFailed   = errors.New("gagal membuat invoice di Duitku")
	ErrDuitkuInvalidWebhook = errors.New("payload webhook Duitku tidak valid")
	ErrInvalidSignature     = errors.New("signature webhook tidak valid")
)

type DuitkuConfig struct {
	Env            string // sandbox | production
	MerchantCode   string
	APIKey         string
	ExpiryMinutes  int
	CallbackURL    string
	ReturnURL      string
	HTTPTimeout    time.Duration
	BaseURLOverride string // untuk pengujian unit via httptest
}

type duitkuProvider struct {
	cfg        DuitkuConfig
	httpClient *http.Client
}

func NewDuitkuProvider(cfg DuitkuConfig) (PaymentProvider, error) {
	if cfg.MerchantCode == "" || cfg.APIKey == "" {
		return nil, ErrDuitkuConfigMissing
	}
	if cfg.ExpiryMinutes <= 0 {
		cfg.ExpiryMinutes = 1440 // default 24 jam
	}
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	return &duitkuProvider{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (d *duitkuProvider) getBaseURL() string {
	if d.cfg.BaseURLOverride != "" {
		return d.cfg.BaseURLOverride
	}
	if strings.ToLower(d.cfg.Env) == "production" {
		return "https://api-prod.duitku.com"
	}
	return "https://api-sandbox.duitku.com"
}

func (d *duitkuProvider) CreateVA(ctx context.Context, invoiceID uuid.UUID, amount int64) (*VACreationResult, error) {
	req := &PaymentRequest{
		InvoiceID:       invoiceID,
		MerchantOrderID: fmt.Sprintf("INV-%s-%d", invoiceID.String()[:8], time.Now().Unix()),
		Amount:          amount,
		ProductDetails:  "Biaya PMB Universitas",
		CustomerName:    "Calon Mahasiswa",
	}
	res, err := d.CreatePayment(ctx, req)
	if err != nil {
		return nil, err
	}
	return &VACreationResult{
		ProviderRef: res.ProviderRef,
		VANumber:    res.VANumber,
		ExpiresAt:   res.ExpiresAt,
	}, nil
}

func (d *duitkuProvider) CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResult, error) {
	baseURL := d.getBaseURL()
	endpoint := fmt.Sprintf("%s/api/merchant/createInvoice", baseURL)

	timestampMs := fmt.Sprintf("%d", time.Now().UnixMilli())
	signature := GenerateDuitkuHeaderSignature(d.cfg.MerchantCode, timestampMs, d.cfg.APIKey)

	// Potong customer name jika melebihi 20 karakter sesuai spesifikasi Duitku POP
	customerName := req.CustomerName
	if len(customerName) > 20 {
		customerName = customerName[:20]
	}
	if customerName == "" {
		customerName = "Pendaftar PMB"
	}

	merchantOrderID := req.MerchantOrderID
	if len(merchantOrderID) > 50 {
		merchantOrderID = merchantOrderID[:50]
	}

	bodyMap := map[string]any{
		"paymentAmount":   req.Amount,
		"merchantOrderId": merchantOrderID,
		"productDetails":  req.ProductDetails,
		"email":           req.Email,
		"phoneNumber":     req.PhoneNumber,
		"customerVaName":  customerName,
		"callbackUrl":     d.cfg.CallbackURL,
		"returnUrl":       d.cfg.ReturnURL,
		"expiryPeriod":    d.cfg.ExpiryMinutes,
	}
	if req.PaymentMethod != "" {
		bodyMap["paymentMethod"] = req.PaymentMethod
	}

	jsonBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("gagal serialize request body Duitku: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-duitku-merchantcode", d.cfg.MerchantCode)
	httpReq.Header.Set("x-duitku-timestamp", timestampMs)
	httpReq.Header.Set("x-duitku-signature", signature)

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("koneksi ke gateway Duitku gagal: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca respon Duitku: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d - %s", ErrDuitkuCreateFailed, resp.StatusCode, string(respBody))
	}

	var duitkuResp struct {
		StatusCode     string `json:"statusCode"`
		StatusMessage  string `json:"statusMessage"`
		Reference      string `json:"reference"`
		PaymentURL     string `json:"paymentUrl"`
		VANumber       string `json:"vaNumber"`
		PaymentCode    string `json:"paymentCode"`
		Amount         string `json:"amount"`
	}

	if err := json.Unmarshal(respBody, &duitkuResp); err != nil {
		return nil, fmt.Errorf("format respon Duitku tidak valid: %w", err)
	}

	if duitkuResp.StatusCode != "00" {
		return nil, fmt.Errorf("%w: kode %s - %s", ErrDuitkuCreateFailed, duitkuResp.StatusCode, duitkuResp.StatusMessage)
	}

	expiresAt := time.Now().Add(time.Duration(d.cfg.ExpiryMinutes) * time.Minute)

	return &PaymentResult{
		ProviderRef:      duitkuResp.Reference,
		PaymentURL:       duitkuResp.PaymentURL,
		VANumber:         duitkuResp.VANumber,
		PaymentCode:      duitkuResp.PaymentCode,
		PublisherOrderID: merchantOrderID,
		Amount:           req.Amount,
		ExpiresAt:        expiresAt,
	}, nil
}

func (d *duitkuProvider) ParseWebhook(ctx context.Context, body []byte) (*WebhookPayload, error) {
	// Duitku callback format is application/x-www-form-urlencoded
	values, err := url.ParseQuery(string(body))
	if err != nil {
		// Fallback parse json jika dikirim sebagai json
		var jsonMap map[string]string
		if jsonErr := json.Unmarshal(body, &jsonMap); jsonErr == nil {
			values = url.Values{}
			for k, v := range jsonMap {
				values.Set(k, v)
			}
		} else {
			return nil, fmt.Errorf("%w: gagal parse form urlencoded", ErrDuitkuInvalidWebhook)
		}
	}

	merchantCode := values.Get("merchantCode")
	amountStr := values.Get("amount")
	merchantOrderID := values.Get("merchantOrderId")
	resultCode := values.Get("resultCode")
	reference := values.Get("reference")
	signature := values.Get("signature")
	paymentCode := values.Get("paymentCode")
	publisherOrderID := values.Get("publisherOrderId")

	if merchantCode == "" || merchantOrderID == "" || reference == "" {
		return nil, fmt.Errorf("%w: parameter wajib hilang (merchantCode, merchantOrderId, reference)", ErrDuitkuInvalidWebhook)
	}

	// Verifikasi kecocokan merchant code
	if merchantCode != d.cfg.MerchantCode {
		return nil, fmt.Errorf("%w: merchant code tidak cocok", ErrDuitkuInvalidWebhook)
	}

	// Verifikasi signature callback
	if !d.VerifySignature(d.cfg.APIKey, signature, body) {
		return nil, ErrInvalidSignature
	}

	amount, _ := strconv.ParseInt(amountStr, 10, 64)
	status := "failed"
	if resultCode == "00" {
		status = "success"
	}

	return &WebhookPayload{
		Provider:         "duitku",
		ProviderRef:      reference,
		VANumber:         paymentCode,
		Amount:           amount,
		Status:           status,
		PaymentCode:      paymentCode,
		PublisherOrderID: publisherOrderID,
		PaidAt:           time.Now(),
	}, nil
}

func (d *duitkuProvider) VerifySignature(secret string, signatureHeader string, body []byte) bool {
	apiKey := secret
	if apiKey == "" {
		apiKey = d.cfg.APIKey
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		var jsonMap map[string]string
		if jsonErr := json.Unmarshal(body, &jsonMap); jsonErr == nil {
			values = url.Values{}
			for k, v := range jsonMap {
				values.Set(k, v)
			}
		} else {
			return false
		}
	}

	merchantCode := values.Get("merchantCode")
	amount := values.Get("amount")
	merchantOrderID := values.Get("merchantOrderId")
	incomingSig := values.Get("signature")
	if incomingSig == "" {
		incomingSig = signatureHeader
	}

	if merchantCode == "" || amount == "" || merchantOrderID == "" || incomingSig == "" {
		return false
	}

	expectedSig := GenerateDuitkuCallbackSignature(merchantCode, amount, merchantOrderID, apiKey)

	return subtle.ConstantTimeCompare([]byte(strings.ToLower(incomingSig)), []byte(strings.ToLower(expectedSig))) == 1
}

func (d *duitkuProvider) CheckStatus(ctx context.Context, merchantOrderID string) (*PaymentStatusResult, error) {
	baseURL := d.getBaseURL()
	endpoint := fmt.Sprintf("%s/api/merchant/transactionStatus", baseURL)

	timestampMs := fmt.Sprintf("%d", time.Now().UnixMilli())
	headerSig := GenerateDuitkuHeaderSignature(d.cfg.MerchantCode, timestampMs, d.cfg.APIKey)

	// Signature body untuk status check: MD5/SHA256(merchantCode + merchantOrderId + apiKey)
	hash := sha256.New()
	hash.Write([]byte(d.cfg.MerchantCode + merchantOrderID + d.cfg.APIKey))
	bodySig := hex.EncodeToString(hash.Sum(nil))

	reqBody := map[string]string{
		"merchantCode":    d.cfg.MerchantCode,
		"merchantOrderId": merchantOrderID,
		"signature":       bodySig,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-duitku-merchantcode", d.cfg.MerchantCode)
	httpReq.Header.Set("x-duitku-timestamp", timestampMs)
	httpReq.Header.Set("x-duitku-signature", headerSig)

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gagal periksa status transaksi Duitku: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var statusResp struct {
		MerchantOrderID string `json:"merchantOrderId"`
		Reference       string `json:"reference"`
		Amount          string `json:"amount"`
		StatusCode      string `json:"statusCode"`
		StatusMessage   string `json:"statusMessage"`
	}

	if err := json.Unmarshal(respBytes, &statusResp); err != nil {
		return nil, fmt.Errorf("format respon periksa status Duitku tidak valid: %w", err)
	}

	amount, _ := strconv.ParseInt(statusResp.Amount, 10, 64)

	return &PaymentStatusResult{
		MerchantOrderID: statusResp.MerchantOrderID,
		StatusCode:      statusResp.StatusCode,
		StatusMessage:   statusResp.StatusMessage,
		Amount:          amount,
		Reference:       statusResp.Reference,
		PaidAt:          time.Now(),
	}, nil
}

// Helper kalkulasi signature HMAC Duitku
func GenerateDuitkuHeaderSignature(merchantCode, timestampMs, apiKey string) string {
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(merchantCode + timestampMs))
	return hex.EncodeToString(mac.Sum(nil))
}

func GenerateDuitkuCallbackSignature(merchantCode, amount, merchantOrderID, apiKey string) string {
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(merchantCode + amount + merchantOrderID))
	return hex.EncodeToString(mac.Sum(nil))
}
